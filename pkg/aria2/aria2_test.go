// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package aria2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		host    string
		want    string
		wantErr bool
	}{
		{name: "url", host: "http://localhost:6800", want: "http://localhost:6800/jsonrpc"},
		{name: "url with trailing slash", host: "http://localhost:6800/", want: "http://localhost:6800/jsonrpc"},
		{name: "url with endpoint", host: "http://localhost:6800/jsonrpc", want: "http://localhost:6800/jsonrpc"},
		{name: "url with base path", host: "https://seedbox.net/aria2", want: "https://seedbox.net/aria2/jsonrpc"},
		{name: "url with base path and endpoint", host: "https://seedbox.net/aria2/jsonrpc", want: "https://seedbox.net/aria2/jsonrpc"},
		{name: "no scheme", host: "localhost:6800", want: "http://localhost:6800/jsonrpc"},
		{name: "websocket scheme", host: "ws://localhost:6800/jsonrpc", want: "http://localhost:6800/jsonrpc"},
		{name: "secure websocket scheme", host: "wss://seedbox.net/jsonrpc", want: "https://seedbox.net/jsonrpc"},
		{name: "empty", host: "", wantErr: true},
		{name: "unsupported scheme", host: "ftp://localhost:6800", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := BuildEndpoint(tt.host)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

type rpcCall struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
}

// newTestServer serves a single canned result and records every decoded request.
func newTestServer(t *testing.T, result any) (*httptest.Server, *[]rpcCall) {
	t.Helper()

	calls := make([]rpcCall, 0)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/jsonrpc", r.URL.Path)

		var call rpcCall
		require.NoError(t, json.NewDecoder(r.Body).Decode(&call))

		calls = append(calls, call)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":      1,
			"jsonrpc": "2.0",
			"result":  result,
		}))
	}))

	t.Cleanup(ts.Close)

	return ts, &calls
}

func TestClientAddURI(t *testing.T) {
	ts, calls := newTestServer(t, "2089b05ecca3d829")

	c, err := NewClient(Config{Host: ts.URL, Secret: "mock-secret"})
	require.NoError(t, err)

	gid, err := c.AddURI(context.Background(), []string{"magnet:?xt=urn:btih:mock"}, Options{"dir": "/downloads"})
	require.NoError(t, err)
	assert.Equal(t, "2089b05ecca3d829", gid)

	require.Len(t, *calls, 1)
	assert.Equal(t, "aria2.addUri", (*calls)[0].Method)
	assert.Equal(t, []any{
		"token:mock-secret",
		[]any{"magnet:?xt=urn:btih:mock"},
		map[string]any{"dir": "/downloads"},
	}, (*calls)[0].Params)
}

func TestClientAddURIWithoutSecret(t *testing.T) {
	ts, calls := newTestServer(t, "2089b05ecca3d829")

	c, err := NewClient(Config{Host: ts.URL})
	require.NoError(t, err)

	_, err = c.AddURI(context.Background(), []string{"magnet:?xt=urn:btih:mock"}, nil)
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Equal(t, []any{
		[]any{"magnet:?xt=urn:btih:mock"},
		map[string]any{},
	}, (*calls)[0].Params)
}

func TestClientAddTorrent(t *testing.T) {
	ts, calls := newTestServer(t, "2089b05ecca3d829")

	c, err := NewClient(Config{Host: ts.URL, Secret: "mock-secret"})
	require.NoError(t, err)

	gid, err := c.AddTorrent(context.Background(), []byte("d4:infod4:name4:mockee"), Options{"pause": "true"})
	require.NoError(t, err)
	assert.Equal(t, "2089b05ecca3d829", gid)

	require.Len(t, *calls, 1)
	assert.Equal(t, "aria2.addTorrent", (*calls)[0].Method)
	assert.Equal(t, []any{
		"token:mock-secret",
		"ZDQ6aW5mb2Q0Om5hbWU0Om1vY2tlZQ==",
		[]any{},
		map[string]any{"pause": "true"},
	}, (*calls)[0].Params)
}

func TestClientTellActive(t *testing.T) {
	ts, _ := newTestServer(t, []map[string]string{
		{"gid": "1", "status": "active", "totalLength": "100", "completedLength": "50"},
		{"gid": "2", "status": "active", "totalLength": "100", "completedLength": "100"},
		{"gid": "3", "status": "active", "totalLength": "0", "completedLength": "0"},
	})

	c, err := NewClient(Config{Host: ts.URL})
	require.NoError(t, err)

	active, err := c.TellActive(context.Background())
	require.NoError(t, err)
	require.Len(t, active, 3)

	assert.True(t, active[0].Downloading())
	assert.False(t, active[1].Downloading(), "a seeding torrent stays active")
	assert.True(t, active[2].Downloading(), "a magnet without metadata has no size yet")
}

func TestClientRPCError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":      1,
			"jsonrpc": "2.0",
			"error":   map[string]any{"code": 1, "message": "Unauthorized"},
		}))
	}))
	defer ts.Close()

	c, err := NewClient(Config{Host: ts.URL, Secret: "wrong"})
	require.NoError(t, err)

	_, err = c.GetVersion(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unauthorized")
}
