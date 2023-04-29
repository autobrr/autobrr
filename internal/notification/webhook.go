package notification

import (
	"crypto/tls"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type webhookMessage struct {
	Token     string    `json:"api_key"`
	User      string    `json:"token"`
	Message   string    `json:"message"`
	Priority  int32     `json:"priority"`
	Title     string    `json:"title"`
	Timestamp time.Time `json:"timestamp"`
	Html      int       `json:"html,omitempty"`
}

type webhookSender struct {
	log      zerolog.Logger
	Settings domain.Notification
}

func NewWebhookSender(log zerolog.Logger, settings domain.Notification) domain.NotificationSender {
	return &webhookSender{
		log:      log.With().Str("sender", "webhook").Logger(),
		Settings: settings,
	}
}

func (s *webhookSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	//m := webhookMessage{
	//	Token:     s.Settings.APIKey,
	//	User:      s.Settings.Token,
	//	Priority:  s.Settings.Priority,
	//	Message:   s.buildMessage(payload),
	//	Title:     s.buildTitle(event),
	//	Timestamp: time.Now(),
	//	Html:      1,
	//}

	data := url.Values{}

	req, err := http.NewRequest(http.MethodPost, s.Settings.Host, strings.NewReader(data.Encode()))
	if err != nil {
		s.log.Error().Err(err).Msgf("webhook client request error: %v", event)
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "autobrr")

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := http.Client{Transport: t, Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		s.log.Error().Err(err).Msgf("webhook client request error: %v", event)
		return errors.Wrap(err, "could not make request: %+v", req)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.log.Error().Err(err).Msgf("webhook client request error: %v", event)
		return errors.Wrap(err, "could not read data")
	}

	defer res.Body.Close()

	s.log.Trace().Msgf("webhook status: %v response: %v", res.StatusCode, string(body))

	if res.StatusCode >= 300 {
		s.log.Error().Err(err).Msgf("webhook client request error: %v", string(body))
		return errors.New("bad status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Msg("notification successfully sent to webhook")

	return nil
}

func (s *webhookSender) newPostRequest(payload domain.NotificationPayload, data url.Values) (*http.Request, error) {
	req, err := http.NewRequest(s.Settings.Method, s.Settings.Host, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", s.Settings.ContentType)

	return req, nil
}

func (s *webhookSender) CanSend(event domain.NotificationEvent) bool {
	if s.isEnabled() && s.isEnabledEvent(event) {
		return true
	}
	return false
}

func (s *webhookSender) isEnabled() bool {
	if s.Settings.Enabled {
		if s.Settings.APIKey == "" {
			s.log.Warn().Msg("webhook missing api key")
			return false
		}

		if s.Settings.Token == "" {
			s.log.Warn().Msg("webhook missing user key")
			return false
		}

		return true
	}

	return false
}

func (s *webhookSender) isEnabledEvent(event domain.NotificationEvent) bool {
	for _, e := range s.Settings.Events {
		if e == string(event) {
			return true
		}
	}

	return false
}

func (s *webhookSender) buildMessage(payload domain.NotificationPayload) string {
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

func (s *webhookSender) buildTitle(event domain.NotificationEvent) string {
	title := ""

	switch event {
	case domain.NotificationEventAppUpdateAvailable:
		title = "Autobrr update available"
	case domain.NotificationEventPushApproved:
		title = "Push Approved"
	case domain.NotificationEventPushRejected:
		title = "Push Rejected"
	case domain.NotificationEventPushError:
		title = "Error"
	case domain.NotificationEventIRCDisconnected:
		title = "IRC Disconnected"
	case domain.NotificationEventIRCReconnected:
		title = "IRC Reconnected"
	case domain.NotificationEventTest:
		title = "Test"
	}

	return title
}
