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

type ConditionMessagePart struct {
	Format    string
	Bits      []interface{}
	Condition bool
}

// MessageBuilderPlainText constructs the body of the notification message in plain text format.
type MessageBuilderPlainText struct{}

// BuildBody constructs the body of the notification message.
func (b *MessageBuilderPlainText) BuildBody(payload domain.NotificationPayload) string {
	messageParts := []ConditionMessagePart{
		{payload.Sender != "", "%v\n", []interface{}{payload.Sender}},
		{payload.Subject != "" && payload.Message != "", "%v\n%v", []interface{}{payload.Subject, payload.Message}},
		{payload.ReleaseName != "", "New release: %v\n", []interface{}{payload.ReleaseName}},
		{payload.Size > 0, "Size: %v\n", []interface{}{humanize.Bytes(payload.Size)}},
		{payload.Status != "", "Status: %v\n", []interface{}{payload.Status.String()}},
		{payload.Indexer != "", "Indexer: %v\n", []interface{}{payload.Indexer}},
		{payload.Filter != "", "Filter: %v\n", []interface{}{payload.Filter}},
		{payload.Action != "", "Action: %v: %v\n", []interface{}{payload.ActionType, payload.Action}},
		{payload.Action != "" && payload.ActionClient != "", "Client: %v\n", []interface{}{payload.ActionClient}},
		{len(payload.Rejections) > 0, "Rejections: %v\n", []interface{}{strings.Join(payload.Rejections, ", ")}},
	}

	return formatMessageContent(messageParts)
}

// MessageBuilderHTML constructs the body of the notification message in HTML format.
type MessageBuilderHTML struct{}

func (b *MessageBuilderHTML) BuildBody(payload domain.NotificationPayload) string {
	messageParts := []ConditionMessagePart{
		{payload.Sender != "", "<b>%v</b>\n", []interface{}{html.EscapeString(payload.Sender)}},
		{payload.Subject != "" && payload.Message != "", "<b>%v</b> %v\n", []interface{}{html.EscapeString(payload.Subject), html.EscapeString(payload.Message)}},
		{payload.ReleaseName != "", "<b>New release:</b> %v\n", []interface{}{html.EscapeString(payload.ReleaseName)}},
		{payload.Size > 0, "<b>Size:</b> %v\n", []interface{}{humanize.Bytes(payload.Size)}},
		{payload.Status != "", "<b>Status:</b> %v\n", []interface{}{html.EscapeString(payload.Status.String())}},
		{payload.Indexer != "", "<b>Indexer:</b> %v\n", []interface{}{html.EscapeString(payload.Indexer)}},
		{payload.Filter != "", "<b>Filter:</b> %v\n", []interface{}{html.EscapeString(payload.Filter)}},
		{payload.Action != "", "<b>Action:</b> %v: %v\n", []interface{}{payload.ActionType, html.EscapeString(payload.Action)}},
		{payload.Action != "" && payload.ActionClient != "", "<b>Client:</b> %v\n", []interface{}{html.EscapeString(payload.ActionClient)}},
		{len(payload.Rejections) > 0, "<b>Rejections:</b> %v\n", []interface{}{html.EscapeString(strings.Join(payload.Rejections, ", "))}},
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
		domain.NotificationEventTest:               "Test",
	}

	if title, ok := titles[event]; ok {
		return title
	}

	return "New Event"
}
