// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncateForDiscord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		limit int
		want  string
	}{
		{
			name:  "shorter than limit is untouched",
			input: "hello",
			limit: 10,
			want:  "hello",
		},
		{
			name:  "exactly at limit is untouched",
			input: "hello",
			limit: 5,
			want:  "hello",
		},
		{
			name:  "keeps both ends and drops the middle",
			input: "abcdefghijklmnop",
			limit: 13,
			want:  "abc [...] nop",
		},
		{
			name:  "limit equal to the marker just cuts",
			input: "abcdefghij",
			limit: 7,
			want:  "abcdefg",
		},
		{
			name:  "one over the marker keeps a single leading rune",
			input: "abcdefghij",
			limit: 8,
			want:  "a [...] ",
		},
		{
			name:  "limit smaller than the marker just cuts",
			input: "abcdefgh",
			limit: 3,
			want:  "abc",
		},
		{
			name:  "zero limit yields empty",
			input: "abcdefgh",
			limit: 0,
			want:  "",
		},
		{
			name:  "negative limit yields empty",
			input: "abcdefgh",
			limit: -1,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := truncateForDiscord(tt.input, tt.limit)

			assert.Equal(t, tt.want, got)
			if tt.limit > 0 {
				assert.LessOrEqual(t, utf8.RuneCountInString(got), tt.limit)
			}
		})
	}
}

func TestTruncateForDiscord_CountsRunesNotBytes(t *testing.T) {
	t.Parallel()

	got := truncateForDiscord(strings.Repeat("日", 500), 100)

	assert.Equal(t, 100, utf8.RuneCountInString(got))
	assert.NotContains(t, got, "�")
}

// A failed *arr push puts the whole http.Request in its error string, which
// reaches the Reasons field well over the 1024 character limit. Discord answers
// 400 {"embeds": ["0"]} and drops the entire notification.
func TestDiscordBuildEmbed_RespectsEmbedLimits(t *testing.T) {
	t.Parallel()

	sender := &discordSender{}

	payload := domain.NotificationPayload{
		Event:        domain.NotificationEventPushError,
		ReleaseName:  strings.Repeat("Very Long Release Name ", 50),
		Filter:       strings.Repeat("f", 2000),
		Indexer:      strings.Repeat("i", 2000),
		Action:       strings.Repeat("a", 2000),
		ActionClient: strings.Repeat("c", 2000),
		Status:       domain.ReleasePushStatusErr,
		ActionType:   domain.ActionTypeRadarr,
		Rejections:   []string{"radarr failed to push release: " + strings.Repeat("x", 4000) + ": i/o timeout"},
	}

	embed := sender.buildEmbed(payload.Event, payload)

	assert.LessOrEqual(t, utf8.RuneCountInString(embed.Title), discordEmbedTitleLimit)
	assert.LessOrEqual(t, utf8.RuneCountInString(embed.Description), discordEmbedDescriptionLimit)

	var reasons string
	for _, f := range embed.Fields {
		assert.LessOrEqual(t, utf8.RuneCountInString(f.Name), discordEmbedFieldNameLimit)
		assert.LessOrEqual(t, utf8.RuneCountInString(f.Value), discordEmbedFieldValueLimit, "field %q", f.Name)

		if f.Name == "Reasons" {
			reasons = f.Value
		}
	}

	require.NotEmpty(t, reasons)
	assert.True(t, strings.HasPrefix(reasons, "```\n"), "opening code fence was cut")
	assert.True(t, strings.HasSuffix(reasons, "\n```"), "closing code fence was cut")
	// the cause is wrapped last, so it is the part that has to survive
	assert.Contains(t, reasons, "i/o timeout")
}

func TestDiscordBuildEmbed_TruncatesLongSubjectAndMessage(t *testing.T) {
	t.Parallel()

	sender := &discordSender{}

	payload := domain.NotificationPayload{
		Event:   domain.NotificationEventTest,
		Subject: strings.Repeat("s", 1000),
		Message: strings.Repeat("m", 9000),
	}

	embed := sender.buildEmbed(payload.Event, payload)

	assert.Equal(t, discordEmbedTitleLimit, utf8.RuneCountInString(embed.Title))
	assert.Equal(t, discordEmbedDescriptionLimit, utf8.RuneCountInString(embed.Description))
}
