package notification

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
)

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type telegramSender struct {
	log      logger.Logger
	Settings domain.Notification
}

func NewTelegramSender(log logger.Logger, settings domain.Notification) domain.NotificationSender {
	return &telegramSender{
		log:      log,
		Settings: settings,
	}
}

func (s *telegramSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	m := TelegramMessage{
		ChatID:    s.Settings.Channel,
		Text:      s.buildMessage(event, payload),
		ParseMode: "HTML",
		//ParseMode: "MarkdownV2",
	}

	jsonData, err := json.Marshal(m)
	if err != nil {
		s.log.Error().Err(err).Msgf("telegram client could not marshal data: %v", m)
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%v/sendMessage", s.Settings.Token)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		s.log.Error().Err(err).Msgf("telegram client request error: %v", event)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("User-Agent", "autobrr")

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := http.Client{Transport: t, Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		s.log.Error().Err(err).Msgf("telegram client request error: %v", event)
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		s.log.Error().Err(err).Msgf("telegram client request error: %v", event)
		return err
	}

	defer res.Body.Close()

	s.log.Trace().Msgf("telegram status: %v response: %v", res.StatusCode, string(body))

	if res.StatusCode != http.StatusOK {
		s.log.Error().Err(err).Msgf("telegram client request error: %v", string(body))
		return fmt.Errorf("err: %v", string(body))
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
	if s.Settings.Enabled && s.Settings.Webhook != "" {
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

	msg += fmt.Sprintf("%v\n<b>%v</b>", payload.Subject, html.EscapeString(payload.Message))
	if payload.Status != "" {
		msg += fmt.Sprintf("\nStatus: %v", payload.Status.String())
	}
	if payload.Indexer != "" {
		msg += fmt.Sprintf("\nIndexer: %v", payload.Indexer)
	}
	if payload.Filter != "" {
		msg += fmt.Sprintf("\nFilter: %v", payload.Filter)
	}
	if payload.Action != "" {
		msg += fmt.Sprintf("\nAction: %v type: %v client: %v", payload.Action, payload.ActionType, payload.ActionClient)
	}
	if len(payload.Rejections) > 0 {
		msg += fmt.Sprintf("\nRejections: %v", strings.Join(payload.Rejections, ", "))
	}

	return msg
}
