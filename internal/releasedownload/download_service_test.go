// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package releasedownload

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type mockIndexerRepo struct{}

func (m *mockIndexerRepo) FindByID(_ context.Context, id int) (*domain.Indexer, error) {
	return &domain.Indexer{ID: int64(id), Identifier: "mock-feed"}, nil
}

type mockProxyService struct{}

func (m *mockProxyService) FindByID(_ context.Context, id int64) (*domain.Proxy, error) {
	return &domain.Proxy{ID: id}, nil
}

func TestDownloadService_ResolveMagnetURI(t *testing.T) {
	const magnetURI = "magnet:?xt=urn:btih:deadbeef"

	tests := []struct {
		name    string
		handler http.HandlerFunc
		// magnetURI on the release before resolving, %s is replaced with the test server url
		before string
		want   string
	}{
		{
			name:   "empty is left alone",
			before: "",
			want:   "",
		},
		{
			name:   "magnet is left alone",
			before: magnetURI,
			want:   magnetURI,
		},
		{
			name: "redirect to a magnet is followed",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, magnetURI, http.StatusFound)
			},
			before: "%s/dl/mock",
			want:   magnetURI,
		},
		{
			name: "magnet in the body is used",
			handler: func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, magnetURI+"\n")
			},
			before: "%s/dl/mock",
			want:   magnetURI,
		},
		{
			name: "a response that is not a magnet is discarded",
			handler: func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, "d8:announce20:https://fake-feed.come")
			},
			before: "%s/dl/mock",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			magnet := tt.before

			if tt.handler != nil {
				srv := httptest.NewServer(tt.handler)
				defer srv.Close()

				magnet = strings.Replace(tt.before, "%s", srv.URL, 1)
			}

			svc := NewDownloadService(zerolog.New(io.Discard), &mockIndexerRepo{}, &mockProxyService{})

			rls := domain.NewRelease(domain.IndexerMinimal{ID: 1, Name: "Mock Feed", Identifier: "mock-feed"})
			rls.MagnetURI = magnet

			err := svc.ResolveMagnetURI(t.Context(), rls)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, rls.MagnetURI)
		})
	}
}
