// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/moistari/rls"
	"github.com/rs/zerolog"
)

type notificationRepo interface {
	List(ctx context.Context) ([]domain.Notification, error)
	Find(ctx context.Context, params domain.NotificationQueryParams) ([]domain.Notification, int, error)
	FindByID(ctx context.Context, notificationID int) (*domain.Notification, error)
	Store(ctx context.Context, notification *domain.Notification) error
	Update(ctx context.Context, notification *domain.Notification) error
	Delete(ctx context.Context, notificationID int) error

	GetNotificationFilters(ctx context.Context, notificationID int) ([]domain.FilterNotification, error)
	GetFilterNotifications(ctx context.Context, filterID int) ([]domain.FilterNotification, error)
	StoreFilterNotifications(ctx context.Context, filterID int, notifications []domain.FilterNotification) error
	DeleteFilterNotifications(ctx context.Context, filterID int) error
}

type Sender interface {
	Name() string
	IsEnabled() bool
	CanSendPayload(payload domain.NotificationPayload) bool
	Send(ctx context.Context, payload domain.NotificationPayload) error
}

type Service struct {
	log  zerolog.Logger
	repo notificationRepo

	mu            sync.RWMutex
	notifications map[int]*domain.Notification
	senders       map[int]Sender
}

func NewService(log zerolog.Logger, repo notificationRepo) *Service {
	s := &Service{
		log:           log.With().Str("module", "notification").Logger(),
		repo:          repo,
		notifications: make(map[int]*domain.Notification),
		senders:       make(map[int]Sender),
	}

	s.registerSenders()

	return s
}

func (s *Service) Find(ctx context.Context, params domain.NotificationQueryParams) ([]domain.Notification, int, error) {
	notifications, count, err := s.repo.Find(ctx, params)
	if err != nil {
		s.log.Error().Err(err).Interface("params", params).Msg("could not find notification with params")
		return nil, 0, err
	}

	for idx, notification := range notifications {
		filters, err := s.repo.GetNotificationFilters(ctx, notification.ID)
		if err != nil {
			s.log.Error().Err(err).Int("notification_id", notification.ID).Msg("could not find filter notifications for notification")
			continue
		}
		notifications[idx].UsedByFilters = filters
	}

	return notifications, count, err
}

func (s *Service) FindByID(ctx context.Context, notificationID int) (*domain.Notification, error) {
	notification, err := s.repo.FindByID(ctx, notificationID)
	if err != nil {
		s.log.Error().Err(err).Int("notification_id", notificationID).Msg("could not find notification by id")
		return nil, err
	}

	return notification, err
}

func (s *Service) Store(ctx context.Context, notification *domain.Notification) error {
	if notification.FilterScope == "" {
		notification.FilterScope = domain.NotificationFilterScopeGlobal
	} else if !notification.FilterScope.Valid() {
		return errors.New("invalid filter_scope: %s", notification.FilterScope)
	}

	err := s.repo.Store(ctx, notification)
	if err != nil {
		s.log.Error().Err(err).Interface("notification", notification).Msg("could not store notification")
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// register a fresh canonical copy so the sender is never bound to the raw
	// request object (whose private filters map is always nil) and so a newly
	// created notification is tracked in s.notifications for later filter saves.
	s.hydrateAndRegister(ctx, notification.ID)

	return nil
}

func (s *Service) Update(ctx context.Context, notification *domain.Notification) error {
	existing, err := s.repo.FindByID(ctx, notification.ID)
	if err != nil {
		s.log.Error().Err(err).Int("notification_id", notification.ID).Msg("could not find notification by id")
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

	if notification.FilterScope == "" {
		notification.FilterScope = domain.NotificationFilterScopeGlobal
	} else if !notification.FilterScope.Valid() {
		return errors.New("invalid filter_scope: %s", notification.FilterScope)
	}

	if err := s.repo.Update(ctx, notification); err != nil {
		s.log.Error().Err(err).Interface("notification", notification).Msg("could not update notification")
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Rebuild the canonical object from the updated global config plus the
	// persisted per-filter rows. The incoming object is decoded from JSON and
	// its private filters map is always nil, so registering it directly would
	// silently drop every per-filter mute/override until restart.
	s.hydrateAndRegister(ctx, notification.ID)

	return nil
}

func (s *Service) Delete(ctx context.Context, notificationID int) error {
	if _, err := s.repo.FindByID(ctx, notificationID); err != nil {
		s.log.Error().Err(err).Int("notification_id", notificationID).Msg("could not find notification by id")
		return err
	}

	if err := s.repo.Delete(ctx, notificationID); err != nil {
		s.log.Error().Err(err).Int("notification_id", notificationID).Msg("could not delete notification")
		return err
	}

	s.mu.Lock()
	delete(s.senders, notificationID)
	delete(s.notifications, notificationID)
	s.mu.Unlock()

	return nil
}

// GetFilterNotifications returns the filter notifications for a given filter
func (s *Service) GetFilterNotifications(ctx context.Context, filterID int) ([]domain.FilterNotification, error) {
	notifications, err := s.repo.GetFilterNotifications(ctx, filterID)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filterID).Msg("could not find filter notifications for filter")
		return nil, err
	}
	return notifications, nil
}

func (s *Service) StoreFilterNotifications(ctx context.Context, filterID int, notifications []domain.FilterNotification) error {
	if err := s.repo.StoreFilterNotifications(ctx, filterID, notifications); err != nil {
		s.log.Error().Err(err).Int("filter_id", filterID).Msg("could not store filter notifications for filter")
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// The affected notifications are those referenced in the new set plus any
	// that currently hold an in-memory entry for this filter, so that removing
	// a notification from the filter (present before, absent now) clears its
	// stale per-filter events too.
	affected := make(map[int]struct{})
	for _, notification := range notifications {
		if notification.NotificationID > 0 {
			affected[notification.NotificationID] = struct{}{}
		}
	}
	for id, n := range s.notifications {
		if n.HasFilterNotifications(filterID) {
			affected[id] = struct{}{}
		}
	}

	for id := range affected {
		s.hydrateAndRegister(ctx, id)
	}

	return nil
}

func (s *Service) DeleteFilterNotifications(ctx context.Context, filterID int) error {
	if err := s.repo.DeleteFilterNotifications(ctx, filterID); err != nil {
		s.log.Error().Err(err).Int("filter_id", filterID).Msg("could not delete filter notifications for filter")
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Rebuild every notification that still holds this filter in memory so its
	// per-filter events are dropped now that the rows are gone from the DB.
	var ids []int
	for id, n := range s.notifications {
		if n.HasFilterNotifications(filterID) {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		s.hydrateAndRegister(ctx, id)
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

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, notification := range notifications {
		s.hydrateAndRegister(ctx, notification.ID)
	}
}

// hydrateAndRegister rebuilds the canonical *domain.Notification for the id
// from the database - global config and per-filter events - stores it in
// s.notifications and (re)registers its sender. The database is authoritative:
// a row deleted by a concurrent request deregisters the sender instead of
// resurrecting it, and a read error keeps the previous state instead of
// registering a half-hydrated object. The previous object is never mutated
// (copy-on-write), so a concurrent Send goroutine reading it cannot race.
// Callers must hold s.mu for writing.
func (s *Service) hydrateAndRegister(ctx context.Context, notificationID int) {
	// a client disconnect right after the preceding commit must not cancel the
	// rebuild and corrupt in-memory state
	ctx = context.WithoutCancel(ctx)

	fresh, err := s.repo.FindByID(ctx, notificationID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			delete(s.senders, notificationID)
			delete(s.notifications, notificationID)
			return
		}

		s.log.Error().Err(err).Int("notification_id", notificationID).Msg("could not find notification by id")
		return
	}

	filterNotifications, err := s.repo.GetNotificationFilters(ctx, notificationID)
	if err != nil {
		s.log.Error().Err(err).Int("notification_id", notificationID).Msg("could not find filter notifications for notification")
		return
	}

	for _, fn := range filterNotifications {
		fresh.SetFilterEvents(fn.FilterID, domain.NewNotificationEventsFromStrings(fn.Events))
	}

	s.notifications[notificationID] = fresh
	s.registerSender(fresh)
}

// registerSender registers an enabled notification via it's id.
// Callers must hold s.mu for writing.
func (s *Service) registerSender(notification *domain.Notification) {
	if !notification.Enabled {
		delete(s.senders, notification.ID)
		return
	}

	sender, err := newSender(s.log, notification)
	if err != nil {
		s.log.Error().Err(err).Str("notification_type", string(notification.Type)).Msg("could not register sender")
		return
	}

	s.senders[notification.ID] = sender
}

// newSender builds the sender for the notification type.
func newSender(log zerolog.Logger, notification *domain.Notification) (Sender, error) {
	switch notification.Type {
	case domain.NotificationTypeDiscord:
		return NewDiscordService(log, notification), nil
	case domain.NotificationTypeGotify:
		return NewGotifyService(log, notification), nil
	case domain.NotificationTypeLunaSea:
		return NewLunaSeaService(log, notification), nil
	case domain.NotificationTypeNotifiarr:
		return NewNotifiarrService(log, notification), nil
	case domain.NotificationTypeNtfy:
		return NewNtfyService(log, notification), nil
	case domain.NotificationTypePushover:
		return NewPushoverService(log, notification), nil
	case domain.NotificationTypeShoutrrr:
		return NewShoutrrrService(log, notification), nil
	case domain.NotificationTypeTelegram:
		return NewTelegramService(log, notification), nil
	case domain.NotificationTypeWebhook:
		return NewWebhookService(log, notification), nil
	}

	return nil, errors.New("unsupported notification type: %s", notification.Type)
}

// Send notifications
func (s *Service) Send(payload domain.NotificationPayload) {
	// Select interested senders under a read lock, then release it before any
	// network I/O. Each sender's CanSendPayload already encodes the full
	// per-(filter, notification) decision - mute, per-filter override, and the
	// per-notification global fallback - so a single uniform pass is correct for
	// both filter-scoped and global (FilterID == 0) events. Evaluating every
	// sender independently ensures one notification's per-filter config (or
	// mute) never suppresses another notification.
	s.mu.RLock()
	var interestedSenders []Sender
	for _, sender := range s.senders {
		if sender.CanSendPayload(payload) {
			interestedSenders = append(interestedSenders, sender)
		}
	}
	s.mu.RUnlock()

	if len(interestedSenders) == 0 {
		s.log.Trace().Str("event", string(payload.Event)).Msg("no interested notification senders for event")
		return
	}

	go func(interested []Sender, payload domain.NotificationPayload) {
		for _, sender := range interested {
			func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				s.log.Debug().Str("sender", sender.Name()).Str("event", string(payload.Event)).Msg("sending notification")

				if err := sender.Send(ctx, payload); err != nil {
					s.log.Error().Err(err).Str("sender", sender.Name()).Str("event", string(payload.Event)).Msg("could not send notification")
				}
			}()
		}
	}(interestedSenders, payload)
}

func (s *Service) Test(ctx context.Context, notification *domain.Notification) error {
	if notification.ID > 0 {
		existing, err := s.repo.FindByID(ctx, notification.ID)
		if err != nil {
			s.log.Error().Err(err).Int("notification_id", notification.ID).Msg("could not find notification by id")
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
	}

	// send test events
	events := []domain.NotificationPayload{
		{
			Event:     domain.NotificationEventTest,
			Subject:   "Test Notification",
			Message:   "autobrr goes brr!!",
			Timestamp: time.Now(),
		},
		{
			Event:          domain.NotificationEventPushApproved,
			Subject:        "New release!",
			Message:        "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
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
			Release: &domain.Release{
				Type:        rls.Episode,
				TorrentName: "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
				Title:       "Best Show Ever",
				Season:      18,
				Episode:     21,
				Year:        2026,
				Resolution:  "1080p",
				Source:      "WEB-DL",
				Codec:       []string{"H.264"},
				Container:   "mkv",
				Audio:       []string{"DDP2.0"},
				Group:       "GROUP",
				Size:        1500000000,
			},
		},
		{
			Event:          domain.NotificationEventPushRejected,
			Subject:        "New release!",
			Message:        "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
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
			Release: &domain.Release{
				TorrentName: "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
				Title:       "Best Show Ever",
				Season:      18,
				Episode:     21,
				Year:        2026,
				Resolution:  "1080p",
				Source:      "WEB-DL",
				Codec:       []string{"H.264"},
				Container:   "mkv",
				Audio:       []string{"DDP2.0"},
				Group:       "GROUP",
				Size:        1500000000,
			},
		},
		{
			Event:          domain.NotificationEventPushError,
			Subject:        "New release!",
			Message:        "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
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
			Release: &domain.Release{
				TorrentName: "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
				Title:       "Best Show Ever",
				Season:      18,
				Episode:     21,
				Year:        2026,
				Resolution:  "1080p",
				Source:      "WEB-DL",
				Codec:       []string{"H.264"},
				Container:   "mkv",
				Audio:       []string{"DDP2.0"},
				Group:       "GROUP",
				Size:        1500000000,
			},
		},
		{
			Event:     domain.NotificationEventIRCDisconnected,
			Subject:   "IRC Disconnected unexpectedly",
			Message:   "Network: P2P-Network",
			Timestamp: time.Now(),
		},
		{
			Event:     domain.NotificationEventIRCReconnected,
			Subject:   "IRC Reconnected",
			Message:   "Network: P2P-Network",
			Timestamp: time.Now(),
		},
		{
			Event:          domain.NotificationEventAppUpdateAvailable,
			Subject:        "New update available!",
			Message:        "v1.6.0",
			CurrentVersion: "v1.6.0",
			NewVersion:     "v1.7.0",
			URL:            "https://github.com/autobrr/autobrr/releases/tag/v1.7.0",
			Timestamp:      time.Now(),
		},
		{
			Event:          domain.NotificationEventReleaseNew,
			Subject:        "New release received!",
			Message:        "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
			ReleaseName:    "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
			Filter:         "TV",
			Indexer:        "MockIndexer",
			Protocol:       domain.ReleaseProtocolTorrent,
			Implementation: domain.ReleaseImplementationIRC,
			Timestamp:      time.Now(),
			Release: &domain.Release{
				TorrentName: "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
				Title:       "Best Show Ever",
				Season:      18,
				Episode:     21,
				Year:        2026,
				Resolution:  "1080p",
				Source:      "WEB-DL",
				Codec:       []string{"H.264"},
				Container:   "mkv",
				Audio:       []string{"DDP2.0"},
				Group:       "GROUP",
				Size:        1500000000,
			},
		},
	}

	sender, err := newSender(s.log, notification)
	if err != nil {
		s.log.Error().Err(err).Str("notification_type", string(notification.Type)).Msg("could not create sender")
		return err
	}

	for _, event := range events {
		if !enabledEvent(notification.Events, event.Event) {
			continue
		}

		if err := sender.Send(ctx, event); err != nil {
			s.log.Error().Err(err).Interface("notification", notification).Msg("error sending test notification")
			return err
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

func enabledEvent(events []string, e domain.NotificationEvent) bool {
	return slices.Contains(events, string(e))
}
