// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/notification/services/discord"
	"github.com/autobrr/autobrr/test/mockdiscord"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A failed *arr push puts the whole http.Request in its error string, which
// reaches the Reasons field well over the 1024 character limit. Discord answers
// 400 {"embeds": ["0"]} and drops the entire notification.
func TestDiscordBuildEmbed_RespectsEmbedLimits(t *testing.T) {
	t.Parallel()

	payload := domain.NotificationPayload{
		Event:        domain.NotificationEventPushError,
		ReleaseName:  strings.Repeat("Very Long Release Name ", 50),
		Release:      &domain.Release{Title: "Very Long Release Name"},
		Filter:       strings.Repeat("f", 2000),
		Indexer:      strings.Repeat("i", 2000),
		Action:       strings.Repeat("a", 2000),
		ActionClient: strings.Repeat("c", 2000),
		Status:       domain.ReleasePushStatusErr,
		ActionType:   domain.ActionTypeRadarr,
		Rejections:   []string{"radarr failed to push release: " + strings.Repeat("x", 4000) + ": i/o timeout"},
	}

	e, err := buildDiscordMessage(payload)
	assert.NoError(t, err)

	assert.NotNil(t, e)
	assert.NotNil(t, e.Embeds)

	embed := e.Embeds[0]

	assert.LessOrEqual(t, utf8.RuneCountInString(embed.Title), discord.EmbedTitleLimit)
	assert.LessOrEqual(t, utf8.RuneCountInString(embed.Description), discord.EmbedDescriptionLimit)

	var reasons string
	for _, f := range embed.Fields {
		assert.LessOrEqual(t, utf8.RuneCountInString(f.Name), discord.EmbedFieldNameLimit)
		assert.LessOrEqual(t, utf8.RuneCountInString(f.Value), discord.EmbedFieldValueLimit, "field %q", f.Name)

		if f.Name == "Error" {
			reasons = f.Value
		}
	}

	require.NotEmpty(t, reasons)
	assert.True(t, strings.HasPrefix(reasons, "```\n"), "opening code fence was cut")
	assert.True(t, strings.HasSuffix(reasons, "\n```"), "closing code fence was cut")
	// the cause is wrapped last, so it is the part that has to survive
	assert.Contains(t, reasons, "i/o timeout")
}

// Sends a real payload through the mockdiscord webhook server so the sender
// and the documented Discord limits are exercised end to end - an oversized
// embed would come back as a Discord-style 400 and fail the send.
func TestDiscordSender_SendThroughMock(t *testing.T) {
	t.Parallel()

	server := &mockdiscord.Server{}
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	settings := &domain.Notification{
		Name:    "Test Discord",
		Type:    domain.NotificationTypeDiscord,
		Enabled: true,
		Webhook: ts.URL + "/api/webhooks/1/mock-token",
		Events:  []string{string(domain.NotificationEventPushRejected)},
	}

	sender := NewDiscordSender(logger.Mock().With().Logger(), settings)

	payload := domain.NotificationPayload{
		Event:       domain.NotificationEventPushRejected,
		Subject:     "New release!",
		Message:     "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
		ReleaseName: "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
		Release:     &domain.Release{Title: "Best Show Ever", TorrentName: "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP"},
		Filter:      "TV",
		Indexer:     "MockIndexer",
		Status:      domain.ReleasePushStatusRejected,
		Rejections:  []string{strings.Repeat("rejection reason ", 500)},
	}

	require.NoError(t, sender.Send(t.Context(), payload))

	messages := server.Messages()
	require.Len(t, messages, 1)
	require.Len(t, messages[0].Message.Embeds, 1)
	assert.NotEmpty(t, messages[0].Message.Embeds[0].Title)
}
