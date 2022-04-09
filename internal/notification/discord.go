package notification

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog/log"
)

type DiscordMessage struct {
	Content  interface{}     `json:"content"`
	Embeds   []DiscordEmbeds `json:"embeds"`
	Username string          `json:"username"`
}

type DiscordEmbeds struct {
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Color       int                   `json:"color"`
	Fields      []DiscordEmbedsFields `json:"fields"`
	Timestamp   time.Time             `json:"timestamp"`
}
type DiscordEmbedsFields struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

func discordNotification(event domain.EventsReleasePushed, webhookURL string) {
	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := http.Client{Transport: t, Timeout: 30 * time.Second}

	color := map[domain.ReleasePushStatus]int{
		domain.ReleasePushStatusApproved: 5814783,
		domain.ReleasePushStatusRejected: 5814783,
		domain.ReleasePushStatusErr:      14026000,
	}

	m := DiscordMessage{
		Content: nil,
		Embeds: []DiscordEmbeds{
			{
				Title:       event.ReleaseName,
				Description: "New release!",
				Color:       color[event.Status],
				Fields: []DiscordEmbedsFields{
					{
						Name:   "Status",
						Value:  event.Status.String(),
						Inline: true,
					},
					{
						Name:   "Indexer",
						Value:  event.Indexer,
						Inline: true,
					},
					{
						Name:   "Filter",
						Value:  event.Filter,
						Inline: true,
					},
					{
						Name:   "Action",
						Value:  event.Action,
						Inline: true,
					},
					{
						Name:   "Action type",
						Value:  string(event.ActionType),
						Inline: true,
					},
					//{
					//	Name:   "Action client",
					//	Value:  event.ActionClient,
					//	Inline: true,
					//},
				},
				Timestamp: time.Now(),
			},
		},
		Username: "brr",
	}

	if event.ActionClient == "" {
		rej := DiscordEmbedsFields{
			Name:   "Action client",
			Value:  "n/a",
			Inline: true,
		}
		m.Embeds[0].Fields = append(m.Embeds[0].Fields, rej)
	} else {
		rej := DiscordEmbedsFields{
			Name:   "Action client",
			Value:  event.ActionClient,
			Inline: true,
		}
		m.Embeds[0].Fields = append(m.Embeds[0].Fields, rej)
	}

	if len(event.Rejections) > 0 {
		rej := DiscordEmbedsFields{
			Name:   "Reasons",
			Value:  fmt.Sprintf("```\n%v\n```", strings.Join(event.Rejections, " ,")),
			Inline: false,
		}
		m.Embeds[0].Fields = append(m.Embeds[0].Fields, rej)
	}

	jsonData, err := json.Marshal(m)
	if err != nil {
		log.Error().Err(err).Msgf("discord client could not marshal data: %v", m)
		return
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error().Err(err).Msgf("discord client request error: %v", event.ReleaseName)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("User-Agent", "autobrr")

	res, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("discord client request error: %v", event.ReleaseName)
		return
	}

	defer res.Body.Close()

	log.Debug().Msg("notification successfully sent to discord")

	return
}
