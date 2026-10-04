package notification

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/r3labs/sse/v2"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSender struct {
	name string
	sent chan struct{}
}

func (s *testSender) Send(context.Context, domain.NotificationPayload) error {
	if s.sent != nil {
		s.sent <- struct{}{}
	}
	return nil
}

func (s *testSender) IsEnabled() bool {
	return true
}

func (s *testSender) Name() string {
	return s.name
}

type testEventBus struct{}

func (testEventBus) OnAppUpdate(func(context.Context, events.AppUpdateEvent) error) func() {
	return func() {}
}

func (testEventBus) OnReleaseNew(func(context.Context, events.ReleaseEvent) error) func() {
	return func() {}
}

func (testEventBus) OnReleasePush(func(context.Context, events.ReleasePushEvent) error) func() {
	return func() {}
}

func (testEventBus) OnIRC(func(context.Context, events.IRCEvent) error) func() {
	return func() {}
}

type testInboxRepo struct {
	mu       sync.Mutex
	messages []*domain.InboxMessage
	cleaned  int64
}

func (r *testInboxRepo) Find(context.Context, domain.InboxQueryParams) (*domain.FindInboxResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return &domain.FindInboxResponse{Data: r.messages, TotalCount: len(r.messages)}, nil
}

func (r *testInboxRepo) Store(_ context.Context, msg *domain.InboxMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	msg.ID = int64(len(r.messages) + 1)
	r.messages = append(r.messages, msg)
	return nil
}

func (r *testInboxRepo) MarkRead(context.Context, []int64, time.Time) error { return nil }

func (r *testInboxRepo) Delete(context.Context, []int64) error { return nil }

func (r *testInboxRepo) DeleteAll(context.Context) error { return nil }

func (r *testInboxRepo) Cleanup(context.Context, domain.InboxCleanupParams) (int64, error) {
	return r.cleaned, nil
}

type testPublisher struct {
	mu     sync.Mutex
	events []*sse.Event
}

func (p *testPublisher) Publish(_ string, event *sse.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.events = append(p.events, event)
}

type testScheduler struct{}

func (testScheduler) ScheduleJob(cron.Job, time.Duration, string) (int, error) {
	return 1, nil
}

func newTestService(repo *testNotificationRepo) *Service {
	return NewService(zerolog.Nop(), testEventBus{}, &testPublisher{}, repo, &testInboxRepo{}, testScheduler{})
}

type testNotificationRepo struct {
	mu                  sync.Mutex
	notifications       map[int]domain.Notification
	filterNotifications map[int][]domain.FilterNotification
	nextID              int
	listErr             error
	filterListErr       error
	cleanupErr          error
}

func newTestNotificationRepo(notifications []domain.Notification, filterNotifications map[int][]domain.FilterNotification) *testNotificationRepo {
	repo := &testNotificationRepo{
		notifications:       make(map[int]domain.Notification, len(notifications)),
		filterNotifications: make(map[int][]domain.FilterNotification, len(filterNotifications)),
		nextID:              1,
	}

	for idx := range notifications {
		notification := *cloneNotification(&notifications[idx])
		repo.notifications[notification.ID] = notification
		if notification.ID >= repo.nextID {
			repo.nextID = notification.ID + 1
		}
	}
	for filterID, routes := range filterNotifications {
		repo.filterNotifications[filterID] = cloneFilterNotifications(routes)
	}

	return repo
}

func cloneFilterNotifications(notifications []domain.FilterNotification) []domain.FilterNotification {
	cloned := make([]domain.FilterNotification, len(notifications))
	for idx, notification := range notifications {
		cloned[idx] = notification
		cloned[idx].Events = append([]string(nil), notification.Events...)
	}
	return cloned
}

func (r *testNotificationRepo) List(context.Context) ([]domain.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.listErr != nil {
		return nil, r.listErr
	}

	notifications := make([]domain.Notification, 0, len(r.notifications))
	for _, notification := range r.notifications {
		notifications = append(notifications, *cloneNotification(&notification))
	}
	return notifications, nil
}

func (r *testNotificationRepo) Find(ctx context.Context, _ domain.NotificationQueryParams) ([]domain.Notification, int, error) {
	notifications, err := r.List(ctx)
	return notifications, len(notifications), err
}

func (r *testNotificationRepo) FindByID(_ context.Context, notificationID int) (*domain.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	notification, ok := r.notifications[notificationID]
	if !ok {
		return nil, domain.ErrRecordNotFound
	}
	return cloneNotification(&notification), nil
}

func (r *testNotificationRepo) Store(_ context.Context, notification *domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if notification.ID == 0 {
		notification.ID = r.nextID
		r.nextID++
	}
	r.notifications[notification.ID] = *cloneNotification(notification)
	return nil
}

func (r *testNotificationRepo) Update(_ context.Context, notification *domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.notifications[notification.ID]; !ok {
		return domain.ErrRecordNotFound
	}
	r.notifications[notification.ID] = *cloneNotification(notification)
	return nil
}

func (r *testNotificationRepo) Delete(_ context.Context, notificationID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.notifications, notificationID)
	for filterID, routes := range r.filterNotifications {
		kept := routes[:0]
		for _, route := range routes {
			if route.NotificationID != notificationID {
				kept = append(kept, route)
			}
		}
		if len(kept) == 0 {
			delete(r.filterNotifications, filterID)
		} else {
			r.filterNotifications[filterID] = cloneFilterNotifications(kept)
		}
	}
	return nil
}

func (r *testNotificationRepo) GetNotificationFilters(_ context.Context, notificationID int) ([]domain.FilterNotification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var notifications []domain.FilterNotification
	for _, routes := range r.filterNotifications {
		for _, route := range routes {
			if route.NotificationID == notificationID {
				notifications = append(notifications, route)
			}
		}
	}
	return cloneFilterNotifications(notifications), nil
}

func (r *testNotificationRepo) GetFilterNotifications(_ context.Context, filterID int) ([]domain.FilterNotification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneFilterNotifications(r.filterNotifications[filterID]), nil
}

func (r *testNotificationRepo) ListFilterNotifications(context.Context) ([]domain.FilterNotification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.filterListErr != nil {
		return nil, r.filterListErr
	}

	var notifications []domain.FilterNotification
	for _, routes := range r.filterNotifications {
		notifications = append(notifications, routes...)
	}
	return cloneFilterNotifications(notifications), nil
}

func (r *testNotificationRepo) StoreFilterNotifications(_ context.Context, filterID int, notifications []domain.FilterNotification) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(notifications) == 0 {
		delete(r.filterNotifications, filterID)
	} else {
		r.filterNotifications[filterID] = cloneFilterNotifications(notifications)
	}
	return nil
}

func (r *testNotificationRepo) DeleteFilterNotifications(_ context.Context, filterID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.filterNotifications, filterID)
	return nil
}

func (r *testNotificationRepo) DeleteOrphanFilterNotifications(context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cleanupErr != nil {
		return r.cleanupErr
	}

	for filterID, routes := range r.filterNotifications {
		kept := routes[:0]
		for _, route := range routes {
			if _, ok := r.notifications[route.NotificationID]; ok {
				kept = append(kept, route)
			}
		}
		if len(kept) == 0 {
			delete(r.filterNotifications, filterID)
		} else {
			r.filterNotifications[filterID] = cloneFilterNotifications(kept)
		}
	}
	return nil
}

func validNotification(id int, name string, events ...domain.NotificationEvent) domain.Notification {
	eventNames := make([]string, len(events))
	for idx, event := range events {
		eventNames[idx] = string(event)
	}

	return domain.Notification{
		ID:      id,
		Name:    name,
		Type:    domain.NotificationTypeWebhook,
		Enabled: true,
		Events:  eventNames,
		Webhook: "https://example.com/notifications",
	}
}

func senderNames(senders []Sender) []string {
	names := make([]string, len(senders))
	for idx, sender := range senders {
		names[idx] = sender.Name()
	}
	sort.Strings(names)
	return names
}

func TestRoutingSnapshotResolve(t *testing.T) {
	pushApproved := domain.NotificationEventPushApproved
	pushRejected := domain.NotificationEventPushRejected

	snapshot := newRoutingSnapshot()
	snapshot.senders[1] = &testSender{name: "one"}
	snapshot.senders[2] = &testSender{name: "two"}
	snapshot.globalEvents[1] = newEventSet([]string{string(pushApproved)})
	snapshot.globalEvents[2] = newEventSet([]string{string(pushRejected)})
	snapshot.filterRoutes[10] = map[int]eventSet{
		1: newEventSet([]string{string(pushRejected)}),
	}
	snapshot.filterRoutes[11] = map[int]eventSet{
		1: newEventSet(nil),
	}
	snapshot.filterRoutes[12] = map[int]eventSet{
		3: newEventSet([]string{string(pushApproved)}),
	}

	tests := []struct {
		name     string
		event    domain.NotificationEvent
		filterID int
		want     []string
	}{
		{name: "global event without filter", event: pushApproved, want: []string{"one"}},
		{name: "filter without override inherits global", event: pushApproved, filterID: 9, want: []string{"one"}},
		{name: "custom event replaces global", event: pushRejected, filterID: 10, want: []string{"one"}},
		{name: "custom route suppresses global event", event: pushApproved, filterID: 10, want: []string{}},
		{name: "empty route mutes filter", event: pushApproved, filterID: 11, want: []string{}},
		{name: "disabled or missing sender does not fall back", event: pushApproved, filterID: 12, want: []string{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, senderNames(snapshot.resolve(test.event, test.filterID)))
		})
	}
}

func TestServiceSend(t *testing.T) {
	t.Run("does not send without a matching route", func(t *testing.T) {
		sender := &testSender{name: "test", sent: make(chan struct{}, 1)}
		snapshot := newRoutingSnapshot()
		snapshot.senders[1] = sender
		snapshot.globalEvents[1] = newEventSet(nil)

		service := &Service{log: zerolog.Nop()}
		service.state.Store(snapshot)
		service.Send(t.Context(), domain.NotificationPayload{Event: domain.NotificationEventReleaseNew})

		select {
		case <-sender.sent:
			require.FailNow(t, "unexpected notification")
		default:
		}
	})

	t.Run("sends to a matching route", func(t *testing.T) {
		sender := &testSender{name: "test", sent: make(chan struct{}, 1)}
		snapshot := newRoutingSnapshot()
		snapshot.senders[1] = sender
		snapshot.globalEvents[1] = newEventSet([]string{string(domain.NotificationEventReleaseNew)})

		service := &Service{log: zerolog.Nop()}
		service.state.Store(snapshot)
		service.Send(t.Context(), domain.NotificationPayload{Event: domain.NotificationEventReleaseNew})

		select {
		case <-sender.sent:
		case <-time.After(time.Second):
			require.FailNow(t, "notification was not sent")
		}
	})
}

func TestServiceRoutingLifecycle(t *testing.T) {
	ctx := t.Context()
	pushApproved := domain.NotificationEventPushApproved

	t.Run("global edit is not reverted by later filter edits", func(t *testing.T) {
		notification := validNotification(1, "webhook", pushApproved)
		repo := newTestNotificationRepo([]domain.Notification{notification}, nil)
		service := newTestService(repo)
		err := service.Start()
		require.NoError(t, err)

		notification.Events = nil
		require.NoError(t, service.Update(ctx, &notification))

		for filterID := 1; filterID <= 4; filterID++ {
			require.NoError(t, service.StoreFilterNotifications(ctx, filterID, []domain.FilterNotification{{
				FilterID:       filterID,
				NotificationID: notification.ID,
				Events:         []string{string(pushApproved)},
			}}))
		}

		assert.Len(t, service.currentSnapshot().resolve(pushApproved, 1), 1)
		assert.Empty(t, service.currentSnapshot().resolve(pushApproved, 5))
		assert.Empty(t, service.currentSnapshot().resolve(pushApproved, 6))

		restarted := newTestService(repo)
		require.NoError(t, restarted.Start())
		for filterID := 1; filterID <= 6; filterID++ {
			assert.Len(t, restarted.currentSnapshot().resolve(pushApproved, filterID), len(service.currentSnapshot().resolve(pushApproved, filterID)))
		}
	})

	t.Run("mute survives notification update", func(t *testing.T) {
		notification := validNotification(1, "webhook", pushApproved)
		repo := newTestNotificationRepo([]domain.Notification{notification}, map[int][]domain.FilterNotification{
			1: {{FilterID: 1, NotificationID: 1, Events: []string{}}},
		})
		service := newTestService(repo)
		err := service.Start()
		require.NoError(t, err)

		notification.Name = "updated webhook"
		require.NoError(t, service.Update(ctx, &notification))
		assert.Empty(t, service.currentSnapshot().resolve(pushApproved, 1))
	})

	t.Run("new notification can be routed without restart", func(t *testing.T) {
		repo := newTestNotificationRepo(nil, nil)
		service := newTestService(repo)
		err := service.Start()
		require.NoError(t, err)

		notification := validNotification(0, "new webhook")
		require.NoError(t, service.Store(ctx, &notification))
		require.NoError(t, service.StoreFilterNotifications(ctx, 1, []domain.FilterNotification{{
			FilterID:       1,
			NotificationID: notification.ID,
			Events:         []string{string(pushApproved)},
		}}))

		assert.Len(t, service.currentSnapshot().resolve(pushApproved, 1), 1)
	})

	t.Run("non-empty replacement removes old routes", func(t *testing.T) {
		notificationOne := validNotification(1, "one")
		notificationTwo := validNotification(2, "two")
		repo := newTestNotificationRepo([]domain.Notification{notificationOne, notificationTwo}, map[int][]domain.FilterNotification{
			1: {
				{FilterID: 1, NotificationID: 1, Events: []string{string(pushApproved)}},
				{FilterID: 1, NotificationID: 2, Events: []string{string(pushApproved)}},
			},
		})
		service := newTestService(repo)
		err := service.Start()
		require.NoError(t, err)

		require.NoError(t, service.StoreFilterNotifications(ctx, 1, []domain.FilterNotification{{
			FilterID:       1,
			NotificationID: 1,
			Events:         []string{string(pushApproved)},
		}}))

		assert.Len(t, service.currentSnapshot().resolve(pushApproved, 1), 1)
		_, oldRouteExists := service.currentSnapshot().filterRoutes[1][2]
		assert.False(t, oldRouteExists)
	})

	t.Run("unknown notification is rejected before persistence", func(t *testing.T) {
		notification := validNotification(1, "one", pushApproved)
		repo := newTestNotificationRepo([]domain.Notification{notification}, nil)
		service := newTestService(repo)
		err := service.Start()
		require.NoError(t, err)

		err = service.StoreFilterNotifications(ctx, 1, []domain.FilterNotification{{
			FilterID:       1,
			NotificationID: 99,
			Events:         []string{string(pushApproved)},
		}})

		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
		stored, findErr := repo.GetFilterNotifications(ctx, 1)
		assert.NoError(t, findErr)
		assert.Empty(t, stored)
	})

	t.Run("disabled service route blocks global fallback", func(t *testing.T) {
		global := validNotification(1, "global", pushApproved)
		disabled := validNotification(2, "disabled")
		disabled.Enabled = false
		repo := newTestNotificationRepo([]domain.Notification{global, disabled}, map[int][]domain.FilterNotification{
			1: {{FilterID: 1, NotificationID: 2, Events: []string{string(pushApproved)}}},
		})
		service := newTestService(repo)
		err := service.Start()
		require.NoError(t, err)

		assert.Empty(t, service.currentSnapshot().resolve(pushApproved, 1))
	})

	t.Run("startup removes orphaned routes", func(t *testing.T) {
		global := validNotification(1, "global", pushApproved)
		repo := newTestNotificationRepo([]domain.Notification{global}, map[int][]domain.FilterNotification{
			1: {{FilterID: 1, NotificationID: 99, Events: []string{string(pushApproved)}}},
		})
		service := newTestService(repo)
		err := service.Start()
		require.NoError(t, err)

		assert.Len(t, service.currentSnapshot().resolve(pushApproved, 1), 1)
	})

	t.Run("deleting final route restores inheritance", func(t *testing.T) {
		global := validNotification(1, "global", pushApproved)
		override := validNotification(2, "override")
		repo := newTestNotificationRepo([]domain.Notification{global, override}, map[int][]domain.FilterNotification{
			1: {{FilterID: 1, NotificationID: 2, Events: []string{}}},
		})
		service := newTestService(repo)
		err := service.Start()
		require.NoError(t, err)

		require.NoError(t, service.Delete(ctx, override.ID))
		assert.Len(t, service.currentSnapshot().resolve(pushApproved, 1), 1)
		_, filterOverrideExists := service.currentSnapshot().filterRoutes[1]
		assert.False(t, filterOverrideExists)
	})

	t.Run("filter deletion does not depend on persisted join rows", func(t *testing.T) {
		global := validNotification(1, "global", pushApproved)
		repo := newTestNotificationRepo([]domain.Notification{global}, map[int][]domain.FilterNotification{
			1: {{FilterID: 1, NotificationID: 1, Events: []string{}}},
		})
		service := newTestService(repo)
		err := service.Start()
		require.NoError(t, err)

		repo.mu.Lock()
		delete(repo.filterNotifications, 1)
		repo.mu.Unlock()

		require.NoError(t, service.DeleteFilterNotifications(ctx, 1))
		assert.Len(t, service.currentSnapshot().resolve(pushApproved, 1), 1)
		_, filterOverrideExists := service.currentSnapshot().filterRoutes[1]
		assert.False(t, filterOverrideExists)
	})
}

func TestServiceStartFailsClosed(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*testNotificationRepo)
	}{
		{
			name: "notification load",
			configure: func(repo *testNotificationRepo) {
				repo.listErr = errors.New("database unavailable")
			},
		},
		{
			name: "filter route load",
			configure: func(repo *testNotificationRepo) {
				repo.filterListErr = errors.New("database unavailable")
			},
		},
		{
			name: "orphan cleanup",
			configure: func(repo *testNotificationRepo) {
				repo.cleanupErr = errors.New("database unavailable")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := newTestNotificationRepo(nil, nil)
			test.configure(repo)

			service := newTestService(repo)
			assert.Error(t, service.Start())
			assert.Nil(t, service.state.Load())
		})
	}
}

func TestServiceConcurrentRoutingUpdates(t *testing.T) {
	ctx := t.Context()
	pushApproved := domain.NotificationEventPushApproved
	notification := validNotification(1, "webhook", pushApproved)
	repo := newTestNotificationRepo([]domain.Notification{notification}, nil)
	service := newTestService(repo)
	err := service.Start()
	require.NoError(t, err)

	var group sync.WaitGroup
	for idx := range 25 {
		group.Add(2)
		go func(filterID int) {
			defer group.Done()
			_ = service.StoreFilterNotifications(ctx, filterID, []domain.FilterNotification{{
				FilterID:       filterID,
				NotificationID: notification.ID,
				Events:         []string{string(pushApproved)},
			}})
		}(idx + 1)
		go func() {
			defer group.Done()
			for filterID := 1; filterID <= 25; filterID++ {
				service.currentSnapshot().resolve(pushApproved, filterID)
			}
		}()
	}
	group.Wait()
}

func TestServiceBuiltinNotification(t *testing.T) {
	ctx := t.Context()
	pushError := domain.NotificationEventPushError

	// Mirrors the row the migrations seed.
	builtinNotification := domain.Notification{
		ID:      1,
		Name:    "Built-in",
		Type:    domain.NotificationTypeBuiltin,
		Enabled: true,
		Events:  []string{string(pushError), string(domain.NotificationEventIRCDisconnected), string(domain.NotificationEventAppUpdateAvailable)},
	}

	t.Run("can not be stored, deleted or change type", func(t *testing.T) {
		repo := newTestNotificationRepo([]domain.Notification{builtinNotification}, nil)
		service := newTestService(repo)
		require.NoError(t, service.Start())

		builtin := builtinNotification
		another := domain.Notification{Name: "Another", Type: domain.NotificationTypeBuiltin, Enabled: true}

		assert.ErrorIs(t, service.Store(ctx, &another), domain.ErrNotificationBuiltin)
		assert.ErrorIs(t, service.Delete(ctx, builtin.ID), domain.ErrNotificationBuiltin)

		changed := builtin
		changed.Type = domain.NotificationTypeWebhook
		assert.ErrorIs(t, service.Update(ctx, &changed), domain.ErrNotificationBuiltin)

		webhook := validNotification(0, "webhook")
		require.NoError(t, service.Store(ctx, &webhook))
		webhook.Type = domain.NotificationTypeBuiltin
		assert.ErrorIs(t, service.Update(ctx, &webhook), domain.ErrNotificationBuiltin)

		builtin.Events = []string{string(domain.NotificationEventPushRejected)}
		require.NoError(t, service.Update(ctx, &builtin))
		assert.Len(t, service.currentSnapshot().resolve(domain.NotificationEventPushRejected, 0), 1)
	})

	t.Run("send stores and publishes inbox message", func(t *testing.T) {
		inbox := &testInboxRepo{}
		publisher := &testPublisher{}
		service := NewService(zerolog.Nop(), testEventBus{}, publisher, newTestNotificationRepo([]domain.Notification{builtinNotification}, nil), inbox, testScheduler{})
		require.NoError(t, service.Start())

		senders := service.currentSnapshot().resolve(pushError, 0)
		require.Len(t, senders, 1)

		require.NoError(t, senders[0].Send(ctx, domain.NotificationPayload{
			Event:        pushError,
			Message:      "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
			ReleaseName:  "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP",
			Filter:       "TV",
			FilterID:     4,
			ActionClient: "Sonarr",
			Rejections:   []string{"error pushing to client"},
		}))

		inbox.mu.Lock()
		messages := append([]*domain.InboxMessage(nil), inbox.messages...)
		inbox.mu.Unlock()

		require.Len(t, messages, 1)
		msg := messages[0]
		assert.Equal(t, "Push Error", msg.Title)
		assert.Empty(t, msg.Message)
		assert.Equal(t, "Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP", msg.ReleaseName)
		assert.Equal(t, "TV", msg.FilterName)
		assert.Equal(t, 4, msg.FilterID)
		assert.Equal(t, "Sonarr", msg.ActionClient)
		assert.Equal(t, []string{"error pushing to client"}, msg.Rejections)

		publisher.mu.Lock()
		published := append([]*sse.Event(nil), publisher.events...)
		publisher.mu.Unlock()

		require.Len(t, published, 1)
		assert.Equal(t, "NOTIFICATION", string(published[0].Event))
	})

	t.Run("inbox changes publish change event", func(t *testing.T) {
		inbox := &testInboxRepo{}
		publisher := &testPublisher{}
		service := NewService(zerolog.Nop(), testEventBus{}, publisher, newTestNotificationRepo(nil, nil), inbox, testScheduler{})

		require.NoError(t, service.MarkInboxRead(ctx, []int64{1}))
		require.NoError(t, service.DeleteInboxMessages(ctx, []int64{1}))
		require.NoError(t, service.DeleteInbox(ctx))

		job := &InboxCleanupJob{log: zerolog.Nop(), repo: inbox, sse: publisher}
		job.Run()

		inbox.cleaned = 2
		job.Run()

		publisher.mu.Lock()
		published := append([]*sse.Event(nil), publisher.events...)
		publisher.mu.Unlock()

		require.Len(t, published, 4)
		for _, event := range published {
			assert.Equal(t, "INBOX_CHANGED", string(event.Event))
			assert.NotEmpty(t, event.Data)
		}
	})
}
