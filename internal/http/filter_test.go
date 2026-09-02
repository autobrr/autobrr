// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

type filterNotificationServiceStub struct {
	filterService
	filterID      int
	notifications []domain.FilterNotification
}

func (s *filterNotificationServiceStub) UpdateNotifications(_ context.Context, filterID int, notifications []domain.FilterNotification) error {
	s.filterID = filterID
	s.notifications = notifications
	return nil
}

func TestFilterHandlerUpdateNotifications(t *testing.T) {
	service := &filterNotificationServiceStub{}
	handler := newFilterHandler(encoder{}, service)
	router := chi.NewRouter()
	handler.Routes(router)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/7/notifications/", strings.NewReader(`[{"notification_id":3,"events":["PUSH_APPROVED"]}]`))
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, 7, service.filterID)
	assert.Equal(t, []domain.FilterNotification{{NotificationID: 3, Events: []string{"PUSH_APPROVED"}}}, service.notifications)
}

type filterExternalTestServiceStub struct {
	filterService
	called   bool
	external *domain.FilterExternal
}

func (s *filterExternalTestServiceStub) TestExternal(_ context.Context, external *domain.FilterExternal) (*domain.FilterExternalTestResult, error) {
	s.called = true
	s.external = external
	return &domain.FilterExternalTestResult{Success: true}, nil
}

func TestFilterHandlerTestExternal(t *testing.T) {
	service := &filterExternalTestServiceStub{}
	handler := newFilterHandler(encoder{}, service)
	router := chi.NewRouter()
	handler.Routes(router)

	post := func(body string) *httptest.ResponseRecorder {
		service.called = false
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/external/test", strings.NewReader(body))
		router.ServeHTTP(recorder, request)
		return recorder
	}

	t.Run("runs the posted external filter", func(t *testing.T) {
		recorder := post(`{"name":"hook","type":"WEBHOOK","webhook_host":"https://mock.local/hook","webhook_expect_status":200}`)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.True(t, service.called)
		assert.Equal(t, "hook", service.external.Name)
		assert.Contains(t, recorder.Body.String(), `"success":true`)
	})

	t.Run("rejects malformed json", func(t *testing.T) {
		recorder := post(`{"type":`)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.False(t, service.called)
	})

	t.Run("rejects null body", func(t *testing.T) {
		recorder := post(`null`)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.False(t, service.called)
	})
}
