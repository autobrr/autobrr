// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package bark

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobrr/autobrr/internal/meta"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_SendMessage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/push", r.URL.Path)
		assert.Equal(t, "application/json; charset=utf-8", r.Header.Get("Content-Type"))
		assert.Equal(t, meta.GetUserAgent(), r.Header.Get("User-Agent"))

		var body map[string]any
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]any{
			"device_key": "mock-key",
			"title":      "Push Approved",
			"body":       "New release: Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP\n",
			"sound":      "minuet",
			"group":      "autobrr",
			"level":      "timeSensitive",
			"icon":       "https://example.com/logo.png",
		}, body)

		_, _ = w.Write([]byte(`{"code":200,"message":"success","timestamp":1700000000}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Host: server.URL, DeviceKey: "mock-key", Name: "mock"})

	message := &Message{
		Title: "Push Approved",
		Body:  "New release: Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP\n",
		Sound: "minuet",
		Group: "autobrr",
		Level: "timeSensitive",
		Icon:  "https://example.com/logo.png",
	}

	assert.NoError(t, client.SendMessage(t.Context(), message))
}

func TestClient_SendMessage_SelfHosted(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/bark/push", r.URL.Path)

		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "user", user)
		assert.Equal(t, "pass", pass)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host := "http://user:pass@" + server.Listener.Addr().String() + "/bark/"
	client := NewSender(zerolog.New(io.Discard), Config{Host: host, DeviceKey: "mock-key", Name: "mock"})

	assert.NoError(t, client.SendMessage(t.Context(), &Message{Title: "Test", Body: "autobrr goes brr!!"}))
}

func TestClient_SendMessage_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErrs   []string
	}{
		{
			name:       "unknown device key",
			statusCode: http.StatusBadRequest,
			body:       `{"code":400,"message":"failed to get device token: failed to get [mock-key] device token from database","timestamp":1700000000}`,
			wantErrs:   []string{"check bark device key", "unexpected status: 400", "failed to get device token"},
		},
		{
			name:       "basic auth rejected",
			statusCode: http.StatusTeapot,
			body:       `I'm a teapot`,
			wantErrs:   []string{"check bark server basic auth credentials", "unexpected status: 418", "I'm a teapot"},
		},
		{
			name:       "apns failure",
			statusCode: http.StatusInternalServerError,
			body:       `{"code":500,"message":"push failed: BadDeviceToken","timestamp":1700000000}`,
			wantErrs:   []string{"unexpected status: 500", "push failed: BadDeviceToken"},
		},
		{
			name:       "empty body",
			statusCode: http.StatusNotFound,
			wantErrs:   []string{"unexpected status: 404"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewSender(zerolog.New(io.Discard), Config{Host: server.URL, DeviceKey: "mock-key", Name: "mock"})

			err := client.SendMessage(t.Context(), &Message{Title: "Test", Body: "autobrr goes brr!!"})
			require.Error(t, err)
			for _, want := range tt.wantErrs {
				assert.ErrorContains(t, err, want)
			}
		})
	}
}
