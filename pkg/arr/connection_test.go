// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package arr_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobrr/autobrr/pkg/arr/lidarr"
	"github.com/autobrr/autobrr/pkg/arr/radarr"
	"github.com/autobrr/autobrr/pkg/arr/readarr"
	"github.com/autobrr/autobrr/pkg/arr/sonarr"
	"github.com/autobrr/autobrr/pkg/arr/sportarr"
	"github.com/autobrr/autobrr/pkg/arr/whisparr"

	"github.com/stretchr/testify/require"
)

func TestClientTestHTTPStatus(t *testing.T) {
	t.Parallel()

	clients := map[string]func(context.Context, string) error{
		"lidarr": func(ctx context.Context, host string) error {
			_, err := lidarr.New(lidarr.Config{Hostname: host}).Test(ctx)
			return err
		},
		"radarr": func(ctx context.Context, host string) error {
			_, err := radarr.New(radarr.Config{Hostname: host}).Test(ctx)
			return err
		},
		"readarr": func(ctx context.Context, host string) error {
			_, err := readarr.New(readarr.Config{Hostname: host}).Test(ctx)
			return err
		},
		"sonarr": func(ctx context.Context, host string) error {
			_, err := sonarr.New(sonarr.Config{Hostname: host}).Test(ctx)
			return err
		},
		"sportarr": func(ctx context.Context, host string) error {
			_, err := sportarr.New(sportarr.Config{Hostname: host}).Test(ctx)
			return err
		},
		"whisparr": func(ctx context.Context, host string) error {
			_, err := whisparr.New(whisparr.Config{Hostname: host}).Test(ctx)
			return err
		},
	}

	for name, test := range clients {
		t.Run(name, func(t *testing.T) {
			for _, status := range []int{http.StatusOK, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError} {
				t.Run(http.StatusText(status), func(t *testing.T) {
					srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(status)
						if status == http.StatusOK {
							_, _ = w.Write([]byte(`{"version":"4.0.1024"}`))
						} else {
							_, _ = w.Write([]byte(`{"message":"upstream error"}`))
						}
					}))
					defer srv.Close()

					err := test(t.Context(), srv.URL)
					switch status {
					case http.StatusOK:
						require.NoError(t, err)
					case http.StatusUnauthorized:
						require.EqualError(t, err, "unauthorized: bad credentials")
					default:
						require.ErrorContains(t, err, "unexpected status code")
					}
				})
			}
		})
	}
}
