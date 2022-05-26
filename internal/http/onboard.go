package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"

	"github.com/autobrr/autobrr/internal/domain"
)

//type authService interface {
//	GetUserCount(ctx context.Context) (int, error)
//	Login(ctx context.Context, username, password string) (*domain.User, error)
//	CreateUser(ctx context.Context, user domain.CreateUserRequest) error
//}

type onboardHandler struct {
	encoder encoder
	config  *domain.Config
	service authService
}

func newOnboardHandler(encoder encoder, config *domain.Config, service authService) *onboardHandler {
	return &onboardHandler{
		encoder: encoder,
		config:  config,
		service: service,
	}
}

func (h onboardHandler) Routes(r chi.Router) {
	r.Post("/", h.onboard)
	r.Get("/", h.canOnboard)
	r.Get("/preferences", h.getOnboardingPreferences)
}

func (h onboardHandler) onboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var data domain.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		h.encoder.StatusResponse(ctx, w, nil, http.StatusBadRequest)
		return
	}

	err := h.config.UpdateConfig(data.LogPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	err = h.service.CreateUser(ctx, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// send empty response as ok
	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}

func (h onboardHandler) canOnboard(w http.ResponseWriter, r *http.Request) {
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

func (h onboardHandler) getOnboardingPreferences(w http.ResponseWriter, r *http.Request) {
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

	logDir, logErrors := h.config.GetPreferredLogDir()
	result := domain.OnboardingPreferences{
		LogDir:    logDir,
		LogErrors: logErrors,
	}

	h.encoder.StatusResponse(r.Context(), w, result, http.StatusOK)
}
