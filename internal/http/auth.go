// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type authService interface {
	GetUserCount(ctx context.Context) (int, error)
	Login(ctx context.Context, username, password string) (*domain.User, error)
	CreateUser(ctx context.Context, req domain.CreateUserRequest) error
	UpdateUser(ctx context.Context, req domain.UpdateUserRequest) error
}

type authHandler struct {
	log            zerolog.Logger
	encoder        encoder
	config         *domain.Config
	service        authService
	oidcService    oidcService
	server         *Server
	sessionManager *scs.SessionManager
	oidcHandler    *OIDCHandler
}

func newAuthHandler(encoder encoder, log zerolog.Logger, server *Server, config *domain.Config, sessionManager *scs.SessionManager, service authService, oidcService oidcService) *authHandler {
	h := &authHandler{
		log:            log,
		encoder:        encoder,
		config:         config,
		service:        service,
		oidcService:    oidcService,
		sessionManager: sessionManager,
		server:         server,
	}

	if h.oidcService.IsEnabled() {
		h.oidcHandler = NewOIDCHandler(encoder, log, config, sessionManager, oidcService)
	}

	return h
}

func (h *authHandler) Routes(r chi.Router) {
	r.Post("/login", h.login)
	r.Post("/onboard", h.onboard)
	r.Get("/onboard", h.canOnboard)

	if h.config.OIDCEnabled && h.oidcHandler != nil {
		r.Route("/oidc", h.oidcHandler.Routes)
	}

	// Group for authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(h.server.IsAuthenticated)

		r.Post("/logout", h.logout)
		r.Get("/validate", h.validate)
		r.Patch("/user/{username}", h.updateUser)
	})
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.config.DisableAuth {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.New("login not available: authentication is disabled"))
		return
	}

	var data domain.UserLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.Wrap(err, "could not decode json"))
		return
	}

	if _, err := h.service.Login(ctx, data.Username, data.Password); err != nil {
		h.log.Error().Err(err).Msgf("Auth: Failed login attempt username: [%s] ip: %s", data.Username, r.RemoteAddr)
		h.encoder.StatusError(w, http.StatusForbidden, errors.New("could not login: bad credentials"))
		return
	}

	// Set cookie options
	h.sessionManager.Cookie.HttpOnly = true
	h.sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	h.sessionManager.Cookie.Path = h.config.BaseURL

	// autobrr does not support serving on TLS / https, so this is only available behind reverse proxy.
	// When forwarded protocol is https we mark the cookie as Secure, but keep SameSite=Lax so OIDC
	// callbacks returning from a different domain still include the session cookie.
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		h.sessionManager.Cookie.Secure = true
	}

	if err := h.sessionManager.RenewToken(ctx); err != nil {
		h.log.Error().Err(err).Msgf("Auth: Failed to renew session token for username: [%s] ip: %s", data.Username, r.RemoteAddr)
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not renew session token"))
		return
	}

	h.sessionManager.RememberMe(ctx, data.RememberMe)

	// Set session values using sessionManager
	h.sessionManager.Put(r.Context(), "authenticated", true)
	h.sessionManager.Put(r.Context(), "username", data.Username)
	h.sessionManager.Put(r.Context(), "created", time.Now().Unix())
	h.sessionManager.Put(r.Context(), "auth_method", "password")

	h.encoder.NoContent(w)
}

func (h *authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.sessionManager.Destroy(r.Context()); err != nil {
		h.log.Error().Err(err).Msgf("could not destroy session: %s", r.RemoteAddr)
		h.encoder.StatusError(w, http.StatusInternalServerError, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h *authHandler) onboard(w http.ResponseWriter, r *http.Request) {
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

func (h *authHandler) canOnboard(w http.ResponseWriter, r *http.Request) {
	if status, err := h.onboardEligible(r.Context()); err != nil {
		if status == http.StatusServiceUnavailable {
			h.encoder.StatusResponse(w, status, err.Error())
			return
		}
		h.encoder.StatusError(w, status, err)
		return
	}

	// send empty response as ok
	// (client can proceed with redirection to onboarding page)
	h.encoder.NoContent(w)
}

// onboardEligible checks if the onboarding process is eligible.
func (h *authHandler) onboardEligible(ctx context.Context) (int, error) {
	if h.config.OIDCEnabled {
		return http.StatusServiceUnavailable, errors.New("onboarding unavailable: using oidc provider")
	}

	if h.config.DisableAuth {
		return http.StatusServiceUnavailable, errors.New("onboarding unavailable: no-auth mode enabled")
	}

	userCount, err := h.service.GetUserCount(ctx)
	if err != nil {
		return http.StatusInternalServerError, errors.New("could not get user count")
	}

	if userCount > 0 {
		h.log.Trace().Msg("onboarding unavailable: user already registered")
		return http.StatusServiceUnavailable, errors.New("onboarding unavailable: user already registered")
	}

	return http.StatusOK, nil
}

// validate sits behind the IsAuthenticated middleware which takes care of checking for a valid session
// If there is a valid session return OK, otherwise the middleware returns early with a 401
func (h *authHandler) validate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.config.DisableAuth {
		response := map[string]interface{}{
			"auth_method": "none",
		}
		h.encoder.StatusResponse(w, http.StatusOK, response)
		return
	}

	username := h.sessionManager.GetString(ctx, "username")
	authMethod := h.sessionManager.GetString(ctx, "auth_method")
	profilePicture := h.sessionManager.GetString(ctx, "profile_picture")

	response := map[string]interface{}{
		"username":    username,
		"auth_method": authMethod,
	}

	if profilePicture != "" {
		response["profile_picture"] = profilePicture
	}

	h.encoder.StatusResponse(w, http.StatusOK, response)
}

func (h *authHandler) updateUser(w http.ResponseWriter, r *http.Request) {
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
