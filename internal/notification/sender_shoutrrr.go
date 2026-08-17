// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification/services/shoutrrr"

	"github.com/rs/zerolog"
)

type shoutrrrService struct {
	baseSender

	client *shoutrrr.Client
}

func NewShoutrrrService(log zerolog.Logger, settings *domain.Notification) Sender {
	return &shoutrrrService{
		baseSender: newBaseSender("shoutrrr", log, settings),
		client: shoutrrr.NewSender(log, shoutrrr.Config{
			URL:  settings.Host,
			Name: settings.Name,
		}),
	}
}

func (s *shoutrrrService) Name() string {
	return "shoutrrr"
}

func (s *shoutrrrService) Send(ctx context.Context, payload domain.NotificationPayload) error {
	return s.client.SendMessage(ctx, buildShoutrrrMessage(payload))
}

func buildShoutrrrMessage(payload domain.NotificationPayload) *shoutrrr.Message {
	return &shoutrrr.Message{
		Message: plainTextBuilder.BuildBody(payload),
	}
}
