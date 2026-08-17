// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification/services/lunasea"

	"github.com/rs/zerolog"
)

type lunaSeaService struct {
	baseSender

	client *lunasea.Client
}

func NewLunaSeaService(log zerolog.Logger, settings *domain.Notification) Sender {
	return &lunaSeaService{
		baseSender: newBaseSender("lunasea", log, settings),
		client: lunasea.NewSender(log, lunasea.Config{
			WebhookURL: settings.Webhook,
			Name:       settings.Name,
		}),
	}
}

func (s *lunaSeaService) Name() string {
	return "lunasea"
}

func (s *lunaSeaService) Send(ctx context.Context, payload domain.NotificationPayload) error {
	return s.client.SendMessage(ctx, buildLunaSeaMessage(payload))
}

const defaultImageURL = "https://raw.githubusercontent.com/autobrr/autobrr/master/.github/images/logo.png"

func buildLunaSeaMessage(payload domain.NotificationPayload) *lunasea.Message {
	return &lunasea.Message{
		Title: BuildTitle(payload.Event),
		Body:  plainTextBuilder.BuildBody(payload),
		Image: defaultImageURL,
	}
}
