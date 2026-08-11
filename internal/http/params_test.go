// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func Test_parseURLParamInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{name: "valid", value: "42", want: 42},
		{name: "negative", value: "-1", want: -1},
		{name: "malformed", value: "not-an-integer", wantErr: true},
		{name: "empty", value: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("indexerID", tt.value)

			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			got, err := parseURLParamInt(r, "indexerID")
			if tt.wantErr {
				assert.EqualError(t, err, "indexerID parameter is invalid")
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_parseQueryParamInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   string
		want    int
		wantErr bool
	}{
		{name: "valid", query: "?limit=42", want: 42},
		{name: "missing_uses_default", query: "", want: 20},
		{name: "malformed", query: "?limit=not-an-integer", wantErr: true},
		{name: "negative", query: "?limit=-1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/"+tt.query, nil)

			got, err := parseQueryParamInt(r, "limit", 20)
			if tt.wantErr {
				assert.EqualError(t, err, "limit parameter is invalid")
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
