// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog"
)

type DiscordMessage struct {
	Content interface{}     `json:"content"`
	Embeds  []DiscordEmbeds `json:"embeds,omitempty"`
}

type DiscordEmbeds struct {
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Color       int                   `json:"color"`
	Fields      []DiscordEmbedsFields `json:"fields,omitempty"`
	Timestamp   time.Time             `json:"timestamp"`
}
type DiscordEmbedsFields struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type EmbedColors int

const (
	LIGHT_BLUE EmbedColors = 5814783  // 58b9ff
	RED        EmbedColors = 15548997 // ed4245
	GREEN      EmbedColors = 5763719  // 57f287
	GRAY       EmbedColors = 10070709 // 99aab5
)

type discordSender struct {
	log      zerolog.Logger
	Settings domain.Notification

	httpClient *http.Client
}

func NewDiscordSender(log zerolog.Logger, settings domain.Notification) domain.NotificationSender {
	return &discordSender{
		log:      log.With().Str("sender", "discord").Logger(),
		Settings: settings,
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (a *discordSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	m := DiscordMessage{
		Content: nil,
		Embeds:  []DiscordEmbeds{a.buildEmbed(event, payload)},
	}

	jsonData, err := json.Marshal(m)
	if err != nil {
		a.log.Error().Err(err).Msgf("discord client could not marshal data: %v", m)
		return errors.Wrap(err, "could not marshal data: %+v", m)
	}

	req, err := http.NewRequest(http.MethodPost, a.Settings.Webhook, bytes.NewBuffer(jsonData))
	if err != nil {
		a.log.Error().Err(err).Msgf("discord client request error: %v", event)
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("User-Agent", "autobrr")

	res, err := a.httpClient.Do(req)
	if err != nil {
		a.log.Error().Err(err).Msgf("discord client request error: %v", event)
		return errors.Wrap(err, "could not make request: %+v", req)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		a.log.Error().Err(err).Msgf("discord client request error: %v", event)
		return errors.Wrap(err, "could not read data")
	}

	a.log.Trace().Msgf("discord status: %v response: %v", res.StatusCode, string(body))

	// discord responds with 204, Notifiarr with 204 so lets take all 200 as ok
	if res.StatusCode >= 300 {
		a.log.Error().Err(err).Msgf("discord client request error: %v", string(body))
		return errors.New("bad status: %v body: %v", res.StatusCode, string(body))
	}

	a.log.Debug().Msg("notification successfully sent to discord")

	return nil
}

func (a *discordSender) CanSend(event domain.NotificationEvent) bool {
	if a.isEnabled() && a.isEnabledEvent(event) {
		return true
	}
	return false
}

func (a *discordSender) isEnabled() bool {
	if a.Settings.Enabled && a.Settings.Webhook != "" {
		return true
	}
	return false
}

func (a *discordSender) isEnabledEvent(event domain.NotificationEvent) bool {
	for _, e := range a.Settings.Events {
		if e == string(event) {
			return true
		}
	}

	return false
}

func (a *discordSender) buildEmbed(event domain.NotificationEvent, payload domain.NotificationPayload) DiscordEmbeds {

	color := LIGHT_BLUE
	switch event {
	case domain.NotificationEventPushApproved:
		color = GREEN
	case domain.NotificationEventPushRejected:
		color = GRAY
	case domain.NotificationEventPushError:
		color = RED
	case domain.NotificationEventIRCDisconnected:
		color = RED
	case domain.NotificationEventIRCReconnected:
		color = GREEN
	case domain.NotificationEventTest:
		color = LIGHT_BLUE
	}

	var fields []DiscordEmbedsFields

	if payload.Status != "" {
		f := DiscordEmbedsFields{
			Name:   "Status",
			Value:  payload.Status.String(),
			Inline: true,
		}
		fields = append(fields, f)
	}
	if payload.Indexer != "" {
		f := DiscordEmbedsFields{
			Name:   "Indexer",
			Value:  payload.Indexer,
			Inline: true,
		}
		fields = append(fields, f)
	}
	if payload.Filter != "" {
		f := DiscordEmbedsFields{
			Name:   "Filter",
			Value:  payload.Filter,
			Inline: true,
		}
		fields = append(fields, f)
	}
	if payload.Action != "" {
		f := DiscordEmbedsFields{
			Name:   "Action",
			Value:  payload.Action,
			Inline: true,
		}
		fields = append(fields, f)
	}
	if payload.ActionType != "" {
		f := DiscordEmbedsFields{
			Name:   "Action type",
			Value:  string(payload.ActionType),
			Inline: true,
		}
		fields = append(fields, f)
	}
	if payload.ActionClient != "" {
		f := DiscordEmbedsFields{
			Name:   "Action client",
			Value:  payload.ActionClient,
			Inline: true,
		}
		fields = append(fields, f)
	}
	if payload.Size > 0 {
		f := DiscordEmbedsFields{
			Name:   "Size",
			Value:  humanize.Bytes(payload.Size),
			Inline: true,
		}
		fields = append(fields, f)
	}
	if len(payload.Protocol) != 0 {
		f := DiscordEmbedsFields{
			Name:   "Protocol",
			Value:  payload.Protocol.String(),
			Inline: true,
		}
		fields = append(fields, f)
	}
	if len(payload.Implementation) != 0 {
		f := DiscordEmbedsFields{
			Name:   "Implementation",
			Value:  payload.Implementation.String(),
			Inline: true,
		}
		fields = append(fields, f)
	}
	if len(payload.Rejections) > 0 {
		f := DiscordEmbedsFields{
			Name:   "Reasons",
			Value:  fmt.Sprintf("```\n%v\n```", strings.Join(payload.Rejections, ", ")),
			Inline: false,
		}
		fields = append(fields, f)
	}

	embed := DiscordEmbeds{
		Title:       payload.ReleaseName,
		Description: "New release!",
		Color:       int(color),
		Fields:      fields,
		Timestamp:   time.Now(),
	}

	if payload.Subject != "" && payload.Message != "" {
		embed.Title = payload.Subject
		embed.Description = payload.Message
	}

	return embed
}
