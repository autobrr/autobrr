// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification/services/pushover"

	"github.com/rs/zerolog"
)

type pushoverService struct {
	baseSender

	client *pushover.Client
}

func NewPushoverService(log zerolog.Logger, settings *domain.Notification) Sender {
	return &pushoverService{
		baseSender: newBaseSender("pushover", log, settings),
		client: pushover.NewSender(log, pushover.Config{
			Token: settings.APIKey,
			User:  settings.Token,
			Name:  settings.Name,
		}),
	}
}

func (s *pushoverService) Name() string {
	return "pushover"
}

func (s *pushoverService) Send(ctx context.Context, payload domain.NotificationPayload) error {
	msg := buildPushoverMessage(payload)
	msg.Priority = s.Settings.Priority
	msg.Sound = s.sound(payload.Event)

	return s.client.SendMessage(ctx, msg)
}

// sound returns the event specific sound, falling back to the notification wide
// one. An empty sound leaves the choice to the user's own Pushover default.
func (s *pushoverService) sound(event domain.NotificationEvent) string {
	if sound, ok := s.Settings.EventSounds[string(event)]; ok && sound != "" {
		return sound
	}

	return s.Settings.Sound
}

func buildPushoverMessage(payload domain.NotificationPayload) *pushover.Message {
	return &pushover.Message{
		Title:     BuildTitle(payload.Event),
		Message:   htmlBuilder.BuildBody(payload),
		Timestamp: time.Now(),
		HTML:      true,
	}
}
