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

	session, err := h.cookieStore.Get(r, "user_session")
	if err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get session"))
		return
	}

	if _, err = h.service.Login(ctx, data.Username, data.Password); err != nil {
		h.log.Error().Err(err).Msgf("Auth: Failed login attempt username: [%s] ip: %s", data.Username, ReadUserIP(r))
		h.encoder.StatusError(w, http.StatusUnauthorized, errors.New("could not login: bad credentials"))
		return
	}

	// Set user as authenticated
	session.Values["authenticated"] = true
	if err := session.Save(r, w); err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.Wrap(err, "could not save session"))
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h authHandler) logout(w http.ResponseWriter, r *http.Request) {
	session, err := h.cookieStore.Get(r, "user_session")
	if err != nil {
		h.log.Error().Err(err).Msg("could not get session")
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get session"))
		return
	}

	if session.IsNew {
		h.encoder.StatusResponse(w, http.StatusNoContent, nil)
		return
	}

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.Wrap(err, "could not save session"))
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h authHandler) onboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	session, _ := h.cookieStore.Get(r, "user_session")

	// Don't proceed if user is authenticated
	if authenticated, ok := session.Values["authenticated"].(bool); ok {
		if ok && authenticated {
			session.Values["authenticated"] = false
			session.Options.MaxAge = -1
			session.Save(r, w)

			h.encoder.StatusError(w, http.StatusForbidden, errors.New("active session found"))
			return
		}
	}

	var req domain.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.Wrap(err, "could not decode json"))
		return
	}

	if err := h.service.CreateUser(ctx, req); err != nil {
		h.encoder.StatusError(w, http.StatusForbidden, err)
		return
	}

	// send response as ok
	h.encoder.StatusResponseMessage(w, http.StatusOK, "user successfully created")
}

func (h authHandler) canOnboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userCount, err := h.service.GetUserCount(ctx)
	if err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get user count"))
		return
	}

	if userCount > 0 {
		h.encoder.StatusError(w, http.StatusForbidden, errors.New("onboarding unavailable"))
		return
	}

	// send empty response as ok
	// (client can proceed with redirection to onboarding page)
	h.encoder.NoContent(w)
}

func (h authHandler) validate(w http.ResponseWriter, r *http.Request) {
	session, err := h.cookieStore.Get(r, "user_session")
	if err != nil {
		h.encoder.StatusError(w, http.StatusInternalServerError, errors.New("could not get session"))
		return
	}

	// Check if user is authenticated
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		session.Values["authenticated"] = false
		session.Options.MaxAge = -1
		session.Save(r, w)
		h.encoder.StatusError(w, http.StatusUnauthorized, errors.New("forbidden: invalid session"))
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
