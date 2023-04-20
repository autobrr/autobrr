package notification

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type telegramSender struct {
	log      zerolog.Logger
	Settings domain.Notification
}

func NewTelegramSender(log zerolog.Logger, settings domain.Notification) domain.NotificationSender {
	return &telegramSender{
		log:      log.With().Str("sender", "telegram").Logger(),
		Settings: settings,
	}
}

func (s *telegramSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	m := TelegramMessage{
		ChatID:    s.Settings.Channel,
		Text:      s.buildMessage(event, payload),
		ParseMode: "HTML",
		// ParseMode: "MarkdownV2",
	}

	jsonData, err := json.Marshal(m)
	if err != nil {
		s.log.Error().Err(err).Msgf("telegram client could not marshal data: %v", m)
		return errors.Wrap(err, "could not marshal data: %+v", m)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%v/sendMessage", s.Settings.Token)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		s.log.Error().Err(err).Msgf("telegram client request error: %v", event)
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("User-Agent", "autobrr")

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := http.Client{Transport: t, Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		s.log.Error().Err(err).Msgf("telegram client request error: %v", event)
		return errors.Wrap(err, "could not make request: %+v", req)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.log.Error().Err(err).Msgf("telegram client request error: %v", event)
		return errors.Wrap(err, "could not read data")
	}

	defer res.Body.Close()

	s.log.Trace().Msgf("telegram status: %v response: %v", res.StatusCode, string(body))

	if res.StatusCode != http.StatusOK {
		s.log.Error().Err(err).Msgf("telegram client request error: %v", string(body))
		return errors.New("bad status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Msg("notification successfully sent to telegram")
	return nil
}

func (s *telegramSender) CanSend(event domain.NotificationEvent) bool {
	if s.isEnabled() && s.isEnabledEvent(event) {
		return true
	}
	return false
}

func (s *telegramSender) isEnabled() bool {
	if s.Settings.Enabled && s.Settings.Token != "" && s.Settings.Channel != "" {
		return true
	}
	return false
}

func (s *telegramSender) isEnabledEvent(event domain.NotificationEvent) bool {
	for _, e := range s.Settings.Events {
		if e == string(event) {
			return true
		}
	}

	return false
}

func (s *telegramSender) buildMessage(event domain.NotificationEvent, payload domain.NotificationPayload) string {
	msg := ""

	if payload.Subject != "" && payload.Message != "" {
		msg += fmt.Sprintf("%v\n<b>%v</b>", payload.Subject, html.EscapeString(payload.Message))
	}
	if payload.ReleaseName != "" {
		msg += fmt.Sprintf("\n<b>New release:</b> %v", html.EscapeString(payload.ReleaseName))
	}
	if payload.Status != "" {
		msg += fmt.Sprintf("\n<b>Status:</b> %v", payload.Status.String())
	}
	if payload.Indexer != "" {
		msg += fmt.Sprintf("\n<b>Indexer:</b> %v", payload.Indexer)
	}
	if payload.Filter != "" {
		msg += fmt.Sprintf("\n<b>Filter:</b> %v", html.EscapeString(payload.Filter))
	}
	if payload.Action != "" {
		action := fmt.Sprintf("\n<b>Action:</b> %v <b>Type:</b> %v", html.EscapeString(payload.Action), payload.ActionType)
		if payload.ActionClient != "" {
			action += fmt.Sprintf(" <b>Client:</b> %v", html.EscapeString(payload.ActionClient))
		}
		msg += action
	}
	if len(payload.Rejections) > 0 {
		msg += fmt.Sprintf("\nRejections: %v", strings.Join(payload.Rejections, ", "))
	}

	return msg
}
