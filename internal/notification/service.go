// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"
	"maps"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/moistari/rls"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
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
	ListFilterNotifications(ctx context.Context) ([]domain.FilterNotification, error)
	StoreFilterNotifications(ctx context.Context, filterID int, notifications []domain.FilterNotification) error
	DeleteFilterNotifications(ctx context.Context, filterID int) error
	DeleteOrphanFilterNotifications(ctx context.Context) error
}

type Sender interface {
	Send(ctx context.Context, payload domain.NotificationPayload) error
	IsEnabled() bool
	Name() string
}

type eventBus interface {
	OnAppUpdate(handler func(context.Context, events.AppUpdateEvent) error) func()
	OnReleaseNew(handler func(context.Context, events.ReleaseEvent) error) func()
	OnReleasePush(handler func(context.Context, events.ReleasePushEvent) error) func()
	OnIRC(handler func(context.Context, events.IRCEvent) error) func()
}

type Service struct {
	log      zerolog.Logger
	eventBus eventBus
	repo     notificationRepo

	stateMu sync.Mutex
	state   atomic.Pointer[routingSnapshot]
}

type eventSet map[domain.NotificationEvent]struct{}

type routingSnapshot struct {
	senders      map[int]Sender
	globalEvents map[int]eventSet
	filterRoutes map[int]map[int]eventSet
}

// NewService loads notification routes and registers event listeners.
func NewService(log zerolog.Logger, eventBus eventBus, repo notificationRepo) *Service {
	s := &Service{
		log:      log.With().Str("module", "notification").Logger(),
		eventBus: eventBus,
		repo:     repo,
	}

	return s
}

func (s *Service) Start() error {
	s.setupEventListeners()

	if err := s.loadRoutingSnapshot(context.Background()); err != nil {
		return err
	}

	return nil
}

func newRoutingSnapshot() *routingSnapshot {
	return &routingSnapshot{
		senders:      make(map[int]Sender),
		globalEvents: make(map[int]eventSet),
		filterRoutes: make(map[int]map[int]eventSet),
	}
}

func (r *routingSnapshot) clone() *routingSnapshot {
	next := newRoutingSnapshot()

	maps.Copy(next.senders, r.senders)
	maps.Copy(next.globalEvents, r.globalEvents)
	maps.Copy(next.filterRoutes, r.filterRoutes)

	return next
}

func newEventSet(events []string) eventSet {
	set := make(eventSet, len(events))
	for _, event := range events {
		set[domain.NotificationEvent(event)] = struct{}{}
	}
	return set
}

func (e eventSet) contains(event domain.NotificationEvent) bool {
	_, ok := e[event]
	return ok
}

func (r *routingSnapshot) resolve(event domain.NotificationEvent, filterID int) []Sender {
	if filterID > 0 {
		if routes, ok := r.filterRoutes[filterID]; ok {
			interested := make([]Sender, 0, len(routes))
			for notificationID, evt := range routes {
				if !evt.contains(event) {
					continue
				}

				if sender, ok := r.senders[notificationID]; ok {
					interested = append(interested, sender)
				}
			}
			return interested
		}
	}

	interested := make([]Sender, 0, len(r.senders))
	for notificationID, evt := range r.globalEvents {
		if !evt.contains(event) {
			continue
		}

		if sender, ok := r.senders[notificationID]; ok {
			interested = append(interested, sender)
		}
	}

	return interested
}

func (s *Service) currentSnapshot() *routingSnapshot {
	if current := s.state.Load(); current != nil {
		return current
	}
	return newRoutingSnapshot()
}

func (s *Service) loadRoutingSnapshot(ctx context.Context) error {
	if err := s.repo.DeleteOrphanFilterNotifications(ctx); err != nil {
		return errors.Wrap(err, "could not delete orphaned filter notifications")
	}

	notifications, err := s.repo.List(ctx)
	if err != nil {
		return errors.Wrap(err, "could not list notifications")
	}

	filterNotifications, err := s.repo.ListFilterNotifications(ctx)
	if err != nil {
		return errors.Wrap(err, "could not list filter notifications")
	}

	snapshot := newRoutingSnapshot()
	for idx := range notifications {
		s.setNotification(snapshot, &notifications[idx])
	}
	for _, notification := range filterNotifications {
		routes, ok := snapshot.filterRoutes[notification.FilterID]
		if !ok {
			routes = make(map[int]eventSet)
			snapshot.filterRoutes[notification.FilterID] = routes
		}
		routes[notification.NotificationID] = newEventSet(notification.Events)
	}

	s.state.Store(snapshot)
	return nil
}

func cloneNotification(notification *domain.Notification) *domain.Notification {
	cloned := *notification
	cloned.Events = append([]string(nil), notification.Events...)
	cloned.UsedByFilters = make([]domain.FilterNotification, len(notification.UsedByFilters))
	for idx, filter := range notification.UsedByFilters {
		cloned.UsedByFilters[idx] = filter
		cloned.UsedByFilters[idx].Events = append([]string(nil), filter.Events...)
	}

	if notification.EventSounds != nil {
		cloned.EventSounds = make(map[string]string, len(notification.EventSounds))
		maps.Copy(cloned.EventSounds, notification.EventSounds)
	}

	return &cloned
}

func (s *Service) setNotification(snapshot *routingSnapshot, notification *domain.Notification) {
	configuration := cloneNotification(notification)
	snapshot.globalEvents[configuration.ID] = newEventSet(configuration.Events)

	sender := s.newSender(configuration)
	if sender == nil || !sender.IsEnabled() {
		delete(snapshot.senders, configuration.ID)
		return
	}

	snapshot.senders[configuration.ID] = sender
}

func (s *Service) setupEventListeners() {
	s.eventBus.OnAppUpdate(func(ctx context.Context, event events.AppUpdateEvent) error {
		payload := domain.NotificationPayload{
			Event:          domain.NotificationEventAppUpdateAvailable,
			Subject:        "New update available!",
			Message:        event.NewVersion,
			CurrentVersion: event.CurrentVersion,
			NewVersion:     event.NewVersion,
			URL:            event.URL,
			Timestamp:      time.Now(),
		}
		s.Send(ctx, payload)

		return nil
	})

	s.eventBus.OnReleaseNew(func(ctx context.Context, event events.ReleaseEvent) error {
		release := event.Release
		payload := domain.NotificationPayload{
			Event:          domain.NotificationEventReleaseNew,
			ReleaseName:    release.TorrentName,
			Indexer:        release.Indexer.Name,
			InfoHash:       release.TorrentHash,
			Size:           release.Size,
			Protocol:       release.Protocol,
			Implementation: release.Implementation,
			Timestamp:      time.Now(),
			Release:        release,
		}
		s.Send(ctx, payload)

		return nil
	})

	s.eventBus.OnReleasePush(func(ctx context.Context, event events.ReleasePushEvent) error {
		release := event.Release
		action := event.Action
		status := event.ActionStatus

		payload := domain.NotificationPayload{
			Event:       domain.NotificationEventPushApproved,
			ReleaseName: release.TorrentName,
			Filter:      release.FilterName,
			FilterID:    release.FilterID,
			Indexer:     release.Indexer.Name,
			InfoHash:    release.TorrentHash,
			Size:        release.Size,
			Status:      domain.ReleasePushStatusApproved,
			Action:      action.Name,
			ActionType:  action.Type,
			//Rejections:     status.Rejections,
			Rejections:     []string{},
			Protocol:       release.Protocol,
			Implementation: release.Implementation,
			Timestamp:      time.Now(),
			Release:        release,
		}

		if action.Client != nil {
			payload.ActionClient = action.Client.Name
		}

		switch event.Type {
		case events.ReleasePushApproved:
			payload.Event = domain.NotificationEventPushApproved
			payload.Status = domain.ReleasePushStatusApproved

		case events.ReleasePushRejected:
			payload.Event = domain.NotificationEventPushRejected
			payload.Status = domain.ReleasePushStatusRejected
			payload.Rejections = status.Rejections

		case events.ReleasePushError:
			payload.Event = domain.NotificationEventPushError
			payload.Status = domain.ReleasePushStatusErr
			payload.Rejections = status.Rejections
		default:
			return nil
		}

		s.Send(ctx, payload)

		return nil
	})

	s.eventBus.OnIRC(func(ctx context.Context, event events.IRCEvent) error {
		var payload domain.NotificationPayload

		switch event.Type {
		case events.IRCReconnected:
			payload = domain.NotificationPayload{
				Event:   domain.NotificationEventIRCReconnected,
				Subject: "IRC Reconnected",
				Message: event.Network,
				//Message: fmt.Sprintf("Network: %s", networkName),
			}

		case events.IRCDisconnected:
			payload = domain.NotificationPayload{
				Event:   domain.NotificationEventIRCDisconnected,
				Subject: "IRC Disconnected",
				Message: event.Network,
			}

		case events.IRCFlapping:
			payload = domain.NotificationPayload{
				Event:   domain.NotificationEventIRCDisconnected,
				Subject: "IRC Stopped",
				Message: event.Message,
			}
		default:
			return nil
		}

		s.Send(ctx, payload)

		return nil
	})
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
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	err := s.repo.Store(ctx, notification)
	if err != nil {
		s.log.Error().Err(err).Interface("notification", notification).Msg("could not store notification")
		return err
	}

	next := s.currentSnapshot().clone()
	s.setNotification(next, notification)
	s.state.Store(next)

	return nil
}

func (s *Service) Update(ctx context.Context, notification *domain.Notification) error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

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

	if err := s.repo.Update(ctx, notification); err != nil {
		s.log.Error().Err(err).Interface("notification", notification).Msg("could not update notification")
		return err
	}

	next := s.currentSnapshot().clone()
	s.setNotification(next, notification)
	s.state.Store(next)

	return nil
}

func (s *Service) Delete(ctx context.Context, notificationID int) error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if _, err := s.repo.FindByID(ctx, notificationID); err != nil {
		s.log.Error().Err(err).Int("notification_id", notificationID).Msg("could not find notification by id")
		return err
	}

	if err := s.repo.Delete(ctx, notificationID); err != nil {
		s.log.Error().Err(err).Int("notification_id", notificationID).Msg("could not delete notification")
		return err
	}

	next := s.currentSnapshot().clone()
	delete(next.senders, notificationID)
	delete(next.globalEvents, notificationID)

	for filterID, currentRoutes := range next.filterRoutes {
		if _, ok := currentRoutes[notificationID]; !ok {
			continue
		}

		routes := make(map[int]eventSet, len(currentRoutes)-1)
		for routeNotificationID, events := range currentRoutes {
			if routeNotificationID != notificationID {
				routes[routeNotificationID] = events
			}
		}

		if len(routes) == 0 {
			delete(next.filterRoutes, filterID)
		} else {
			next.filterRoutes[filterID] = routes
		}
	}

	s.state.Store(next)

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
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	current := s.currentSnapshot()
	for _, notification := range notifications {
		if _, ok := current.globalEvents[notification.NotificationID]; !ok {
			return errors.Wrap(domain.ErrNotificationNotFound, "notification with id %d does not exist", notification.NotificationID)
		}
	}

	if err := s.repo.StoreFilterNotifications(ctx, filterID, notifications); err != nil {
		s.log.Error().Err(err).Int("filter_id", filterID).Msg("could not store filter notifications for filter")
		return err
	}

	next := current.clone()
	if len(notifications) == 0 {
		delete(next.filterRoutes, filterID)
	} else {
		routes := make(map[int]eventSet, len(notifications))
		for _, notification := range notifications {
			routes[notification.NotificationID] = newEventSet(notification.Events)
		}
		next.filterRoutes[filterID] = routes
	}
	s.state.Store(next)

	return nil
}

func (s *Service) DeleteFilterNotifications(ctx context.Context, filterID int) error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if err := s.repo.DeleteFilterNotifications(ctx, filterID); err != nil {
		s.log.Error().Err(err).Int("filter_id", filterID).Msg("could not delete filter notifications for filter")
		return err
	}

	next := s.currentSnapshot().clone()
	delete(next.filterRoutes, filterID)
	s.state.Store(next)

	return nil
}

func (s *Service) newSender(notification *domain.Notification) Sender {
	switch notification.Type {
	case domain.NotificationTypeDiscord:
		return NewDiscordSender(s.log, notification)
	case domain.NotificationTypeGotify:
		return NewGotifySender(s.log, notification)
	case domain.NotificationTypeLunaSea:
		return NewLunaSeaSender(s.log, notification)
	case domain.NotificationTypeNotifiarr:
		return NewNotifiarrSender(s.log, notification)
	case domain.NotificationTypeNtfy:
		return NewNtfySender(s.log, notification)
	case domain.NotificationTypePushover:
		return NewPushoverSender(s.log, notification)
	case domain.NotificationTypeShoutrrr:
		return NewShoutrrrSender(s.log, notification)
	case domain.NotificationTypeTelegram:
		return NewTelegramSender(s.log, notification)
	case domain.NotificationTypeWebhook:
		return NewWebhookSender(s.log, notification)
	default:
		s.log.Error().Str("notification_type", string(notification.Type)).Msg("unsupported notification type")
		return nil
	}
}

func (s *Service) Send(ctx context.Context, payload domain.NotificationPayload) {
	snapshot := s.currentSnapshot()
	if len(snapshot.senders) == 0 {
		s.log.Trace().Msg("no notification senders registered")
		return
	}

	interestedSenders := snapshot.resolve(payload.Event, payload.FilterID)
	if len(interestedSenders) == 0 {
		s.log.Trace().Str("event", string(payload.Event)).Msg("no interested notification senders for event")
		return
	}

	go func(interested []Sender, payload domain.NotificationPayload) {
		for _, sender := range interested {
			func() {
				senderCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
				s.log.Debug().Str("sender", sender.Name()).Str("event", string(payload.Event)).Msg("sending notification")

				if err := sender.Send(senderCtx, payload); err != nil {
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

	var agent Sender

	// send test events
	testEvents := []domain.NotificationPayload{
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
		{
			Subject:        "New release received!",
			Message:        "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
			Event:          domain.NotificationEventReleaseNew,
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

	agent = s.newSender(notification)
	if agent == nil {
		return errors.New("unsupported notification type")
	}

	g, gCtx := errgroup.WithContext(ctx)

	for _, event := range testEvents {
		if !enabledEvent(notification.Events, event.Event) {
			continue
		}

		if err := agent.Send(gCtx, event); err != nil {
			s.log.Error().Err(err).Interface("notification", notification).Msg("error sending test notification")
			return err
		}

		time.Sleep(1 * time.Second)
	}

	if err := g.Wait(); err != nil {
		s.log.Error().Err(err).Str("notification_type", string(notification.Type)).Msg("something went wrong sending test notifications")
		return err
	}

	return nil
}

func enabledEvent(events []string, e domain.NotificationEvent) bool {
	return slices.Contains(events, string(e))
}
