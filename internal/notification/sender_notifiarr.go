// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification/services/notifiarr"

	"github.com/rs/zerolog"
)

type notifiarrService struct {
	baseSender

	client *notifiarr.Client
}

func NewNotifiarrService(log zerolog.Logger, settings *domain.Notification) Sender {
	return &notifiarrService{
		baseSender: newBaseSender("notifiarr", log, settings),
		client: notifiarr.NewSender(log, notifiarr.Config{
			APIKey: settings.APIKey,
			Name:   settings.Name,
		}),
	}
}

func (s *notifiarrService) Name() string {
	return "notifiarr"
}

func (s *notifiarrService) Send(ctx context.Context, payload domain.NotificationPayload) error {
	return s.client.SendMessage(ctx, buildNotifiarrMessage(payload))
}

func buildNotifiarrMessage(payload domain.NotificationPayload) *notifiarr.Message {
	data := notifiarr.MessageData{
		Event:     string(payload.Event),
		Timestamp: payload.Timestamp,
	}

	if payload.Subject != "" && payload.Message != "" {
		data.Subject = payload.Subject
		data.Message = payload.Message
	}
	if payload.ReleaseName != "" {
		data.ReleaseName = &payload.ReleaseName
	}
	if payload.Status != "" {
		status := string(payload.Status)
		data.Status = &status
	}
	if payload.Indexer != "" {
		data.Indexer = &payload.Indexer
	}
	if payload.Filter != "" {
		data.Filter = &payload.Filter
	}
	if payload.Action != "" || payload.ActionClient != "" {
		data.Action = &payload.Action

		if payload.ActionClient != "" {
			data.ActionClient = &payload.ActionClient
		}
	}
	if len(payload.Rejections) > 0 {
		data.Rejections = payload.Rejections
	}

	return &notifiarr.Message{
		Event: string(payload.Event),
		Data:  data,
	}
}
