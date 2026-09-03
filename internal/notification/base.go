// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"github.com/autobrr/autobrr/internal/domain"
)

type baseSender struct {
	Settings   *domain.Notification
	senderName string
}

func newBaseSender(name string, settings *domain.Notification) baseSender {
	return baseSender{
		Settings:   settings,
		senderName: name,
	}
}

func (s *baseSender) IsEnabled() bool {
	return s.Settings.IsEnabled()
}

func (s *baseSender) Name() string {
	return s.senderName
}
