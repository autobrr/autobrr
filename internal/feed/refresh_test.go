// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRefreshJob struct {
	err error
}

func (j *stubRefreshJob) Run() {}

func (j *stubRefreshJob) RunE(context.Context) error {
	return j.err
}

func TestServiceRefreshEmitsOutcome(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantType  events.EventType
		wantError string
	}{
		{name: "success", wantType: events.FeedRefreshSuccess},
		{name: "error", err: errors.New("unexpected status code: 503"), wantType: events.FeedRefreshError, wantError: "unexpected status code: 503"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := events.NewEventBus(zerolog.Nop())

			var got []events.FeedRefreshEvent
			bus.OnFeedRefresh(func(_ context.Context, event events.FeedRefreshEvent) error {
				got = append(got, event)
				return nil
			})

			s := &Service{log: zerolog.Nop(), eventBus: bus}
			f := &domain.Feed{ID: 4, Name: "Mock Indexer"}

			err := s.refresh(context.Background(), f, &stubRefreshJob{err: tt.err})
			assert.Equal(t, tt.err, err)

			require.Len(t, got, 1)
			assert.Equal(t, tt.wantType, got[0].Type)
			assert.Equal(t, f, got[0].Feed)
			assert.Equal(t, tt.wantError, got[0].Error)
		})
	}
}
