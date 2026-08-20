// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package lunasea

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_rewriteWebhookURL(t *testing.T) {
	t.Parallel()

	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "radarr",
			args: args{url: "https://notify.lunasea.app/v1/radarr/user/mock-id"},
			want: "https://notify.lunasea.app/v1/custom/user/mock-id",
		},
		{
			name: "sonarr",
			args: args{url: "https://notify.lunasea.app/v1/sonarr/device/mock-id"},
			want: "https://notify.lunasea.app/v1/custom/device/mock-id",
		},
		{
			name: "custom",
			args: args{url: "https://notify.lunasea.app/v1/custom/user/mock-id"},
			want: "https://notify.lunasea.app/v1/custom/user/mock-id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, rewriteWebhookURL(tt.args.url), "rewriteWebhookURL(%v)", tt.args.url)
		})
	}
}

func TestClient_SendMessage(t *testing.T) {
	t.Parallel()

	var got Message

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{WebhookURL: server.URL, Name: "mock"})

	message := &Message{
		Title: "Push Approved",
		Body:  "New release: Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
		Image: "https://mock.local/logo.png",
	}

	assert.NoError(t, client.SendMessage(t.Context(), message))
	assert.Equal(t, *message, got)
}
