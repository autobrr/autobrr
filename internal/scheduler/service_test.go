// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Stop must hold shutdown until a running job finishes, since the database is
// closed right after it.
func TestService_StopWaitsForRunningJob(t *testing.T) {
	s := NewService(zerolog.Nop(), nil, &domain.Config{}, nil)

	running := make(chan struct{})
	release := make(chan struct{})

	_, err := s.ScheduleJob(cron.FuncJob(func() {
		close(running)
		<-release
	}), time.Second, "blocking")
	require.NoError(t, err)

	require.NoError(t, s.Start(t.Context()))

	select {
	case <-running:
	case <-time.After(3 * time.Second):
		t.Fatal("job did not run")
	}

	stopped := make(chan error, 1)
	go func() {
		stopped <- s.Stop(context.Background())
	}()

	select {
	case <-stopped:
		t.Fatal("Stop returned while the job was still running")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)

	select {
	case err := <-stopped:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Stop did not return after the job finished")
	}
}

func TestService_StopGivesUpWhenContextEnds(t *testing.T) {
	s := NewService(zerolog.Nop(), nil, &domain.Config{}, nil)

	running := make(chan struct{})
	release := make(chan struct{})
	defer close(release)

	_, err := s.ScheduleJob(cron.FuncJob(func() {
		close(running)
		<-release
	}), time.Second, "blocking")
	require.NoError(t, err)

	require.NoError(t, s.Start(t.Context()))

	select {
	case <-running:
	case <-time.After(3 * time.Second):
		t.Fatal("job did not run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	assert.ErrorIs(t, s.Stop(ctx), context.DeadlineExceeded)
}
