// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification/services/bark"

	"github.com/rs/zerolog"
)

type barkSender struct {
	baseSender
	client *bark.Client
}

func NewBarkSender(log zerolog.Logger, settings *domain.Notification) *barkSender {
	return &barkSender{
		baseSender: newBaseSender("bark", settings),
		client: bark.NewSender(log, bark.Config{
			Host:      settings.Host,
			DeviceKey: settings.Token,
			Name:      settings.Name,
		}),
	}
}

// barkLevels maps the notification priority onto Bark interruption levels
// using the Pushover scale (-1 quiet, 1 high, 2 emergency); anything else
// leaves the level unset so Bark falls back to active.
var barkLevels = map[int32]string{
	-1: "passive",
	1:  "timeSensitive",
	2:  "critical",
}

func (s *barkSender) Send(ctx context.Context, payload domain.NotificationPayload) error {
	msg := buildBarkMessage(payload)
	msg.Sound = s.Settings.Sound
	msg.Group = s.Settings.Topic
	msg.Level = barkLevels[s.Settings.Priority]

	return s.client.SendMessage(ctx, msg)
}

func buildBarkMessage(payload domain.NotificationPayload) *bark.Message {
	return &bark.Message{
		Title: BuildTitle(payload.Event),
		Body:  plainTextBuilder.BuildBody(payload),
		Icon:  defaultImageURL,
		URL:   payload.URL,
	}
}
