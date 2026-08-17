// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification/services/webhook"
	"github.com/google/uuid"

	"github.com/rs/zerolog"
)

type webhookService struct {
	baseSender

	client *webhook.Client
}

func NewWebhookService(log zerolog.Logger, settings *domain.Notification) Sender {
	return &webhookService{
		baseSender: newBaseSender("webhook", log, settings),
		client: webhook.NewSender(log, webhook.Config{
			URL:     settings.Webhook,
			Method:  settings.Method,
			Headers: settings.Headers,
			Name:    settings.Name,
		}),
	}
}

func (s *webhookService) Name() string {
	return "webhook"
}

func (s *webhookService) Send(ctx context.Context, payload domain.NotificationPayload) error {
	return s.client.SendMessage(ctx, buildWebhookMessage(payload))
}

func buildWebhookMessage(payload domain.NotificationPayload) *webhook.Message {
	event := domain.NewWebhookEvent(payload.Event, payload, uuid.New().String())

	return &webhook.Message{
		Event:   string(event.Event),
		Payload: event,
	}
}
