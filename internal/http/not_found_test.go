// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

type notFoundListService struct {
	listService
}

func (notFoundListService) RefreshList(context.Context, int64) error {
	return domain.ErrRecordNotFound
}

type historyMatrixService struct {
	matrixService
	err error
}

func (s historyMatrixService) GetMessageHistory(context.Context, int64, string) ([]domain.MatrixMessage, error) {
	return nil, s.err
}

type notFoundReleaseService struct {
	releaseService
}

func (notFoundReleaseService) DeleteReleaseProfileDuplicate(context.Context, int64) error {
	return domain.ErrRecordNotFound
}

func TestHandlers_StatusForServiceErrors(t *testing.T) {
	tests := []struct {
		name   string
		routes func(chi.Router)
		method string
		path   string
		want   int
	}{
		{
			name:   "webhook trigger for missing list",
			routes: newWebhookHandler(encoder{}, notFoundListService{}).Routes,
			method: http.MethodPost,
			path:   "/lists/trigger/1",
			want:   http.StatusNotFound,
		},
		{
			name:   "matrix history for missing room",
			routes: newMatrixHandler(encoder{}, nil, historyMatrixService{err: errors.Wrap(domain.ErrMatrixRoomNotFound, "room: '#x'")}).Routes,
			method: http.MethodGet,
			path:   "/network/1/room/history?room=x",
			want:   http.StatusNotFound,
		},
		{
			name:   "matrix history failure",
			routes: newMatrixHandler(encoder{}, nil, historyMatrixService{err: errors.New("boom")}).Routes,
			method: http.MethodGet,
			path:   "/network/1/room/history?room=x",
			want:   http.StatusInternalServerError,
		},
		{
			name:   "delete missing duplicate release profile",
			routes: newReleaseHandler(encoder{}, notFoundReleaseService{}).Routes,
			method: http.MethodDelete,
			path:   "/profiles/duplicate/1",
			want:   http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Use(asCaller(testUser))
			r.Route("/", tt.routes)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			assert.Equal(t, tt.want, rec.Code, rec.Body.String())
		})
	}
}
