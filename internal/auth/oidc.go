// Copyright (c) 2021-2024, Ludvig Lundgren and the autobrr contributors.
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
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

type OIDCConfig struct {
	Enabled      bool
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

type OIDCHandler struct {
	config      *OIDCConfig
	provider    *oidc.Provider
	verifier    *oidc.IDTokenVerifier
	oauthConfig *oauth2.Config
	log         zerolog.Logger
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
		return nil, fmt.Errorf("OIDC issuer is required")
	}

	if cfg.OIDCClientID == "" {
		log.Error().Msg("OIDC client ID is empty")
		return nil, fmt.Errorf("OIDC client ID is required")
	}

	if cfg.OIDCClientSecret == "" {
		log.Error().Msg("OIDC client secret is empty")
		return nil, fmt.Errorf("OIDC client secret is required")
	}

	if cfg.OIDCRedirectURL == "" {
		log.Error().Msg("OIDC redirect URL is empty")
		return nil, fmt.Errorf("OIDC redirect URL is required")
	}

	scopes := []string{"openid", "profile", "email"}

	issuer := strings.TrimRight(cfg.OIDCIssuer, "/")

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize OIDC provider")
		return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
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
		log.Debug().
			Str("authorization_endpoint", claims.AuthURL).
			Str("token_endpoint", claims.TokenURL).
			Str("jwks_uri", claims.JWKSURL).
			Str("userinfo_endpoint", claims.UserURL).
			Msg("discovered OIDC provider endpoints")
	}

	oidcConfig := &oidc.Config{
		ClientID: cfg.OIDCClientID,
	}

	handler := &OIDCHandler{
		log: log,
		config: &OIDCConfig{
			Enabled:      cfg.OIDCEnabled,
			Issuer:       cfg.OIDCIssuer,
			ClientID:     cfg.OIDCClientID,
			ClientSecret: cfg.OIDCClientSecret,
			RedirectURL:  cfg.OIDCRedirectURL,
			Scopes:       scopes,
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
	h.log.Debug().
		Bool("enabled", h.config.Enabled).
		Str("issuer", h.config.Issuer).
		Msg("returning OIDC config")
	return h.config
}

func (h *OIDCHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateRandomState()

	h.SetStateCookie(w, r, state)

	authURL := h.oauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *OIDCHandler) HandleCallback(w http.ResponseWriter, r *http.Request) (string, error) {
	h.log.Debug().Msg("handling OIDC callback")

	state, err := r.Cookie("state")
	if err != nil {
		h.log.Error().Err(err).Msg("state cookie not found")
		return "", fmt.Errorf("state cookie not found")
	}
	if r.URL.Query().Get("state") != state.Value {
		h.log.Error().
			Str("expected", state.Value).
			Str("got", r.URL.Query().Get("state")).
			Msg("state did not match")
		return "", fmt.Errorf("state did not match")
	}

	oauth2Token, err := h.oauthConfig.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		h.log.Error().Err(err).Msg("failed to exchange token")
		return "", fmt.Errorf("failed to exchange token: %w", err)
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		h.log.Error().Msg("no id_token found in oauth2 token")
		return "", fmt.Errorf("no id_token found in oauth2 token")
	}

	idToken, err := h.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to verify ID Token")
		return "", fmt.Errorf("failed to verify ID Token: %w", err)
	}

	var claims struct {
		Email    string `json:"email"`
		Username string `json:"preferred_username"`
		Sub      string `json:"sub"`
	}
	if err := idToken.Claims(&claims); err != nil {
		h.log.Error().Err(err).Msg("failed to parse claims")
		return "", fmt.Errorf("failed to parse claims: %w", err)
	}

	// Use preferred_username if available, otherwise use email, sub, or a default value
	// This is used for frontend display
	username := claims.Username
	if username == "" {
		username = claims.Email
		if username == "" {
			username = claims.Sub
			if username == "" {
				username = "oidc_user"
			}
		}
	}

	h.log.Debug().
		Str("username", username).
		Str("email", claims.Email).
		Str("sub", claims.Sub).
		Msg("successfully processed OIDC callback")

	return username, nil
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
	Enabled          bool   `json:"enabled"`
	AuthorizationURL string `json:"authorizationUrl"`
	State            string `json:"state"`
}

func (h *OIDCHandler) GetConfigResponse() GetConfigResponse {
	if h == nil {
		return GetConfigResponse{
			Enabled: false,
		}
	}

	state := generateRandomState()
	authURL := h.oauthConfig.AuthCodeURL(state)
	h.log.Debug().
		Bool("enabled", h.config.Enabled).
		Str("authorization_url", authURL).
		Str("state", state).
		Msg("returning OIDC config response")

	return GetConfigResponse{
		Enabled:          h.config.Enabled,
		AuthorizationURL: authURL,
		State:            state,
	}
}

// SetStateCookie sets a secure cookie containing the OIDC state parameter.
// The state parameter is verified when the OAuth provider redirects back to our callback.
// Short expiration ensures the authentication flow must be completed in a reasonable timeframe.
func (h *OIDCHandler) SetStateCookie(w http.ResponseWriter, r *http.Request, state string) {
	isSecure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	http.SetCookie(w, &http.Cookie{
		Name:     "state",
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
