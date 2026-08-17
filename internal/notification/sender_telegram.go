// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification/services/telegram"

	"github.com/rs/zerolog"
)

type telegramService struct {
	baseSender

	client *telegram.Client
}

func NewTelegramService(log zerolog.Logger, settings *domain.Notification) Sender {
	threadID := 0
	if settings.Topic != "" {
		var err error
		threadID, err = strconv.Atoi(settings.Topic)
		if err != nil {
			log.Error().Err(err).Str("topic", settings.Topic).Msg("could not parse specified topic as an integer")
		}
	}

	return &telegramService{
		baseSender: newBaseSender("telegram", log, settings),
		client: telegram.NewSender(log, telegram.Config{
			Host:     settings.Host,
			Token:    settings.Token,
			ChatID:   settings.Channel,
			ThreadID: threadID,
			Name:     settings.Name,
		}),
	}
}

func (s *telegramService) Name() string {
	return "telegram"
}

func (s *telegramService) Send(ctx context.Context, payload domain.NotificationPayload) error {
	payload.Sender = s.Settings.Username

	return s.client.SendMessage(ctx, buildTelegramMessage(payload))
}

func buildTelegramMessage(payload domain.NotificationPayload) *telegram.Message {
	return &telegram.Message{
		Text:      htmlBuilder.BuildBody(payload),
		ParseMode: "HTML",
	}
}
