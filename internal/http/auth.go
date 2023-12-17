// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"net/http"

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
}

type authHandler struct {
	log     zerolog.Logger
	encoder encoder
	config  *domain.Config
	service authService

	cookieStore *sessions.CookieStore
}

func newAuthHandler(encoder encoder, log zerolog.Logger, config *domain.Config, cookieStore *sessions.CookieStore, service authService) *authHandler {
	return &authHandler{
		log:         log,
		encoder:     encoder,
		config:      config,
		service:     service,
		cookieStore: cookieStore,
	}
}

func (h authHandler) Routes(r chi.Router) {
	r.Post("/login", h.login)
	r.Post("/logout", h.logout)
	r.Post("/onboard", h.onboard)
	r.Get("/onboard", h.canOnboard)
	r.Get("/validate", h.validate)
}

func (h authHandler) login(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.User
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.Wrap(err, "could not decode json"))
		return
	}

	h.cookieStore.Options.HttpOnly = true
	h.cookieStore.Options.SameSite = http.SameSiteLaxMode
	h.cookieStore.Options.Path = h.config.BaseURL

	// autobrr does not support serving on TLS / https, so this is only available behind reverse proxy
	// if forwarded protocol is https then set cookie secure
	// SameSite Strict can only be set with a secure cookie. So we overwrite it here if possible.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite
	fwdProto := r.Header.Get("X-Forwarded-Proto")
	if fwdProto == "https" {
		h.cookieStore.Options.Secure = true
		h.cookieStore.Options.SameSite = http.SameSiteStrictMode
	}

	if _, err := h.service.Login(ctx, data.Username, data.Password); err != nil {
		h.log.Error().Err(err).Msgf("Auth: Failed login attempt username: [%s] ip: %s", data.Username, ReadUserIP(r))
		h.encoder.StatusError(w, http.StatusUnauthorized, errors.New("could not login: bad credentials"))
		return
	}

	// create new session
	session, err := h.cookieStore.New(r, "user_session")
	if err != nil {
		h.log.Error().Err(err).Msgf("Auth: Failed to parse cookies with attempt username: [%s] ip: %s", data.Username, ReadUserIP(r))
		h.encoder.StatusError(w, http.StatusUnauthorized, errors.New("could not parse cookies"))
		return
	}

	// Set user as authenticated
	session.Values["authenticated"] = true
	h.cookieStore.Save(r, w, session)
	if err := session.Save(r, w); err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.Wrap(err, "could not save session"))
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if session := h.getLoginOrInvalidate(w, r); session == nil {
		return
	}

	h.logoutUser(w, r)
	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h authHandler) onboard(w http.ResponseWriter, r *http.Request) {
	if !h.onboardEligible(w, r) {
		return
	}

	var req domain.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.Wrap(err, "could not decode json"))
		return
	}

	ctx := r.Context()
	if err := h.service.CreateUser(ctx, req); err != nil {
		h.encoder.StatusError(w, http.StatusForbidden, err)
		return
	}

	// send response as ok
	h.encoder.StatusResponseMessage(w, http.StatusOK, "user successfully created")
}

func (h authHandler) canOnboard(w http.ResponseWriter, r *http.Request) {
	if !h.onboardEligible(w, r) {
		return
	}

	// send empty response as ok
	// (client can proceed with redirection to onboarding page)
	h.encoder.NoContent(w)
}

func (h authHandler) onboardEligible(w http.ResponseWriter, r *http.Request) bool {
	ctx := r.Context()

	userCount, err := h.service.GetUserCount(ctx)
	if err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get user count"))
		return false
	}

	if userCount > 0 {
		h.encoder.StatusError(w, http.StatusForbidden, errors.New("onboarding unavailable"))
		return false
	}

	return true
}

func (h authHandler) validate(w http.ResponseWriter, r *http.Request) {
	if session := h.getLoginOrInvalidate(w, r); session == nil {
		return
	}

	// send empty response as ok
	h.encoder.NoContent(w)
}

func ReadUserIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}

func (h authHandler) getLoginOrInvalidate(w http.ResponseWriter, r *http.Request) *sessions.Session {
	session, err := h.cookieStore.Get(r, "user_session")
	if err == nil {
		if auth, ok := session.Values["authenticated"].(bool); ok && auth {
			return session
		} else {
			h.log.Error().Err(err).Msgf("Invalid session from ip: %s", ReadUserIP(r))
		}
	} else {
		h.log.Error().Err(err).Msgf("Failed to parse cookies from ip: %s", ReadUserIP(r))
	}

	h.logoutUser(w, r)
	h.encoder.StatusError(w, http.StatusUnauthorized, errors.New("forbidden: invalid session"))
	return nil
}

func (h authHandler) logoutUser(w http.ResponseWriter, r *http.Request) {
	session, err := h.cookieStore.New(r, "user_session")
	if session != nil {
		h.cookieStore.Save(r, w, session)
		session.Save(r, w)
	} else if err != nil {
		h.log.Error().Err(err).Msgf("Failed to reset cookie store from ip: %s", ReadUserIP(r))
	}
}
