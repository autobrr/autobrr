// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

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
	Get2FAStatus(ctx context.Context, username string) (bool, error)
	Enable2FA(ctx context.Context, username string) (string, string, error)
	Verify2FA(ctx context.Context, username string, code string) error
	Verify2FALogin(ctx context.Context, username string, code string) error
	Disable2FA(ctx context.Context, username string) error
}

type authHandler struct {
	log     zerolog.Logger
	encoder encoder
	config  *domain.Config
	service authService
	server  Server

	cookieStore *sessions.CookieStore
}

func newAuthHandler(encoder encoder, log zerolog.Logger, server Server, config *domain.Config, cookieStore *sessions.CookieStore, service authService) *authHandler {
	return &authHandler{
		log:         log,
		encoder:     encoder,
		config:      config,
		service:     service,
		cookieStore: cookieStore,
		server:      server,
	}
}

func (h authHandler) Routes(r chi.Router) {
	r.Post("/login", h.login)
	r.Post("/onboard", h.onboard)
	r.Get("/onboard", h.canOnboard)

	// 2FA verification endpoint - not behind authentication
	r.Post("/2fa/verify", h.verify2FA)

	// Group for authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(h.server.IsAuthenticated)

		r.Post("/logout", h.logout)
		r.Get("/validate", h.validate)
		r.Patch("/user/{username}", h.updateUser)

		// 2FA routes that require authentication
		r.Get("/2fa/status", h.get2FAStatus)
		r.Post("/2fa/enable", h.enable2FA)
		r.Post("/2fa/disable", h.disable2FA)
	})
}

func (h authHandler) login(w http.ResponseWriter, r *http.Request) {
	var data domain.User
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.Wrap(err, "could not decode json"))
		return
	}

	user, err := h.service.Login(r.Context(), data.Username, data.Password)
	if err != nil {
		h.log.Error().Err(err).Msgf("Auth: Failed login attempt username: [%s] ip: %s", data.Username, r.RemoteAddr)
		h.encoder.StatusError(w, http.StatusForbidden, errors.New("could not login: bad credentials"))
		return
	}

	// Check if 2FA is enabled
	requires2FA, err := h.service.Get2FAStatus(r.Context(), data.Username)
	if err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.Wrap(err, "could not check 2FA status"))
		return
	}

	if requires2FA {
		// Create temporary session for 2FA verification
		session, err := h.cookieStore.Get(r, "user_session")
		if err != nil {
			h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not create session"))
			return
		}

		session.Values["temp_username"] = user.Username
		session.Values["awaiting_2fa"] = true
		session.Values["created"] = time.Now().Unix()

		if err := session.Save(r, w); err != nil {
			h.encoder.StatusError(w, http.StatusInternalServerError, errors.Wrap(err, "could not save session"))
			return
		}

		h.encoder.StatusResponse(w, http.StatusOK, map[string]bool{"requires2FA": true})
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
	session.Values["username"] = user.Username

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

	h.encoder.StatusResponse(w, http.StatusOK, map[string]bool{"requires2FA": false})
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
		session.Values["username"] = ""
		session.Values["awaiting_2fa"] = false
		session.Values["temp_username"] = ""

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
	session, ok := r.Context().Value("session").(*sessions.Session)
	if !ok {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get session from context"))
		return
	}

	if session != nil {
		h.log.Debug().Msgf("found user session: %+v", session)
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

func (h authHandler) get2FAStatus(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value("session").(*sessions.Session)
	if !ok {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get session from context"))
		return
	}

	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get username from session"))
		return
	}

	enabled, err := h.service.Get2FAStatus(r.Context(), username)
	if err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.Wrap(err, "could not get 2FA status"))
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, map[string]bool{"enabled": enabled})
}

func (h authHandler) enable2FA(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value("session").(*sessions.Session)
	if !ok {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get session from context"))
		return
	}

	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get username from session"))
		return
	}

	url, secret, err := h.service.Enable2FA(r.Context(), username)
	if err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.Wrap(err, "could not enable 2FA"))
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, map[string]string{
		"url":    url,
		"secret": secret,
	})
}

func (h authHandler) verify2FA(w http.ResponseWriter, r *http.Request) {
	session, err := h.cookieStore.Get(r, "user_session")
	if err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get session"))
		return
	}

	var data struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.Wrap(err, "could not decode json"))
		return
	}

	// Check if this is a login verification
	if awaiting2FA, ok := session.Values["awaiting_2fa"].(bool); ok && awaiting2FA {
		username, ok := session.Values["temp_username"].(string)
		if !ok || username == "" {
			h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get username from session"))
			return
		}

		h.log.Debug().
			Str("username", username).
			Str("code", data.Code).
			Bool("awaiting_2fa", awaiting2FA).
			Msg("attempting 2FA login verification")

		if err := h.service.Verify2FALogin(r.Context(), username, data.Code); err != nil {
			h.encoder.StatusError(w, http.StatusBadRequest, errors.Wrap(err, "could not verify 2FA code"))
			return
		}

		// 2FA verified, create full session
		session.Values["authenticated"] = true
		session.Values["username"] = username
		session.Values["awaiting_2fa"] = false
		session.Values["temp_username"] = ""
		session.Values["created"] = time.Now().Unix()

		// Set cookie options
		session.Options.HttpOnly = true
		session.Options.SameSite = http.SameSiteLaxMode
		session.Options.Path = h.config.BaseURL

		if r.Header.Get("X-Forwarded-Proto") == "https" {
			session.Options.Secure = true
			session.Options.SameSite = http.SameSiteStrictMode
		}

		if err := session.Save(r, w); err != nil {
			h.encoder.StatusError(w, http.StatusInternalServerError, errors.Wrap(err, "could not save session"))
			return
		}

		h.encoder.StatusResponseMessage(w, http.StatusOK, "2FA verification successful")
		return
	}

	// Regular 2FA setup verification
	usernameValue, exists := session.Values["username"]
	if !exists {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("username not found in session"))
		return
	}

	username, ok := usernameValue.(string)
	if !ok || username == "" {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("invalid username in session"))
		return
	}
	if err := h.service.Verify2FA(r.Context(), username, data.Code); err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.Wrap(err, "could not verify 2FA code"))
		return
	}

	h.encoder.StatusResponseMessage(w, http.StatusOK, "2FA successfully verified")
}

func (h authHandler) disable2FA(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value("session").(*sessions.Session)
	if !ok {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get session from context"))
		return
	}

	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get username from session"))
		return
	}

	if err := h.service.Disable2FA(r.Context(), username); err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.Wrap(err, "could not disable 2FA"))
		return
	}

	h.encoder.StatusResponseMessage(w, http.StatusOK, "2FA successfully disabled")
}
