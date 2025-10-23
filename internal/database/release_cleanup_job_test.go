// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
)

func getMockReleaseCleanupJob() *domain.ReleaseCleanupJob {
	return &domain.ReleaseCleanupJob{
		Name:          "Test Cleanup Job",
		Enabled:       true,
		Schedule:      "0 3 * * *",
		OlderThan:     720,
		Indexers:      "btn,ptp",
		Statuses:      "PUSH_REJECTED,PUSH_ERROR",
		LastRun:       time.Now(),
		LastRunStatus: domain.ReleaseCleanupStatusSuccess,
		LastRunData:   `{"deleted": 10}`,
	}
}

func TestReleaseCleanupJobRepo_Store(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseCleanupJobRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("Store_Succeeds [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			assert.NotZero(t, mockData.ID)

			// Verify
			job, err := repo.FindByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, mockData.Name, job.Name)
			assert.Equal(t, mockData.Enabled, job.Enabled)
			assert.Equal(t, mockData.Schedule, job.Schedule)
			assert.Equal(t, mockData.OlderThan, job.OlderThan)
			assert.Equal(t, mockData.Indexers, job.Indexers)
			assert.Equal(t, mockData.Statuses, job.Statuses)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
		})
	}
}

func TestReleaseCleanupJobRepo_FindByID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseCleanupJobRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("FindByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Execute
			job, err := repo.FindByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, mockData.Name, job.Name)
			assert.Equal(t, mockData.ID, job.ID)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_Not_Found [%s]", dbType), func(t *testing.T) {
			// Execute
			_, err := repo.FindByID(context.Background(), 99999)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestReleaseCleanupJobRepo_List(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseCleanupJobRepo(log, db)

		t.Run(fmt.Sprintf("List_Returns_All_Jobs [%s]", dbType), func(t *testing.T) {
			// Setup - create multiple jobs
			job1 := getMockReleaseCleanupJob()
			job1.Name = "Job 1"
			job2 := getMockReleaseCleanupJob()
			job2.Name = "Job 2"
			job3 := getMockReleaseCleanupJob()
			job3.Name = "Job 3"

			err := repo.Store(context.Background(), job1)
			assert.NoError(t, err)
			err = repo.Store(context.Background(), job2)
			assert.NoError(t, err)
			err = repo.Store(context.Background(), job3)
			assert.NoError(t, err)

			// Execute
			jobs, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, len(jobs), 3)

			// Verify - find our test jobs
			foundJobs := 0
			for _, job := range jobs {
				if job.Name == "Job 1" || job.Name == "Job 2" || job.Name == "Job 3" {
					foundJobs++
				}
			}
			assert.Equal(t, 3, foundJobs)

			// Cleanup
			_ = repo.Delete(context.Background(), job1.ID)
			_ = repo.Delete(context.Background(), job2.ID)
			_ = repo.Delete(context.Background(), job3.ID)
		})

		t.Run(fmt.Sprintf("List_Empty_Table [%s]", dbType), func(t *testing.T) {
			// Execute
			jobs, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, jobs)
		})
	}
}

func TestReleaseCleanupJobRepo_Update(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseCleanupJobRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("Update_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Update data
			mockData.Name = "Updated Name"
			mockData.Schedule = "0 4 * * *"
			mockData.OlderThan = 168
			mockData.Enabled = false
			mockData.Indexers = "hdt,blu"
			mockData.Statuses = "PUSH_APPROVED"

			// Execute
			err = repo.Update(context.Background(), mockData)
			assert.NoError(t, err)

			// Verify
			updatedJob, err := repo.FindByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, "Updated Name", updatedJob.Name)
			assert.Equal(t, "0 4 * * *", updatedJob.Schedule)
			assert.Equal(t, 168, updatedJob.OlderThan)
			assert.Equal(t, false, updatedJob.Enabled)
			assert.Equal(t, "hdt,blu", updatedJob.Indexers)
			assert.Equal(t, "PUSH_APPROVED", updatedJob.Statuses)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
		})

		t.Run(fmt.Sprintf("Update_Fails_Non_Existing_Job [%s]", dbType), func(t *testing.T) {
			// Setup
			nonExistingJob := getMockReleaseCleanupJob()
			nonExistingJob.ID = 99999

			// Execute
			err := repo.Update(context.Background(), nonExistingJob)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestReleaseCleanupJobRepo_UpdateLastRun(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseCleanupJobRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("UpdateLastRun_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Update last run data
			newLastRun := time.Now().Add(-1 * time.Hour)
			mockData.LastRun = newLastRun
			mockData.LastRunStatus = domain.ReleaseCleanupStatusError
			mockData.LastRunData = `{"error": "test error"}`

			// Execute
			err = repo.UpdateLastRun(context.Background(), mockData)
			assert.NoError(t, err)

			// Verify
			updatedJob, err := repo.FindByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, domain.ReleaseCleanupStatusError, updatedJob.LastRunStatus)
			assert.Equal(t, `{"error": "test error"}`, updatedJob.LastRunData)
			assert.WithinDuration(t, newLastRun, updatedJob.LastRun, 2*time.Second)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
		})

		t.Run(fmt.Sprintf("UpdateLastRun_Fails_Non_Existing_Job [%s]", dbType), func(t *testing.T) {
			// Setup
			nonExistingJob := getMockReleaseCleanupJob()
			nonExistingJob.ID = 99999

			// Execute
			err := repo.UpdateLastRun(context.Background(), nonExistingJob)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestReleaseCleanupJobRepo_ToggleEnabled(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseCleanupJobRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("ToggleEnabled_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData.Enabled = true
			err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Execute - disable
			err = repo.ToggleEnabled(context.Background(), mockData.ID, false)
			assert.NoError(t, err)

			// Verify
			job, err := repo.FindByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.False(t, job.Enabled)

			// Execute - enable
			err = repo.ToggleEnabled(context.Background(), mockData.ID, true)
			assert.NoError(t, err)

			// Verify
			job, err = repo.FindByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.True(t, job.Enabled)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
		})

		t.Run(fmt.Sprintf("ToggleEnabled_Fails_Non_Existing_Job [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.ToggleEnabled(context.Background(), 99999, false)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestReleaseCleanupJobRepo_Delete(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseCleanupJobRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Execute
			err = repo.Delete(context.Background(), mockData.ID)
			assert.NoError(t, err)

			// Verify
			_, err = repo.FindByID(context.Background(), mockData.ID)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})

		t.Run(fmt.Sprintf("Delete_Fails_Non_Existing_Job [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.Delete(context.Background(), 99999)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}
