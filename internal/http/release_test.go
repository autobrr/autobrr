// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// Mock releaseService for testing cleanup job endpoints
type releaseServiceMock struct {
	cleanupJobs map[int]*domain.ReleaseCleanupJob
	nextID      int
}

func newReleaseServiceMock() *releaseServiceMock {
	return &releaseServiceMock{
		cleanupJobs: make(map[int]*domain.ReleaseCleanupJob),
		nextID:      1,
	}
}

// Cleanup job methods
func (m *releaseServiceMock) ListCleanupJobs(ctx context.Context) ([]*domain.ReleaseCleanupJob, error) {
	jobs := make([]*domain.ReleaseCleanupJob, 0, len(m.cleanupJobs))
	for _, job := range m.cleanupJobs {
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (m *releaseServiceMock) GetCleanupJob(ctx context.Context, id int) (*domain.ReleaseCleanupJob, error) {
	job, ok := m.cleanupJobs[id]
	if !ok {
		return nil, domain.ErrRecordNotFound
	}
	return job, nil
}

func (m *releaseServiceMock) StoreCleanupJob(ctx context.Context, job *domain.ReleaseCleanupJob) error {
	job.ID = m.nextID
	m.nextID++
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	m.cleanupJobs[job.ID] = job
	return nil
}

func (m *releaseServiceMock) UpdateCleanupJob(ctx context.Context, job *domain.ReleaseCleanupJob) error {
	if _, ok := m.cleanupJobs[job.ID]; !ok {
		return domain.ErrRecordNotFound
	}
	job.UpdatedAt = time.Now()
	m.cleanupJobs[job.ID] = job
	return nil
}

func (m *releaseServiceMock) DeleteCleanupJob(ctx context.Context, id int) error {
	if _, ok := m.cleanupJobs[id]; !ok {
		return domain.ErrRecordNotFound
	}
	delete(m.cleanupJobs, id)
	return nil
}

func (m *releaseServiceMock) ToggleCleanupJobEnabled(ctx context.Context, id int, enabled bool) error {
	job, ok := m.cleanupJobs[id]
	if !ok {
		return domain.ErrRecordNotFound
	}
	job.Enabled = enabled
	job.UpdatedAt = time.Now()
	return nil
}

func (m *releaseServiceMock) ForceRunCleanupJob(ctx context.Context, id int) error {
	job, ok := m.cleanupJobs[id]
	if !ok {
		return domain.ErrRecordNotFound
	}
	job.LastRun = time.Now()
	job.LastRunStatus = domain.ReleaseCleanupStatusSuccess
	job.LastRunData = `{"test": true}`
	return nil
}

// Stub implementations for other required interface methods
func (m *releaseServiceMock) Find(ctx context.Context, query domain.ReleaseQueryParams) (*domain.FindReleasesResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *releaseServiceMock) Get(ctx context.Context, req *domain.GetReleaseRequest) (*domain.Release, error) {
	return nil, errors.New("not implemented")
}

func (m *releaseServiceMock) GetIndexerOptions(ctx context.Context) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (m *releaseServiceMock) Stats(ctx context.Context) (*domain.ReleaseStats, error) {
	return nil, errors.New("not implemented")
}

func (m *releaseServiceMock) Delete(ctx context.Context, req *domain.DeleteReleaseRequest) error {
	return errors.New("not implemented")
}

func (m *releaseServiceMock) Retry(ctx context.Context, req *domain.ReleaseActionRetryReq) error {
	return errors.New("not implemented")
}

func (m *releaseServiceMock) ProcessManual(ctx context.Context, req *domain.ReleaseProcessReq) error {
	return errors.New("not implemented")
}

func (m *releaseServiceMock) StoreReleaseProfileDuplicate(ctx context.Context, profile *domain.DuplicateReleaseProfile) error {
	return errors.New("not implemented")
}

func (m *releaseServiceMock) FindDuplicateReleaseProfiles(ctx context.Context) ([]*domain.DuplicateReleaseProfile, error) {
	return nil, errors.New("not implemented")
}

func (m *releaseServiceMock) DeleteReleaseProfileDuplicate(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func setupReleaseHandler(service releaseService) chi.Router {
	encoder := encoder{}
	handler := newReleaseHandler(encoder, service)

	r := chi.NewRouter()
	r.Route("/api/releases", handler.Routes)

	return r
}

func TestReleaseHandler_ListCleanupJobs(t *testing.T) {
	t.Parallel()

	service := newReleaseServiceMock()
	service.cleanupJobs[1] = &domain.ReleaseCleanupJob{
		ID:        1,
		Name:      "Test Job",
		Enabled:   true,
		Schedule:  "0 3 * * *",
		OlderThan: 720,
		Indexers:  "btn",
		Statuses:  "PUSH_REJECTED",
	}

	router := setupReleaseHandler(service)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/api/releases/cleanup-jobs")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jobs []*domain.ReleaseCleanupJob
	err = json.NewDecoder(resp.Body).Decode(&jobs)
	assert.NoError(t, err)
	assert.Len(t, jobs, 1)

	// Verify ALL fields for returned job
	assert.Equal(t, 1, jobs[0].ID)
	assert.Equal(t, "Test Job", jobs[0].Name)
	assert.True(t, jobs[0].Enabled)
	assert.Equal(t, "0 3 * * *", jobs[0].Schedule)
	assert.Equal(t, 720, jobs[0].OlderThan)
	assert.Equal(t, "btn", jobs[0].Indexers)
	assert.Equal(t, "PUSH_REJECTED", jobs[0].Statuses)
}

func TestReleaseHandler_GetCleanupJob(t *testing.T) {
	t.Parallel()

	service := newReleaseServiceMock()
	service.cleanupJobs[1] = &domain.ReleaseCleanupJob{
		ID:        1,
		Name:      "Test Job",
		Enabled:   true,
		Schedule:  "0 3 * * *",
		OlderThan: 720,
		Indexers:  "btn,ptp",
		Statuses:  "PUSH_REJECTED,PUSH_ERROR",
	}

	router := setupReleaseHandler(service)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/api/releases/cleanup-jobs/1")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var job domain.ReleaseCleanupJob
	err = json.NewDecoder(resp.Body).Decode(&job)
	assert.NoError(t, err)

	// Verify ALL fields
	assert.Equal(t, 1, job.ID)
	assert.Equal(t, "Test Job", job.Name)
	assert.True(t, job.Enabled)
	assert.Equal(t, "0 3 * * *", job.Schedule)
	assert.Equal(t, 720, job.OlderThan)
	assert.Equal(t, "btn,ptp", job.Indexers)
	assert.Equal(t, "PUSH_REJECTED,PUSH_ERROR", job.Statuses)
}

func TestReleaseHandler_GetCleanupJob_NotFound(t *testing.T) {
	t.Parallel()

	service := newReleaseServiceMock()

	router := setupReleaseHandler(service)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/api/releases/cleanup-jobs/999")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestReleaseHandler_StoreCleanupJob(t *testing.T) {
	t.Parallel()

	service := newReleaseServiceMock()

	router := setupReleaseHandler(service)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	jobData := &domain.ReleaseCleanupJob{
		Name:      "New Job",
		Enabled:   false,
		Schedule:  "0 2 * * *",
		OlderThan: 168,
		Indexers:  "btn,ptp",
		Statuses:  "PUSH_REJECTED",
	}

	body, err := json.Marshal(jobData)
	assert.NoError(t, err)

	resp, err := http.Post(testServer.URL+"/api/releases/cleanup-jobs", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var created domain.ReleaseCleanupJob
	err = json.NewDecoder(resp.Body).Decode(&created)
	assert.NoError(t, err)

	// Verify ALL fields in response
	assert.Equal(t, 1, created.ID) // First auto-generated ID
	assert.Equal(t, "New Job", created.Name)
	assert.False(t, created.Enabled)
	assert.Equal(t, "0 2 * * *", created.Schedule)
	assert.Equal(t, 168, created.OlderThan)
	assert.Equal(t, "btn,ptp", created.Indexers)
	assert.Equal(t, "PUSH_REJECTED", created.Statuses)

	// Verify job was actually stored in service with all fields
	storedJob, exists := service.cleanupJobs[created.ID]
	assert.True(t, exists, "job should exist in service storage")
	assert.Equal(t, 1, storedJob.ID)
	assert.Equal(t, "New Job", storedJob.Name)
	assert.False(t, storedJob.Enabled)
	assert.Equal(t, "0 2 * * *", storedJob.Schedule)
	assert.Equal(t, 168, storedJob.OlderThan)
	assert.Equal(t, "btn,ptp", storedJob.Indexers)
	assert.Equal(t, "PUSH_REJECTED", storedJob.Statuses)
}

func TestReleaseHandler_UpdateCleanupJob(t *testing.T) {
	t.Parallel()

	service := newReleaseServiceMock()
	service.cleanupJobs[1] = &domain.ReleaseCleanupJob{
		ID:        1,
		Name:      "Original Name",
		Enabled:   false,
		Schedule:  "0 3 * * *",
		OlderThan: 720,
		Indexers:  "btn",
		Statuses:  "PUSH_REJECTED",
	}

	router := setupReleaseHandler(service)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	updateData := &domain.ReleaseCleanupJob{
		ID:        1,
		Name:      "Updated Name",
		Enabled:   true,
		Schedule:  "0 4 * * *",
		OlderThan: 168,
		Indexers:  "btn,ptp",
		Statuses:  "PUSH_ERROR",
	}

	body, err := json.Marshal(updateData)
	assert.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, testServer.URL+"/api/releases/cleanup-jobs/1", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var updated domain.ReleaseCleanupJob
	err = json.NewDecoder(resp.Body).Decode(&updated)
	assert.NoError(t, err)

	// Verify ALL fields in response
	assert.Equal(t, 1, updated.ID)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.True(t, updated.Enabled)
	assert.Equal(t, "0 4 * * *", updated.Schedule)
	assert.Equal(t, 168, updated.OlderThan)
	assert.Equal(t, "btn,ptp", updated.Indexers)
	assert.Equal(t, "PUSH_ERROR", updated.Statuses)

	// Verify ALL fields actually updated in service storage
	storedJob := service.cleanupJobs[1]
	assert.Equal(t, 1, storedJob.ID)
	assert.Equal(t, "Updated Name", storedJob.Name)
	assert.True(t, storedJob.Enabled)
	assert.Equal(t, "0 4 * * *", storedJob.Schedule)
	assert.Equal(t, 168, storedJob.OlderThan)
	assert.Equal(t, "btn,ptp", storedJob.Indexers)
	assert.Equal(t, "PUSH_ERROR", storedJob.Statuses)
}

func TestReleaseHandler_UpdateCleanupJob_NotFound(t *testing.T) {
	t.Parallel()

	service := newReleaseServiceMock()

	router := setupReleaseHandler(service)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	updateData := &domain.ReleaseCleanupJob{
		ID:   999,
		Name: "Does Not Exist",
	}

	body, err := json.Marshal(updateData)
	assert.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, testServer.URL+"/api/releases/cleanup-jobs/999", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestReleaseHandler_DeleteCleanupJob(t *testing.T) {
	t.Parallel()

	service := newReleaseServiceMock()
	service.cleanupJobs[1] = &domain.ReleaseCleanupJob{
		ID:   1,
		Name: "Job To Delete",
	}

	router := setupReleaseHandler(service)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	req, err := http.NewRequest(http.MethodDelete, testServer.URL+"/api/releases/cleanup-jobs/1", nil)
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify deleted
	assert.Empty(t, service.cleanupJobs)
}

func TestReleaseHandler_DeleteCleanupJob_NotFound(t *testing.T) {
	t.Parallel()

	service := newReleaseServiceMock()

	router := setupReleaseHandler(service)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	req, err := http.NewRequest(http.MethodDelete, testServer.URL+"/api/releases/cleanup-jobs/999", nil)
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestReleaseHandler_ToggleCleanupJobEnabled(t *testing.T) {
	t.Parallel()

	service := newReleaseServiceMock()
	service.cleanupJobs[1] = &domain.ReleaseCleanupJob{
		ID:      1,
		Name:    "Toggle Test",
		Enabled: false,
	}

	router := setupReleaseHandler(service)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	toggleData := map[string]bool{"enabled": true}
	body, err := json.Marshal(toggleData)
	assert.NoError(t, err)

	req, err := http.NewRequest(http.MethodPatch, testServer.URL+"/api/releases/cleanup-jobs/1/enabled", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify toggled
	assert.True(t, service.cleanupJobs[1].Enabled)
}

func TestReleaseHandler_ToggleCleanupJobEnabled_NotFound(t *testing.T) {
	t.Parallel()

	service := newReleaseServiceMock()

	router := setupReleaseHandler(service)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	toggleData := map[string]bool{"enabled": true}
	body, err := json.Marshal(toggleData)
	assert.NoError(t, err)

	req, err := http.NewRequest(http.MethodPatch, testServer.URL+"/api/releases/cleanup-jobs/999/enabled", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestReleaseHandler_ForceRunCleanupJob(t *testing.T) {
	t.Parallel()

	service := newReleaseServiceMock()
	service.cleanupJobs[1] = &domain.ReleaseCleanupJob{
		ID:   1,
		Name: "Force Run Test",
	}

	router := setupReleaseHandler(service)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	resp, err := http.Post(testServer.URL+"/api/releases/cleanup-jobs/1/run", "application/json", nil)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify job was "run" (all run-related fields updated)
	job := service.cleanupJobs[1]
	assert.NotZero(t, job.LastRun)
	assert.Equal(t, domain.ReleaseCleanupStatusSuccess, job.LastRunStatus)
	assert.Equal(t, `{"test": true}`, job.LastRunData)
}

func TestReleaseHandler_ForceRunCleanupJob_NotFound(t *testing.T) {
	t.Parallel()

	service := newReleaseServiceMock()

	router := setupReleaseHandler(service)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	resp, err := http.Post(testServer.URL+"/api/releases/cleanup-jobs/999/run", "application/json", nil)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
