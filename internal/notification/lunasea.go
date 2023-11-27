package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/dustin/go-humanize"

	"github.com/rs/zerolog"
)

// unsure if this is the best approach to send an image with the notification
const defaultImageURL = "https://raw.githubusercontent.com/autobrr/autobrr/master/.github/images/logo.png"

type LunaSeaMessage struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Image string `json:"image,omitempty"`
}

type lunaSeaSender struct {
	log      zerolog.Logger
	Settings domain.Notification
}

func (s *lunaSeaSender) rewriteWebhookURL(url string) string {
	re := regexp.MustCompile(`/(radarr|sonarr|lidarr|tautulli|overseerr)/`)
	return re.ReplaceAllString(url, "/custom/")
} // `custom` is not mentioned in their docs, so I thought this would be a good idea to add to avoid user errors

func NewLunaSeaSender(log zerolog.Logger, settings domain.Notification) domain.NotificationSender {
	return &lunaSeaSender{
		log:      log.With().Str("sender", "lunasea").Logger(),
		Settings: settings,
	}
}

func (s *lunaSeaSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	m := s.buildMessage(event, payload)

	jsonData, err := json.Marshal(m)
	if err != nil {
		s.log.Error().Err(err).Msg("lunasea client could not marshal data")
		return errors.Wrap(err, "could not marshal data")
	}

	rewrittenURL := s.rewriteWebhookURL(s.Settings.Webhook)

	req, err := http.NewRequest(http.MethodPost, rewrittenURL, bytes.NewBuffer(jsonData))
	if err != nil {
		s.log.Error().Err(err).Msg("lunasea client request error")
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		s.log.Error().Err(err).Msg("lunasea client request error")
		return errors.Wrap(err, "could not make request")
	}

	defer res.Body.Close()

	if res.StatusCode >= 300 {
		s.log.Error().Msgf("bad status from lunasea: %v", res.StatusCode)
		return errors.New("bad status: %v", res.StatusCode)
	}

	s.log.Debug().Msg("notification successfully sent to lunasea")

	return nil
}

func (s *lunaSeaSender) CanSend(event domain.NotificationEvent) bool {
	if s.Settings.Enabled && s.Settings.Webhook != "" && s.isEnabledEvent(event) {
		return true
	}
	return false
}

func (s *lunaSeaSender) isEnabledEvent(event domain.NotificationEvent) bool {
	for _, e := range s.Settings.Events {
		if e == string(event) {
			return true
		}
	}
	return false
}

func (s *lunaSeaSender) buildMessage(event domain.NotificationEvent, payload domain.NotificationPayload) LunaSeaMessage {
	title := s.buildTitle(event)
	body := s.buildBody(payload)

	return LunaSeaMessage{
		Title: title,
		Body:  body,
		Image: defaultImageURL, // Unsure if this is the right approach.
	}
}

func (s *lunaSeaSender) buildBody(payload domain.NotificationPayload) string {
	var parts []string

	if payload.Subject != "" && payload.Message != "" {
		parts = append(parts, fmt.Sprintf("%v\n%v", payload.Subject, payload.Message))
	}
	if payload.ReleaseName != "" {
		parts = append(parts, fmt.Sprintf("New release: %v", payload.ReleaseName))
	}
	if payload.Size > 0 {
		parts = append(parts, fmt.Sprintf("Size: %v", humanize.Bytes(payload.Size)))
	}
	if payload.Status != "" {
		parts = append(parts, fmt.Sprintf("Status: %v", payload.Status.String()))
	}
	if payload.Indexer != "" {
		parts = append(parts, fmt.Sprintf("Indexer: %v", payload.Indexer))
	}
	if payload.Filter != "" {
		parts = append(parts, fmt.Sprintf("Filter: %v", payload.Filter))
	}
	if payload.Action != "" {
		action := fmt.Sprintf("Action: %v Type: %v", payload.Action, payload.ActionType)
		if payload.ActionClient != "" {
			action += fmt.Sprintf(" Client: %v", payload.ActionClient)
		}
		parts = append(parts, action)
	}
	if len(payload.Rejections) > 0 {
		parts = append(parts, fmt.Sprintf("Rejections: %v", strings.Join(payload.Rejections, ", ")))
	}

	return strings.Join(parts, "\n")
}

func (s *lunaSeaSender) buildTitle(event domain.NotificationEvent) string {
	switch event {
	case domain.NotificationEventAppUpdateAvailable:
		return "Autobrr update available"
	case domain.NotificationEventPushApproved:
		return "Push Approved"
	case domain.NotificationEventPushRejected:
		return "Push Rejected"
	case domain.NotificationEventPushError:
		return "Error"
	case domain.NotificationEventIRCDisconnected:
		return "IRC Disconnected"
	case domain.NotificationEventIRCReconnected:
		return "IRC Reconnected"
	case domain.NotificationEventTest:
		return "Test"
	default:
		return "New Event"
	}
}
