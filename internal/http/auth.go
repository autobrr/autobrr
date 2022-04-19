package http

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path"

	"github.com/go-chi/chi"
	"github.com/gorilla/sessions"

	"github.com/autobrr/autobrr/internal/domain"
)

type authService interface {
	GetUserCount(ctx context.Context) (int, error)
	Login(ctx context.Context, username, password string) (*domain.User, error)
	CreateUser(ctx context.Context, user domain.CreateUserRequest) error
}

type authHandler struct {
	encoder encoder
	config  *domain.Config
	service authService

	cookieStore *sessions.CookieStore
}

func newAuthHandler(encoder encoder, config *domain.Config, cookieStore *sessions.CookieStore, service authService) *authHandler {
	return &authHandler{
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
	r.Get("/onboard/preferences", h.getOnboardingPreferences)
	r.Get("/validate", h.validate)
}

func (h authHandler) login(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.User
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		h.encoder.StatusResponse(ctx, w, nil, http.StatusBadRequest)
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

	session, _ := h.cookieStore.Get(r, "user_session")

	_, err := h.service.Login(ctx, data.Username, data.Password)
	if err != nil {
		h.encoder.StatusResponse(ctx, w, nil, http.StatusUnauthorized)
		return
	}

	// Set user as authenticated
	session.Values["authenticated"] = true
	session.Save(r, w)

	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}

func (h authHandler) logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	session, _ := h.cookieStore.Get(r, "user_session")

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Save(r, w)

	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}

func (h authHandler) onboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var data domain.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		h.encoder.StatusResponse(ctx, w, nil, http.StatusBadRequest)
		return
	}

	err := h.service.CreateUser(ctx, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// send empty response as ok
	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}

func (h authHandler) canOnboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userCount, err := h.service.GetUserCount(ctx)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if userCount > 0 {
		// send 503 service onboarding unavailable
		http.Error(w, "Onboarding unavailable", http.StatusServiceUnavailable)
		return
	}

	// send empty response as ok
	// (client can proceed with redirection to onboarding page)
	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}

func GetPreferredLogDir() (string, []string) {
	// 0. Check if ~/.config/autobrr/ is accessible to the current user
	// 1. Check if ~/.config/autobrr/log/ is accessible to the current user
	// 2. Check if golang can find the temp directory and use that.
	// 3. If neither 1 nor 2 were successful, bail with an error message.
	// NOTE: If neither $XDG_CONFIG_HOME nor $HOME are defined, UserConfigDir will return an error.
	configDir, err := os.UserConfigDir()

	// Keep track of errors, if any. Might help diagnose misconfiguration problems and such.
	var discoveredErrors []string
	if err == nil {
		// If we managed to find the user config directory,
		// then return ~/.config/autobrr/logs as the preferred log dir
		logDir := path.Join(configDir, "autobrr", "logs")
		return logDir, discoveredErrors
	} else {
		discoveredErrors = append(discoveredErrors, err.Error())
	}

	for _, dir := range [3]string{"/var/log/", "/opt/", os.TempDir()} {
		if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
			return path.Join(dir, "autobrr", "logs"), discoveredErrors
		} else {
			discoveredErrors = append(discoveredErrors, err.Error())
		}
	}

	return "", discoveredErrors
}

func (h authHandler) getOnboardingPreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userCount, err := h.service.GetUserCount(ctx)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if userCount > 0 {
		// send 503 service onboarding unavailable
		http.Error(w, "Onboarding unavailable", http.StatusServiceUnavailable)
		return
	}

	logDir, logErrors := GetPreferredLogDir()
	result := domain.OnboardingPreferences{
		LogDir:    logDir,
		LogErrors: logErrors,
	}

	h.encoder.StatusResponse(r.Context(), w, result, http.StatusOK)
}

func (h authHandler) validate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session, _ := h.cookieStore.Get(r, "user_session")

	// Check if user is authenticated
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Error(w, "Forbidden", http.StatusUnauthorized)
		return
	}

	// send empty response as ok
	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}
