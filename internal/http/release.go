// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
)

type releaseService interface {
	Find(ctx context.Context, query domain.ReleaseQueryParams) (*domain.FindReleasesResponse, error)
	Get(ctx context.Context, req *domain.GetReleaseRequest) (*domain.Release, error)
	GetIndexerOptions(ctx context.Context) ([]string, error)
	Stats(ctx context.Context) (*domain.ReleaseStats, error)
	Delete(ctx context.Context, req *domain.DeleteReleaseRequest) error
	Retry(ctx context.Context, req *domain.ReleaseActionRetryReq) error
	ProcessManual(ctx context.Context, req *domain.ReleaseProcessReq) error

	StoreReleaseProfileDuplicate(ctx context.Context, profile *domain.DuplicateReleaseProfile) error
	FindDuplicateReleaseProfiles(ctx context.Context) ([]*domain.DuplicateReleaseProfile, error)
	DeleteReleaseProfileDuplicate(ctx context.Context, id int64) error

	ListCleanupJobs(ctx context.Context) ([]*domain.ReleaseCleanupJob, error)
	GetCleanupJob(ctx context.Context, id int) (*domain.ReleaseCleanupJob, error)
	StoreCleanupJob(ctx context.Context, job *domain.ReleaseCleanupJob) error
	UpdateCleanupJob(ctx context.Context, job *domain.ReleaseCleanupJob) error
	DeleteCleanupJob(ctx context.Context, id int) error
	ToggleCleanupJobEnabled(ctx context.Context, id int, enabled bool) error
	ForceRunCleanupJob(ctx context.Context, id int) error
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
	r.Get("/recent", h.findRecentReleases)
	r.Get("/stats", h.getStats)
	r.Get("/indexers", h.getIndexerOptions)
	r.Delete("/", h.deleteReleases)

	//r.Post("/process", h.retryAction)

	r.Route("/{releaseID}", func(r chi.Router) {
		r.Get("/", h.getReleaseByID)
		r.Post("/actions/{actionStatusID}/retry", h.retryAction)
	})

	r.Route("/profiles/duplicate", func(r chi.Router) {
		r.Get("/", h.findReleaseProfileDuplicate)
		r.Post("/", h.storeReleaseProfileDuplicate)

		r.Delete("/{profileId}", h.deleteReleaseProfileDuplicate)
	})

	r.Route("/cleanup-jobs", func(r chi.Router) {
		r.Get("/", h.listCleanupJobs)
		r.Post("/", h.storeCleanupJob)

		r.Route("/{jobID}", func(r chi.Router) {
			r.Get("/", h.getCleanupJob)
			r.Put("/", h.updateCleanupJob)
			r.Delete("/", h.deleteCleanupJob)
			r.Patch("/enabled", h.toggleCleanupJobEnabled)
			r.Post("/run", h.forceRunCleanupJob)
		})
	})
}

func (h releaseHandler) findReleases(w http.ResponseWriter, r *http.Request) {
	limitP := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitP)
	if err != nil && limitP != "" {
		h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]any{
			"code":    "BAD_REQUEST_PARAMS",
			"message": "limit parameter is invalid",
		})
		return
	}
	if limit == 0 {
		limit = 20
	}

	offsetP := r.URL.Query().Get("offset")
	offset, err := strconv.Atoi(offsetP)
	if err != nil && offsetP != "" {
		h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]any{
			"code":    "BAD_REQUEST_PARAMS",
			"message": "offset parameter is invalid",
		})
		return
	}

	cursorP := r.URL.Query().Get("cursor")
	cursor := 0
	if cursorP != "" {
		cursor, err = strconv.Atoi(cursorP)
		if err != nil && cursorP != "" {
			h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]any{
				"code":    "BAD_REQUEST_PARAMS",
				"message": "cursor parameter is invalid",
			})
		}
		return
	}

	u, err := url.Parse(r.URL.String())
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]any{
			"code":    "BAD_REQUEST_PARAMS",
			"message": "indexer parameter is invalid",
		})
		return
	}
	vals := u.Query()
	indexer := vals["indexer"]

	pushStatus := r.URL.Query().Get("push_status")
	if pushStatus != "" {
		if !domain.ValidReleasePushStatus(pushStatus) {
			h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]any{
				"code":    "BAD_REQUEST_PARAMS",
				"message": fmt.Sprintf("push_status parameter is of invalid type: %v", pushStatus),
			})
			return
		}
	}

	search := r.URL.Query().Get("q")

	query := domain.ReleaseQueryParams{
		Limit:  uint64(limit),
		Offset: uint64(offset),
		Cursor: uint64(cursor),
		Sort:   nil,
		Filters: struct {
			Indexers   []string
			PushStatus string
		}{Indexers: indexer, PushStatus: pushStatus},
		Search: search,
	}

	resp, err := h.service.Find(r.Context(), query)
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusInternalServerError, map[string]any{
			"code":    "INTERNAL_SERVER_ERROR",
			"message": err.Error(),
		})
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, resp)
}

func (h releaseHandler) findRecentReleases(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.Find(r.Context(), domain.ReleaseQueryParams{Limit: 10})
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusInternalServerError, map[string]any{
			"code":    "INTERNAL_SERVER_ERROR",
			"message": err.Error(),
		})
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, resp)
}

func (h releaseHandler) getReleaseByID(w http.ResponseWriter, r *http.Request) {
	releaseID, err := strconv.Atoi(chi.URLParam(r, "releaseID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	release, err := h.service.Get(r.Context(), &domain.GetReleaseRequest{Id: releaseID})
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("could not find release with id %d", releaseID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, release)
}

func (h releaseHandler) getIndexerOptions(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetIndexerOptions(r.Context())
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusInternalServerError, map[string]any{
			"code":    "INTERNAL_SERVER_ERROR",
			"message": err.Error(),
		})
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, stats)
}

func (h releaseHandler) getStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.Stats(r.Context())
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusInternalServerError, map[string]any{
			"code":    "INTERNAL_SERVER_ERROR",
			"message": err.Error(),
		})
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, stats)
}

func (h releaseHandler) deleteReleases(w http.ResponseWriter, r *http.Request) {
	req := domain.DeleteReleaseRequest{}

	olderThanParam := r.URL.Query().Get("olderThan")
	if olderThanParam != "" {
		duration, err := strconv.Atoi(olderThanParam)
		if err != nil {
			h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]any{
				"code":    "BAD_REQUEST_PARAMS",
				"message": "olderThan parameter is invalid",
			})
			return
		}
		req.OlderThan = duration
	}

	indexers := r.URL.Query()["indexer"]
	if len(indexers) > 0 {
		req.Indexers = indexers
	}

	releaseStatuses := r.URL.Query()["releaseStatus"]
	var filteredStatuses []string
	for _, status := range releaseStatuses {
		if domain.ValidDeletableReleasePushStatus(status) {
			filteredStatuses = append(filteredStatuses, status)
		} else {
			h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]any{
				"code":    "INVALID_RELEASE_STATUS",
				"message": "releaseStatus contains invalid value",
			})
			return
		}
	}
	req.ReleaseStatuses = filteredStatuses

	if err := h.service.Delete(r.Context(), &req); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h releaseHandler) process(w http.ResponseWriter, r *http.Request) {
	var req *domain.ReleaseProcessReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if req.IndexerIdentifier == "" {
		h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]any{
			"code":    "VALIDATION_ERROR",
			"message": "field indexer_identifier empty",
		})
	}

	if len(req.AnnounceLines) == 0 {
		h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]any{
			"code":    "VALIDATION_ERROR",
			"message": "field announce_lines empty",
		})
	}

	err = h.service.ProcessManual(r.Context(), req)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h releaseHandler) retryAction(w http.ResponseWriter, r *http.Request) {
	releaseID, err := strconv.Atoi(chi.URLParam(r, "releaseID"))
	if err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, err)
		return
	}

	actionStatusId, err := strconv.Atoi(chi.URLParam(r, "actionStatusID"))
	if err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, err)
		return
	}

	req := &domain.ReleaseActionRetryReq{
		ReleaseId:      releaseID,
		ActionStatusId: actionStatusId,
	}

	if err := h.service.Retry(r.Context(), req); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h releaseHandler) storeReleaseProfileDuplicate(w http.ResponseWriter, r *http.Request) {
	var data *domain.DuplicateReleaseProfile

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.StoreReleaseProfileDuplicate(r.Context(), data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusCreatedData(w, data)
}

func (h releaseHandler) findReleaseProfileDuplicate(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.service.FindDuplicateReleaseProfiles(r.Context())
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"code":    "INTERNAL_SERVER_ERROR",
			"message": err.Error(),
		})
		return
	}

	//ret := struct {
	//	Data       []*domain.DuplicateReleaseProfile `json:"data"`
	//}{
	//	Data:       profiles,
	//}

	h.encoder.StatusResponse(w, http.StatusOK, profiles)
}

func (h releaseHandler) deleteReleaseProfileDuplicate(w http.ResponseWriter, r *http.Request) {
	//profileIdParam := chi.URLParam(r, "releaseId")

	profileId, err := strconv.Atoi(chi.URLParam(r, "profileId"))
	if err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.service.DeleteReleaseProfileDuplicate(r.Context(), int64(profileId)); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

// Cleanup job handlers

func (h releaseHandler) listCleanupJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.service.ListCleanupJobs(r.Context())
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, jobs)
}

func (h releaseHandler) getCleanupJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := strconv.Atoi(chi.URLParam(r, "jobID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	job, err := h.service.GetCleanupJob(r.Context(), jobID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("could not find cleanup job with id %d", jobID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, job)
}

func (h releaseHandler) storeCleanupJob(w http.ResponseWriter, r *http.Request) {
	var data *domain.ReleaseCleanupJob
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.StoreCleanupJob(r.Context(), data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, data)
}

func (h releaseHandler) updateCleanupJob(w http.ResponseWriter, r *http.Request) {
	var data *domain.ReleaseCleanupJob
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.UpdateCleanupJob(r.Context(), data); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("could not find cleanup job with id %d", data.ID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, data)
}

func (h releaseHandler) deleteCleanupJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := strconv.Atoi(chi.URLParam(r, "jobID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.DeleteCleanupJob(r.Context(), jobID); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("could not find cleanup job with id %d", jobID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h releaseHandler) toggleCleanupJobEnabled(w http.ResponseWriter, r *http.Request) {
	jobID, err := strconv.Atoi(chi.URLParam(r, "jobID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	var data struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.ToggleCleanupJobEnabled(r.Context(), jobID, data.Enabled); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("could not find cleanup job with id %d", jobID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h releaseHandler) forceRunCleanupJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := strconv.Atoi(chi.URLParam(r, "jobID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.ForceRunCleanupJob(r.Context(), jobID); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("could not find cleanup job with id %d", jobID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}
