// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestInitAndRemoveIndexers(t *testing.T) {
	def1 := &domain.IndexerDefinition{
		Identifier: "abc",
		IRC: &domain.IndexerIRC{
			Channels:   []string{"1", "2"},
			Announcers: []string{"3", "4"},
			Parse: &domain.IndexerIRCParse{
				Lines: []domain.IndexerIRCParseLine{},
			},
		},
	}
	def2 := &domain.IndexerDefinition{
		Identifier: "def",
		IRC: &domain.IndexerIRC{
			Channels:   []string{"5", "6"},
			Announcers: []string{"7", "8"},
			Parse: &domain.IndexerIRCParse{
				Lines: []domain.IndexerIRCParseLine{},
			},
		},
	}
	definitions := []*domain.IndexerDefinition{def1, def2}

	log := zerolog.Logger{}
	sse := new(sse.Server)
	network := domain.IrcNetwork{}
	releaseSvc := (release.Service)(nil)
	notificationSvc := (notification.Service)(nil)

	h := NewHandler(log, sse, network, definitions, releaseSvc, notificationSvc)

	channelsLen := len(def1.IRC.Channels) + len(def2.IRC.Channels)
	announcersLen := len(def1.IRC.Announcers) + len(def2.IRC.Announcers)
	assert.Len(t, h.definitions, len(definitions))
	assert.Len(t, h.announceProcessors, channelsLen)
	assert.Len(t, h.channelHealth, channelsLen)
	assert.Len(t, h.validChannels, channelsLen)
	assert.Len(t, h.validAnnouncers, announcersLen)

	h.removeIndexer(def1)
	channelsLen = len(def2.IRC.Channels)
	announcersLen = len(def2.IRC.Announcers)
	assert.Len(t, h.definitions, 1)
	assert.Len(t, h.announceProcessors, channelsLen)
	assert.Len(t, h.channelHealth, channelsLen)
	assert.Len(t, h.validChannels, channelsLen)
	assert.Len(t, h.validAnnouncers, announcersLen)

	h.removeIndexer(def2)
	channelsLen = 0
	announcersLen = 0
	assert.Len(t, h.definitions, 0)
	assert.Len(t, h.announceProcessors, channelsLen)
	assert.Len(t, h.channelHealth, channelsLen)
	assert.Len(t, h.validChannels, channelsLen)
	assert.Len(t, h.validAnnouncers, announcersLen)
}
