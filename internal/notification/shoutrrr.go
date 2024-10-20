package notification

import (
	"github.com/autobrr/autobrr/internal/domain"

	"github.com/containrrr/shoutrrr"
	"github.com/rs/zerolog"
)

type shoutrrrSender struct {
	log      zerolog.Logger
	Settings *domain.Notification
	builder  MessageBuilderPlainText
}

func (s *shoutrrrSender) Name() string {
	return "shoutrrr"
}

func NewShoutrrrSender(log zerolog.Logger, settings *domain.Notification) domain.NotificationSender {
	return &shoutrrrSender{
		log:      log.With().Str("sender", "shoutrrr").Logger(),
		Settings: settings,
		builder:  MessageBuilderPlainText{},
	}
}

func (s *shoutrrrSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	message := s.builder.BuildBody(payload)

	if err := shoutrrr.Send(s.Settings.Host, message); err != nil {
		return err
	}

	s.log.Debug().Msg("notification successfully sent to via shoutrrr")

	return nil
}

func (s *shoutrrrSender) CanSend(event domain.NotificationEvent) bool {
	if s.isEnabled() && s.isEnabledEvent(event) {
		return true
	}
	return false
}

func (s *shoutrrrSender) isEnabled() bool {
	if s.Settings.Enabled {
		if s.Settings.Host == "" {
			s.log.Warn().Msg("shoutrrr missing host")
			return false
		}

		return true
	}

	return false
}

func (s *shoutrrrSender) isEnabledEvent(event domain.NotificationEvent) bool {
	for _, e := range s.Settings.Events {
		if e == string(event) {
			return true
		}
	}

	return false
}
