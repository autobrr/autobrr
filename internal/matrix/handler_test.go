// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package matrix

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"

	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRoomAlias = "#blu-announce:matrix.watchtower.pm"
	testRoomID    = "!announce:matrix.watchtower.pm"
	testAnnouncer = "@blubot:matrix.watchtower.pm"
	// blubot wraps every announce over three lines of one message
	testAnnounce = "[Blutopia] [NEW] bearfish has uploaded True Story with Hamish & Andy S02 1080p STAN WEB-DL AAC 2.0 H.265-NYXIS grab it now!\n[Category: TV Show] [Type: WEB-DL] [Size: 4.75 GiB] [Freeleech: 0%] [DoubleUpload: No] [Internal: No] [PersonalRelease: Yes] [TMDB: 72242] [IMDB: 6974706] [TMDB vote average: 8.5] [TMDB vote count: 2]\n[Link: https://blutopia.cc/torrents/402579]"
)

type fakeReleaseService struct {
	releases chan *domain.Release
}

func (f *fakeReleaseService) Process(_ context.Context, release *domain.Release) {
	f.releases <- release
}

type fakeSSE struct {
	m      sync.Mutex
	events map[string][][]byte
}

func (f *fakeSSE) Publish(id string, event *sse.Event) {
	f.m.Lock()
	defer f.m.Unlock()

	if id != EventStreamKey {
		return
	}

	if f.events == nil {
		f.events = map[string][][]byte{}
	}

	f.events[string(event.Event)] = append(f.events[string(event.Event)], event.Data)
}

func (f *fakeSSE) count(event string) int {
	f.m.Lock()
	defer f.m.Unlock()

	return len(f.events[event])
}

type fakeTokenStore struct {
	m       sync.Mutex
	tokens  []domain.MatrixTokens
	roomIDs map[int64]string
}

func (f *fakeTokenStore) UpdateTokens(_ context.Context, _ int64, tokens domain.MatrixTokens) error {
	f.m.Lock()
	defer f.m.Unlock()

	f.tokens = append(f.tokens, tokens)

	return nil
}

func (f *fakeTokenStore) UpdateRoomID(_ context.Context, id int64, roomID string) error {
	f.m.Lock()
	defer f.m.Unlock()

	f.roomIDs[id] = roomID

	return nil
}

func messageEvent(id, sender, body string, edit bool) map[string]any {
	content := map[string]any{"msgtype": "m.notice", "body": body}
	if edit {
		content["m.relates_to"] = map[string]any{"rel_type": "m.replace", "event_id": "$orig"}
	}

	return map[string]any{"type": "m.room.message", "event_id": id, "sender": sender, "content": content}
}

func syncBody(nextBatch string, events ...map[string]any) map[string]any {
	return map[string]any{
		"next_batch": nextBatch,
		"rooms": map[string]any{
			"join": map[string]any{
				testRoomID: map[string]any{"timeline": map[string]any{"events": events}},
			},
		},
	}
}

// fakeHomeserver serves one access token, which the first sync rejects to force a refresh.
type fakeHomeserver struct {
	t         *testing.T
	refreshed atomic.Bool
}

func (f *fakeHomeserver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	switch {
	case r.URL.Path == "/oauth2/token":
		require.NoError(f.t, r.ParseForm())
		assert.Equal(f.t, "refresh_token", r.Form.Get("grant_type"))
		assert.Equal(f.t, "refresh-1", r.Form.Get("refresh_token"))
		f.refreshed.Store(true)
		writeJSON(w, http.StatusOK, map[string]any{"access_token": "access-2", "refresh_token": "refresh-2", "expires_in": 300})
		return

	case token == "access-1" && strings.HasSuffix(r.URL.Path, "/sync"):
		writeJSON(w, http.StatusUnauthorized, map[string]any{"errcode": "M_UNKNOWN_TOKEN", "error": "expired", "soft_logout": true})
		return

	case token != "access-1" && token != "access-2":
		writeJSON(w, http.StatusUnauthorized, map[string]any{"errcode": "M_UNKNOWN_TOKEN", "error": "unknown"})
		return
	}

	switch {
	case r.URL.Path == "/_matrix/client/v3/account/whoami":
		writeJSON(w, http.StatusOK, map[string]any{"user_id": "@autobrr:matrix.watchtower.pm", "device_id": "DEVICE"})

	case r.URL.Path == "/_matrix/client/v3/join/"+testRoomAlias:
		writeJSON(w, http.StatusOK, map[string]any{"room_id": testRoomID})

	case r.URL.Path == "/_matrix/client/v3/sync":
		var filter struct {
			Room struct {
				Rooms []string `json:"rooms"`
			} `json:"room"`
		}
		require.NoError(f.t, json.Unmarshal([]byte(r.URL.Query().Get("filter")), &filter))
		assert.Equal(f.t, []string{testRoomID}, filter.Room.Rooms)

		switch r.URL.Query().Get("since") {
		case "":
			// history from before the connect must never be announced
			writeJSON(w, http.StatusOK, syncBody("s1", messageEvent("$old", testAnnouncer, testAnnounce, false)))
		case "s1":
			writeJSON(w, http.StatusOK, syncBody("s2",
				messageEvent("$spoof", "@someone:matrix.watchtower.pm", testAnnounce, false),
				messageEvent("$edit", testAnnouncer, testAnnounce, true),
				messageEvent("$new", testAnnouncer, testAnnounce, false),
			))
		default:
			<-r.Context().Done()
		}

	default:
		f.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func TestHandler_AnnounceFlow(t *testing.T) {
	definition, err := indexer.OpenAndProcessDefinition("../indexer/definitions/blutopia.yaml")
	require.NoError(t, err)

	definition.ID = 1
	definition.Enabled = true
	definition.SettingsMap = map[string]string{"rsskey": "secret"}

	homeserver := &fakeHomeserver{t: t}
	server := httptest.NewServer(homeserver)
	defer server.Close()

	expiresAt := time.Now().Add(time.Hour)
	network := domain.MatrixNetwork{
		ID:                 1,
		Name:               "Watchtower",
		Enabled:            true,
		Homeserver:         server.URL,
		AuthMethod:         domain.MatrixAuthMethodOAuth,
		AccessToken:        "access-1",
		RefreshToken:       "refresh-1",
		TokenExpiresAt:     &expiresAt,
		OAuthClientID:      "client",
		OAuthTokenEndpoint: server.URL + "/oauth2/token",
		Rooms:              []domain.MatrixRoom{{ID: 7, Enabled: true, Name: testRoomAlias}},
	}

	releases := &fakeReleaseService{releases: make(chan *domain.Release, 10)}
	store := &fakeTokenStore{roomIDs: map[int64]string{}}

	stream := &fakeSSE{}

	h := newHandler(zerolog.Nop(), stream, store, server.Client(), releases, network, []*domain.IndexerDefinition{definition})
	h.Start()
	defer h.Stop()

	select {
	case rls := <-releases.releases:
		assert.Equal(t, "True Story with Hamish & Andy S02 1080p STAN WEB-DL AAC 2.0 H.265-NYXIS", rls.TorrentName)
		assert.Equal(t, "https://blutopia.cc/torrent/download/402579.secret", rls.DownloadURL)
		assert.Equal(t, "https://blutopia.cc/torrents/402579", rls.InfoURL)
		assert.Equal(t, "bearfish", rls.Uploader)
	case <-time.After(5 * time.Second):
		t.Fatal("no release processed")
	}

	select {
	case rls := <-releases.releases:
		t.Fatalf("only the announcer's new message should be processed, got extra release: %s", rls.TorrentName)
	case <-time.After(200 * time.Millisecond):
	}

	assert.True(t, homeserver.refreshed.Load())

	store.m.Lock()
	defer store.m.Unlock()

	require.NotEmpty(t, store.tokens)
	assert.Equal(t, "access-2", store.tokens[0].AccessToken)
	assert.Equal(t, "refresh-2", store.tokens[0].RefreshToken)
	assert.Equal(t, testRoomID, store.roomIDs[7])

	networkHealth, roomHealth := h.Health()
	assert.True(t, networkHealth.Connected)
	assert.True(t, networkHealth.Healthy)
	assert.True(t, roomHealth[testRoomAlias].Joined)
	assert.NotNil(t, roomHealth[testRoomAlias].LastAnnounce)

	// the log view shows every new room message, announcer or not, but not edits or history from before connect
	history, err := h.History(testRoomAlias)
	require.NoError(t, err)
	require.Len(t, history, 2)
	assert.Equal(t, "@someone:matrix.watchtower.pm", history[0].Sender)
	assert.Equal(t, testAnnouncer, history[1].Sender)
	assert.True(t, strings.HasPrefix(history[1].Message, "[Blutopia] [NEW] bearfish"))
	assert.NotContains(t, history[1].Message, "\n")

	assert.Equal(t, 2, stream.count(eventMessage))
	assert.Positive(t, stream.count(eventHealth))
}

func TestHandler_TokenRejectedWithoutRefreshStops(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"errcode": "M_UNKNOWN_TOKEN", "error": "unknown"})
	}))
	defer server.Close()

	network := domain.MatrixNetwork{
		ID:          1,
		Name:        "Watchtower",
		Enabled:     true,
		Homeserver:  server.URL,
		AuthMethod:  domain.MatrixAuthMethodToken,
		AccessToken: "pasted",
		Rooms:       []domain.MatrixRoom{{ID: 1, Enabled: true, Name: testRoomAlias}},
	}

	h := newHandler(zerolog.Nop(), &fakeSSE{}, &fakeTokenStore{roomIDs: map[int64]string{}}, server.Client(), &fakeReleaseService{}, network, nil)
	h.Start()

	select {
	case <-h.done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler kept retrying a rejected token")
	}

	h.Stop()

	networkHealth, _ := h.Health()
	assert.False(t, networkHealth.Connected)
	assert.Equal(t, reasonTokenRejected, networkHealth.Error)
}

func TestAnnounceLine(t *testing.T) {
	assert.Equal(t, "first line second", announceLine("first\u00a0line\n\n  second  \r\n"))
	assert.Empty(t, announceLine(" \n "))
}

func TestMessageBufferKeepsNewest(t *testing.T) {
	var b messageBuffer
	for i := range maxRoomMessages + 5 {
		b.add(domain.MatrixMessage{Message: strconv.Itoa(i)})
	}

	messages := b.list()
	require.Len(t, messages, maxRoomMessages)
	assert.Equal(t, "5", messages[0].Message)
	assert.Equal(t, strconv.Itoa(maxRoomMessages+4), messages[len(messages)-1].Message)
}

// A room that fails to join is retried while the joined room keeps syncing, and the
// history it arrives with on joining is not announced.
func TestHandler_RetriesFailedJoin(t *testing.T) {
	const otherAlias = "#other:matrix.watchtower.pm"
	const otherID = "!other:matrix.watchtower.pm"

	definition, err := indexer.OpenAndProcessDefinition("../indexer/definitions/blutopia.yaml")
	require.NoError(t, err)

	definition.SettingsMap = map[string]string{"rsskey": "secret"}
	definition.Matrix.Rooms = append(definition.Matrix.Rooms, domain.IndexerIRCV2Channel{
		Name:       otherAlias,
		Announcers: definition.Matrix.Rooms[0].Announcers,
		Parse:      definition.Matrix.Rooms[0].Parse,
	})
	definition.Prepare()

	var joinAttempts, batch atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/_matrix/client/v3/account/whoami":
			writeJSON(w, http.StatusOK, map[string]any{"user_id": "@autobrr:matrix.watchtower.pm"})
		case r.URL.Path == "/_matrix/client/v3/join/"+testRoomAlias:
			writeJSON(w, http.StatusOK, map[string]any{"room_id": testRoomID})
		case r.URL.Path == "/_matrix/client/v3/join/"+otherAlias:
			if joinAttempts.Add(1) == 1 {
				writeJSON(w, http.StatusTooManyRequests, map[string]any{"errcode": "M_LIMIT_EXCEEDED"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"room_id": otherID})
		case r.URL.Path == "/_matrix/client/v3/sync":
			n := batch.Add(1)
			body := map[string]any{"next_batch": "s" + strconv.Itoa(int(n)), "rooms": map[string]any{"join": map[string]any{}}}
			join := body["rooms"].(map[string]any)["join"].(map[string]any)

			if strings.Contains(r.URL.Query().Get("filter"), otherID) {
				if !strings.Contains(r.URL.Query().Get("since"), "joined") {
					// first sync after the late join carries the room's history
					join[otherID] = map[string]any{"timeline": map[string]any{"events": []any{messageEvent("$old", testAnnouncer, testAnnounce, false)}}}
					body["next_batch"] = "joined"
				} else if r.URL.Query().Get("since") == "joined" {
					join[otherID] = map[string]any{"timeline": map[string]any{"events": []any{messageEvent("$new", testAnnouncer, testAnnounce, false)}}}
					body["next_batch"] = "joined-2"
				} else {
					<-r.Context().Done()
					return
				}
			} else {
				time.Sleep(20 * time.Millisecond)
			}

			writeJSON(w, http.StatusOK, body)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	network := domain.MatrixNetwork{
		ID: 1, Name: "Watchtower", Enabled: true, Homeserver: server.URL,
		AuthMethod: domain.MatrixAuthMethodToken, AccessToken: "token",
		Rooms: []domain.MatrixRoom{{ID: 1, Enabled: true, Name: testRoomAlias}, {ID: 2, Enabled: true, Name: otherAlias}},
	}

	releases := &fakeReleaseService{releases: make(chan *domain.Release, 10)}

	h := newHandler(zerolog.Nop(), &fakeSSE{}, &fakeTokenStore{roomIDs: map[int64]string{}}, server.Client(), releases, network, []*domain.IndexerDefinition{definition})
	h.joinRetryInitial = 50 * time.Millisecond
	h.Start()
	defer h.Stop()

	select {
	case <-releases.releases:
	case <-time.After(5 * time.Second):
		t.Fatal("room was not joined on retry")
	}

	select {
	case rls := <-releases.releases:
		t.Fatalf("history of the late-joined room was announced: %s", rls.TorrentName)
	case <-time.After(200 * time.Millisecond):
	}

	assert.Equal(t, int32(2), joinAttempts.Load())

	_, roomHealth := h.Health()
	assert.True(t, roomHealth[otherAlias].Joined)
	assert.Empty(t, roomHealth[otherAlias].Error)
}
