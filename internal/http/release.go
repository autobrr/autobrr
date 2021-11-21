package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/go-chi/chi"
)

type releaseService interface {
	Find(ctx context.Context, query domain.QueryParams) (res []domain.Release, nextCursor int64, err error)
	Stats(ctx context.Context) (*domain.ReleaseStats, error)
	Store(release domain.Release) error
}

type releaseHandler struct {
	encoder encoder
	service releaseService
}

func newReleaseHandler(encoder encoder, service releaseService) *releaseHandler {
	return &releaseHandler{
		encoder: encoder,
		service: service,
	}
}

func (h releaseHandler) Routes(r chi.Router) {
	r.Get("/", h.findReleases)
	r.Get("/stats", h.getStats)
}

func (h releaseHandler) findReleases(w http.ResponseWriter, r *http.Request) {

	limitP := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitP)
	if err != nil && limitP != "" {
		h.encoder.StatusResponse(r.Context(), w, map[string]interface{}{
			"code":    "BAD_REQUEST_PARAMS",
			"message": "limit parameter is invalid",
		}, http.StatusBadRequest)
	}
	if limit == 0 {
		limit = 20
	}

	cursorP := r.URL.Query().Get("cursor")
	cursor, err := strconv.Atoi(cursorP)
	if err != nil && cursorP != "" {
		h.encoder.StatusResponse(r.Context(), w, map[string]interface{}{
			"code":    "BAD_REQUEST_PARAMS",
			"message": "cursor parameter is invalid",
		}, http.StatusBadRequest)
	}

	//r.URL.Query().
	//f, _ := url.ParseQuery(r.URL.RawQuery)
	//log.Info().Msgf("url", f)

	query := domain.QueryParams{
		Limit:  uint64(limit),
		Cursor: uint64(cursor),
		Sort:   nil,
		//Filter: "",
	}

	releases, nextCursor, err := h.service.Find(r.Context(), query)
	if err != nil {
		h.encoder.StatusNotFound(r.Context(), w)
		return
	}

	ret := struct {
		Data       []domain.Release `json:"data"`
		NextCursor int64            `json:"next_cursor"`
	}{
		Data:       releases,
		NextCursor: nextCursor,
	}

	h.encoder.StatusResponse(r.Context(), w, ret, http.StatusOK)
}

func (h releaseHandler) getStats(w http.ResponseWriter, r *http.Request) {

	stats, err := h.service.Stats(r.Context())
	if err != nil {
		h.encoder.StatusNotFound(r.Context(), w)
		return
	}

	//ret := struct {
	//	Data       []domain.Release `json:"data"`
	//	NextCursor int64            `json:"next_cursor"`
	//}{
	//	Data:       releases,
	//	NextCursor: nextCursor,
	//}

	//h.encoder.StatusResponse(ctx, w, releases, http.StatusOK)
	h.encoder.StatusResponse(r.Context(), w, stats, http.StatusOK)
}
