// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/go-chi/chi/v5"
)

type releaseService interface {
	Find(ctx context.Context, query domain.ReleaseQueryParams) (res []*domain.Release, nextCursor int64, count int64, err error)
	FindRecent(ctx context.Context) (res []*domain.Release, err error)
	GetIndexerOptions(ctx context.Context) ([]string, error)
	Stats(ctx context.Context) (*domain.ReleaseStats, error)
	Delete(ctx context.Context) error
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
	r.Delete("/all", h.deleteReleases)
}

func (h releaseHandler) findReleases(w http.ResponseWriter, r *http.Request) {

	limitP := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitP)
	if err != nil && limitP != "" {
		h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]interface{}{
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
		h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]interface{}{
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
			h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]interface{}{
				"code":    "BAD_REQUEST_PARAMS",
				"message": "cursor parameter is invalid",
			})
		}
		return
	}

	u, err := url.Parse(r.URL.String())
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]interface{}{
			"code":    "BAD_REQUEST_PARAMS",
			"message": "indexer parameter is invalid",
		})
		return
	}
	vals := u.Query()
	indexer := vals["indexer"]

	pushStatus := r.URL.Query().Get("push_status")
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

	releases, nextCursor, count, err := h.service.Find(r.Context(), query)
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"code":    "INTERNAL_SERVER_ERROR",
			"message": err.Error(),
		})
		return
	}

	ret := struct {
		Data       []*domain.Release `json:"data"`
		NextCursor int64             `json:"next_cursor"`
		Count      int64             `json:"count"`
	}{
		Data:       releases,
		NextCursor: nextCursor,
		Count:      count,
	}

	h.encoder.StatusResponse(w, http.StatusOK, ret)
}

func (h releaseHandler) findRecentReleases(w http.ResponseWriter, r *http.Request) {

	releases, err := h.service.FindRecent(r.Context())
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"code":    "INTERNAL_SERVER_ERROR",
			"message": err.Error(),
		})
		return
	}

	ret := struct {
		Data []*domain.Release `json:"data"`
	}{
		Data: releases,
	}

	h.encoder.StatusResponse(w, http.StatusOK, ret)
}

func (h releaseHandler) getIndexerOptions(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetIndexerOptions(r.Context())
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusInternalServerError, map[string]interface{}{
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
		h.encoder.StatusResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"code":    "INTERNAL_SERVER_ERROR",
			"message": err.Error(),
		})
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, stats)
}

func (h releaseHandler) deleteReleases(w http.ResponseWriter, r *http.Request) {
	err := h.service.Delete(r.Context())
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"code":    "INTERNAL_SERVER_ERROR",
			"message": err.Error(),
		})
		return
	}

	h.encoder.NoContent(w)
}
