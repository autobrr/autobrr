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
