// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type stubAPIKeyService struct{}

func (stubAPIKeyService) List(context.Context) ([]domain.APIKey, error) { return nil, nil }
func (stubAPIKeyService) Store(context.Context, *domain.APIKey) error   { return nil }
func (stubAPIKeyService) Delete(context.Context, string) error          { return nil }
func (stubAPIKeyService) ValidateAPIKey(context.Context, string) bool   { return true }

func newAPIKeyTestRouter(cfg *domain.Config) chi.Router {
	h := newAPIKeyHandler(newEncoder(zerolog.Nop()), stubAPIKeyService{}, cfg)
	r := chi.NewRouter()
	r.Route("/keys", h.Routes)
	return r
}

func TestAPIKeyHandler_BlockedWhenAuthDisabled(t *testing.T) {
	cfg := &domain.Config{
		AuthDisabled:                true,
		AuthDisabledAcknowledgement: domain.AuthDisabledAcknowledgementValue,
	}
	r := newAPIKeyTestRouter(cfg)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/keys/"},
		{http.MethodPost, "/keys/"},
		{http.MethodDelete, "/keys/some-key"},
	}

	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusForbidden, rr.Code)
		})
	}
}

func TestAPIKeyHandler_AllowedWhenAuthEnabled(t *testing.T) {
	r := newAPIKeyTestRouter(&domain.Config{})

	req := httptest.NewRequest(http.MethodGet, "/keys/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
