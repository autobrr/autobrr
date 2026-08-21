// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification/services/ntfy"

	"github.com/rs/zerolog"
)

type ntfySender struct {
	baseSender
	client *ntfy.Client
}

func NewNtfySender(log zerolog.Logger, settings *domain.Notification) *ntfySender {
	return &ntfySender{
		baseSender: newBaseSender("ntfy", settings),
		client: ntfy.NewSender(log, ntfy.Config{
			Host:     settings.Host,
			Token:    settings.Token,
			Username: settings.Username,
			Password: settings.Password,
			Name:     settings.Name,
		}),
	}
}

func (s *ntfySender) Send(ctx context.Context, payload domain.NotificationPayload) error {
	msg := buildNtfyMessage(payload)
	msg.Priority = s.Settings.Priority
	msg.Tags = s.Settings.Topic

	return s.client.SendMessage(ctx, msg)
}

func buildNtfyMessage(payload domain.NotificationPayload) *ntfy.Message {
	return &ntfy.Message{
		Title:   BuildTitle(payload.Event),
		Message: plainTextBuilder.BuildBody(payload),
	}
}
