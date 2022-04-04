package notification

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
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

	client := http.Client{Transport: t, Timeout: 15 * time.Second}

	m := DiscordMessage{
		Content: nil,
		Embeds: []DiscordEmbeds{
			{
				Title:       event.ReleaseName,
				Description: "New release!",
				Color:       5814783,
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
						Inline: false,
					},
				},
				Timestamp: time.Now(),
			},
		},
		Username: "brr",
	}

	jsonData, err := json.Marshal(m)
	if err != nil {
		log.Error().Err(err).Msgf("discord client could not marshal data: %v", m)
		return
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		//log.Error().Err(err).Msgf("webhook client request error: %v", action.WebhookHost)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("User-Agent", "autobrr")

	res, err := client.Do(req)
	if err != nil {
		//log.Error().Err(err).Msgf("webhook client request error: %v", action.WebhookHost)
		return
	}

	defer res.Body.Close()

	log.Debug().Msg("notification successfully sent to discord")

	return
}
