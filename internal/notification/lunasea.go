package notification

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

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
	builder  NotificationBuilderPlainText

	httpClient *http.Client
}

func (s *lunaSeaSender) rewriteWebhookURL(url string) string {
	re := regexp.MustCompile(`/(radarr|sonarr|lidarr|tautulli|overseerr)/`)
	return re.ReplaceAllString(url, "/custom/")
} // `custom` is not mentioned in their docs, so I thought this would be a good idea to add to avoid user errors

func NewLunaSeaSender(log zerolog.Logger, settings domain.Notification) domain.NotificationSender {
	return &lunaSeaSender{
		log:      log.With().Str("sender", "lunasea").Logger(),
		Settings: settings,
		builder:  NotificationBuilderPlainText{},
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (s *lunaSeaSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	m := LunaSeaMessage{
		Title: s.builder.BuildTitle(event),
		Body:  s.builder.BuildBody(payload),
		Image: defaultImageURL,
	}

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

	res, err := s.httpClient.Do(req)
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
