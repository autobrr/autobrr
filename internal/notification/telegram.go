package notification

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
)

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

func (s *service) telegramNotification(event *domain.EventsReleasePushed, chatID string, token string) error {
	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := http.Client{Transport: t, Timeout: 30 * time.Second}

	text := fmt.Sprintf("Hello from *autobrr\\!*\nthis was a test\\!")

	m := TelegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "MarkdownV2",
	}

	jsonData, err := json.Marshal(m)
	if err != nil {
		s.log.Error().Err(err).Msgf("telegram client could not marshal data: %v", m)
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%v/sendMessage", token)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		s.log.Error().Err(err).Msgf("telegram client request error: %v", event.ReleaseName)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("User-Agent", "autobrr")

	res, err := client.Do(req)
	if err != nil {
		s.log.Error().Err(err).Msgf("telegram client request error: %v", event.ReleaseName)
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		s.log.Error().Err(err).Msgf("telegram client request error: %v", event.ReleaseName)
		return err
	}

	defer res.Body.Close()

	s.log.Trace().Msgf("telegram status: %v response: %v", res.StatusCode, string(body))

	s.log.Debug().Msg("notification successfully sent to telegram")
	return nil
}
