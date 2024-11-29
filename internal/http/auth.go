// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/auth"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog"
)

type authService interface {
	GetUserCount(ctx context.Context) (int, error)
	Login(ctx context.Context, username, password string) (*domain.User, error)
	CreateUser(ctx context.Context, req domain.CreateUserRequest) error
	UpdateUser(ctx context.Context, req domain.UpdateUserRequest) error
}

type authHandler struct {
	log         zerolog.Logger
	encoder     encoder
	config      *domain.Config
	service     authService
	server      Server
	cookieStore *sessions.CookieStore
	oidcHandler *auth.OIDCHandler
}

func newAuthHandler(encoder encoder, log zerolog.Logger, server Server, config *domain.Config, cookieStore *sessions.CookieStore, service authService) *authHandler {
	oidcHandler, err := auth.NewOIDCHandler(config, log)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize OIDC handler")
	}

	return &authHandler{
		log:         log,
		encoder:     encoder,
		config:      config,
		service:     service,
		cookieStore: cookieStore,
		server:      server,
		oidcHandler: oidcHandler,
	}
}

func (h authHandler) Routes(r chi.Router) {
	r.Post("/login", h.login)
	r.Post("/onboard", h.onboard)
	r.Get("/onboard", h.canOnboard)

	r.Route("/oidc", func(r chi.Router) {
		r.Get("/config", h.getOIDCConfig)
		r.Get("/callback", h.handleOIDCCallback)
	})

	// Group for authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(h.server.IsAuthenticated)

		r.Post("/logout", h.logout)
		r.Get("/validate", h.validate)
		r.Patch("/user/{username}", h.updateUser)
	})
}

func (h authHandler) login(w http.ResponseWriter, r *http.Request) {
	var data domain.User
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.Wrap(err, "could not decode json"))
		return
	}

	if _, err := h.service.Login(r.Context(), data.Username, data.Password); err != nil {
		h.log.Error().Err(err).Msgf("Auth: Failed login attempt username: [%s] ip: %s", data.Username, r.RemoteAddr)
		h.encoder.StatusError(w, http.StatusForbidden, errors.New("could not login: bad credentials"))
		return
	}

	// create new session
	session, err := h.cookieStore.Get(r, "user_session")
	if err != nil {
		h.log.Error().Err(err).Msgf("Auth: Failed to create cookies with attempt username: [%s] ip: %s", data.Username, r.RemoteAddr)
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not create cookies"))
		return
	}

	// Set user as authenticated
	session.Values["authenticated"] = true
	session.Values["created"] = time.Now().Unix()

	// Set cookie options
	session.Options.HttpOnly = true
	session.Options.SameSite = http.SameSiteLaxMode
	session.Options.Path = h.config.BaseURL

	// autobrr does not support serving on TLS / https, so this is only available behind reverse proxy
	// if forwarded protocol is https then set cookie secure
	// SameSite Strict can only be set with a secure cookie. So we overwrite it here if possible.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		session.Options.Secure = true
		session.Options.SameSite = http.SameSiteStrictMode
	}

	if err := session.Save(r, w); err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.Wrap(err, "could not save session"))
		return
	}

	h.encoder.NoContent(w)
}

func (h authHandler) logout(w http.ResponseWriter, r *http.Request) {
	// get session from context
	session, ok := r.Context().Value("session").(*sessions.Session)
	if !ok {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get session from context"))
		return
	}

	if session != nil {
		session.Values["authenticated"] = false

		// MaxAge<0 means delete cookie immediately
		session.Options.MaxAge = -1

		session.Options.Path = h.config.BaseURL

		if err := session.Save(r, w); err != nil {
			h.log.Error().Err(err).Msgf("could not store session: %s", r.RemoteAddr)
			h.encoder.StatusError(w, http.StatusInternalServerError, err)
			return
		}
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h authHandler) onboard(w http.ResponseWriter, r *http.Request) {
	if status, err := h.onboardEligible(r.Context()); err != nil {
		h.encoder.StatusError(w, status, err)
		return
	}

	var req domain.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.Wrap(err, "could not decode json"))
		return
	}

	if err := h.service.CreateUser(r.Context(), req); err != nil {
		h.encoder.StatusError(w, http.StatusForbidden, err)
		return
	}

	// send response as ok
	h.encoder.StatusResponseMessage(w, http.StatusOK, "user successfully created")
}

func (h authHandler) canOnboard(w http.ResponseWriter, r *http.Request) {
	if status, err := h.onboardEligible(r.Context()); err != nil {
		h.encoder.StatusError(w, status, err)
		return
	}

	// send empty response as ok
	// (client can proceed with redirection to onboarding page)
	h.encoder.NoContent(w)
}

// onboardEligible checks if the onboarding process is eligible.
func (h authHandler) onboardEligible(ctx context.Context) (int, error) {
	userCount, err := h.service.GetUserCount(ctx)
	if err != nil {
		return http.StatusInternalServerError, errors.New("could not get user count")
	}

	if userCount > 0 {
		return http.StatusServiceUnavailable, errors.New("onboarding unavailable")
	}

	return http.StatusOK, nil
}

// validate sits behind the IsAuthenticated middleware which takes care of checking for a valid session
// If there is a valid session return OK, otherwise the middleware returns early with a 401
func (h authHandler) validate(w http.ResponseWriter, r *http.Request) {
	session := r.Context().Value("session").(*sessions.Session)
	if session != nil {
		h.log.Debug().Msgf("found user session: %+v", session)
		// Return username if available in session
		if username, ok := session.Values["username"].(string); ok {
			h.encoder.StatusResponse(w, http.StatusOK, map[string]interface{}{
				"username": username,
			})
			return
		}
	}
	// send empty response as ok
	h.encoder.NoContent(w)
}

func (h authHandler) updateUser(w http.ResponseWriter, r *http.Request) {
	var data domain.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.Wrap(err, "could not decode json"))
		return
	}

	data.UsernameCurrent = chi.URLParam(r, "username")

	if err := h.service.UpdateUser(r.Context(), data); err != nil {
		h.encoder.StatusError(w, http.StatusForbidden, err)
		return
	}

	// send response as ok
	h.encoder.StatusResponseMessage(w, http.StatusOK, "user successfully updated")
}

func (h authHandler) getOIDCConfig(w http.ResponseWriter, r *http.Request) {
	h.log.Debug().Msg("getting OIDC config")

	if h.oidcHandler == nil {
		h.log.Debug().Msg("OIDC handler is nil, returning disabled config")
		h.encoder.StatusResponse(w, http.StatusOK, auth.GetConfigResponse{
			Enabled: false,
		})
		return
	}

	config := h.oidcHandler.GetConfigResponse()
	h.log.Debug().
		Bool("enabled", config.Enabled).
		Str("authorization_url", config.AuthorizationURL).
		Str("state", config.State).
		Msg("returning OIDC config")

	h.oidcHandler.SetStateCookie(w, r, config.State)

	h.encoder.StatusResponse(w, http.StatusOK, config)
}

func (h authHandler) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if h.oidcHandler == nil {
		h.encoder.StatusError(w, http.StatusServiceUnavailable, errors.New("OIDC not configured"))
		return
	}

	username, err := h.oidcHandler.HandleCallback(w, r)
	if err != nil {
		h.encoder.StatusError(w, http.StatusUnauthorized, errors.Wrap(err, "OIDC authentication failed"))
		return
	}

	// Create new session
	session, err := h.cookieStore.Get(r, "user_session")
	if err != nil {
		h.log.Error().Err(err).Msgf("Auth: Failed to create cookies with attempt username: [%s] ip: %s", username, r.RemoteAddr)
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not create cookies"))
		return
	}

	// Set user as authenticated
	session.Values["authenticated"] = true
	session.Values["created"] = time.Now().Unix()
	session.Values["username"] = username

	// Set cookie options
	session.Options.HttpOnly = true
	session.Options.SameSite = http.SameSiteLaxMode
	session.Options.Path = h.config.BaseURL

	// If forwarded protocol is https then set cookie secure
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		session.Options.Secure = true
		session.Options.SameSite = http.SameSiteStrictMode
	}

	if err := session.Save(r, w); err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.Wrap(err, "could not save session"))
		return
	}

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

	h.log.Debug().
		Str("redirect_url", frontendURL).
		Str("x_forwarded_proto", r.Header.Get("X-Forwarded-Proto")).
		Str("x_forwarded_host", r.Header.Get("X-Forwarded-Host")).
		Str("host", r.Host).
		Msg("redirecting to frontend after OIDC callback")

	http.Redirect(w, r, frontendURL, http.StatusFound)
}
