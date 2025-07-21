// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type Service interface {
	Find(ctx context.Context, params domain.NotificationQueryParams) ([]domain.Notification, int, error)
	FindByID(ctx context.Context, id int) (*domain.Notification, error)
	Store(ctx context.Context, notification *domain.Notification) error
	Update(ctx context.Context, notification *domain.Notification) error
	Delete(ctx context.Context, id int) error
	Send(event domain.NotificationEvent, payload domain.NotificationPayload)
	SendFilterNotifications(event domain.NotificationEvent, payload domain.NotificationPayload, filterNotifications []domain.FilterNotification)
	Test(ctx context.Context, notification *domain.Notification) error
}

type service struct {
	log     zerolog.Logger
	repo    domain.NotificationRepo
	senders map[int]domain.NotificationSender
}

func NewService(log logger.Logger, repo domain.NotificationRepo) Service {
	s := &service{
		log:     log.With().Str("module", "notification").Logger(),
		repo:    repo,
		senders: make(map[int]domain.NotificationSender),
	}

	s.registerSenders()

	return s
}

func (s *service) Find(ctx context.Context, params domain.NotificationQueryParams) ([]domain.Notification, int, error) {
	notifications, count, err := s.repo.Find(ctx, params)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find notification with params: %+v", params)
		return nil, 0, err
	}

	return notifications, count, err
}

func (s *service) FindByID(ctx context.Context, id int) (*domain.Notification, error) {
	notification, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find notification by id: %v", id)
		return nil, err
	}

	return notification, err
}

func (s *service) Store(ctx context.Context, notification *domain.Notification) error {
	err := s.repo.Store(ctx, notification)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not store notification: %+v", notification)
		return err
	}

	// register sender
	s.registerSender(notification)

	return nil
}

func (s *service) Update(ctx context.Context, notification *domain.Notification) error {
	err := s.repo.Update(ctx, notification)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not update notification: %+v", notification)
		return err
	}

	// register sender
	s.registerSender(notification)

	return nil
}

func (s *service) Delete(ctx context.Context, id int) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not delete notification: %v", id)
		return err
	}

	// delete sender
	delete(s.senders, id)

	return nil
}

func (s *service) registerSenders() {
	notificationSenders, err := s.repo.List(context.Background())
	if err != nil {
		s.log.Error().Err(err).Msg("could not find notifications")
		return
	}

	for _, notificationSender := range notificationSenders {
		s.registerSender(&notificationSender)
	}

	return
}

// registerSender registers an enabled notification via it's id
func (s *service) registerSender(notification *domain.Notification) {
	if !notification.Enabled {
		delete(s.senders, notification.ID)
		return
	}

	switch notification.Type {
	case domain.NotificationTypeDiscord:
		s.senders[notification.ID] = NewDiscordSender(s.log, notification)
	case domain.NotificationTypeGotify:
		s.senders[notification.ID] = NewGotifySender(s.log, notification)
	case domain.NotificationTypeLunaSea:
		s.senders[notification.ID] = NewLunaSeaSender(s.log, notification)
	case domain.NotificationTypeNotifiarr:
		s.senders[notification.ID] = NewNotifiarrSender(s.log, notification)
	case domain.NotificationTypeNtfy:
		s.senders[notification.ID] = NewNtfySender(s.log, notification)
	case domain.NotificationTypePushover:
		s.senders[notification.ID] = NewPushoverSender(s.log, notification)
	case domain.NotificationTypeShoutrrr:
		s.senders[notification.ID] = NewShoutrrrSender(s.log, notification)
	case domain.NotificationTypeTelegram:
		s.senders[notification.ID] = NewTelegramSender(s.log, notification)
	}

	return
}

// Send notifications
func (s *service) Send(event domain.NotificationEvent, payload domain.NotificationPayload) {
	if len(s.senders) > 0 {
		s.log.Debug().Msgf("sending notification for %v", string(event))
	}

	go func() {
		for _, sender := range s.senders {
			// check if sender is active and have notification types
			if sender.CanSend(event) {
				if err := sender.Send(event, payload); err != nil {
					s.log.Error().Err(err).Msgf("could not send %s notification for %v", sender.Name(), string(event))
				}
			}
		}
	}()

	return
}

func (s *service) SendFilterNotifications(event domain.NotificationEvent, payload domain.NotificationPayload, filterNotifications []domain.FilterNotification) {
	// If no filter-specific notifications, fall back to global notifications
	if len(filterNotifications) == 0 {
		s.Send(event, payload)
		return
	}

	s.log.Debug().Msgf("sending filter-specific notifications for %v", string(event))

	go func() {
		// Send to filter-specific notifications
		for _, fn := range filterNotifications {
			// Check if this notification should handle this event
			eventEnabled := false
			for _, e := range fn.Events {
				if e == string(event) {
					eventEnabled = true
					break
				}
			}

			if !eventEnabled {
				continue
			}

			// Find the sender for this notification
			sender, exists := s.senders[fn.NotificationID]
			if !exists {
				s.log.Warn().Msgf("notification sender %d not found for filter notification", fn.NotificationID)
				continue
			}

			// Send the notification
			if err := sender.Send(event, payload); err != nil {
				s.log.Error().Err(err).Msgf("could not send %s filter notification for %v", sender.Name(), string(event))
			}
		}
	}()

	return
}

func (s *service) Test(ctx context.Context, notification *domain.Notification) error {
	var agent domain.NotificationSender

	// send test events
	events := []domain.NotificationPayload{
		{
			Subject:   "Test Notification",
			Message:   "autobrr goes brr!!",
			Event:     domain.NotificationEventTest,
			Timestamp: time.Now(),
		},
		{
			Subject:        "New release!",
			Message:        "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
			Event:          domain.NotificationEventPushApproved,
			ReleaseName:    "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
			Filter:         "TV",
			Indexer:        "MockIndexer",
			Status:         domain.ReleasePushStatusApproved,
			Action:         "Send to qBittorrent",
			ActionType:     domain.ActionTypeQbittorrent,
			ActionClient:   "qBittorrent",
			Rejections:     nil,
			Protocol:       domain.ReleaseProtocolTorrent,
			Implementation: domain.ReleaseImplementationIRC,
			Timestamp:      time.Now(),
		},
		{
			Subject:        "New release!",
			Message:        "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
			Event:          domain.NotificationEventPushRejected,
			ReleaseName:    "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
			Filter:         "TV",
			Indexer:        "MockIndexer",
			Status:         domain.ReleasePushStatusRejected,
			Action:         "Send to Sonarr",
			ActionType:     domain.ActionTypeSonarr,
			ActionClient:   "Sonarr",
			Rejections:     []string{"Unknown Series"},
			Protocol:       domain.ReleaseProtocolTorrent,
			Implementation: domain.ReleaseImplementationIRC,
			Timestamp:      time.Now(),
		},
		{
			Subject:        "New release!",
			Message:        "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
			Event:          domain.NotificationEventPushError,
			ReleaseName:    "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
			Filter:         "TV",
			Indexer:        "MockIndexer",
			Status:         domain.ReleasePushStatusErr,
			Action:         "Send to Sonarr",
			ActionType:     domain.ActionTypeSonarr,
			ActionClient:   "Sonarr",
			Rejections:     []string{"error pushing to client"},
			Protocol:       domain.ReleaseProtocolTorrent,
			Implementation: domain.ReleaseImplementationIRC,
			Timestamp:      time.Now(),
		},
		{
			Subject:   "IRC Disconnected unexpectedly",
			Message:   "Network: P2P-Network",
			Event:     domain.NotificationEventIRCDisconnected,
			Timestamp: time.Now(),
		},
		{
			Subject:   "IRC Reconnected",
			Message:   "Network: P2P-Network",
			Event:     domain.NotificationEventIRCReconnected,
			Timestamp: time.Now(),
		},
		{
			Subject:   "New update available!",
			Message:   "v1.6.0",
			Event:     domain.NotificationEventAppUpdateAvailable,
			Timestamp: time.Now(),
		},
	}

	switch notification.Type {
	case domain.NotificationTypeDiscord:
		agent = NewDiscordSender(s.log, notification)
	case domain.NotificationTypeGotify:
		agent = NewGotifySender(s.log, notification)
	case domain.NotificationTypeLunaSea:
		agent = NewLunaSeaSender(s.log, notification)
	case domain.NotificationTypeNotifiarr:
		agent = NewNotifiarrSender(s.log, notification)
	case domain.NotificationTypeNtfy:
		agent = NewNtfySender(s.log, notification)
	case domain.NotificationTypePushover:
		agent = NewPushoverSender(s.log, notification)
	case domain.NotificationTypeShoutrrr:
		agent = NewShoutrrrSender(s.log, notification)
	case domain.NotificationTypeTelegram:
		agent = NewTelegramSender(s.log, notification)
	default:
		s.log.Error().Msgf("unsupported notification type: %v", notification.Type)
		return errors.New("unsupported notification type")
	}

	g, _ := errgroup.WithContext(ctx)

	for _, event := range events {
		e := event

		if !enabledEvent(notification.Events, e.Event) {
			continue
		}

		if err := agent.Send(e.Event, e); err != nil {
			s.log.Error().Err(err).Msgf("error sending test notification: %#v", notification)
			return err
		}

		time.Sleep(1 * time.Second)
	}

	if err := g.Wait(); err != nil {
		s.log.Error().Err(err).Msgf("Something went wrong sending test notifications to %v", notification.Type)
		return err
	}

	return nil
}

func enabledEvent(events []string, e domain.NotificationEvent) bool {
	for _, v := range events {
		if v == string(e) {
			return true
		}
	}

	return false
}
