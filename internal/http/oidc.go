// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/argon2id"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/alexedwards/scs/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	log            zerolog.Logger
	encoder        encoder
	config         *domain.Config
	oidcConfig     *OIDCConfig
	provider       *oidc.Provider
	verifier       *oidc.IDTokenVerifier
	oauthConfig    *oauth2.Config
	sessionManager *scs.SessionManager
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

func NewOIDCHandler(encoder encoder, log zerolog.Logger, cfg *domain.Config, sessionManager *scs.SessionManager) (*OIDCHandler, error) {
	log.Debug().
		Bool("oidc_enabled", cfg.OIDCEnabled).
		Str("oidc_issuer", cfg.OIDCIssuer).
		Str("oidc_client_id", cfg.OIDCClientID).
		Str("oidc_redirect_url", cfg.OIDCRedirectURL).
		Str("oidc_scopes", cfg.OIDCScopes).
		Msg("initializing OIDC handler with oidcConfig")

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

	handler := &OIDCHandler{
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
			Scopes:              scopes,
		},
		provider:       provider,
		verifier:       provider.Verifier(oidcConfig),
		sessionManager: sessionManager,
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

func (h *OIDCHandler) Routes(r chi.Router) {
	r.Use(middleware.ThrottleBacklog(1, 1, time.Second))
	r.Get("/config", h.getConfig)
	r.Get("/callback", h.handleCallback)
}

func (h *OIDCHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	// Get the config first
	config := h.GetConfigResponse()

	// Check for existing session
	authenticated := h.sessionManager.GetBool(r.Context(), "authenticated")
	if authenticated {
		h.log.Debug().Msg("user already has valid session, skipping OIDC state cookie")
		// Still return enabled=true, just don't set the cookie
		h.encoder.StatusResponse(w, http.StatusOK, config)
		return
	}

	h.log.Debug().Bool("enabled", config.Enabled).Str("authorization_url", config.AuthorizationURL).Str("state", config.State).Msg("returning OIDC config")

	// Only set state cookie if user is not already authenticated
	h.SetStateCookie(w, r, config.State)

	h.encoder.StatusResponse(w, http.StatusOK, config)
}

func (h *OIDCHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	h.log.Debug().Msg("handling OIDC callback")

	// Get and validate state parameter
	expectedState := h.sessionManager.GetString(r.Context(), "oidc_state")
	if expectedState == "" {
		h.log.Error().Msg("no state found in session")
		h.encoder.StatusError(w, http.StatusBadRequest, errors.New("invalid state: no state found in session"))
		return
	}

	actualState := r.URL.Query().Get("state")
	if actualState != expectedState {
		h.log.Error().Str("expected", expectedState).Str("got", actualState).Msg("state did not match")
		h.encoder.StatusError(w, http.StatusBadRequest, errors.New("invalid state: state mismatch"))
		return
	}

	// Clear the state from session after successful validation
	h.sessionManager.Remove(r.Context(), "oidc_state")

	code := r.URL.Query().Get("code")
	if code == "" {
		h.log.Error().Msg("authorization code is missing from callback request")
		h.encoder.StatusError(w, http.StatusBadRequest, errors.New("authorization code is missing from callback request"))
		return
	}

	oauth2Token, err := h.oauthConfig.Exchange(r.Context(), code)
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

	idToken, err := h.verifier.Verify(r.Context(), rawIDToken)
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

	userInfo, err := h.provider.UserInfo(r.Context(), oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		h.log.Error().Err(err).Msg("failed to get userinfo")
	}

	if userInfo != nil {
		var userInfoClaims struct {
			Email    string `json:"email"`
			Username string `json:"preferred_username"`
			Name     string `json:"name"`
			Nickname string `json:"nickname"`
			Picture  string `json:"picture"`
		}
		if err := userInfo.Claims(&userInfoClaims); err != nil {
			h.log.Warn().Err(err).Msg("failed to parse claims from userinfo endpoint, proceeding with ID token claims if available")
		} else {
			h.log.Debug().Str("userinfo_email", userInfoClaims.Email).Str("userinfo_username", userInfoClaims.Username).Str("userinfo_name", userInfoClaims.Name).Str("userinfo_nickname", userInfoClaims.Nickname).Msg("successfully parsed claims from userinfo endpoint")

			if userInfoClaims.Email != "" {
				claims.Email = userInfoClaims.Email
			}
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

	// Create new session
	if err := h.sessionManager.RenewToken(r.Context()); err != nil {
		h.log.Error().Err(err).Msgf("Auth: Failed to renew session token for username: [%s] ip: %s", claims.Username, r.RemoteAddr)
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not renew session token"))
		return
	}

	// Set cookie options
	h.sessionManager.Cookie.HttpOnly = true
	h.sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	h.sessionManager.Cookie.Path = h.config.BaseURL

	// If forwarded protocol is https then set cookie secure
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		h.sessionManager.Cookie.Secure = true
		h.sessionManager.Cookie.SameSite = http.SameSiteStrictMode
	}

	// Set session values using sessionManager
	h.sessionManager.Put(r.Context(), "authenticated", true)
	h.sessionManager.Put(r.Context(), "username", claims.Username)
	h.sessionManager.Put(r.Context(), "created", time.Now().Unix())
	h.sessionManager.Put(r.Context(), "auth_method", "oidc")
	h.sessionManager.Put(r.Context(), "profile_picture", claims.Picture)
	h.sessionManager.RememberMe(r.Context(), true)

	// Redirect to the frontend
	frontendURL := h.config.BaseURL
	if frontendURL == "/" {
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			host := r.Header.Get("X-Forwarded-Host")
			if host == "" {
				host = r.Host
			}
			frontendURL = fmt.Sprintf("%s://%s", proto, host)
		}
	}

	h.log.Debug().Str("redirect_url", frontendURL).Str("x_forwarded_proto", r.Header.Get("X-Forwarded-Proto")).Str("x_forwarded_host", r.Header.Get("X-Forwarded-Host")).Str("host", r.Host).Msg("redirecting to frontend after OIDC callback")

	http.Redirect(w, r, frontendURL, http.StatusFound)
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
	state := generateRandomState()
	authURL := h.oauthConfig.AuthCodeURL(state)

	h.log.Debug().Bool("enabled", h.oidcConfig.Enabled).Str("authorization_url", authURL).Str("state", state).Bool("disable_built_in_login", h.oidcConfig.DisableBuiltInLogin).Str("issuer_url", h.oidcConfig.Issuer).Msg("returning OIDC oidcConfig response")

	return GetConfigResponse{
		Enabled:             h.oidcConfig.Enabled,
		AuthorizationURL:    authURL,
		State:               state,
		DisableBuiltInLogin: h.oidcConfig.DisableBuiltInLogin,
		IssuerURL:           h.oidcConfig.Issuer,
	}
}

// SetStateCookie sets a secure cookie containing the OIDC state parameter.
// The state parameter is verified when the OAuth provider redirects back to our callback.
// Short expiration ensures the authentication flow must be completed in a reasonable timeframe.
func (h *OIDCHandler) SetStateCookie(_ http.ResponseWriter, r *http.Request, state string) {
	// Store the state in the session for later validation
	h.sessionManager.Put(r.Context(), "oidc_state", state)

	h.log.Debug().Str("state", state).Msg("stored OIDC state in session")
}
