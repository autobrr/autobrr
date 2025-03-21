// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/argon2id"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

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
	config      *OIDCConfig
	provider    *oidc.Provider
	verifier    *oidc.IDTokenVerifier
	oauthConfig *oauth2.Config
	log         zerolog.Logger
	cookieStore *sessions.CookieStore
}

// OIDCClaims represents the claims returned from the OIDC provider
type OIDCClaims struct {
	Email     string `json:"email"`
	Username  string `json:"preferred_username"`
	Name      string `json:"name"`
	GivenName string `json:"given_name"`
	Nickname  string `json:"nickname"`
	Sub       string `json:"sub"`
	Picture   string `json:"picture"`
}

func NewOIDCHandler(cfg *domain.Config, log zerolog.Logger) (*OIDCHandler, error) {
	log.Debug().
		Bool("oidc_enabled", cfg.OIDCEnabled).
		Str("oidc_issuer", cfg.OIDCIssuer).
		Str("oidc_client_id", cfg.OIDCClientID).
		Str("oidc_redirect_url", cfg.OIDCRedirectURL).
		Str("oidc_scopes", cfg.OIDCScopes).
		Msg("initializing OIDC handler with config")

	//if !cfg.OIDCEnabled {
	//	log.Debug().Msg("OIDC is not enabled, returning nil handler")
	//	return nil, nil
	//}

	if cfg.OIDCIssuer == "" {
		log.Error().Msg("OIDC issuer is empty")
		return nil, errors.New("OIDC issuer is required")
	}

	if cfg.OIDCClientID == "" {
		log.Error().Msg("OIDC client ID is empty")
		return nil, errors.New("OIDC client ID is required")
	}

	if cfg.OIDCClientSecret == "" {
		log.Error().Msg("OIDC client secret is empty")
		return nil, errors.New("OIDC client secret is required")
	}

	if cfg.OIDCRedirectURL == "" {
		log.Error().Msg("OIDC redirect URL is empty")
		return nil, errors.New("OIDC redirect URL is required")
	}

	scopes := []string{"openid", "profile", "email"}

	issuer := cfg.OIDCIssuer
	ctx := context.Background()

	// First try with original issuer
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		// If failed and issuer ends with slash, try without
		if strings.HasSuffix(issuer, "/") {
			withoutSlash := strings.TrimRight(issuer, "/")
			log.Debug().
				Str("original_issuer", issuer).
				Str("retry_issuer", withoutSlash).
				Msg("retrying OIDC provider initialization without trailing slash")

			provider, err = oidc.NewProvider(ctx, withoutSlash)
		} else {
			// If failed and issuer doesn't end with slash, try with
			withSlash := issuer + "/"
			log.Debug().Str("original_issuer", issuer).Str("retry_issuer", withSlash).Msg("retrying OIDC provider initialization with trailing slash")

			provider, err = oidc.NewProvider(ctx, withSlash)
		}

		if err != nil {
			log.Error().Err(err).Msg("failed to initialize OIDC provider")
			return nil, errors.Wrap(err, "failed to initialize OIDC provider")
		}
	}

	var claims struct {
		AuthURL  string `json:"authorization_endpoint"`
		TokenURL string `json:"token_endpoint"`
		JWKSURL  string `json:"jwks_uri"`
		UserURL  string `json:"userinfo_endpoint"`
	}
	if err := provider.Claims(&claims); err != nil {
		log.Warn().Err(err).Msg("failed to parse provider claims for endpoints")
	} else {
		log.Debug().Str("authorization_endpoint", claims.AuthURL).Str("token_endpoint", claims.TokenURL).Str("jwks_uri", claims.JWKSURL).Str("userinfo_endpoint", claims.UserURL).Msg("discovered OIDC provider endpoints")
	}

	oidcConfig := &oidc.Config{
		ClientID: cfg.OIDCClientID,
	}

	stateSecret := generateRandomState()

	handler := &OIDCHandler{
		log: log,
		config: &OIDCConfig{
			Enabled:             cfg.OIDCEnabled,
			Issuer:              cfg.OIDCIssuer,
			ClientID:            cfg.OIDCClientID,
			ClientSecret:        cfg.OIDCClientSecret,
			RedirectURL:         cfg.OIDCRedirectURL,
			DisableBuiltInLogin: cfg.OIDCDisableBuiltInLogin,
			Scopes:              scopes,
		},
		provider: provider,
		verifier: provider.Verifier(oidcConfig),
		oauthConfig: &oauth2.Config{
			ClientID:     cfg.OIDCClientID,
			ClientSecret: cfg.OIDCClientSecret,
			RedirectURL:  cfg.OIDCRedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       scopes,
		},
		cookieStore: sessions.NewCookieStore([]byte(stateSecret)),
	}

	log.Debug().Msg("OIDC handler initialized successfully")
	return handler, nil
}

func (h *OIDCHandler) GetConfig() *OIDCConfig {
	if h == nil {
		return &OIDCConfig{
			Enabled: false,
		}
	}
	h.log.Debug().Bool("enabled", h.config.Enabled).Str("issuer", h.config.Issuer).Msg("returning OIDC config")
	return h.config
}

func (h *OIDCHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	session, err := h.cookieStore.Get(r, "user_session")
	if err != nil {
		h.log.Error().Err(err).Msg("failed to get user session")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if session.Values["authenticated"] == true {
		h.log.Debug().Msg("user already has valid session, skipping OIDC login")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	state := generateRandomState()
	h.SetStateCookie(w, r, state)

	authURL := h.oauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *OIDCHandler) HandleCallback(w http.ResponseWriter, r *http.Request) (*OIDCClaims, error) {
	h.log.Debug().Msg("handling OIDC callback")

	// get state from session
	session, err := h.cookieStore.Get(r, "oidc_state")
	if err != nil {
		h.log.Error().Err(err).Msg("state session not found")
		return nil, errors.New("state session not found")
	}

	expectedState, ok := session.Values["state"].(string)
	if !ok {
		h.log.Error().Msg("state not found in session")
		return nil, errors.New("state not found in session")
	}

	if r.URL.Query().Get("state") != expectedState {
		h.log.Error().Str("expected", expectedState).Str("got", r.URL.Query().Get("state")).Msg("state did not match")
		return nil, errors.New("state did not match")
	}

	// clear the state session after use
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		h.log.Error().Err(err).Msg("failed to clear state session")
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		h.log.Error().Msg("authorization code is missing from callback request")
		return nil, errors.New("authorization code is missing from callback request")
	}

	oauth2Token, err := h.oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to exchange token")
		return nil, errors.Wrap(err, "failed to exchange token")
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		h.log.Error().Msg("no id_token found in oauth2 token")
		return nil, errors.New("no id_token found in oauth2 token")
	}

	idToken, err := h.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to verify ID Token")
		return nil, errors.Wrap(err, "failed to verify ID Token")
	}

	var claims OIDCClaims
	if err := idToken.Claims(&claims); err != nil {
		h.log.Error().Err(err).Msg("failed to parse claims")
		return nil, errors.Wrap(err, "failed to parse claims")
	}

	// Try different claims in order of preference for username
	// This is solely used for frontend display
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

	h.log.Debug().
		Str("username", claims.Username).
		Str("email", claims.Email).
		Str("nickname", claims.Nickname).
		Str("name", claims.Name).
		Str("sub", claims.Sub).
		Str("picture", claims.Picture).
		Msg("successfully processed OIDC claims")

	return &claims, nil
}

func generateRandomState() string {
	b, err := argon2id.GenerateRandomBytes(32)
	if err != nil {
		b = make([]byte, 32)
		rand.Read(b)
	}
	return fmt.Sprintf("%x", b)
}

func (h *OIDCHandler) GetAuthorizationURL() string {
	if h == nil {
		return ""
	}
	state := generateRandomState()
	return h.oauthConfig.AuthCodeURL(state)
}

type GetConfigResponse struct {
	Enabled             bool   `json:"enabled"`
	AuthorizationURL    string `json:"authorizationUrl"`
	State               string `json:"state"`
	DisableBuiltInLogin bool   `json:"disableBuiltInLogin"`
	IssuerURL           string `json:"issuerUrl"`
}

func (h *OIDCHandler) GetConfigResponse() GetConfigResponse {
	if h == nil {
		return GetConfigResponse{
			Enabled:             false,
			DisableBuiltInLogin: false,
			IssuerURL:           "",
		}
	}

	state := generateRandomState()
	authURL := h.oauthConfig.AuthCodeURL(state)

	h.log.Debug().Bool("enabled", h.config.Enabled).Str("authorization_url", authURL).Str("state", state).Bool("disable_built_in_login", h.config.DisableBuiltInLogin).Str("issuer_url", h.config.Issuer).Msg("returning OIDC config response")

	return GetConfigResponse{
		Enabled:             h.config.Enabled,
		AuthorizationURL:    authURL,
		State:               state,
		DisableBuiltInLogin: h.config.DisableBuiltInLogin,
		IssuerURL:           h.config.Issuer,
	}
}

// SetStateCookie sets a secure cookie containing the OIDC state parameter.
// The state parameter is verified when the OAuth provider redirects back to our callback.
// Short expiration ensures the authentication flow must be completed in a reasonable timeframe.
func (h *OIDCHandler) SetStateCookie(w http.ResponseWriter, r *http.Request, state string) {
	session, _ := h.cookieStore.New(r, "oidc_state")
	session.Values["state"] = state
	session.Options.MaxAge = 300
	session.Options.HttpOnly = true
	session.Options.Secure = r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	session.Options.SameSite = http.SameSiteLaxMode
	session.Options.Path = "/"

	if err := session.Save(r, w); err != nil {
		h.log.Error().Err(err).Msg("failed to save state session")
	}
}
