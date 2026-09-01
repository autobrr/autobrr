// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package events

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"uuid"

	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/maniartech/signals"
	"github.com/rs/zerolog"
)

// Topic.Emit and Topic.On are always reached through an Emit<Event>/On<Event>
// method on EventBus, so two frames up is the code that actually emitted or
// subscribed. Reaching a Topic directly would need a skip of one.
const callerSkipWrapper = 2

var keyCounter atomic.Uint64

// generateKey generates a unique key for event listeners
func generateKey() string {
	return fmt.Sprintf("listener_%d", keyCounter.Add(1))
}

// caller returns the file:line skip frames above it, formatted the way zerolog
// formats its own caller field so the log viewer renders it as a source link.
func caller(skip int) string {
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return ""
	}

	return zerolog.CallerMarshalFunc(pc, file, line)
}

// EventConstraint ensures T embeds Event, so the bus can log the concrete event
// type without reflecting over the payload.
type EventConstraint interface {
	GetType() EventType
}

// Topic carries every event sharing a payload type. Its logger is stamped with
// the topic once, at construction, so dispatch only adds what varies per event.
type Topic[T EventConstraint] struct {
	log    zerolog.Logger
	signal *signals.SyncSignal[T]
}

// NewTopic returns a Topic dispatching events of type T under the given label.
func NewTopic[T EventConstraint](log zerolog.Logger, topic string) *Topic[T] {
	return &Topic[T]{
		log:    log.With().Str("topic", topic).Logger(),
		signal: signals.NewSync[T](),
	}
}

// On registers handler and returns a function that unregisters it.
func (t *Topic[T]) On(handler func(context.Context, T) error) func() {
	key := generateKey()

	// the listener logs at dispatch time, long after registration, so the
	// subscriber that registered it is stamped once here
	l := t.log.With().Str("listener", key).Str("caller", caller(callerSkipWrapper)).Logger()

	l.Trace().Msg("registering event handler")

	count := t.signal.AddListenerWithErr(func(ctx context.Context, event T) error {
		// a panicking subscriber must not take down the emitter, so every listener
		// recovers its own; Emit deliberately has no second recover
		defer func() {
			if r := recover(); r != nil {
				l.Error().Str("event", string(event.GetType())).Str("event_uuid", GetEventUUID(ctx)).Interface("panic", r).Str("stack", string(debug.Stack())).Msg("event handler panic")
			}
		}()

		// the emitter already logged the payload, this line only has to correlate
		l.Trace().Str("event", string(event.GetType())).Str("event_uuid", GetEventUUID(ctx)).Msg("<-- receiving event")

		return handler(ctx, event)
	}, key)

	l.Trace().Int("listener_count", count).Msg("event handler registered")

	return func() {
		t.signal.RemoveListener(key)

		l.Trace().Msg("event handler unregistered")
	}
}

// Emit dispatches event to every listener synchronously.
func (t *Topic[T]) Emit(ctx context.Context, event T) error {
	// an event describes something that already happened, so the emitter's deadline
	// must not decide whether subscribers observe it: TryEmit bails on a cancelled
	// ctx, which would drop reconciliation when an HTTP client aborts mid-request.
	// Values (event UUID, logger) still ride along.
	ctx = context.WithoutCancel(ctx)
	ctx = ContextWithEventUUID(ctx)

	l := t.log.With().Str("event", string(event.GetType())).Str("event_uuid", GetEventUUID(ctx)).Logger()

	// caller unwinds the stack and the payload marshals the whole event, so both
	// stay behind the level check rather than being computed for a discarded line
	if e := l.Trace(); e != nil {
		e.Str("caller", caller(callerSkipWrapper)).Interface("payload", event).Msg("--> emitting event")
	}

	if err := t.signal.TryEmit(ctx, event); err != nil {
		// We log at warn level to avoid noisy error logs for expected cancellations
		l.Warn().Err(err).Str("caller", caller(callerSkipWrapper)).Msg("event emission error")
		return errors.Wrap(err, "could not emit event")
	}

	return nil
}

type EventBus struct {
	appUpdate   *Topic[AppUpdateEvent]
	indexer     *Topic[IndexerChangeEvent]
	irc         *Topic[IRCEvent]
	release     *Topic[ReleaseEvent]
	releasePush *Topic[ReleasePushEvent]
}

func NewEventBus(log zerolog.Logger) *EventBus {
	log = log.With().Str("module", "eventbus").Logger()

	return &EventBus{
		appUpdate:   NewTopic[AppUpdateEvent](log, "app_update"),
		indexer:     NewTopic[IndexerChangeEvent](log, "indexer"),
		irc:         NewTopic[IRCEvent](log, "irc"),
		release:     NewTopic[ReleaseEvent](log, "release"),
		releasePush: NewTopic[ReleasePushEvent](log, "release_push"),
	}
}

func (eb *EventBus) EmitAppUpdate(ctx context.Context, event AppUpdateEvent) {
	eb.appUpdate.Emit(ctx, event)
}

func (eb *EventBus) OnAppUpdate(handler func(context.Context, AppUpdateEvent) error) func() {
	return eb.appUpdate.On(handler)
}

func (eb *EventBus) EmitIndexer(ctx context.Context, event IndexerChangeEvent) {
	eb.indexer.Emit(ctx, event)
}

func (eb *EventBus) OnIndexer(handler func(context.Context, IndexerChangeEvent) error) func() {
	return eb.indexer.On(handler)
}

func (eb *EventBus) EmitIRC(ctx context.Context, event IRCEvent) {
	eb.irc.Emit(ctx, event)
}

func (eb *EventBus) OnIRC(handler func(context.Context, IRCEvent) error) func() {
	return eb.irc.On(handler)
}

func (eb *EventBus) EmitReleaseNew(ctx context.Context, event ReleaseEvent) {
	eb.release.Emit(ctx, event)
}

func (eb *EventBus) OnReleaseNew(handler func(context.Context, ReleaseEvent) error) func() {
	return eb.release.On(handler)
}

func (eb *EventBus) EmitReleasePush(ctx context.Context, event ReleasePushEvent) {
	eb.releasePush.Emit(ctx, event)
}

func (eb *EventBus) OnReleasePush(handler func(context.Context, ReleasePushEvent) error) func() {
	return eb.releasePush.On(handler)
}

type eventUUIDKey struct{}

// GetEventUUID retrieves the UUID from context
func GetEventUUID(ctx context.Context) string {
	id, _ := ctx.Value(eventUUIDKey{}).(string)
	return id
}

// ContextWithEventUUID returns a new context with the event UUID set
func ContextWithEventUUID(ctx context.Context) context.Context {
	if GetEventUUID(ctx) == "" {
		return context.WithValue(ctx, eventUUIDKey{}, uuid.NewV7().String())
	}
	return ctx
}
