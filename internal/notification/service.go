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

type Sender interface {
	Send(event domain.NotificationEvent, payload domain.NotificationPayload)
}

type Tester interface {
	Test(ctx context.Context, notification *domain.Notification) error
}

type Storer interface {
	Find(ctx context.Context, params domain.NotificationQueryParams) ([]domain.Notification, int, error)
	FindByID(ctx context.Context, notificationID int) (*domain.Notification, error)
	Store(ctx context.Context, notification *domain.Notification) error
	Update(ctx context.Context, notification *domain.Notification) error
	Delete(ctx context.Context, notificationID int) error
}

type FilterStorer interface {
	GetFilterNotifications(ctx context.Context, filterID int) ([]domain.FilterNotification, error)
	StoreFilterNotifications(ctx context.Context, filterID int, notifications []domain.FilterNotification) error
	DeleteFilterNotifications(ctx context.Context, filterID int) error
}

type FullService interface {
	FilterStorer
	Storer
	Sender
	Tester
}

type Service struct {
	log  zerolog.Logger
	repo domain.NotificationRepo

	notifications map[int]*domain.Notification
	senders       map[int]domain.NotificationSender
}

func NewService(log logger.Logger, repo domain.NotificationRepo) *Service {
	s := &Service{
		log:           log.With().Str("module", "notification").Logger(),
		repo:          repo,
		notifications: make(map[int]*domain.Notification),
		senders:       make(map[int]domain.NotificationSender),
	}

	s.registerSenders()

	return s
}

func (s *Service) Find(ctx context.Context, params domain.NotificationQueryParams) ([]domain.Notification, int, error) {
	notifications, count, err := s.repo.Find(ctx, params)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find notification with params: %+v", params)
		return nil, 0, err
	}

	for idx, notification := range notifications {
		filters, err := s.repo.GetNotificationFilters(ctx, notification.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("could not find filter notifications for notification: %v", notification.ID)
			continue
		}
		notifications[idx].UsedByFilters = filters
	}

	return notifications, count, err
}

func (s *Service) FindByID(ctx context.Context, notificationID int) (*domain.Notification, error) {
	notification, err := s.repo.FindByID(ctx, notificationID)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find notification by id: %v", notificationID)
		return nil, err
	}

	return notification, err
}

func (s *Service) Store(ctx context.Context, notification *domain.Notification) error {
	err := s.repo.Store(ctx, notification)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not store notification: %+v", notification)
		return err
	}

	// register sender
	s.registerSender(notification)

	return nil
}

func (s *Service) Update(ctx context.Context, notification *domain.Notification) error {
	existing, err := s.repo.FindByID(ctx, notification.ID)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find notification by id: %v", notification.ID)
		return err
	}

	if domain.IsRedactedString(notification.Password) {
		notification.Password = existing.Password
	}
	if domain.IsRedactedString(notification.Token) {
		notification.Token = existing.Token
	}
	if domain.IsRedactedString(notification.APIKey) {
		notification.APIKey = existing.APIKey
	}

	if err := s.repo.Update(ctx, notification); err != nil {
		s.log.Error().Err(err).Msgf("could not update notification: %+v", notification)
		return err
	}

	// register sender
	s.registerSender(notification)

	return nil
}

func (s *Service) Delete(ctx context.Context, id int) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not delete notification: %v", id)
		return err
	}

	// delete sender
	delete(s.senders, id)

	return nil
}

// GetFilterNotifications returns the filter notifications for a given filter
func (s *Service) GetFilterNotifications(ctx context.Context, filterID int) ([]domain.FilterNotification, error) {
	notifications, err := s.repo.GetFilterNotifications(ctx, filterID)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find filter notifications for filter: %v", filterID)
		return nil, err
	}
	return notifications, nil
}

func (s *Service) StoreFilterNotifications(ctx context.Context, filterID int, notifications []domain.FilterNotification) error {
	if err := s.repo.StoreFilterNotifications(ctx, filterID, notifications); err != nil {
		s.log.Error().Err(err).Msgf("could not store filter notifications for filter: %v", filterID)
		return err
	}

	if len(notifications) == 0 {
		for _, notification := range s.notifications {
			notification.RemoveFilterEvents(filterID)
		}
	}

	for _, notification := range notifications {
		if notification.NotificationID == 0 {
			continue
		}

		n, ok := s.notifications[notification.NotificationID]
		if ok {
			n.SetFilterEvents(filterID, domain.NewNotificationEventsFromStrings(notification.Events))
		}
	}

	return nil
}

func (s *Service) DeleteFilterNotifications(ctx context.Context, filterID int) error {
	notifications, err := s.repo.GetFilterNotifications(ctx, filterID)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find filter notifications for filter: %v", filterID)
		return err
	}

	if err := s.repo.DeleteFilterNotifications(ctx, filterID); err != nil {
		s.log.Error().Err(err).Msgf("could not delete filter notifications for filter: %v", filterID)
		return err
	}

	for _, notification := range notifications {
		if notification.NotificationID == 0 {
			continue
		}
		n, ok := s.notifications[notification.NotificationID]
		if ok {
			n.RemoveFilterEvents(filterID)
		}
	}

	return nil
}

func (s *Service) registerSenders() {
	ctx := context.Background()
	notifications, err := s.repo.List(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("could not find notifications")
		return
	}

	for _, notificationSender := range notifications {
		f, err := s.repo.GetNotificationFilters(ctx, notificationSender.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("could not find filter notifications for notification: %v", notificationSender.ID)
			continue
		}
		for _, notification := range f {
			notificationSender.SetFilterEvents(notification.FilterID, domain.NewNotificationEventsFromStrings(notification.Events))
		}

		s.notifications[notificationSender.ID] = &notificationSender

		s.registerSender(&notificationSender)
	}

	return
}

// registerSender registers an enabled notification via it's id
func (s *Service) registerSender(notification *domain.Notification) {
	if !notification.Enabled {
		delete(s.senders, notification.ID)
		return
	}

	switch notification.Type {
	case domain.NotificationTypeDiscord:
		s.senders[notification.ID] = NewDiscordSender(s.log, notification)
		break
	case domain.NotificationTypeGotify:
		s.senders[notification.ID] = NewGotifySender(s.log, notification)
		break
	case domain.NotificationTypeLunaSea:
		s.senders[notification.ID] = NewLunaSeaSender(s.log, notification)
		break
	case domain.NotificationTypeNotifiarr:
		s.senders[notification.ID] = NewNotifiarrSender(s.log, notification)
		break
	case domain.NotificationTypeNtfy:
		s.senders[notification.ID] = NewNtfySender(s.log, notification)
		break
	case domain.NotificationTypePushover:
		s.senders[notification.ID] = NewPushoverSender(s.log, notification)
		break
	case domain.NotificationTypeShoutrrr:
		s.senders[notification.ID] = NewShoutrrrSender(s.log, notification)
		break
	case domain.NotificationTypeTelegram:
		s.senders[notification.ID] = NewTelegramSender(s.log, notification)
		break
	default:
		s.log.Error().Msgf("unsupported notification type: %v", notification.Type)
		return
	}

	return
}

// Send notifications
func (s *Service) Send(event domain.NotificationEvent, payload domain.NotificationPayload) {
	if len(s.senders) == 0 {
		s.log.Trace().Msg("no notification senders registered")
		return
	}

	go func(event domain.NotificationEvent, payload domain.NotificationPayload) {
		for _, sender := range s.senders {
			// check if the sender is active and have notification types
			if sender.CanSendPayload(event, payload) {
				s.log.Debug().Str("sender", sender.Name()).Str("event", string(event)).Msg("sending notification")

				if err := sender.Send(event, payload); err != nil {
					s.log.Error().Err(err).Msgf("could not send %s notification for %v", sender.Name(), string(event))
				}
			}
		}
	}(event, payload)

	return
}

func (s *Service) Test(ctx context.Context, notification *domain.Notification) error {
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
		if !enabledEvent(notification.Events, event.Event) {
			continue
		}

		if err := agent.Send(event.Event, event); err != nil {
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
