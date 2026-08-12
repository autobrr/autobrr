// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package release

import (
	"context"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type mockCleanupRepo struct {
	deleteCalled bool
	deleteReq    *domain.DeleteReleaseRequest
	updatedJob   *domain.ReleaseCleanupJob
}

func (m *mockCleanupRepo) UpdateCleanupJobLastRun(_ context.Context, job *domain.ReleaseCleanupJob) error {
	m.updatedJob = job
	return nil
}

func (m *mockCleanupRepo) Delete(_ context.Context, req *domain.DeleteReleaseRequest) error {
	m.deleteCalled = true
	m.deleteReq = req
	return nil
}

func TestCleanupJobRun_AllStatusesInvalid_AbortsWithoutDelete(t *testing.T) {
	repo := &mockCleanupRepo{}
	job := &domain.ReleaseCleanupJob{Name: "test", OlderThan: 24, Statuses: "LEGACY_UNKNOWN"}

	NewCleanupJob(zerolog.Nop(), repo, job).Run()

	assert.False(t, repo.deleteCalled)
	assert.Equal(t, domain.ReleaseCleanupStatusError, job.LastRunStatus)
	assert.Contains(t, job.LastRunData, "no valid statuses")
}

func TestCleanupJobRun_MixedStatuses_DeletesWithValidOnly(t *testing.T) {
	repo := &mockCleanupRepo{}
	job := &domain.ReleaseCleanupJob{Name: "test", OlderThan: 24, Statuses: "PENDING, LEGACY_UNKNOWN"}

	NewCleanupJob(zerolog.Nop(), repo, job).Run()

	assert.True(t, repo.deleteCalled)
	assert.Equal(t, []string{"PENDING"}, repo.deleteReq.ReleaseStatuses)
	assert.Equal(t, domain.ReleaseCleanupStatusSuccess, job.LastRunStatus)
}

func TestCleanupJobRun_NoStatusFilter_Deletes(t *testing.T) {
	repo := &mockCleanupRepo{}
	job := &domain.ReleaseCleanupJob{Name: "test", OlderThan: 24}

	NewCleanupJob(zerolog.Nop(), repo, job).Run()

	assert.True(t, repo.deleteCalled)
	assert.Empty(t, repo.deleteReq.ReleaseStatuses)
	assert.Equal(t, domain.ReleaseCleanupStatusSuccess, job.LastRunStatus)
}
