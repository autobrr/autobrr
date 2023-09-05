package notification

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/rs/zerolog"
	"net"
	"net/smtp"
	"strings"
)

type emailSender struct {
	log      zerolog.Logger
	Settings domain.Notification
}

func NewEmailSender(log zerolog.Logger, settings domain.Notification) domain.NotificationSender {
	return &emailSender{
		log:      log.With().Str("sender", "email").Logger(),
		Settings: settings,
	}
}

func (s *emailSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	if !s.isEnabled() || !s.isEnabledEvent(event) {
		return errors.New("notification or event not enabled")
	}

	to := s.Settings.RecipientAddresses
	from := s.Settings.FromAddress
	subject := payload.Subject
	body := s.buildMessage(event, payload)

	msg := fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\n\r\n%s",
		strings.Join(to, ","),
		from,
		subject,
		body,
	)

	hostWithPort := fmt.Sprintf("%s:%d", s.Settings.Host, s.Settings.SmtpPort)
	hostOnly := strings.Split(hostWithPort, ":")[0]
	auth := smtp.PlainAuth("", s.Settings.Username, s.Settings.Password, s.Settings.Host)

	if s.Settings.RequireEncryption {
		tlsConfig := &tls.Config{
			ServerName: hostOnly,
		}

		if s.Settings.SmtpPort == 465 {
			// Explicit SSL
			conn, err := tls.Dial("tcp", hostWithPort, tlsConfig)
			if err != nil {
				s.log.Error().Err(err).Msg("failed to establish a secure connection")
				return err
			}
			defer conn.Close()

			c, err := smtp.NewClient(conn, s.Settings.Host)
			if err != nil {
				s.log.Error().Err(err).Msg("failed to create new SMTP client")
				return err
			}
			defer c.Quit()

			// Auth and send
			err = sendMail(c, auth, from, to, msg)
			if err != nil {
				s.log.Error().Err(err).Msg("failed to send email")
				return err
			}
		} else {
			// StartTLS
			conn, err := net.Dial("tcp", hostWithPort)
			if err != nil {
				s.log.Error().Err(err).Msg("failed to establish a secure connection")
				return err
			}
			defer conn.Close()

			c, err := smtp.NewClient(conn, s.Settings.Host)
			if err != nil {
				s.log.Error().Err(err).Msg("failed to create new SMTP client")
				return err
			}
			defer c.Quit()

			err = c.StartTLS(tlsConfig)
			if err != nil {
				s.log.Error().Err(err).Msg("failed to start TLS")
				return err
			}

			// Auth and send
			err = sendMail(c, auth, from, to, msg)
			if err != nil {
				s.log.Error().Err(err).Msg("failed to send email")
				return err
			}
		}
	} else {
		// Non-encrypted
		err := smtp.SendMail(hostWithPort, auth, from, to, []byte(msg))
		if err != nil {
			s.log.Error().Err(err).Msg("failed to send the email")
			return err
		}
	}

	s.log.Debug().Msg("notification successfully sent via email")
	return nil
}

func (s *emailSender) CanSend(event domain.NotificationEvent) bool {
	return s.isEnabled() && s.isEnabledEvent(event)
}

func sendMail(c *smtp.Client, auth smtp.Auth, from string, to []string, msg string) error {
	if err := c.Auth(auth); err != nil {
		return err
	}

	if err := c.Mail(from); err != nil {
		return err
	}

	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return err
		}
	}

	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return nil
}

func (s *emailSender) isEnabled() bool {
	return s.Settings.Enabled &&
		s.Settings.Host != "" &&
		len(s.Settings.RecipientAddresses) > 0 &&
		s.Settings.FromAddress != ""
}

func (s *emailSender) isEnabledEvent(event domain.NotificationEvent) bool {
	for _, e := range s.Settings.Events {
		if e == string(event) {
			return true
		}
	}
	return false
}

func (s *emailSender) buildMessage(event domain.NotificationEvent, payload domain.NotificationPayload) string {
	var msgBuffer bytes.Buffer

	if payload.Subject != "" && payload.Message != "" {
		msgBuffer.WriteString(fmt.Sprintf("%s\n%s", payload.Subject, payload.Message))
	}
	if payload.ReleaseName != "" {
		msgBuffer.WriteString(fmt.Sprintf("\nNew release: %s", payload.ReleaseName))
	}
	if payload.Status != "" {
		msgBuffer.WriteString(fmt.Sprintf("\nStatus: %s", payload.Status))
	}
	if payload.Indexer != "" {
		msgBuffer.WriteString(fmt.Sprintf("\nIndexer: %s", payload.Indexer))
	}
	if payload.Filter != "" {
		msgBuffer.WriteString(fmt.Sprintf("\nFilter: %s", payload.Filter))
	}
	if payload.Action != "" {
		action := fmt.Sprintf("\nAction: %s Type: %s", payload.Action, payload.ActionType)
		if payload.ActionClient != "" {
			action += fmt.Sprintf(" Client: %s", payload.ActionClient)
		}
		msgBuffer.WriteString(action)
	}
	if len(payload.Rejections) > 0 {
		msgBuffer.WriteString(fmt.Sprintf("\nRejections: %s", strings.Join(payload.Rejections, ", ")))
	}

	return msgBuffer.String()
}
