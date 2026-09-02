// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification/services/shoutrrr"

	"github.com/rs/zerolog"
)

type shoutrrrSender struct {
	baseSender
	client *shoutrrr.Client
}

func NewShoutrrrSender(log zerolog.Logger, settings *domain.Notification) *shoutrrrSender {
	return &shoutrrrSender{
		baseSender: newBaseSender("shoutrrr", settings),
		client: shoutrrr.NewSender(log, shoutrrr.Config{
			URL:  settings.Host,
			Name: settings.Name,
		}),
	}
}
func (s *shoutrrrSender) Send(ctx context.Context, payload domain.NotificationPayload) error {
	return s.client.SendMessage(ctx, buildShoutrrrMessage(payload))
}

func buildShoutrrrMessage(payload domain.NotificationPayload) *shoutrrr.Message {
	return &shoutrrr.Message{
		Message: plainTextBuilder.BuildBody(payload),
	}
}
