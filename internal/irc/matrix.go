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
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/rs/zerolog"
)

const matrixTokenStoreFile = "/config/matrix_tokens.json"

type matrixStoredCreds struct {
	ClientID     string    `json:"client_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenURI     string    `json:"token_uri"`
	DeviceURI    string    `json:"device_uri"`
	ExpiresAt    time.Time `json:"expires_at"`
}

var matrixStoreMu sync.Mutex

func loadMatrixCreds(server string) (*matrixStoredCreds, error) {
	matrixStoreMu.Lock()
	defer matrixStoreMu.Unlock()

	data, err := os.ReadFile(matrixTokenStoreFile)
	if err != nil {
		return nil, err
	}
	var all map[string]*matrixStoredCreds
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, err
	}
	creds, ok := all[strings.ToLower(server)]
	if !ok {
		return nil, os.ErrNotExist
	}
	return creds, nil
}

func saveMatrixCreds(server string, creds *matrixStoredCreds) error {
	matrixStoreMu.Lock()
	defer matrixStoreMu.Unlock()

	all := make(map[string]*matrixStoredCreds)
	if data, err := os.ReadFile(matrixTokenStoreFile); err == nil {
		_ = json.Unmarshal(data, &all)
	}
	all[strings.ToLower(server)] = creds
	out, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(matrixTokenStoreFile, out, 0600)
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

func (h *Handler) serveMatrixPipe(conn net.Conn, network *domain.IrcNetwork) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		_ = conn.Close()
	}()

	serverHost := strings.TrimSpace(network.Server)
	serverHost = strings.TrimPrefix(serverHost, "matrix://")
	serverHost = strings.TrimPrefix(serverHost, "https://")
	serverHost = strings.TrimPrefix(serverHost, "http://")
	serverHost = strings.TrimSuffix(serverHost, "/")
	baseURL := "https://" + serverHost

	httpClient := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: network.TLSSkipVerify,
			},
		},
	}

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

		parts := strings.SplitN(line, " ", 3)
		cmd := strings.ToUpper(parts[0])

		switch cmd {
		case "CAP":
			if len(parts) >= 2 && strings.EqualFold(parts[1], "LS") {
				_ = sendIRC(fmt.Sprintf(":%s CAP * LS :", serverHost))
			}
		case "NICK":
			if len(parts) >= 2 {
				clientNick = strings.TrimPrefix(parts[1], ":")
				nickGot = true
			}
		case "USER":
			userGot = true
		case "PING":
			arg := serverHost
			if len(parts) >= 2 {
				arg = strings.TrimPrefix(parts[1], ":")
			}
			_ = sendIRC(fmt.Sprintf(":%s PONG %s :%s", serverHost, serverHost, arg))
		case "QUIT":
			return
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
					tok, err := h.ensureMatrixAccessToken(ctx, httpClient, serverHost, baseURL, rawPass)
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

			go h.matrixSyncLoop(ctx, httpClient, serverHost, baseURL, rawPass, &roomMu, roomToChan, sendIRC)
		}
	}
}

var matrixAuthMu sync.Mutex

func (h *Handler) ensureMatrixAccessToken(ctx context.Context, client *http.Client, serverHost, baseURL, rawPass string) (string, error) {
	matrixAuthMu.Lock()
	defer matrixAuthMu.Unlock()

	creds, _ := loadMatrixCreds(serverHost)
	if creds == nil {
		creds = &matrixStoredCreds{
			ClientID:  "01M514EYEAN1WBJ9QCJZG448J0",
			TokenURI:  "https://auth.watchtower.pm/oauth2/token",
			DeviceURI: "https://auth.watchtower.pm/oauth2/device",
		}
	}

	// Check if rawPass supplies a refresh token (mrt_..., mar_..., or client_id|mar_...)
	if strings.HasPrefix(rawPass, "mrt_") || strings.HasPrefix(rawPass, "mar_") {
		if creds.RefreshToken != rawPass && creds.RefreshToken == "" {
			creds.RefreshToken = rawPass
		}
	} else if strings.Contains(rawPass, "|mrt_") || strings.Contains(rawPass, "|mar_") {
		sp := strings.SplitN(rawPass, "|", 2)
		creds.ClientID = sp[0]
		if creds.RefreshToken == "" {
			creds.RefreshToken = sp[1]
		}
	} else if strings.HasPrefix(rawPass, "mat_") || strings.HasPrefix(rawPass, "syt_") || strings.HasPrefix(rawPass, "mct_") {
		// Test if rawPass is currently active
		if checkMatrixTokenActive(ctx, client, baseURL, rawPass) {
			creds.AccessToken = rawPass
			_ = saveMatrixCreds(serverHost, creds)
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
		tok, err := refreshMatrixOAuthToken(ctx, client, creds)
		if err == nil {
			_ = saveMatrixCreds(serverHost, creds)
			return tok, nil
		}
		h.log.Warn().Err(err).Msg("matrix: refresh token failed, falling back")
	}

	if creds.AccessToken != "" && checkMatrixTokenActive(ctx, client, baseURL, creds.AccessToken) {
		return creds.AccessToken, nil
	}

	// Initiate or resume OAuth2 Device Authorization Flow so the user only needs 1 click
	tok, err := h.waitForMatrixDeviceAuth(ctx, client, serverHost, creds)
	if err != nil {
		return "", err
	}
	return tok, nil
}

type matrixDevicePending struct {
	ClientID        string    `json:"client_id"`
	DeviceCode      string    `json:"device_code"`
	UserCode        string    `json:"user_code"`
	VerificationURI string    `json:"verification_uri"`
	CompleteURI     string    `json:"verification_uri_complete"`
	Interval        int       `json:"interval"`
	ExpiresAt       time.Time `json:"expires_at"`
}

const matrixDevicePendingFile = "/config/matrix_device_pending.json"

func (h *Handler) waitForMatrixDeviceAuth(ctx context.Context, client *http.Client, serverHost string, creds *matrixStoredCreds) (string, error) {
	var pending matrixDevicePending
	if data, err := os.ReadFile(matrixDevicePendingFile); err == nil {
		_ = json.Unmarshal(data, &pending)
	}

	if pending.DeviceCode == "" || time.Now().After(pending.ExpiresAt) {
		form := url.Values{}
		form.Set("client_id", creds.ClientID)
		form.Set("scope", "openid urn:matrix:org.matrix.msc2967.client:api:* urn:matrix:org.matrix.msc2967.client:device:AUTOBRRBLU01")

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, creds.DeviceURI, strings.NewReader(form.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("device auth HTTP %d", resp.StatusCode)
		}

		var dRes struct {
			DeviceCode      string `json:"device_code"`
			UserCode        string `json:"user_code"`
			VerificationURI string `json:"verification_uri"`
			ExpiresIn       int    `json:"expires_in"`
			Interval        int    `json:"interval"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&dRes); err != nil {
			return "", err
		}
		pending = matrixDevicePending{
			ClientID:        creds.ClientID,
			DeviceCode:      dRes.DeviceCode,
			UserCode:        dRes.UserCode,
			VerificationURI: dRes.VerificationURI,
			CompleteURI:     fmt.Sprintf("%s?code=%s", dRes.VerificationURI, dRes.UserCode),
			Interval:        dRes.Interval,
			ExpiresAt:       time.Now().Add(time.Duration(dRes.ExpiresIn) * time.Second),
		}
		if out, err := json.MarshalIndent(pending, "", "  "); err == nil {
			_ = os.WriteFile(matrixDevicePendingFile, out, 0600)
		}
	}

	msg := fmt.Sprintf("Matrix OAuth2: Aprove o acesso em %s (Código: %s)", pending.CompleteURI, pending.UserCode)
	h.log.Warn().Str("url", pending.CompleteURI).Str("code", pending.UserCode).Msg("matrix: awaiting user device approval")
	h.addConnectError(msg)
	h.broadcastHealth()

	interval := pending.Interval
	if interval < 5 {
		interval = 5
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			if time.Now().After(pending.ExpiresAt) {
				_ = os.Remove(matrixDevicePendingFile)
				return "", fmt.Errorf("matrix device code expired")
			}

			form := url.Values{}
			form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
			form.Set("client_id", pending.ClientID)
			form.Set("device_code", pending.DeviceCode)

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, creds.TokenURI, strings.NewReader(form.Encode()))
			if err != nil {
				continue
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, err := client.Do(req)
			if err != nil {
				continue
			}

			if resp.StatusCode == http.StatusOK {
				var tRes struct {
					AccessToken  string `json:"access_token"`
					RefreshToken string `json:"refresh_token"`
					ExpiresIn    int    `json:"expires_in"`
				}
				err := json.NewDecoder(resp.Body).Decode(&tRes)
				_ = resp.Body.Close()
				if err == nil && tRes.AccessToken != "" {
					creds.AccessToken = tRes.AccessToken
					creds.RefreshToken = tRes.RefreshToken
					if tRes.ExpiresIn > 60 {
						creds.ExpiresAt = time.Now().Add(time.Duration(tRes.ExpiresIn-60) * time.Second)
					} else {
						creds.ExpiresAt = time.Now().Add(4 * time.Minute)
					}
					_ = saveMatrixCreds(serverHost, creds)
					_ = os.Remove(matrixDevicePendingFile)
					h.clearConnectErrors()
					h.broadcastHealth()
					h.log.Info().Msg("matrix: device authorization approved and refresh token saved!")
					return creds.AccessToken, nil
				}
				continue
			}
			_ = resp.Body.Close()
		}
	}
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

func refreshMatrixOAuthToken(ctx context.Context, client *http.Client, creds *matrixStoredCreds) (string, error) {
	tokenURI := creds.TokenURI
	if tokenURI == "" {
		tokenURI = "https://auth.watchtower.pm/oauth2/token"
	}
	clientID := creds.ClientID
	if clientID == "" {
		clientID = "01M514EYEAN1WBJ9QCJZG448J0"
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", clientID)
	form.Set("refresh_token", creds.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("oauth2 refresh status %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
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
	// Known Watchtower Blutopia room fast-path fallback
	if strings.EqualFold(alias, "#blu-announce:matrix.watchtower.pm") {
		return "!xbWdAiMmRUgepJMvfV:matrix.watchtower.pm", nil
	}

	dirURL := fmt.Sprintf("%s/_matrix/client/v3/directory/room/%s", baseURL, url.PathEscape(alias))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dirURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("directory lookup HTTP %d", resp.StatusCode)
	}
	var out struct {
		RoomID string `json:"room_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.RoomID == "" {
		return "", fmt.Errorf("empty room_id")
	}
	return out.RoomID, nil
}

func (h *Handler) matrixSyncLoop(
	ctx context.Context,
	client *http.Client,
	serverHost, baseURL, rawPass string,
	roomMu *sync.RWMutex,
	roomToChan map[string]string,
	sendIRC func(string) error,
) {
	var nextBatch string
	seenEvents := make(map[string]struct{})

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		token, err := h.ensureMatrixAccessToken(ctx, client, serverHost, baseURL, rawPass)
		if err != nil {
			h.log.Error().Err(err).Msg("matrix sync: token refresh failed, retrying in 15s")
			select {
			case <-ctx.Done():
				return
			case <-time.After(15 * time.Second):
				continue
			}
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
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			h.log.Warn().Err(err).Msg("matrix sync request error, retrying in 5s")
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusUnauthorized {
			_ = resp.Body.Close()
			// Force refresh on next iteration
			if creds, err := loadMatrixCreds(serverHost); err == nil && creds != nil {
				creds.ExpiresAt = time.Time{}
				_ = saveMatrixCreds(serverHost, creds)
			}
			time.Sleep(2 * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		var syncData matrixSyncResponse
		if err := json.NewDecoder(resp.Body).Decode(&syncData); err != nil {
			_ = resp.Body.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		_ = resp.Body.Close()

		isInitial := nextBatch == ""
		if syncData.NextBatch != "" {
			nextBatch = syncData.NextBatch
		}

		if isInitial {
			h.log.Info().Str("next_batch", nextBatch).Msg("matrix sync initialized, listening for live announces")
			continue
		}

		for roomID, roomData := range syncData.Rooms.Join {
			roomMu.RLock()
			ircChan, monitored := roomToChan[roomID]
			roomMu.RUnlock()
			if !monitored {
				continue
			}

			for _, ev := range roomData.Timeline.Events {
				if ev.Type != "m.room.message" || ev.Content.Body == "" {
					continue
				}
				if ev.EventID != "" {
					if _, dup := seenEvents[ev.EventID]; dup {
						continue
					}
					seenEvents[ev.EventID] = struct{}{}
				}

				nick := strings.TrimPrefix(ev.Sender, "@")
				if idx := strings.IndexByte(nick, ':'); idx > 0 {
					nick = nick[:idx]
				}
				if nick == "" {
					nick = "blubot"
				}

				cleanBody := strings.ReplaceAll(ev.Content.Body, "\u00a0", " ")
				cleanBody = strings.ReplaceAll(cleanBody, "\r\n", " ")
				cleanBody = strings.ReplaceAll(cleanBody, "\n", " ")
				cleanBody = strings.Join(strings.Fields(cleanBody), " ")

				if err := sendIRC(fmt.Sprintf(":%s!%s@%s PRIVMSG %s :%s", nick, nick, serverHost, ircChan, cleanBody)); err != nil {
					h.log.Error().Err(err).Msg("matrix: failed writing PRIVMSG to pipe")
					return
				}
			}
		}
	}
}

var _ = zerolog.InfoLevel
