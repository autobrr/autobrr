// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"
	"uuid"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification/services/webhook"

	"github.com/rs/zerolog"
)

type webhookSender struct {
	baseSender
	client *webhook.Client
}

func NewWebhookSender(log zerolog.Logger, settings *domain.Notification) *webhookSender {
	return &webhookSender{
		baseSender: newBaseSender("webhook", settings),
		client: webhook.NewSender(log, webhook.Config{
			URL:     settings.Webhook,
			Method:  settings.Method,
			Headers: settings.Headers,
			Name:    settings.Name,
		}),
	}
}

func (s *webhookSender) Send(ctx context.Context, payload domain.NotificationPayload) error {
	return s.client.SendMessage(ctx, buildWebhookMessage(payload))
}

func buildWebhookMessage(payload domain.NotificationPayload) *webhook.Message {
	event := domain.NewWebhookEvent(payload.Event, payload, uuid.NewV7().String())

	return &webhook.Message{
		Event:   string(event.Event),
		Payload: event,
	}
}
