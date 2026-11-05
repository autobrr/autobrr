// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	loginLimiter   *clientRateLimiter
}

// Only failed logins draw from the per-client bucket (keyed by
// Server.limiterKey), so someone who knows the password is never locked out by
// their own logins.
const (
	loginFailureBurst    = 5
	loginFailureInterval = 20 * time.Second
)

const (
	loginBodyLimit       = 4 << 10
	loginBodyReadTimeout = 10 * time.Second
)

func newAuthHandler(encoder encoder, log zerolog.Logger, server *Server, config *domain.Config, sessionManager *scs.SessionManager, service authService, oidcService oidcService) *authHandler {
	h := &authHandler{
		log:            log,
		encoder:        encoder,
		config:         config,
		service:        service,
		oidcService:    oidcService,
		sessionManager: sessionManager,
		server:         server,
		loginLimiter:   newClientRateLimiter(loginFailureInterval, loginFailureBurst),
	}

	if h.oidcService.IsEnabled() {
		h.oidcHandler = NewOIDCHandler(encoder, log, config, sessionManager, oidcService, server.limiterKey)
	}

	return h
}

func (h *authHandler) Routes(r chi.Router) {
	// One login at a time bounds argon2 memory (64 MiB per comparison) and makes
	// the failure limiter's check-then-record in login atomic. The backlog is
	// deep because rate limited requests are rejected before hashing, so a
	// guessing client cannot hold the queue and lock everyone else out. The body
	// is read before taking the slot, otherwise a client that stalls mid-body
	// holds it indefinitely.
	r.With(h.bufferLoginBody, middleware.ThrottleBacklog(1, 64, 10*time.Second)).Post("/login", h.login)
	r.Post("/onboard", h.onboard)
	r.Get("/onboard", h.canOnboard)

	if h.config.AuthCapabilities().OIDCLogin && h.oidcHandler != nil {
		r.Route("/oidc", h.oidcHandler.Routes)
	}

	// Group for authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(h.server.IsAuthenticated)

		r.Post("/logout", h.logout)
		r.Get("/validate", h.validate)
		r.With(requireUser).Patch("/user/{username}", h.updateUser)
	})
}

// bufferLoginBody reads the whole request body into memory under a size and
// time bound. The read deadline is cleared only on success, because a deadline
// left on the connection cancels the request context mid-handler. On failure it
// must stay expired: net/http drains the unread body before writing the
// response, and without a deadline that drain blocks on a stalled client.
func (h *authHandler) bufferLoginBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)
		_ = rc.SetReadDeadline(time.Now().Add(loginBodyReadTimeout))

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, loginBodyLimit))
		if err != nil {
			h.encoder.StatusError(w, http.StatusBadRequest, errors.Wrap(err, "could not read request body"))
			return
		}

		_ = rc.SetReadDeadline(time.Time{})

		r.Body = io.NopCloser(bytes.NewReader(body))

		next.ServeHTTP(w, r)
	})
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	if !h.config.AuthCapabilities().PasswordLogin {
		h.encoder.StatusError(w, http.StatusForbidden, errors.New("password login is unavailable in %s authentication mode", h.config.AuthMode()))
		return
	}

	ip := h.server.limiterKey(r)
	if wait := h.loginLimiter.retryAfter(ip, time.Now()); wait > 0 {
		h.log.Error().Str("remote_addr", r.RemoteAddr).Dur("retry_after", wait).Msg("login rate limited after repeated failures")
		setRetryAfter(w, wait)
		h.encoder.StatusError(w, http.StatusTooManyRequests, errors.New("too many failed login attempts"))
		return
	}

	ctx := r.Context()
	var data domain.UserLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.Wrap(err, "could not decode json"))
		return
	}

	if _, err := h.service.Login(ctx, data.Username, data.Password); err != nil {
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			h.log.Error().Err(err).Str("username", data.Username).Str("remote_addr", r.RemoteAddr).Msg("could not check login credentials")
			h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not login"))
			return
		}

		h.loginLimiter.record(ip, time.Now())
		h.log.Error().Err(err).Str("username", data.Username).Str("remote_addr", r.RemoteAddr).Msg("failed login attempt")
		h.encoder.StatusError(w, http.StatusForbidden, errors.New("could not login: bad credentials"))
		return
	}

	h.loginLimiter.reset(ip)

	if err := h.sessionManager.RenewToken(ctx); err != nil {
		h.log.Error().Err(err).Str("username", data.Username).Str("remote_addr", r.RemoteAddr).Msg("failed to renew session token")
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
	if caps := h.config.AuthCapabilities(); !caps.PasswordLogin && !caps.OIDCLogin {
		h.encoder.StatusError(w, http.StatusForbidden, errors.New("logout is unavailable in %s authentication mode", h.config.AuthMode()))
		return
	}

	if err := h.sessionManager.Destroy(r.Context()); err != nil {
		h.log.Error().Err(err).Str("remote_addr", r.RemoteAddr).Msg("failed to destroy session")
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
	if !h.config.AuthCapabilities().Onboarding {
		return http.StatusServiceUnavailable, errors.New("onboarding unavailable in %s authentication mode", h.config.AuthMode())
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

// validate returns the caller IsAuthenticated resolved; requests it rejects never get here.
func (h *authHandler) validate(w http.ResponseWriter, r *http.Request) {
	caller, ok := callerFromContext(r.Context())
	if !ok {
		h.encoder.StatusError(w, http.StatusForbidden, errors.New("no authenticated caller"))
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, caller)
}

func (h *authHandler) updateUser(w http.ResponseWriter, r *http.Request) {
	if !h.config.AuthCapabilities().AccountManagement {
		h.encoder.StatusError(w, http.StatusForbidden, errors.New("account management is unavailable in %s authentication mode", h.config.AuthMode()))
		return
	}

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

	if data.PasswordNew != "" {
		h.revokeOtherSessions(r.Context())
	}

	// send response as ok
	h.encoder.StatusResponseMessage(w, http.StatusOK, "user successfully updated")
}

// revokeOtherSessions destroys every session except the current one, so a
// password change invalidates sessions an attacker may already hold. Iterate
// stops at the first callback error, so a failed Destroy is logged and
// swallowed to keep the remaining sessions from surviving.
func (h *authHandler) revokeOtherSessions(ctx context.Context) {
	currentToken := h.sessionManager.Token(ctx)

	if err := h.sessionManager.Iterate(ctx, func(sessionCtx context.Context) error {
		if h.sessionManager.Token(sessionCtx) == currentToken {
			return nil
		}

		if err := h.sessionManager.Destroy(sessionCtx); err != nil {
			h.log.Error().Err(err).Msg("could not destroy session after password change")
		}

		return nil
	}); err != nil {
		h.log.Error().Err(err).Msg("could not revoke other sessions after password change")
	}
}
