// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"
	"fmt"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification/services/discord"

	"github.com/rs/zerolog"
)

type discordService struct {
	baseSender

	client *discord.Client
}

func NewDiscordService(log zerolog.Logger, settings *domain.Notification) Sender {
	return &discordService{
		baseSender: newBaseSender("discord", log, settings),
		client: discord.NewSender(log, discord.Config{
			WebHookURL: settings.Webhook,
			Name:       settings.Name,
		}),
	}
}

func (s *discordService) Name() string {
	return "discord"
}

func (s *discordService) Send(ctx context.Context, payload domain.NotificationPayload) error {
	msg, err := buildDiscordMessage(payload)
	if err != nil {
		return err
	}

	return s.client.SendMessage(ctx, msg)
}

func buildDiscordMessage(payload domain.NotificationPayload) (*discord.Message, error) {
	m := discord.NewMessage()
	switch payload.Event {
	case domain.NotificationEventTest:
		e := discord.NewEmbed()
		e.SetAuthor("autobrr: notification test", "", "")
		e.SetTitle("Test event!")
		e.SetColor(discord.LIGHT_BLUE)
		e.AddFields(
			discord.TextField("Message", payload.Message, true),
		)
		m.AddEmbed(e)
		break

	case domain.NotificationEventAppUpdateAvailable:
		e := discord.NewEmbed()
		e.SetAuthor("autobrr: New application update available", "", payload.URL)
		e.SetColor(discord.LIGHT_BLUE)
		e.AddFields(
			discord.TextField("Current version", payload.CurrentVersion, true),
			discord.LinksField("New version", true, discord.NewLink(payload.NewVersion, payload.URL)),
			discord.LinksField("Links", true, discord.NewLink("Changelog", payload.URL)),
		)
		m.AddEmbed(e)
		break

	case domain.NotificationEventPushApproved:
		e := discord.NewEmbed()
		e.SetAuthor("autobrr: release approved", "", "")
		e.SetTitle(payload.Release.Title)
		e.SetColor(discord.GREEN)
		e.SetFooter(&discord.Footer{Text: payload.ActionClient})
		e.SetTimestamp(payload.Timestamp)
		e.AddFields(
			discord.TextField("Release", payload.ReleaseName, false),
			discord.SizeField("Size", payload.Size, true),
			discord.TextField("Indexer", payload.Indexer, true),
			discord.TextField("Filter", payload.Filter, true),
			discord.TextField("Action", payload.Action, true),
			discord.TextField("Protocol", string(payload.Protocol), true),
			discord.TextField("Implementation", string(payload.Implementation), true),
		)
		m.AddEmbed(e)
		break

	case domain.NotificationEventPushRejected:
		e := discord.NewEmbed()
		e.SetAuthor("autobrr: release rejected", "", "")
		e.SetTitle(payload.Release.Title)
		e.SetColor(discord.YELLOW)
		e.SetFooter(&discord.Footer{Text: payload.ActionClient})
		e.SetTimestamp(payload.Timestamp)
		e.AddFields(
			discord.TextField("Release", payload.ReleaseName, false),
			discord.SizeField("Size", payload.Size, true),
			discord.TextField("Indexer", payload.Indexer, true),
			discord.TextField("Filter", payload.Filter, true),
			discord.TextField("Action", payload.Action, true),
			discord.TextField("Protocol", string(payload.Protocol), true),
			discord.TextField("Implementation", string(payload.Implementation), true),
			discord.CodeField("Reason", strings.Join(payload.Rejections, ", "), false),
		)
		m.AddEmbed(e)
		break

	case domain.NotificationEventPushError:
		e := discord.NewEmbed()
		e.SetAuthor("autobrr: release error", "", "")
		e.SetTitle(payload.Release.Title)
		e.SetColor(discord.RED)
		e.SetFooter(&discord.Footer{Text: payload.ActionClient})
		e.SetTimestamp(payload.Timestamp)
		e.AddFields(
			discord.TextField("Release", payload.ReleaseName, false),
			discord.SizeField("Size", payload.Size, true),
			discord.TextField("Indexer", payload.Indexer, true),
			discord.TextField("Filter", payload.Filter, true),
			discord.TextField("Action", payload.Action, true),
			discord.TextField("Protocol", string(payload.Protocol), true),
			discord.TextField("Implementation", string(payload.Implementation), true),
			discord.CodeField("Error", strings.Join(payload.Rejections, ", "), false),
		)
		m.AddEmbed(e)
		break

	case domain.NotificationEventReleaseNew:
		e := discord.NewEmbed()
		e.SetAuthor("autobrr: new release", "", "")
		e.SetTitle(payload.Release.Title)
		e.SetColor(discord.LIGHT_BLUE)
		e.SetTimestamp(payload.Timestamp)
		e.AddFields(
			discord.TextField("Release", payload.ReleaseName, false),
			discord.TextField("Indexer", payload.Indexer, true),
			discord.SizeField("Size", payload.Size, true),
			discord.TextField("Protocol", string(payload.Protocol), true),
			discord.TextField("Implementation", string(payload.Implementation), true),
		)
		m.AddEmbed(e)
		break

	case domain.NotificationEventIRCDisconnected:
		e := discord.NewEmbed()
		e.SetTitle("IRC Disconnected unexpectedly")
		e.SetColor(discord.RED)
		e.AddFields(
			discord.TextField("Network", payload.Message, true),
		)
		m.AddEmbed(e)
		break

	case domain.NotificationEventIRCReconnected:
		e := discord.NewEmbed()
		e.SetTitle("IRC Reconnected")
		e.SetColor(discord.GREEN)
		e.AddFields(
			discord.TextField("Network", payload.Message, true),
		)
		m.AddEmbed(e)
		break

	default:
		return nil, fmt.Errorf("unknown event: %s", payload.Event)
	}

	return m, nil
}
