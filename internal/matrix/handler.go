// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package matrix

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/announce"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/matrix"

	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
)

// EventStreamKey is the SSE stream carrying room messages and network health.
const EventStreamKey = "matrix"

// SSE event types on EventStreamKey.
const (
	eventMessage = "MESSAGE"
	eventHealth  = "HEALTH"
)

const (
	syncTimeout        = 30 * time.Second
	syncTimelineLimit  = 100
	tokenRefreshMargin = time.Minute
	backoffMin         = 2 * time.Second
	backoffMax         = 5 * time.Minute
	joinRetryMin       = 30 * time.Second
	joinRetryMax       = 10 * time.Minute
	tokenStoreTimeout  = 10 * time.Second
)

// Health reasons shown in the UI.
const (
	reasonTokenRejected   = "access token was rejected, log in again"
	reasonRefreshRejected = "refresh token was rejected, log in again"
	reasonNotLoggedIn     = "not logged in"
	reasonNoRooms         = "no enabled rooms"
	reasonNoIndexer       = "no indexer definition uses this room"
)

type ssePublisher interface {
	Publish(id string, event *sse.Event)
}

type tokenStore interface {
	UpdateTokens(ctx context.Context, networkID int64, tokens domain.MatrixTokens) error
	UpdateRoomID(ctx context.Context, id int64, roomID string) error
}

// roomSubscriber is an indexer listening to a room.
type roomSubscriber struct {
	processor  *announce.MatrixProcessor
	announcers map[string]struct{}
}

type room struct {
	config      domain.MatrixRoom
	queue       string
	subscribers []roomSubscriber
	history     messageBuffer
}

// Handler runs the sync loop of one network.
type Handler struct {
	log     zerolog.Logger
	sse     ssePublisher
	repo    tokenStore
	network domain.MatrixNetwork
	client  *matrix.Client
	oauth   *matrix.OAuthClient

	rooms      []*room
	processors []*announce.MatrixProcessor

	// joinRetryInitial is the first delay before retrying rooms that failed to join
	joinRetryInitial time.Duration
	// tokensUnsaved is owned by the sync loop goroutine
	tokensUnsaved bool

	m              sync.RWMutex
	connected      bool
	connectedSince *time.Time
	lastErr        string
	loginRequired  bool
	roomHealth     map[string]*domain.MatrixRoomHealth

	cancel context.CancelFunc
	done   chan struct{}
}

func newHandler(log zerolog.Logger, publisher ssePublisher, repo tokenStore, httpClient *http.Client, releaseSvc releaseService, indexerSvc indexerStatus, network domain.MatrixNetwork, definitions []*domain.IndexerDefinition) *Handler {
	h := &Handler{
		log:              log.With().Str("network", network.Name).Int64("network_id", network.ID).Logger(),
		sse:              publisher,
		repo:             repo,
		network:          network,
		client:           matrix.NewClient(network.Homeserver, httpClient),
		oauth:            matrix.NewOAuthClient(httpClient),
		roomHealth:       map[string]*domain.MatrixRoomHealth{},
		joinRetryInitial: joinRetryMin,
	}

	h.client.SetAccessToken(network.AccessToken)

	processors := map[string]*announce.MatrixProcessor{}

	for _, cfg := range network.Rooms {
		if !cfg.Enabled {
			continue
		}

		r := &room{config: cfg, queue: strings.ToLower(cfg.Name)}

		for _, definition := range definitions {
			roomDef, ok := definition.Matrix.GetRoom(cfg.Name)
			if !ok {
				continue
			}

			processor, ok := processors[definition.Identifier]
			if !ok {
				processor = announce.NewMatrixAnnounceProcessor(h.log, releaseSvc, indexerSvc, definition)
				processors[definition.Identifier] = processor
				h.processors = append(h.processors, processor)
			}

			// user ids are case-sensitive, so senders must match exactly
			announcers := make(map[string]struct{}, len(roomDef.Announcers))
			for _, announcer := range roomDef.Announcers {
				announcers[announcer] = struct{}{}
			}

			r.subscribers = append(r.subscribers, roomSubscriber{processor: processor, announcers: announcers})
		}

		h.rooms = append(h.rooms, r)
		h.roomHealth[cfg.Name] = &domain.MatrixRoomHealth{}

		if len(r.subscribers) == 0 {
			h.roomHealth[cfg.Name].Error = reasonNoIndexer
		}
	}

	return h
}

// Start runs the sync loop until Stop.
func (h *Handler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	h.done = make(chan struct{})

	go func() {
		defer close(h.done)

		h.run(ctx)
	}()
}

// Stop ends the sync loop and the announce processors and waits for the loop to exit.
func (h *Handler) Stop() {
	if h.cancel != nil {
		h.cancel()
		<-h.done
	}

	for _, processor := range h.processors {
		processor.Stop()
	}

	h.processors = nil
}

func (h *Handler) run(ctx context.Context) {
	if !h.network.HasCredentials() {
		h.setLoginRequired(reasonNotLoggedIn)
		return
	}

	if len(h.rooms) == 0 {
		h.setError(reasonNoRooms)
		return
	}

	defer func() {
		if h.tokensUnsaved {
			h.storeTokens(context.Background())
		}
	}()

	backoff := backoffMin
	since := ""

	for ctx.Err() == nil {
		next, connected, err := h.session(ctx, since)
		since = next

		if ctx.Err() != nil {
			return
		}

		// earlier outages must not slow down recovery from this one
		if connected {
			backoff = backoffMin
		}

		h.setDisconnected(err)

		// every fatal stop is a rejected login
		if fatal, ok := errors.AsType[*fatalError](err); ok {
			h.log.Error().Err(fatal.err).Msg("matrix network stopped")
			h.setLoginRequired(fatal.reason)
			return
		}

		wait := backoff
		if merr, ok := errors.AsType[*matrix.Error](err); ok && merr.RetryAfter > 0 {
			wait = merr.RetryAfter
		}

		h.log.Warn().Err(err).Dur("retry_in", wait).Msg("matrix sync failed")

		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}

		backoff = min(backoff*2, backoffMax)
	}
}

// fatalError stops the sync loop until the user logs in again.
type fatalError struct {
	reason string
	err    error
}

func (e *fatalError) Error() string {
	return e.reason
}

// session connects and syncs until an error. It returns the last sync token so a
// reconnect resumes where the previous session stopped, and whether it connected; an
// empty since starts fresh and skips the room history.
func (h *Handler) session(ctx context.Context, since string) (string, bool, error) {
	if err := h.ensureToken(ctx); err != nil {
		return since, false, err
	}

	whoami, err := callAuthed(ctx, h, func() (*matrix.WhoamiResponse, error) { return h.client.Whoami(ctx) })
	if err != nil {
		return since, false, err
	}

	h.log.Debug().Str("user_id", whoami.UserID).Msg("matrix authenticated")

	roomIDs, failed, retryAfter, err := h.joinRooms(ctx, h.rooms)
	if err != nil {
		return since, false, err
	}

	if len(roomIDs) == 0 {
		return since, false, errors.New("could not join any room")
	}

	// connected means a sync succeeded, so a session that only gets as far as joining keeps backing off
	connected := false
	markConnected := func() {
		connected = true
		h.setConnected()
		h.log.Info().Int("rooms", len(roomIDs)).Msg("matrix network connected")
	}

	// rooms joined after since may arrive with their history, which must not be announced
	lateJoined := map[string]struct{}{}

	if since == "" {
		resp, err := h.sync(ctx, matrix.SyncRequest{Rooms: roomIDs, TimelineLimit: 1})
		if err != nil {
			return since, false, err
		}

		since = resp.NextBatch
		markConnected()
	} else {
		// on a reconnect any room may have been joined since the last sync, for example
		// one that failed to join before; rooms that stayed joined carry no join event
		for _, id := range roomIDs {
			lateJoined[id] = struct{}{}
		}
	}

	joinRetry := h.joinRetryInitial
	nextJoin := time.Now().Add(max(joinRetry, retryAfter))

	for {
		if len(failed) > 0 && !time.Now().Before(nextJoin) {
			joined, stillFailed, retryAfter, err := h.joinRooms(ctx, failed)
			if err != nil {
				return since, connected, err
			}

			for _, id := range joined {
				lateJoined[id] = struct{}{}
			}

			roomIDs = append(roomIDs, joined...)
			failed = stillFailed

			joinRetry = min(joinRetry*2, joinRetryMax)
			nextJoin = time.Now().Add(max(joinRetry, retryAfter))
		}

		resp, err := h.sync(ctx, matrix.SyncRequest{Since: since, Timeout: syncTimeout, Rooms: roomIDs, TimelineLimit: syncTimelineLimit})
		if err != nil {
			return since, connected, err
		}

		if !connected {
			markConnected()
		}

		for id := range lateJoined {
			joinedRoom, ok := resp.Rooms.Join[id]
			if !ok {
				continue
			}

			joinedRoom.Timeline.Events = eventsAfterJoin(joinedRoom.Timeline.Events, whoami.UserID)
			resp.Rooms.Join[id] = joinedRoom
			delete(lateJoined, id)
		}

		h.handleSync(resp)

		since = resp.NextBatch

		if h.tokensUnsaved {
			h.storeTokens(ctx)
		}
	}
}

func (h *Handler) sync(ctx context.Context, req matrix.SyncRequest) (*matrix.SyncResponse, error) {
	return callAuthed(ctx, h, func() (*matrix.SyncResponse, error) { return h.client.Sync(ctx, req) })
}

// callAuthed runs an authenticated request and retries it once after refreshing a rejected token.
func callAuthed[T any](ctx context.Context, h *Handler, fn func() (T, error)) (T, error) {
	resp, err := fn()
	if !isUnauthorized(err) {
		return resp, err
	}

	if !h.network.CanRefresh() {
		return resp, &fatalError{reason: reasonTokenRejected, err: err}
	}

	if err := h.refreshToken(ctx); err != nil {
		return resp, err
	}

	resp, err = fn()
	if isUnauthorized(err) {
		return resp, &fatalError{reason: reasonTokenRejected, err: err}
	}

	return resp, err
}

func isUnauthorized(err error) bool {
	merr, ok := errors.AsType[*matrix.Error](err)

	return ok && merr.IsUnauthorized()
}

func (h *Handler) ensureToken(ctx context.Context) error {
	expiresAt := h.network.TokenExpiresAt
	if expiresAt == nil || !h.network.CanRefresh() {
		return nil
	}

	if time.Until(*expiresAt) > tokenRefreshMargin {
		return nil
	}

	return h.refreshToken(ctx)
}

func (h *Handler) refreshToken(ctx context.Context) error {
	token, err := h.oauth.RefreshToken(ctx, h.network.OAuthTokenEndpoint, h.network.OAuthClientID, h.network.RefreshToken)
	if err != nil {
		if oerr, ok := errors.AsType[*matrix.OAuthError](err); ok && (oerr.Code == matrix.OAuthErrInvalidGrant || oerr.StatusCode == http.StatusUnauthorized) {
			return &fatalError{reason: reasonRefreshRejected, err: err}
		}

		return errors.Wrap(err, "could not refresh access token")
	}

	h.network.AccessToken = token.AccessToken
	h.network.RefreshToken = token.RefreshToken
	if expiresAt := token.ExpiresAt(time.Now()); !expiresAt.IsZero() {
		h.network.TokenExpiresAt = &expiresAt
	} else {
		h.network.TokenExpiresAt = nil
	}

	h.client.SetAccessToken(token.AccessToken)

	h.storeTokens(ctx)

	h.log.Trace().Msg("matrix access token refreshed")

	return nil
}

// storeTokens persists the current token pair. The old refresh token is spent once
// refreshed, so this runs even while stopping, bounded so it cannot hold up a stop. A
// failed write is retried after the next sync and when the loop exits.
func (h *Handler) storeTokens(ctx context.Context) {
	storeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), tokenStoreTimeout)
	defer cancel()

	if err := h.repo.UpdateTokens(storeCtx, h.network.ID, tokensOf(h.network)); err != nil {
		h.log.Error().Err(err).Msg("could not store refreshed matrix tokens")
		h.tokensUnsaved = true
		return
	}

	h.tokensUnsaved = false
}

func tokensOf(network domain.MatrixNetwork) domain.MatrixTokens {
	return domain.MatrixTokens{
		AuthMethod:         network.AuthMethod,
		UserID:             network.UserID,
		DeviceID:           network.DeviceID,
		AccessToken:        network.AccessToken,
		RefreshToken:       network.RefreshToken,
		TokenExpiresAt:     network.TokenExpiresAt,
		OAuthClientID:      network.OAuthClientID,
		OAuthTokenEndpoint: network.OAuthTokenEndpoint,
	}
}

// joinRooms joins rooms and returns the room ids it is in, the rooms that failed and
// the longest retry delay the homeserver asked for. Joining an already joined room
// succeeds, so this also resolves aliases on every connect. A rejected login is
// returned instead of being reported per room.
func (h *Handler) joinRooms(ctx context.Context, rooms []*room) ([]string, []*room, time.Duration, error) {
	roomIDs := make([]string, 0, len(rooms))

	var (
		failed     []*room
		retryAfter time.Duration
	)

	for _, r := range rooms {
		roomID, err := callAuthed(ctx, h, func() (string, error) { return h.client.JoinRoom(ctx, r.config.Name) })
		if err != nil {
			if _, ok := errors.AsType[*fatalError](err); ok {
				return nil, nil, 0, err
			}

			if merr, ok := errors.AsType[*matrix.Error](err); ok {
				retryAfter = max(retryAfter, merr.RetryAfter)
			}

			h.log.Error().Err(err).Str("room", r.config.Name).Msg("could not join matrix room")
			h.setRoomError(r.config.Name, err.Error())
			failed = append(failed, r)
			continue
		}

		if roomID != r.config.RoomID {
			r.config.RoomID = roomID

			if err := h.repo.UpdateRoomID(ctx, r.config.ID, roomID); err != nil {
				h.log.Error().Err(err).Str("room", r.config.Name).Msg("could not store matrix room id")
			}
		}

		h.setRoomJoined(r.config.Name)
		roomIDs = append(roomIDs, roomID)
	}

	return roomIDs, failed, retryAfter, nil
}

// eventsAfterJoin drops the history a room arrives with when it is joined mid-session:
// everything up to the user's own join event. A room the user was already in carries
// no join event and is returned unchanged.
func eventsAfterJoin(events []matrix.Event, userID string) []matrix.Event {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].IsJoinOf(userID) {
			return events[i+1:]
		}
	}

	return events
}

func (h *Handler) handleSync(resp *matrix.SyncResponse) {
	for roomID, joined := range resp.Rooms.Join {
		r := h.roomByID(roomID)
		if r == nil {
			continue
		}

		if joined.Timeline.Limited {
			h.log.Warn().Str("room", r.config.Name).Msg("matrix sync timeline was truncated, announces may have been missed")
		}

		for _, event := range joined.Timeline.Events {
			h.handleEvent(r, event)
		}
	}
}

func (h *Handler) roomByID(roomID string) *room {
	for _, r := range h.rooms {
		if r.config.RoomID == roomID {
			return r
		}
	}

	return nil
}

func (h *Handler) handleEvent(r *room, event matrix.Event) {
	if event.Type != matrix.EventTypeRoomMessage {
		return
	}

	msg, err := event.Message()
	if err != nil {
		h.log.Debug().Err(err).Str("event_id", event.EventID).Msg("could not read matrix message")
		return
	}

	if msg.MsgType != matrix.MsgTypeText && msg.MsgType != matrix.MsgTypeNotice {
		return
	}

	// edits repeat the original body and would announce the release twice
	if msg.IsEdit() {
		return
	}

	line := announceLine(msg.Body)
	if line == "" {
		return
	}

	h.recordMessage(r, event, line)

	queued := false

	for _, subscriber := range r.subscribers {
		if _, ok := subscriber.announcers[event.Sender]; !ok {
			continue
		}

		if err := subscriber.processor.AddLineToQueue(r.queue, line); err != nil {
			h.log.Error().Err(err).Str("room", r.config.Name).Msg("could not queue matrix announce")
			continue
		}

		queued = true
	}

	if queued {
		h.setRoomAnnounce(r.config.Name)
	} else {
		h.log.Trace().Str("room", r.config.Name).Str("sender", event.Sender).Msg("ignoring matrix message from non-announcer")
	}
}

// announceLine joins the lines of a message body with spaces. A Matrix message is
// one whole announce, unlike IRC where an announce spanning lines needs a multi-line
// parse, and announcers wrap long announces over several lines of one message.
func announceLine(body string) string {
	var parts []string

	for line := range strings.SplitSeq(strings.ReplaceAll(body, "\u00a0", " "), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			parts = append(parts, line)
		}
	}

	return strings.Join(parts, " ")
}

// recordMessage keeps a room message in the room history and streams it to the UI.
func (h *Handler) recordMessage(r *room, event matrix.Event, line string) {
	msg := domain.MatrixMessage{
		ID:      event.EventID,
		Network: h.network.ID,
		Room:    r.config.Name,
		Sender:  event.Sender,
		Message: line,
		Time:    time.Now(),
	}

	if event.OriginServerTS > 0 {
		msg.Time = time.UnixMilli(event.OriginServerTS)
	}

	r.history.add(msg)
	h.publish(eventMessage, msg)
}

// History returns the recorded messages of a room.
func (h *Handler) History(roomName string) ([]domain.MatrixMessage, error) {
	for _, r := range h.rooms {
		if strings.EqualFold(r.config.Name, roomName) {
			return r.history.list(), nil
		}
	}

	return nil, errors.Wrap(domain.ErrMatrixRoomNotFound, "could not find history for room: '%s'", roomName)
}

func (h *Handler) publish(event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		h.log.Error().Err(err).Str("event", event).Msg("could not marshal matrix event")
		return
	}

	h.sse.Publish(EventStreamKey, &sse.Event{Event: []byte(event), Data: payload})
}

// broadcastHealth streams the current health so the UI updates without waiting for its poll.
// It must be called without holding h.m.
func (h *Handler) broadcastHealth() {
	networkHealth, roomHealth := h.Health()

	h.publish(eventHealth, domain.MatrixHealthEvent{Network: h.network.ID, Health: networkHealth, Rooms: roomHealth})
}

// QueueAnnounce feeds a line into a room's processors as if an announcer sent it.
func (h *Handler) QueueAnnounce(roomName, line string) error {
	for _, r := range h.rooms {
		if !strings.EqualFold(r.config.Name, roomName) {
			continue
		}

		if len(r.subscribers) == 0 {
			return errors.New("no indexer for room: %s", roomName)
		}

		for _, subscriber := range r.subscribers {
			if err := subscriber.processor.AddLineToQueue(r.queue, line); err != nil {
				return err
			}
		}

		return nil
	}

	return errors.New("room not found: %s", roomName)
}

func (h *Handler) setConnected() {
	defer h.broadcastHealth()

	h.m.Lock()
	defer h.m.Unlock()

	now := time.Now()
	h.connected = true
	h.connectedSince = &now
	h.lastErr = ""
}

func (h *Handler) setDisconnected(err error) {
	defer h.broadcastHealth()

	h.m.Lock()
	defer h.m.Unlock()

	h.connected = false
	h.connectedSince = nil

	if err != nil {
		h.lastErr = err.Error()
	}

	for _, health := range h.roomHealth {
		health.Joined = false
		health.JoinedSince = nil
	}
}

// setLoginRequired stops reporting the network as merely failing: only a new login
// brings it back, so the UI offers one.
func (h *Handler) setLoginRequired(reason string) {
	defer h.broadcastHealth()

	h.m.Lock()
	defer h.m.Unlock()

	h.lastErr = reason
	h.loginRequired = true
}

func (h *Handler) setError(reason string) {
	defer h.broadcastHealth()

	h.m.Lock()
	defer h.m.Unlock()

	h.lastErr = reason
}

func (h *Handler) setRoomJoined(name string) {
	defer h.broadcastHealth()

	h.m.Lock()
	defer h.m.Unlock()

	health := h.roomHealth[name]
	now := time.Now()
	health.Joined = true
	health.JoinedSince = &now

	if len(h.roomByName(name).subscribers) > 0 {
		health.Error = ""
	}
}

func (h *Handler) roomByName(name string) *room {
	for _, r := range h.rooms {
		if r.config.Name == name {
			return r
		}
	}

	return nil
}

func (h *Handler) setRoomError(name, reason string) {
	defer h.broadcastHealth()

	h.m.Lock()
	defer h.m.Unlock()

	h.roomHealth[name].Error = reason
}

func (h *Handler) setRoomAnnounce(name string) {
	defer h.broadcastHealth()

	h.m.Lock()
	defer h.m.Unlock()

	now := time.Now()
	h.roomHealth[name].LastAnnounce = &now
}

// Health returns the network and per-room state.
func (h *Handler) Health() (domain.MatrixNetworkHealth, map[string]domain.MatrixRoomHealth) {
	h.m.RLock()
	defer h.m.RUnlock()

	network := domain.MatrixNetworkHealth{
		Connected:      h.connected,
		ConnectedSince: h.connectedSince,
		Error:          h.lastErr,
		LoginRequired:  h.loginRequired,
	}

	rooms := make(map[string]domain.MatrixRoomHealth, len(h.roomHealth))
	healthy := h.connected

	for name, health := range h.roomHealth {
		rooms[name] = *health

		if !health.Joined || health.Error != "" {
			healthy = false
		}
	}

	network.Healthy = healthy

	return network, rooms
}
