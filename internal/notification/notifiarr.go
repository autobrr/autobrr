package notification

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type notifiarrMessage struct {
	Event string               `json:"event"`
	Data  notifiarrMessageData `json:"data"`
}

type notifiarrMessageData struct {
	Subject        string                        `json:"subject"`
	Message        string                        `json:"message"`
	Event          domain.NotificationEvent      `json:"event"`
	ReleaseName    *string                       `json:"release_name,omitempty"`
	Filter         *string                       `json:"filter,omitempty"`
	Indexer        *string                       `json:"indexer,omitempty"`
	InfoHash       *string                       `json:"info_hash,omitempty"`
	Size           *uint64                       `json:"size,omitempty"`
	Status         *domain.ReleasePushStatus     `json:"status,omitempty"`
	Action         *string                       `json:"action,omitempty"`
	ActionType     *domain.ActionType            `json:"action_type,omitempty"`
	ActionClient   *string                       `json:"action_client,omitempty"`
	Rejections     []string                      `json:"rejections,omitempty"`
	Protocol       *domain.ReleaseProtocol       `json:"protocol,omitempty"`       // torrent
	Implementation *domain.ReleaseImplementation `json:"implementation,omitempty"` // irc, rss, api
	Timestamp      time.Time                     `json:"timestamp"`
}

type notifiarrSender struct {
	log      zerolog.Logger
	Settings domain.Notification
}

func NewNotifiarrSender(log zerolog.Logger, settings domain.Notification) domain.NotificationSender {
	return &notifiarrSender{
		log:      log.With().Str("sender", "notifiarr").Logger(),
		Settings: settings,
	}
}

func (s *notifiarrSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	m := notifiarrMessage{
		Event: string(event),
		Data:  s.buildMessage(payload),
	}

	jsonData, err := json.Marshal(m)
	if err != nil {
		s.log.Error().Err(err).Msgf("notifiarr client could not marshal data: %v", m)
		return errors.Wrap(err, "could not marshal data: %+v", m)
	}

	url := fmt.Sprintf("https://notifiarr.com/api/v1/notification/autobrr/%v", s.Settings.APIKey)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		s.log.Error().Err(err).Msgf("notifiarr client request error: %v", event)
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := http.Client{Transport: t, Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		s.log.Error().Err(err).Msgf("notifiarr client request error: %v", event)
		return errors.Wrap(err, "could not make request: %+v", req)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.log.Error().Err(err).Msgf("notifiarr client request error: %v", event)
		return errors.Wrap(err, "could not read data")
	}

	defer res.Body.Close()

	s.log.Trace().Msgf("notifiarr status: %v response: %v", res.StatusCode, string(body))

	if res.StatusCode != http.StatusOK {
		s.log.Error().Err(err).Msgf("notifiarr client request error: %v", string(body))
		return errors.New("bad status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Msg("notification successfully sent to notifiarr")

	return nil
}

func (s *notifiarrSender) CanSend(event domain.NotificationEvent) bool {
	if s.isEnabled() && s.isEnabledEvent(event) {
		return true
	}
	return false
}

func (s *notifiarrSender) isEnabled() bool {
	if s.Settings.Enabled && s.Settings.Webhook != "" {
		return true
	}
	return false
}

func (s *notifiarrSender) isEnabledEvent(event domain.NotificationEvent) bool {
	for _, e := range s.Settings.Events {
		if e == string(event) {
			return true
		}
	}

	return false
}

func (s *notifiarrSender) buildMessage(payload domain.NotificationPayload) notifiarrMessageData {
	m := notifiarrMessageData{
		Event:     payload.Event,
		Timestamp: payload.Timestamp,
	}

	if payload.Subject != "" && payload.Message != "" {
		m.Subject = payload.Subject
		m.Message = payload.Message
	}
	if payload.ReleaseName != "" {
		m.ReleaseName = &payload.ReleaseName
	}
	if payload.Status != "" {
		m.Status = &payload.Status
	}
	if payload.Indexer != "" {
		m.Indexer = &payload.Indexer
	}
	if payload.Filter != "" {
		m.Filter = &payload.Filter
	}
	if payload.Action != "" || payload.ActionClient != "" {
		m.Action = &payload.Action

		if payload.ActionClient != "" {
			m.ActionClient = &payload.ActionClient
		}
	}
	if len(payload.Rejections) > 0 {
		m.Rejections = payload.Rejections
	}

	return m
}
