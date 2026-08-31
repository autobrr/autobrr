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
	"github.com/stretchr/testify/assert"
)

type ircAuthValidationService struct {
	storeCalls  int
	updateCalls int
}

func (s *ircAuthValidationService) ListNetworks(context.Context) ([]domain.IrcNetwork, error) {
	return nil, nil
}

func (s *ircAuthValidationService) GetNetworksWithHealth(context.Context) ([]domain.IrcNetworkWithHealth, error) {
	return nil, nil
}

func (s *ircAuthValidationService) DeleteNetwork(context.Context, int64) error { return nil }

func (s *ircAuthValidationService) GetNetworkByID(context.Context, int64) (*domain.IrcNetwork, error) {
	return nil, nil
}

func (s *ircAuthValidationService) StoreNetwork(context.Context, *domain.IrcNetwork) error {
	s.storeCalls++
	return nil
}

func (s *ircAuthValidationService) UpdateNetwork(context.Context, *domain.IrcNetwork) error {
	s.updateCalls++
	return nil
}

func (s *ircAuthValidationService) StoreChannel(context.Context, int64, *domain.IrcChannel) error {
	return nil
}

func (s *ircAuthValidationService) RestartNetwork(context.Context, int64) error { return nil }

func (s *ircAuthValidationService) SendCmd(context.Context, *domain.SendIrcCmdRequest) error {
	return nil
}

func (s *ircAuthValidationService) ManualProcessAnnounce(context.Context, *domain.IRCManualProcessRequest) error {
	return nil
}

func (s *ircAuthValidationService) GetMessageHistory(context.Context, int64, string) ([]domain.IrcMessage, error) {
	return nil, nil
}

func TestIRCHandlerRejectsInvalidAuthBeforeService(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "unknown mechanism", body: `{"auth":{"mechanism":"TYPO","password":"secret"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ircAuthValidationService{}
			handler := newIrcHandler(encoder{}, nil, service)

			for _, method := range []string{http.MethodPost, http.MethodPut} {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(method, "/", strings.NewReader(tt.body))
				if method == http.MethodPost {
					handler.storeNetwork(recorder, request)
				} else {
					handler.updateNetwork(recorder, request)
				}

				assert.Equal(t, http.StatusBadRequest, recorder.Code, method)
			}

			assert.Zero(t, service.storeCalls)
			assert.Zero(t, service.updateCalls)
		})
	}
}

func TestIRCHandlerAcceptsCompatibleAuth(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "omitted mechanism", body: `{"auth":{"password":"secret"}}`},
		{name: "legacy incomplete sasl", body: `{"auth":{"mechanism":"SASL_PLAIN"}}`},
		{name: "password only nickserv", body: `{"auth":{"mechanism":"NICKSERV","password":"secret"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ircAuthValidationService{}
			handler := newIrcHandler(encoder{}, nil, service)

			for _, method := range []string{http.MethodPost, http.MethodPut} {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(method, "/", strings.NewReader(tt.body))
				if method == http.MethodPost {
					handler.storeNetwork(recorder, request)
				} else {
					handler.updateNetwork(recorder, request)
				}

				assert.Equal(t, http.StatusNoContent, recorder.Code, method)
			}

			assert.Equal(t, 1, service.storeCalls)
			assert.Equal(t, 1, service.updateCalls)
		})
	}
}
