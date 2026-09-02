// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"fmt"
	"html"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/dustin/go-humanize"
)

type MessageBuilder interface {
	BuildBody(payload domain.NotificationPayload) string
}

// The builders are stateless, so the per-service message builders share these.
var (
	plainTextBuilder = &MessageBuilderPlainText{}
	htmlBuilder      = &MessageBuilderHTML{}
)

type ConditionMessagePart struct {
	Condition bool
	Format    string
	Bits      []any
}

// MessageBuilderPlainText constructs the body of the notification message in plain text format.
type MessageBuilderPlainText struct{}

// BuildBody constructs the body of the notification message.
func (b *MessageBuilderPlainText) BuildBody(payload domain.NotificationPayload) string {
	messageParts := []ConditionMessagePart{
		{payload.Sender != "", "%v\n", []any{payload.Sender}},
		{payload.Subject != "" && payload.Message != "", "%v\n%v", []any{payload.Subject, payload.Message}},
		{payload.ReleaseName != "", "New release: %v\n", []any{payload.ReleaseName}},
		{payload.Size > 0, "Size: %v\n", []any{humanize.Bytes(payload.Size)}},
		{payload.Status != "", "Status: %v\n", []any{payload.Status.String()}},
		{payload.Indexer != "", "Indexer: %v\n", []any{payload.Indexer}},
		{payload.Filter != "", "Filter: %v\n", []any{payload.Filter}},
		{payload.Action != "", "Action: %v: %v\n", []any{payload.ActionType, payload.Action}},
		{payload.Action != "" && payload.ActionClient != "", "Client: %v\n", []any{payload.ActionClient}},
		{len(payload.Rejections) > 0, "Rejections: %v\n", []any{strings.Join(payload.Rejections, ", ")}},
	}

	return formatMessageContent(messageParts)
}

// MessageBuilderHTML constructs the body of the notification message in HTML format.
type MessageBuilderHTML struct{}

func (b *MessageBuilderHTML) BuildBody(payload domain.NotificationPayload) string {
	messageParts := []ConditionMessagePart{
		{payload.Sender != "", "<b>%v</b>\n", []any{html.EscapeString(payload.Sender)}},
		{payload.Subject != "" && payload.Message != "", "<b>%v</b> %v\n", []any{html.EscapeString(payload.Subject), html.EscapeString(payload.Message)}},
		{payload.ReleaseName != "", "<b>New release:</b> %v\n", []any{html.EscapeString(payload.ReleaseName)}},
		{payload.Size > 0, "<b>Size:</b> %v\n", []any{humanize.Bytes(payload.Size)}},
		{payload.Status != "", "<b>Status:</b> %v\n", []any{html.EscapeString(payload.Status.String())}},
		{payload.Indexer != "", "<b>Indexer:</b> %v\n", []any{html.EscapeString(payload.Indexer)}},
		{payload.Filter != "", "<b>Filter:</b> %v\n", []any{html.EscapeString(payload.Filter)}},
		{payload.Action != "", "<b>Action:</b> %v: %v\n", []any{payload.ActionType, html.EscapeString(payload.Action)}},
		{payload.Action != "" && payload.ActionClient != "", "<b>Client:</b> %v\n", []any{html.EscapeString(payload.ActionClient)}},
		{len(payload.Rejections) > 0, "<b>Rejections:</b> %v\n", []any{html.EscapeString(strings.Join(payload.Rejections, ", "))}},
	}

	return formatMessageContent(messageParts)
}

func formatMessageContent(messageParts []ConditionMessagePart) string {
	var builder strings.Builder
	for _, part := range messageParts {
		if part.Condition {
			builder.WriteString(fmt.Sprintf(part.Format, part.Bits...))
		}
	}
	return builder.String()
}

// BuildTitle constructs the title of the notification message.
func BuildTitle(event domain.NotificationEvent) string {
	titles := map[domain.NotificationEvent]string{
		domain.NotificationEventAppUpdateAvailable: "Autobrr update available",
		domain.NotificationEventPushApproved:       "Push Approved",
		domain.NotificationEventPushRejected:       "Push Rejected",
		domain.NotificationEventPushError:          "Push Error",
		domain.NotificationEventIRCDisconnected:    "IRC Disconnected",
		domain.NotificationEventIRCReconnected:     "IRC Reconnected",
		domain.NotificationEventReleaseNew:         "New Release",
		domain.NotificationEventTest:               "Test",
	}

	if title, ok := titles[event]; ok {
		return title
	}

	return "New Event"
}
