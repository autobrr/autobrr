// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package update

import (
	"context"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/version"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeChecker struct {
	release *version.Release
	err     error
}

func (f *fakeChecker) CheckNewVersion(_ context.Context, _ string) (bool, *version.Release, error) {
	if f.err != nil {
		return false, nil, f.err
	}

	return f.release != nil, f.release, nil
}

type recordingBus struct {
	events []events.AppUpdateEvent
}

func (b *recordingBus) EmitAppUpdate(_ context.Context, event events.AppUpdateEvent) {
	b.events = append(b.events, event)
}

func newTestService(checker *fakeChecker) (*Service, *recordingBus) {
	bus := &recordingBus{}

	s := NewUpdate(zerolog.Nop(), bus, &domain.Config{})
	s.releaseChecker = checker

	return s, bus
}

// The startup check, the scheduled job, the http handler and the tray all call
// CheckUpdateAvailable. Whichever finds a release first must not hide it from
// the others, and the event must fire once per release.
func TestService_CheckUpdateAvailable(t *testing.T) {
	checker := &fakeChecker{release: &version.Release{TagName: "v1.90.0", HtmlURL: "https://example.com/v1.90.0"}}
	s, bus := newTestService(checker)

	for range 3 {
		rel, err := s.CheckUpdateAvailable(t.Context())
		require.NoError(t, err)
		require.NotNil(t, rel)
		assert.Equal(t, "v1.90.0", rel.TagName)
	}

	require.Len(t, bus.events, 1)
	assert.Equal(t, "v1.90.0", bus.events[0].NewVersion)
	assert.Equal(t, "https://example.com/v1.90.0", bus.events[0].URL)
	assert.Equal(t, "v1.90.0", s.GetLatestRelease(t.Context()).TagName)

	checker.release = &version.Release{TagName: "v1.91.0"}

	rel, err := s.CheckUpdateAvailable(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "v1.91.0", rel.TagName)
	require.Len(t, bus.events, 2)
	assert.Equal(t, "v1.91.0", bus.events[1].NewVersion)
}

func TestService_CheckUpdateAvailable_UpToDate(t *testing.T) {
	s, bus := newTestService(&fakeChecker{})

	rel, err := s.CheckUpdateAvailable(t.Context())
	require.NoError(t, err)
	assert.Nil(t, rel)
	assert.Empty(t, bus.events)
}

func TestService_CheckUpdateAvailable_Error(t *testing.T) {
	s, bus := newTestService(&fakeChecker{err: errors.New("connection refused")})

	rel, err := s.CheckUpdateAvailable(t.Context())
	require.Error(t, err)
	assert.Nil(t, rel)
	assert.Empty(t, bus.events)
}
