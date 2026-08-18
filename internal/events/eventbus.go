// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package events

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/maniartech/signals"
	"github.com/rs/zerolog"
	"gitlab.com/tozd/go/errors"
)

// emitEvent and onEvent are always reached through an Emit<Event>/On<Event>
// method, so two frames up is the code that actually emitted or subscribed.
const callerSkipWrapper = 2

var keyCounter uint64

// generateKey generates a unique key for event listeners
func generateKey() string {
	return fmt.Sprintf("listener_%d", atomic.AddUint64(&keyCounter, 1))
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

type EventBus struct {
	log zerolog.Logger
	ctx context.Context

	appUpdate   signals.SyncSignal[AppUpdateEvent]
	indexer     signals.SyncSignal[IndexerChangeEvent]
	irc         signals.SyncSignal[IRCEvent]
	release     signals.SyncSignal[ReleaseEvent]
	releasePush signals.SyncSignal[ReleasePushEvent]
}

func NewEventBus(log zerolog.Logger, ctx context.Context) *EventBus {
	return &EventBus{
		log:         log.With().Str("module", "eventbus").Logger(),
		ctx:         ctx,
		appUpdate:   *signals.NewSync[AppUpdateEvent](),
		indexer:     *signals.NewSync[IndexerChangeEvent](),
		irc:         *signals.NewSync[IRCEvent](),
		release:     *signals.NewSync[ReleaseEvent](),
		releasePush: *signals.NewSync[ReleasePushEvent](),
	}
}

// EventConstraint ensures that T embeds Event
type EventConstraint interface {
	GetEvent() *Event
}

// Generic internal methods for event handling
func onEvent[T any](log zerolog.Logger, signal signals.SyncSignal[T], eventType EventType, handler func(context.Context, T) errors.E) func() {
	key := generateKey()

	// the listener logs at dispatch time, long after registration, so the event
	// it belongs to and the subscriber that registered it are stamped once here
	l := log.With().Str("event", string(eventType)).Str("listener", key).Str("caller", caller(callerSkipWrapper)).Logger()

	l.Trace().Msg("registering event handler")

	count := signal.AddListenerWithErr(func(ctx context.Context, event T) error {
		// Panic/exception safety
		defer func() {
			if r := recover(); r != nil {
				l.Error().Str("event_uuid", GetEventUUID(ctx)).Interface("panic", r).Str("stack", string(debug.Stack())).Msg("event handler panic")
			}
		}()

		l.Trace().Str("event_uuid", GetEventUUID(ctx)).Str("type", fmt.Sprintf("%T", event)).Interface("payload", event).Msg("<-- receiving event")

		return handler(ctx, event)
	}, key)

	l.Trace().Int("listener_count", count).Msg("event handler registered")

	return func() {
		signal.RemoveListener(key)

		l.Trace().Msg("event handler unregistered")
	}
}

func emitEvent[T any](ctx context.Context, log zerolog.Logger, signal signals.SyncSignal[T], event T) errors.E {
	// Add UUID to context if not already present
	ctx = ContextWithEventUUID(ctx)

	l := log.With().Str("event_uuid", GetEventUUID(ctx)).Str("type", fmt.Sprintf("%T", event)).Str("caller", caller(callerSkipWrapper)).Logger()

	l.Trace().Interface("payload", event).Msg("--> emitting event")

	// Emit synchronously; recover panic inside signal dispatch and log emission errors
	defer func() {
		if r := recover(); r != nil {
			l.Error().Interface("panic", r).Str("stack", string(debug.Stack())).Interface("payload", event).Msg("panic emitting event")
		}
	}()

	if err := signal.TryEmit(ctx, event); err != nil {
		// We log at warn level to avoid noisy error logs for expected cancellations
		l.Warn().Err(err).Interface("payload", event).Msg("event emission error")
		return errors.WithStack(err)
	}

	return nil
}

func (eb *EventBus) EmitAppUpdate(event AppUpdateEvent) {
	emitEvent(eb.ctx, eb.log, eb.appUpdate, event)
}

func (eb *EventBus) OnAppUpdate(handler func(context.Context, AppUpdateEvent) errors.E) func() {
	return onEvent(eb.log, eb.appUpdate, "AppUpdate", handler)
}

func (eb *EventBus) EmitIndexer(event IndexerChangeEvent) {
	emitEvent(eb.ctx, eb.log, eb.indexer, event)
}

func (eb *EventBus) OnIndexer(handler func(context.Context, IndexerChangeEvent) errors.E) func() {
	return onEvent(eb.log, eb.indexer, "Indexer", handler)
}

func (eb *EventBus) EmitIRC(event IRCEvent) {
	emitEvent(eb.ctx, eb.log, eb.irc, event)
}

func (eb *EventBus) OnIRC(handler func(context.Context, IRCEvent) errors.E) func() {
	return onEvent(eb.log, eb.irc, "IRC", handler)
}

func (eb *EventBus) EmitReleaseNew(event ReleaseEvent) {
	emitEvent(eb.ctx, eb.log, eb.release, event)
}

func (eb *EventBus) OnReleaseNew(handler func(context.Context, ReleaseEvent) errors.E) func() {
	return onEvent(eb.log, eb.release, "ReleaseNew", handler)
}

func (eb *EventBus) EmitReleasePush(event ReleasePushEvent) {
	emitEvent(eb.ctx, eb.log, eb.releasePush, event)
}

func (eb *EventBus) OnReleasePush(handler func(context.Context, ReleasePushEvent) errors.E) func() {
	return onEvent(eb.log, eb.releasePush, "ReleasePush", handler)
}

// GetEventUUID retrieves the UUID from context
func GetEventUUID(ctx context.Context) string {
	if val := ctx.Value("event_uuid"); val != nil {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// ContextWithEventUUID returns a new context with the event UUID set
func ContextWithEventUUID(ctx context.Context) context.Context {
	if GetEventUUID(ctx) == "" {
		return context.WithValue(ctx, "event_uuid", uuid.New().String())
	}
	return ctx
}
