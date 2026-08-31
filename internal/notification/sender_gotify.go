// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification/services/gotify"

	"github.com/rs/zerolog"
)

type gotifySender struct {
	baseSender
	client *gotify.Client
}

func NewGotifySender(log zerolog.Logger, settings *domain.Notification) *gotifySender {
	return &gotifySender{
		baseSender: newBaseSender("gotify", settings),
		client: gotify.NewSender(log, gotify.Config{
			Host:  settings.Host,
			Token: settings.Token,
			Name:  settings.Name,
		}),
	}
}

func (s *gotifySender) Send(ctx context.Context, payload domain.NotificationPayload) error {
	return s.client.SendMessage(ctx, buildGotifyMessage(payload))
}

func buildGotifyMessage(payload domain.NotificationPayload) *gotify.Message {
	return &gotify.Message{
		Title:   BuildTitle(payload.Event),
		Message: plainTextBuilder.BuildBody(payload),
	}
}
