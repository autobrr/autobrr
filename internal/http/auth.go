package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/gorilla/sessions"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/domain"
)

type authService interface {
	Login(username, password string) (*domain.User, error)
}

type authHandler struct {
	encoder     encoder
	authService authService
}

var (
	// key will only be valid as long as it's running.
	key   = []byte(config.Config.SessionSecret)
	store = sessions.NewCookieStore(key)
)

func (h authHandler) Routes(r chi.Router) {
	r.Post("/login", h.login)
	r.Post("/logout", h.logout)
	r.Get("/test", h.test)
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

	session, _ := store.Get(r, "user_session")

	_, err := h.authService.Login(data.Username, data.Password)
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

	session, _ := store.Get(r, "user_session")

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Save(r, w)

	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}

func (h authHandler) test(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session, _ := store.Get(r, "user_session")

	// Check if user is authenticated
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// send empty response as ok
	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}
