// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/argon2id"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/alexedwards/scs/v2"
	"github.com/autobrr/go-cache/ttlcache"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

type oidcService interface {
	IsEnabled() bool
	IsReady() bool
	ValidateConfig() error
	UserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (*oidc.UserInfo, error)
	VerifyIDToken(ctx context.Context, idToken string) (*oidc.IDToken, error)
	GetEndpoint() oauth2.Endpoint
	OAuthExchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
	OauthAuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string
	SupportsPKCE() bool
}

type OIDCConfig struct {
	Enabled             bool
	Issuer              string
	ClientID            string
	ClientSecret        string
	RedirectURL         string
	DisableBuiltInLogin bool
	Scopes              []string
}

type OIDCHandler struct {
	log            zerolog.Logger
	encoder        encoder
	config         *domain.Config
	oidcConfig     *OIDCConfig
	oidcService    oidcService
	sessionManager *scs.SessionManager
	limiterKey     func(r *http.Request) string
	pendingLogins  *ttlcache.Cache[string, pendingOIDCLogin]
	// m makes taking a pending login atomic, so a state works for one callback only.
	m sync.Mutex
}

// pendingOIDCLogin is a login that was sent to the provider and whose callback
// has not arrived yet, keyed by its state.
type pendingOIDCLogin struct {
	verifier string
	target   string
}

// OIDCClaims represents the claims returned from the OIDC provider
type OIDCClaims struct {
	Email         string   `json:"email"`
	EmailVerified oidcBool `json:"email_verified"`
	Username      string   `json:"preferred_username"`
	Name          string   `json:"name"`
	GivenName     string   `json:"given_name"`
	Nickname      string   `json:"nickname"`
	Sub           string   `json:"sub"`
	Picture       string   `json:"picture"`
}

// oidcBool decodes a boolean claim that some providers send as the string "true".
type oidcBool bool

func (b *oidcBool) UnmarshalJSON(data []byte) error {
	*b = string(data) == "true" || string(data) == `"true"`

	return nil
}

// VerifiedEmail returns the email only when the provider asserts it is verified.
func (c OIDCClaims) VerifiedEmail() string {
	if !c.EmailVerified {
		return ""
	}

	return c.Email
}

// NewOIDCHandler builds the OIDC login handler; limiterKey identifies the client
// its request limiter buckets on.
func NewOIDCHandler(encoder encoder, log zerolog.Logger, cfg *domain.Config, sessionManager *scs.SessionManager, service oidcService, limiterKey func(r *http.Request) string) *OIDCHandler {
	return &OIDCHandler{
		encoder: encoder,
		log:     log,
		config:  cfg,
		oidcConfig: &OIDCConfig{
			Enabled:             cfg.OIDCEnabled,
			Issuer:              cfg.OIDCIssuer,
			ClientID:            cfg.OIDCClientID,
			ClientSecret:        cfg.OIDCClientSecret,
			RedirectURL:         cfg.OIDCRedirectURL,
			DisableBuiltInLogin: cfg.OIDCDisableBuiltInLogin,
		},
		oidcService:    service,
		sessionManager: sessionManager,
		limiterKey:     limiterKey,
		pendingLogins:  ttlcache.New[string, pendingOIDCLogin](ttlcache.SetDefaultTTL(oidcLoginTimeout), ttlcache.DisableUpdateTime(true)),
	}
}

const (
	oidcRequestBurst    = 20
	oidcRequestInterval = 3 * time.Second

	oidcCallbackLimit   = 4
	oidcCallbackBacklog = 32
	oidcProviderTimeout = 30 * time.Second

	oidcLoginTimeout      = 30 * time.Minute
	oidcStateCookiePrefix = "autobrr_oidc_"

	// /config is public, so these bound what unauthenticated callers can make the
	// server hold: per-client rate limits do not cap clients spread over many
	// addresses, and a return page would otherwise only be limited by the header size.
	oidcMaxPendingLogins  = 1024
	oidcMaxRedirectLength = 2048
)

func (h *OIDCHandler) Routes(r chi.Router) {
	// Every /config call stores fresh state in a new session and every callback
	// hits the provider, so bound both per client instead of with a global
	// throttle that one client could hold to block every OIDC login.
	r.Use(newClientRateLimiter(oidcRequestInterval, oidcRequestBurst).Middleware(h.limiterKey))

	r.Get("/config", h.getConfig)
	// The per-client limiter bounds request rate, not concurrent provider calls,
	// so the callback also gets a global cap; the provider timeout keeps a slow
	// identity provider from holding a slot indefinitely.
	r.With(middleware.ThrottleBacklog(oidcCallbackLimit, oidcCallbackBacklog, oidcProviderTimeout)).Get("/callback", h.handleCallback)
}

func (h *OIDCHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	// Until the provider is discovered there is no authorization URL to hand out;
	// an empty one tells the login page to keep the OIDC button disabled.
	if !h.oidcService.IsReady() {
		h.log.Debug().Str("issuer", h.oidcConfig.Issuer).Msg("OIDC provider not discovered yet, login page shows it as unavailable")
		h.encoder.StatusResponse(w, http.StatusOK, GetConfigResponse{
			Enabled:             h.oidcConfig.Enabled,
			DisableBuiltInLogin: h.oidcConfig.DisableBuiltInLogin,
			IssuerURL:           h.oidcConfig.Issuer,
		})
		return
	}

	config := h.GetConfigResponse()

	// Check for existing session
	authenticated := h.sessionManager.GetBool(r.Context(), "authenticated")
	if authenticated {
		config.AutoRedirect = false
		h.log.Debug().Msg("user already has valid session, skipping OIDC state cookie")
		// Still return enabled=true, just don't set the cookie
		h.encoder.StatusResponse(w, http.StatusOK, config)
		return
	}

	h.log.Debug().Bool("enabled", config.Enabled).Str("authorization_url", config.AuthorizationURL).Str("state", config.State).Msg("returning OIDC config")

	login := pendingOIDCLogin{verifier: config.PKCEVerifier}
	if target := r.URL.Query().Get("redirect"); target != "" && len(target) <= oidcMaxRedirectLength && isRedirectUnderBase(target, h.config.BaseURL) {
		login.target = target
	}

	h.startLogin(w, r, config.State, login)

	h.encoder.StatusResponse(w, http.StatusOK, config)
}

func (h *OIDCHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	h.log.Debug().Msg("handling OIDC callback")

	if !h.oidcService.IsReady() {
		h.log.Error().Str("issuer", h.oidcConfig.Issuer).Msg("OIDC callback arrived before the provider was discovered")
		h.encoder.StatusError(w, http.StatusServiceUnavailable, errors.New("OIDC provider is not available"))
		return
	}

	login, ok := h.finishLogin(w, r, r.URL.Query().Get("state"))
	if !ok {
		h.log.Error().Msg("OIDC callback state is unknown, expired or not from this browser")
		h.encoder.StatusError(w, http.StatusBadRequest, errors.New("invalid state: login expired or was started in another browser"))
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		h.log.Error().Msg("authorization code is missing from callback request")
		h.encoder.StatusError(w, http.StatusBadRequest, errors.New("authorization code is missing from callback request"))
		return
	}

	var exchangeOpts []oauth2.AuthCodeOption
	if login.verifier != "" {
		exchangeOpts = append(exchangeOpts, oauth2.VerifierOption(login.verifier))
	}

	providerCtx, cancel := context.WithTimeout(r.Context(), oidcProviderTimeout)
	defer cancel()

	oauth2Token, err := h.oidcService.OAuthExchange(providerCtx, code, exchangeOpts...)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to exchange token")
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.Wrap(err, "failed to exchange token"))
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		h.log.Error().Msg("no id_token found in oauth2 token")
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("no id_token found in oauth2 token"))
		return
	}

	idToken, err := h.oidcService.VerifyIDToken(providerCtx, rawIDToken)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to verify ID Token")
		h.encoder.StatusError(w, http.StatusUnauthorized, errors.Wrap(err, "failed to verify ID Token"))
		return
	}

	var claims OIDCClaims
	if err := idToken.Claims(&claims); err != nil {
		h.log.Error().Err(err).Msg("failed to parse claims from ID token")
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.Wrap(err, "failed to parse claims from ID token"))
		return
	}

	userInfo, userInfoErr := h.oidcService.UserInfo(providerCtx, oauth2.StaticTokenSource(oauth2Token))
	if userInfoErr != nil {
		h.log.Error().Err(userInfoErr).Msg("failed to get userinfo")
	}

	// OIDC Core 5.3.4: userinfo for another subject must not be used, or its
	// verified email could satisfy oidcAllowedEmail for this login
	if userInfo != nil && userInfo.Subject != claims.Sub {
		h.log.Warn().Str("sub", claims.Sub).Str("userinfo_sub", userInfo.Subject).Msg("userinfo subject does not match ID token subject, ignoring userinfo")
		userInfo = nil
	}

	if userInfo != nil {
		// take the userinfo email and its verification status together, never an
		// unverified userinfo email paired with the ID token's verified flag
		if userInfo.Email != "" {
			claims.Email = userInfo.Email
			claims.EmailVerified = oidcBool(userInfo.EmailVerified)
		}

		var userInfoClaims struct {
			Username string `json:"preferred_username"`
			Name     string `json:"name"`
			Nickname string `json:"nickname"`
			Picture  string `json:"picture"`
		}
		if err := userInfo.Claims(&userInfoClaims); err != nil {
			h.log.Warn().Err(err).Msg("failed to parse claims from userinfo endpoint, proceeding with ID token claims if available")
		} else {
			h.log.Debug().Str("userinfo_email", userInfo.Email).Str("userinfo_username", userInfoClaims.Username).Str("userinfo_name", userInfoClaims.Name).Str("userinfo_nickname", userInfoClaims.Nickname).Msg("successfully parsed claims from userinfo endpoint")

			if userInfoClaims.Username != "" {
				claims.Username = userInfoClaims.Username
			}
			if userInfoClaims.Name != "" {
				claims.Name = userInfoClaims.Name
			}
			if userInfoClaims.Nickname != "" {
				claims.Nickname = userInfoClaims.Nickname
			}
			if userInfoClaims.Picture != "" {
				claims.Picture = userInfoClaims.Picture
			}
		}
	}

	if claims.Username == "" {
		if claims.Nickname != "" {
			claims.Username = claims.Nickname
		} else if claims.Name != "" {
			claims.Username = claims.Name
		} else if claims.Email != "" {
			claims.Username = claims.Email
		} else if claims.Sub != "" {
			claims.Username = claims.Sub
		} else {
			claims.Username = "oidc_user"
		}
	}

	h.log.Debug().Str("email", claims.Email).Str("preferred_username", claims.Username).Str("nickname", claims.Nickname).Str("name", claims.Name).Str("sub", claims.Sub).Msg("successfully processed OIDC claims")

	if !h.config.IsOIDCUserAllowed(claims.Sub, claims.VerifiedEmail()) {
		// only the email check depends on userinfo; the error itself is logged
		// above and may carry the provider's response body
		if userInfoErr != nil && h.config.OIDCAllowedEmail != "" {
			h.encoder.StatusError(w, http.StatusBadGateway, errors.New("could not verify user: failed to get userinfo"))
			return
		}

		h.log.Warn().Str("sub", claims.Sub).Str("email", claims.Email).Bool("email_verified", bool(claims.EmailVerified)).Str("remote_addr", r.RemoteAddr).Msg("OIDC user is not allowed to log in")
		h.encoder.StatusError(w, http.StatusForbidden, errors.New("user is not allowed to log in"))
		return
	}

	// Create new session
	if err := h.sessionManager.RenewToken(r.Context()); err != nil {
		h.log.Error().Err(err).Str("username", claims.Username).Str("remote_addr", r.RemoteAddr).Msg("failed to renew session token")
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not renew session token"))
		return
	}

	// Set session values using sessionManager
	h.sessionManager.Put(r.Context(), "authenticated", true)
	h.sessionManager.Put(r.Context(), "username", claims.Username)
	h.sessionManager.Put(r.Context(), "created", time.Now().Unix())
	h.sessionManager.Put(r.Context(), "auth_method", "oidc")
	h.sessionManager.Put(r.Context(), "oidc_subject", claims.Sub)
	h.sessionManager.Put(r.Context(), "oidc_verified_email", claims.VerifiedEmail())
	h.sessionManager.Put(r.Context(), "profile_picture", claims.Picture)
	h.sessionManager.RememberMe(r.Context(), true)

	if login.target != "" {
		h.log.Debug().Str("redirect_url", login.target).Msg("redirecting to requested page after OIDC callback")
		http.Redirect(w, r, login.target, http.StatusFound)
		return
	}

	// Redirect to the frontend
	frontendURL := h.config.BaseURL
	if frontendURL == "/" {
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			host := cmp.Or(r.Header.Get("X-Forwarded-Host"), r.Host)
			frontendURL = fmt.Sprintf("%s://%s", proto, host)
		}
	}

	h.log.Debug().Str("redirect_url", frontendURL).Str("x_forwarded_proto", r.Header.Get("X-Forwarded-Proto")).Str("x_forwarded_host", r.Header.Get("X-Forwarded-Host")).Str("host", r.Host).Msg("redirecting to frontend after OIDC callback")

	http.Redirect(w, r, frontendURL, http.StatusFound)
}

// isRedirectUnderBase reports whether target is a plain path inside the base URL.
// Browsers read a backslash as a slash and strip tabs and newlines, which can turn
// a path into "//host", so those are rejected outright.
func isRedirectUnderBase(target, baseURL string) bool {
	if strings.HasPrefix(target, "//") || strings.Contains(target, "\\") {
		return false
	}

	for _, c := range target {
		if c < 0x20 || c == 0x7f {
			return false
		}
	}

	u, err := url.Parse(target)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return false
	}

	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return strings.HasPrefix(target, baseURL) && strings.HasPrefix(path.Clean(u.Path)+"/", baseURL)
}

func generateRandomState() string {
	b, err := argon2id.GenerateRandomBytes(32)
	if err != nil {
		b = make([]byte, 32)
		_, _ = rand.Read(b)
	}
	return fmt.Sprintf("%x", b)
}

func (h *OIDCHandler) GetAuthorizationURL() string {
	state := generateRandomState()
	return h.oidcService.OauthAuthCodeURL(state)
}

type GetConfigResponse struct {
	Enabled             bool   `json:"enabled"`
	AutoRedirect        bool   `json:"autoRedirect"`
	AuthorizationURL    string `json:"authorizationUrl"`
	State               string `json:"state"`
	PKCEVerifier        string `json:"-"`
	DisableBuiltInLogin bool   `json:"disableBuiltInLogin"`
	IssuerURL           string `json:"issuerUrl"`
}

func (h *OIDCHandler) GetConfigResponse() GetConfigResponse {
	state := generateRandomState()

	var authURL, verifier string
	if h.oidcService.SupportsPKCE() {
		verifier = oauth2.GenerateVerifier()
		authURL = h.oidcService.OauthAuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	} else {
		authURL = h.oidcService.OauthAuthCodeURL(state)
	}

	h.log.Debug().Bool("enabled", h.oidcConfig.Enabled).Str("authorization_url", authURL).Str("state", state).Bool("disable_built_in_login", h.oidcConfig.DisableBuiltInLogin).Str("issuer_url", h.oidcConfig.Issuer).Msg("returning OIDC oidcConfig response")

	return GetConfigResponse{
		Enabled:             h.oidcConfig.Enabled,
		AutoRedirect:        h.oidcConfig.Enabled && h.config.OIDCAutoRedirect,
		AuthorizationURL:    authURL,
		State:               state,
		PKCEVerifier:        verifier,
		DisableBuiltInLogin: h.oidcConfig.DisableBuiltInLogin,
		IssuerURL:           h.oidcConfig.Issuer,
	}
}

// startLogin records a login sent to the provider and binds its state to this
// browser with a cookie of its own. Keeping logins out of the session lets
// several tabs log in at once: their first requests carry no session cookie, so
// each would start its own session and only the last cookie set would survive.
func (h *OIDCHandler) startLogin(w http.ResponseWriter, r *http.Request, state string, login pendingOIDCLogin) {
	h.m.Lock()
	h.evictOldestLoginIfFull()
	h.pendingLogins.Set(state, login, ttlcache.DefaultTTL)
	h.m.Unlock()

	http.SetCookie(w, h.stateCookie(r, state, int(oidcLoginTimeout.Seconds())))

	h.log.Debug().Str("state", state).Msg("started OIDC login")
}

// evictOldestLoginIfFull drops the login closest to expiry once the store is full.
// Evicting costs that login a retry; refusing new logins instead would let anyone
// who can reach /config block every login. Callers hold h.m.
func (h *OIDCHandler) evictOldestLoginIfFull() {
	keys := h.pendingLogins.GetKeys()
	if len(keys) < oidcMaxPendingLogins {
		return
	}

	var oldest string
	var oldestExpiry time.Time
	for _, key := range keys {
		item, ok := h.pendingLogins.GetItem(key)
		if !ok {
			continue
		}
		if oldest == "" || item.GetTime().Before(oldestExpiry) {
			oldest, oldestExpiry = key, item.GetTime()
		}
	}

	h.pendingLogins.Delete(oldest)
	h.log.Warn().Int("pending_logins", len(keys)).Msg("too many pending OIDC logins, dropped the oldest")
}

// finishLogin returns the pending login for state when this browser started it.
// The state cookie keeps a callback URL minted for another browser from logging
// this one in, and each state works once.
func (h *OIDCHandler) finishLogin(w http.ResponseWriter, r *http.Request, state string) (pendingOIDCLogin, bool) {
	if state == "" {
		return pendingOIDCLogin{}, false
	}

	cookie, err := r.Cookie(stateCookieName(state))
	if err != nil || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(state)) != 1 {
		return pendingOIDCLogin{}, false
	}

	http.SetCookie(w, h.stateCookie(r, state, -1))

	h.m.Lock()
	defer h.m.Unlock()

	login, ok := h.pendingLogins.Get(state)
	if !ok {
		return pendingOIDCLogin{}, false
	}

	h.pendingLogins.Delete(state)

	return login, true
}

func (h *OIDCHandler) stateCookie(r *http.Request, state string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     stateCookieName(state),
		Value:    state,
		Path:     h.stateCookiePath(),
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.Header.Get("X-Forwarded-Proto") == "https",
	}
}

// stateCookiePath scopes the state cookies to the callback, the only request that
// reads them, so abandoned logins do not ride along on every API request. The
// provider sends the browser to the configured redirect URL, so its path is the
// one the callback arrives on, including any reverse proxy prefix.
func (h *OIDCHandler) stateCookiePath() string {
	basePath := h.sessionManager.Cookie.Path
	if !strings.HasSuffix(basePath, "/") {
		basePath += "/"
	}

	// A redirect URL outside the base URL only works if a proxy forwards or
	// redirects it into autobrr, and a redirect would leave the narrow path behind.
	if u, err := url.Parse(h.config.OIDCRedirectURL); err == nil && strings.HasPrefix(u.Path, basePath) {
		return u.Path
	}

	return h.sessionManager.Cookie.Path
}

// stateCookieName gives every pending login its own cookie, so concurrent logins
// in several tabs do not overwrite each other.
func stateCookieName(state string) string {
	return oidcStateCookiePrefix + state[:min(len(state), 16)]
}
