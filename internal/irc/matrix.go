// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"golang.org/x/net/proxy"
)

var (
	matrixConfigDirMu sync.RWMutex
	matrixConfigDir   string
)

// SetMatrixConfigDir overrides the storage directory for Matrix tokens and pending auth files.
func SetMatrixConfigDir(dir string) {
	matrixConfigDirMu.Lock()
	defer matrixConfigDirMu.Unlock()
	matrixConfigDir = dir
}

func getMatrixConfigDir() string {
	matrixConfigDirMu.RLock()
	dir := matrixConfigDir
	matrixConfigDirMu.RUnlock()
	if dir != "" {
		return dir
	}
	if env := os.Getenv("AUTOBRR__CONFIG_PATH"); env != "" {
		return env
	}
	if fi, err := os.Stat("/config"); err == nil && fi.IsDir() {
		return "/config"
	}
	if userConfigDir, err := os.UserConfigDir(); err == nil {
		d := filepath.Join(userConfigDir, "autobrr")
		_ = os.MkdirAll(d, 0755)
		return d
	}
	return "."
}

func getMatrixTokenStoreFile() string {
	return filepath.Join(getMatrixConfigDir(), "matrix_tokens.json")
}

func getMatrixDevicePendingFile(serverHost, nick string) string {
	safeServer := strings.ReplaceAll(serverHost, ":", "_")
	safeServer = strings.ReplaceAll(safeServer, "/", "_")
	safeNick := strings.ReplaceAll(nick, ":", "_")
	safeNick = strings.ReplaceAll(safeNick, "/", "_")
	if safeNick != "" {
		return filepath.Join(getMatrixConfigDir(), fmt.Sprintf("matrix_device_pending_%s_%s.json", safeServer, safeNick))
	}
	return filepath.Join(getMatrixConfigDir(), fmt.Sprintf("matrix_device_pending_%s.json", safeServer))
}

type matrixStoredCreds struct {
	Server       string    `json:"server,omitempty"`
	ClientID     string    `json:"client_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	SourcePass   string    `json:"source_pass,omitempty"`
	TokenURI     string    `json:"token_uri"`
	DeviceURI    string    `json:"device_uri"`
	ExpiresAt    time.Time `json:"expires_at"`
}

var matrixStoreMu sync.Mutex

func accountKey(server, nick string) string {
	s := strings.ToLower(strings.TrimSpace(server))
	n := strings.ToLower(strings.TrimSpace(nick))
	if n != "" {
		return s + "/" + n
	}
	return s
}

func loadMatrixCreds(server, nick string) (*matrixStoredCreds, error) {
	matrixStoreMu.Lock()
	defer matrixStoreMu.Unlock()

	data, err := os.ReadFile(getMatrixTokenStoreFile())
	if err != nil {
		return nil, err
	}
	var all map[string]*matrixStoredCreds
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, errors.Wrap(err, "failed to decode matrix credentials file")
	}

	if nick != "" {
		if creds, ok := all[accountKey(server, nick)]; ok {
			return creds, nil
		}
	}
	creds, ok := all[strings.ToLower(strings.TrimSpace(server))]
	if !ok {
		return nil, os.ErrNotExist
	}
	return creds, nil
}

func saveMatrixCreds(server, nick string, creds *matrixStoredCreds) error {
	matrixStoreMu.Lock()
	defer matrixStoreMu.Unlock()

	all := make(map[string]*matrixStoredCreds)
	storeFile := getMatrixTokenStoreFile()
	if data, err := os.ReadFile(storeFile); err == nil {
		_ = json.Unmarshal(data, &all)
	}

	creds.Server = server
	key := accountKey(server, nick)
	all[key] = creds

	out, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal matrix credentials")
	}
	return os.WriteFile(storeFile, out, 0600)
}

type matrixDevicePending struct {
	ClientID        string    `json:"client_id"`
	DeviceCode      string    `json:"device_code"`
	UserCode        string    `json:"user_code"`
	VerificationURI string    `json:"verification_uri"`
	CompleteURI     string    `json:"complete_uri"`
	Interval        int       `json:"interval"`
	ExpiresAt       time.Time `json:"expires_at"`
}

func saveMatrixPendingAuth(serverHost, nick string, pending *matrixDevicePending) error {
	out, err := json.MarshalIndent(pending, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getMatrixDevicePendingFile(serverHost, nick), out, 0600)
}

func clearMatrixPendingAuth(serverHost, nick string) error {
	return os.Remove(getMatrixDevicePendingFile(serverHost, nick))
}

func isMatrixNetwork(server string) bool {
	s := strings.ToLower(strings.TrimSpace(server))
	return strings.HasPrefix(s, "matrix.") ||
		strings.HasPrefix(s, "matrix://") ||
		strings.Contains(s, ".matrix.") ||
		strings.HasSuffix(s, ".watchtower.pm")
}

func (h *Handler) isMatrix() bool {
	netw := h.GetNetwork()
	if netw == nil {
		return false
	}
	return isMatrixNetwork(netw.Server)
}

func (h *Handler) dialMatrixContext(_ context.Context, _, _ string) (net.Conn, error) {
	clientConn, serverConn := net.Pipe()
	netw := h.GetNetwork()
	go h.serveMatrixPipe(serverConn, netw)
	return clientConn, nil
}

type matrixSyncResponse struct {
	NextBatch string `json:"next_batch"`
	Rooms     struct {
		Join map[string]struct {
			Timeline struct {
				Events []struct {
					Type           string `json:"type"`
					Sender         string `json:"sender"`
					EventID        string `json:"event_id"`
					OriginServerTS int64  `json:"origin_server_ts"`
					Content        struct {
						MsgType string `json:"msgtype"`
						Body    string `json:"body"`
					} `json:"content"`
				} `json:"events"`
			} `json:"timeline"`
		} `json:"join"`
	} `json:"rooms"`
}

func buildMatrixHTTPClient(network *domain.IrcNetwork) (*http.Client, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: network.TLSSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	if network.UseProxy {
		if network.Proxy == nil {
			return nil, errors.New("use_proxy enabled for matrix network '%s', but no proxy attached", network.Server)
		}

		proxyURL, err := url.Parse(network.Proxy.Addr)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse proxy url: %s", network.Proxy.Addr)
		}

		if network.Proxy.User != "" && network.Proxy.Pass != "" {
			proxyURL.User = url.UserPassword(network.Proxy.User, network.Proxy.Pass)
		}

		switch proxyURL.Scheme {
		case "http", "https":
			transport.Proxy = http.ProxyURL(proxyURL)
		default:
			dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
			if err != nil {
				return nil, errors.Wrap(err, "could not create proxy dialer from url: %s", network.Proxy.Addr)
			}
			if cd, ok := dialer.(proxy.ContextDialer); ok {
				transport.DialContext = cd.DialContext
			} else {
				transport.DialContext = func(ctx context.Context, netw, addr string) (net.Conn, error) {
					return dialer.Dial(netw, addr)
				}
			}
		}
	}

	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
	}, nil
}

func (h *Handler) serveMatrixPipe(conn net.Conn, network *domain.IrcNetwork) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		_ = conn.Close()
	}()

	httpClient, err := buildMatrixHTTPClient(network)
	if err != nil {
		h.log.Error().Err(err).Msg("matrix: failed to configure HTTP client")
		h.addConnectError(err.Error())
		h.broadcastHealth()
		return
	}

	serverHost := strings.TrimSpace(network.Server)
	serverHost = strings.TrimPrefix(serverHost, "matrix://")
	serverHost = strings.TrimPrefix(serverHost, "https://")
	serverHost = strings.TrimPrefix(serverHost, "http://")
	serverHost = strings.TrimSuffix(serverHost, "/")

	scheme := "https"
	if !network.TLS {
		scheme = "http"
	}
	host := serverHost
	if network.Port > 0 && !strings.Contains(serverHost, ":") {
		if (scheme == "https" && network.Port != 443) || (scheme == "http" && network.Port != 80) {
			host = fmt.Sprintf("%s:%d", serverHost, network.Port)
		}
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, host)

	var writeMu sync.Mutex
	sendIRC := func(line string) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
		_, err := io.WriteString(conn, line+"\r\n")
		return err
	}

	rawPass := strings.TrimSpace(network.Pass)
	if rawPass == "" {
		rawPass = strings.TrimSpace(network.Auth.Password)
	}

	var (
		clientNick = network.Nick
		nickGot    = false
		userGot    = false
		welcomed   = false
		roomMu     sync.RWMutex
		roomToChan = make(map[string]string)
	)
	if clientNick == "" {
		clientNick = "autobrr"
	}

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, " ")
		cmd := strings.ToUpper(parts[0])

		switch cmd {
		case "NICK":
			if len(parts) > 1 {
				clientNick = parts[1]
			}
			nickGot = true
		case "USER":
			userGot = true
		case "PING":
			arg := serverHost
			if len(parts) > 1 {
				arg = strings.TrimPrefix(parts[1], ":")
			}
			_ = sendIRC(fmt.Sprintf(":%s PONG %s :%s", serverHost, serverHost, arg))
		case "QUIT":
			return
		case "PART":
			if len(parts) >= 2 {
				chanList := strings.Split(parts[1], ",")
				for _, chName := range chanList {
					chName = strings.TrimSpace(chName)
					if chName == "" {
						continue
					}
					target := strings.ToLower(chName)
					roomMu.Lock()
					var toDelete []string
					for rID, ch := range roomToChan {
						if strings.EqualFold(ch, target) {
							toDelete = append(toDelete, rID)
						}
					}
					for _, rID := range toDelete {
						delete(roomToChan, rID)
					}
					roomMu.Unlock()
					_ = sendIRC(fmt.Sprintf(":%s!matrix@%s PART %s", clientNick, serverHost, chName))
				}
			}
		case "JOIN":
			if len(parts) < 2 {
				continue
			}
			chanList := strings.Split(parts[1], ",")
			for _, chName := range chanList {
				chName = strings.TrimSpace(chName)
				if chName == "" {
					continue
				}
				go func(chName string) {
					tok, err := h.ensureMatrixAccessToken(ctx, httpClient, serverHost, clientNick, baseURL, rawPass)
					if err != nil {
						h.log.Error().Err(err).Str("channel", chName).Msg("matrix: auth failed before room join")
						return
					}
					roomID, rErr := resolveMatrixRoom(ctx, httpClient, baseURL, serverHost, tok, chName)
					if rErr != nil {
						h.log.Error().Err(rErr).Str("channel", chName).Msg("matrix: failed to resolve room")
						_ = sendIRC(fmt.Sprintf(":%s 473 %s %s :Cannot join room (%s)", serverHost, clientNick, chName, rErr.Error()))
						return
					}
					roomMu.Lock()
					roomToChan[roomID] = strings.ToLower(chName)
					roomMu.Unlock()

					h.log.Info().Str("channel", chName).Str("room_id", roomID).Msg("matrix: monitoring room")
					_ = sendIRC(fmt.Sprintf(":%s!matrix@%s JOIN %s", clientNick, serverHost, chName))
					_ = sendIRC(fmt.Sprintf(":%s 332 %s %s :Matrix Room %s", serverHost, clientNick, chName, roomID))
					_ = sendIRC(fmt.Sprintf(":%s 353 %s = %s :%s @blubot", serverHost, clientNick, chName, clientNick))
					_ = sendIRC(fmt.Sprintf(":%s 366 %s %s :End of /NAMES list.", serverHost, clientNick, chName))
				}(chName)
			}
		}

		if nickGot && userGot && !welcomed {
			welcomed = true
			_ = sendIRC(fmt.Sprintf(":%s 001 %s :Welcome to the Matrix-IRC Native Gateway %s", serverHost, clientNick, clientNick))
			_ = sendIRC(fmt.Sprintf(":%s 002 %s :Your host is %s, running autobrr-matrix-native", serverHost, clientNick, serverHost))
			_ = sendIRC(fmt.Sprintf(":%s 376 %s :End of /MOTD command.", serverHost, clientNick))
			_ = sendIRC(fmt.Sprintf(":%s!matrix@%s MODE %s :+r", clientNick, serverHost, clientNick))

			go h.matrixSyncLoop(ctx, httpClient, serverHost, clientNick, baseURL, rawPass, &roomMu, roomToChan, sendIRC)
		}
	}
}

var matrixAuthMu sync.Mutex

func (h *Handler) ensureMatrixAccessToken(ctx context.Context, client *http.Client, serverHost, clientNick, baseURL, rawPass string) (string, error) {
	matrixAuthMu.Lock()
	defer matrixAuthMu.Unlock()

	creds, _ := loadMatrixCreds(serverHost, clientNick)
	if creds == nil {
		creds = &matrixStoredCreds{
			Server: serverHost,
		}
		if serverHost == "matrix.watchtower.pm" || strings.HasSuffix(serverHost, ".watchtower.pm") {
			creds.ClientID = "01M514EYEAN1WBJ9QCJZG448J0"
			creds.TokenURI = "https://auth.watchtower.pm/oauth2/token"
			creds.DeviceURI = "https://auth.watchtower.pm/oauth2/device"
		}
	}

	// Update credentials when user inputs or changes password in settings
	if rawPass != "" && creds.SourcePass != rawPass {
		creds.SourcePass = rawPass
		if strings.HasPrefix(rawPass, "mrt_") || strings.HasPrefix(rawPass, "mar_") {
			creds.RefreshToken = rawPass
			creds.AccessToken = ""
			creds.ExpiresAt = time.Time{}
		} else if strings.Contains(rawPass, "|mrt_") || strings.Contains(rawPass, "|mar_") {
			sp := strings.SplitN(rawPass, "|", 2)
			creds.ClientID = sp[0]
			creds.RefreshToken = sp[1]
			creds.AccessToken = ""
			creds.ExpiresAt = time.Time{}
		} else if strings.HasPrefix(rawPass, "mat_") || strings.HasPrefix(rawPass, "syt_") || strings.HasPrefix(rawPass, "mct_") {
			creds.AccessToken = rawPass
			creds.RefreshToken = ""
			creds.ExpiresAt = time.Now().Add(24 * 365 * time.Hour)
		}
	}

	// Check if active access token is provided directly
	if strings.HasPrefix(rawPass, "mat_") || strings.HasPrefix(rawPass, "syt_") || strings.HasPrefix(rawPass, "mct_") {
		if checkMatrixTokenActive(ctx, client, baseURL, rawPass) {
			creds.AccessToken = rawPass
			if err := saveMatrixCreds(serverHost, clientNick, creds); err != nil {
				h.log.Warn().Err(err).Msg("matrix: failed to persist credentials")
			}
			return rawPass, nil
		}
	}

	// If stored access token is still valid, verify it
	if creds.AccessToken != "" && time.Now().Before(creds.ExpiresAt) {
		if checkMatrixTokenActive(ctx, client, baseURL, creds.AccessToken) {
			return creds.AccessToken, nil
		}
	}

	// If we have a refresh token, refresh via OAuth2
	if creds.RefreshToken != "" {
		newToken, err := refreshMatrixOAuthToken(ctx, client, creds)
		if err == nil {
			creds.AccessToken = newToken
			creds.ExpiresAt = time.Now().Add(4 * time.Minute)
			if sErr := saveMatrixCreds(serverHost, clientNick, creds); sErr != nil {
				h.log.Warn().Err(sErr).Msg("matrix: failed to persist refreshed credentials")
			}
			return newToken, nil
		}
		h.log.Warn().Err(err).Msg("matrix: refresh token failed, falling back to device auth")
	}

	// If not Watchtower and no DeviceURI configured, fail gracefully
	if creds.DeviceURI == "" {
		if serverHost == "matrix.watchtower.pm" || strings.HasSuffix(serverHost, ".watchtower.pm") {
			creds.ClientID = "01M514EYEAN1WBJ9QCJZG448J0"
			creds.TokenURI = "https://auth.watchtower.pm/oauth2/token"
			creds.DeviceURI = "https://auth.watchtower.pm/oauth2/device"
		} else {
			return "", errors.New("matrix: authentication failed for %s, please provide a valid access token in settings", serverHost)
		}
	}

	pending, err := startMatrixDeviceAuth(ctx, client, creds)
	if err != nil {
		return "", errors.Wrap(err, "matrix: failed to start device auth")
	}

	_ = saveMatrixPendingAuth(serverHost, clientNick, pending)

	msg := fmt.Sprintf("Matrix OAuth2: approve access at %s (code: %s)", pending.CompleteURI, pending.UserCode)
	h.log.Warn().Str("url", pending.CompleteURI).Str("code", pending.UserCode).Msg("matrix: awaiting user device approval")
	h.addConnectError(msg)
	h.broadcastHealth()

	tokenResp, err := waitForMatrixDeviceAuth(ctx, client, creds, pending)
	if err != nil {
		return "", errors.Wrap(err, "matrix: device authorization failed or timed out")
	}

	_ = clearMatrixPendingAuth(serverHost, clientNick)
	creds.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		creds.RefreshToken = tokenResp.RefreshToken
	}
	creds.ExpiresAt = time.Now().Add(4 * time.Minute)
	if sErr := saveMatrixCreds(serverHost, clientNick, creds); sErr != nil {
		h.log.Warn().Err(sErr).Msg("matrix: failed to persist authorized credentials")
	}

	return creds.AccessToken, nil
}

func checkMatrixTokenActive(ctx context.Context, client *http.Client, baseURL, token string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/_matrix/client/v3/account/whoami", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

type matrixDeviceAuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func startMatrixDeviceAuth(ctx context.Context, client *http.Client, creds *matrixStoredCreds) (*matrixDevicePending, error) {
	deviceURI := creds.DeviceURI
	if deviceURI == "" {
		if creds.Server == "matrix.watchtower.pm" || strings.HasSuffix(creds.Server, ".watchtower.pm") {
			deviceURI = "https://auth.watchtower.pm/oauth2/device"
		}
	}
	clientID := creds.ClientID
	if clientID == "" {
		if creds.Server == "matrix.watchtower.pm" || strings.HasSuffix(creds.Server, ".watchtower.pm") {
			clientID = "01M514EYEAN1WBJ9QCJZG448J0"
		}
	}
	if deviceURI == "" {
		return nil, errors.New("matrix: no device endpoint available for oauth")
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", "urn:matrix:org.matrix.msc2967.client:api:*")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deviceURI, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create device auth request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute device auth request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("device auth HTTP %d", resp.StatusCode)
	}

	var dRes struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dRes); err != nil {
		return nil, errors.Wrap(err, "failed to decode device auth response")
	}

	pending := &matrixDevicePending{
		ClientID:        clientID,
		DeviceCode:      dRes.DeviceCode,
		UserCode:        dRes.UserCode,
		VerificationURI: dRes.VerificationURI,
		CompleteURI:     fmt.Sprintf("%s?code=%s", dRes.VerificationURI, dRes.UserCode),
		Interval:        dRes.Interval,
		ExpiresAt:       time.Now().Add(time.Duration(dRes.ExpiresIn) * time.Second),
	}
	return pending, nil
}

func waitForMatrixDeviceAuth(ctx context.Context, client *http.Client, creds *matrixStoredCreds, pending *matrixDevicePending) (*matrixDeviceAuthResponse, error) {
	interval := pending.Interval
	if interval < 5 {
		interval = 5
	}

	tokenURI := creds.TokenURI
	if tokenURI == "" {
		if creds.Server == "matrix.watchtower.pm" || strings.HasSuffix(creds.Server, ".watchtower.pm") {
			tokenURI = "https://auth.watchtower.pm/oauth2/token"
		}
	}
	if tokenURI == "" {
		return nil, errors.New("matrix: no token endpoint available for oauth")
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(pending.ExpiresAt) {
				return nil, errors.New("matrix device code expired")
			}

			form := url.Values{}
			form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
			form.Set("client_id", pending.ClientID)
			form.Set("device_code", pending.DeviceCode)

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURI, strings.NewReader(form.Encode()))
			if err != nil {
				continue
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, err := client.Do(req)
			if err != nil {
				continue
			}

			if resp.StatusCode == http.StatusOK {
				var tokenResp matrixDeviceAuthResponse
				if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err == nil {
					resp.Body.Close()
					return &tokenResp, nil
				}
			}
			resp.Body.Close()
		}
	}
}

func refreshMatrixOAuthToken(ctx context.Context, client *http.Client, creds *matrixStoredCreds) (string, error) {
	tokenURI := creds.TokenURI
	if tokenURI == "" {
		if creds.Server == "matrix.watchtower.pm" || strings.HasSuffix(creds.Server, ".watchtower.pm") {
			tokenURI = "https://auth.watchtower.pm/oauth2/token"
		}
	}
	clientID := creds.ClientID
	if clientID == "" {
		if creds.Server == "matrix.watchtower.pm" || strings.HasSuffix(creds.Server, ".watchtower.pm") {
			clientID = "01M514EYEAN1WBJ9QCJZG448J0"
		}
	}
	if tokenURI == "" {
		return "", errors.New("matrix: no token endpoint available for oauth refresh")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", clientID)
	form.Set("refresh_token", creds.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", errors.Wrap(err, "failed to create oauth2 refresh request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to execute oauth2 refresh request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", errors.New("oauth2 refresh status %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", errors.Wrap(err, "failed to decode oauth2 refresh response")
	}
	creds.AccessToken = res.AccessToken
	if res.RefreshToken != "" {
		creds.RefreshToken = res.RefreshToken
	}
	if res.ExpiresIn > 60 {
		creds.ExpiresAt = time.Now().Add(time.Duration(res.ExpiresIn-60) * time.Second)
	} else {
		creds.ExpiresAt = time.Now().Add(4 * time.Minute)
	}
	return creds.AccessToken, nil
}

func resolveMatrixRoom(ctx context.Context, client *http.Client, baseURL, serverHost, token, chName string) (string, error) {
	if strings.HasPrefix(chName, "!") {
		return chName, nil
	}
	alias := chName
	if !strings.Contains(alias, ":") {
		alias = alias + ":" + serverHost
	}
	if strings.EqualFold(alias, "#blu-announce:matrix.watchtower.pm") {
		return "!xbWdAiMmRUgepJMvfV:matrix.watchtower.pm", nil
	}

	dirURL := fmt.Sprintf("%s/_matrix/client/v3/directory/room/%s", baseURL, url.PathEscape(alias))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dirURL, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create directory lookup request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to execute directory lookup request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("directory lookup HTTP %d", resp.StatusCode)
	}
	var out struct {
		RoomID string `json:"room_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.RoomID == "" {
		return "", errors.New("failed to resolve room id for alias %s", alias)
	}
	return out.RoomID, nil
}

func (h *Handler) matrixSyncLoop(
	ctx context.Context,
	client *http.Client,
	serverHost, clientNick, baseURL, rawPass string,
	roomMu *sync.RWMutex,
	roomToChan map[string]string,
	sendIRC func(string) error,
) {
	var (
		nextBatch   string
		activeToken string
		tokenExpiry time.Time
	)

	// Bounded FIFO ring buffer for seen event IDs to prevent unbounded memory growth
	const maxSeenEvents = 200
	seenEvents := make(map[string]struct{}, maxSeenEvents)
	seenOrder := make([]string, 0, maxSeenEvents)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Re-acquire token only if expired or not yet obtained
		if activeToken == "" || (!tokenExpiry.IsZero() && time.Now().After(tokenExpiry)) {
			tok, err := h.ensureMatrixAccessToken(ctx, client, serverHost, clientNick, baseURL, rawPass)
			if err != nil {
				h.log.Error().Err(err).Msg("matrix sync: token refresh failed, retrying in 15s")
				select {
				case <-ctx.Done():
					return
				case <-time.After(15 * time.Second):
					continue
				}
			}
			activeToken = tok
			tokenExpiry = time.Now().Add(4 * time.Minute)
		}

		syncURL := fmt.Sprintf("%s/_matrix/client/v3/sync?timeout=25000", baseURL)
		if nextBatch == "" {
			filter := `{"room":{"timeline":{"limit":1}}}`
			syncURL = fmt.Sprintf("%s/_matrix/client/v3/sync?timeout=0&filter=%s", baseURL, url.QueryEscape(filter))
		} else {
			syncURL = fmt.Sprintf("%s&since=%s", syncURL, url.QueryEscape(nextBatch))
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, syncURL, nil)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		req.Header.Set("Authorization", "Bearer "+activeToken)

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			h.log.Warn().Err(err).Msg("matrix sync: request error, backing off 5s")
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			h.log.Warn().Msg("matrix sync: token unauthorized (401), refreshing token")
			activeToken = ""
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			h.log.Warn().Int("status", resp.StatusCode).Msg("matrix sync: unexpected status, retrying in 5s")
			time.Sleep(5 * time.Second)
			continue
		}

		var syncData matrixSyncResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&syncData)
		resp.Body.Close()
		if decodeErr != nil {
			h.log.Error().Err(decodeErr).Msg("matrix sync: failed to decode sync response")
			time.Sleep(5 * time.Second)
			continue
		}

		if syncData.NextBatch != "" {
			nextBatch = syncData.NextBatch
		}

		roomMu.RLock()
		for roomID, roomObj := range syncData.Rooms.Join {
			chanName, ok := roomToChan[roomID]
			if !ok {
				continue
			}

			for _, ev := range roomObj.Timeline.Events {
				if ev.Type != "m.room.message" || ev.Content.MsgType != "m.text" {
					continue
				}

				if ev.EventID != "" {
					if _, seen := seenEvents[ev.EventID]; seen {
						continue
					}
					if len(seenOrder) >= maxSeenEvents {
						oldest := seenOrder[0]
						seenOrder = seenOrder[1:]
						delete(seenEvents, oldest)
					}
					seenEvents[ev.EventID] = struct{}{}
					seenOrder = append(seenOrder, ev.EventID)
				}

				body := ev.Content.Body
				if body == "" {
					continue
				}

				nick := strings.TrimPrefix(ev.Sender, "@")
				if idx := strings.IndexByte(nick, ':'); idx > 0 {
					nick = nick[:idx]
				}
				if nick == "" {
					nick = "blubot"
				}

				cleanBody := strings.ReplaceAll(body, "\u00a0", " ")
				cleanBody = strings.ReplaceAll(cleanBody, "\r\n", " ")
				cleanBody = strings.ReplaceAll(cleanBody, "\n", " ")
				cleanBody = strings.Join(strings.Fields(cleanBody), " ")

				if err := sendIRC(fmt.Sprintf(":%s!%s@%s PRIVMSG %s :%s", nick, nick, serverHost, chanName, cleanBody)); err != nil {
					h.log.Error().Err(err).Msg("matrix: failed writing PRIVMSG to pipe")
					roomMu.RUnlock()
					return
				}
			}
		}
		roomMu.RUnlock()
	}
}
