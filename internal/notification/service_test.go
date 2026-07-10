package notification

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
)

type mockSender struct {
	mock.Mock
}

func (m *mockSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	args := m.Called(event, payload)
	return args.Error(0)
}

func (m *mockSender) CanSend(event domain.NotificationEvent) bool {
	args := m.Called(event)
	return args.Bool(0)
}

func (m *mockSender) CanSendPayload(event domain.NotificationEvent, payload domain.NotificationPayload) bool {
	args := m.Called(event, payload)
	return args.Bool(0)
}

func (m *mockSender) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockSender) Name() string {
	return "mock"
}

func (m *mockSender) HasFilterEvents(filterID int) bool {
	args := m.Called(filterID)
	return args.Bool(0)
}

func TestService_Send_Optimization(t *testing.T) {
	log := zerolog.Nop()

	t.Run("should not spawn goroutine if no senders interested", func(t *testing.T) {
		sender := new(mockSender)
		svc := &Service{
			log: log,
			senders: map[int]Sender{
				1: sender,
			},
		}

		event := domain.NotificationEventReleaseNew
		payload := domain.NotificationPayload{Event: event}

		// Configure mock to say it's NOT interested
		sender.On("CanSendPayload", event, payload).Return(false)

		svc.Send(event, payload)

		// Wait a bit to ensure no goroutine work happened
		time.Sleep(50 * time.Millisecond)

		sender.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
	})

	t.Run("should send if sender is interested", func(t *testing.T) {
		sender := new(mockSender)
		svc := &Service{
			log: log,
			senders: map[int]Sender{
				1: sender,
			},
		}

		event := domain.NotificationEventReleaseNew
		payload := domain.NotificationPayload{Event: event}

		// Configure mock to say it IS interested
		sender.On("CanSendPayload", event, payload).Return(true)
		sender.On("Send", event, payload).Return(nil)

		svc.Send(event, payload)

		// Wait for goroutine
		time.Sleep(100 * time.Millisecond)

		sender.AssertExpectations(t)
	})

	// Regression for #2235/#2417: a per-filter override (or mute) on one
	// notification must not suppress another notification. Every sender is
	// evaluated independently via CanSendPayload; there is no set-wide gate.
	t.Run("filter event evaluates every sender independently", func(t *testing.T) {
		muted := new(mockSender)  // has a (muted) per-filter row -> declines
		global := new(mockSender) // no per-filter config -> global fallback fires

		svc := &Service{
			log: log,
			senders: map[int]Sender{
				1: muted,
				2: global,
			},
		}

		event := domain.NotificationEventPushApproved
		payload := domain.NotificationPayload{Event: event, FilterID: 42}

		muted.On("CanSendPayload", event, payload).Return(false)
		global.On("CanSendPayload", event, payload).Return(true)
		global.On("Send", event, payload).Return(nil)

		svc.Send(event, payload)

		time.Sleep(100 * time.Millisecond)

		global.AssertExpectations(t)
		muted.AssertCalled(t, "CanSendPayload", event, payload)
		muted.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
	})
}

// fakeNotificationRepo is a minimal in-memory notificationRepo for exercising
// the sender lifecycle without a database.
type fakeNotificationRepo struct {
	notifications map[int]*domain.Notification
	// filterByNotification holds filter_notification rows keyed by notification id.
	filterByNotification map[int][]domain.FilterNotification
}

func (f *fakeNotificationRepo) List(_ context.Context) ([]domain.Notification, error) {
	out := make([]domain.Notification, 0, len(f.notifications))
	for _, n := range f.notifications {
		out = append(out, *n)
	}
	return out, nil
}

func (f *fakeNotificationRepo) Find(_ context.Context, _ domain.NotificationQueryParams) ([]domain.Notification, int, error) {
	return nil, 0, nil
}

func (f *fakeNotificationRepo) FindByID(_ context.Context, id int) (*domain.Notification, error) {
	n, ok := f.notifications[id]
	if !ok {
		return nil, domain.ErrRecordNotFound
	}
	c := *n
	return &c, nil
}

func (f *fakeNotificationRepo) Store(_ context.Context, n *domain.Notification) error {
	c := *n
	f.notifications[n.ID] = &c
	return nil
}

func (f *fakeNotificationRepo) Update(_ context.Context, n *domain.Notification) error {
	c := *n
	f.notifications[n.ID] = &c
	return nil
}

func (f *fakeNotificationRepo) Delete(_ context.Context, id int) error {
	delete(f.notifications, id)
	delete(f.filterByNotification, id)
	return nil
}

func (f *fakeNotificationRepo) GetNotificationFilters(_ context.Context, notificationID int) ([]domain.FilterNotification, error) {
	return f.filterByNotification[notificationID], nil
}

func (f *fakeNotificationRepo) GetFilterNotifications(_ context.Context, filterID int) ([]domain.FilterNotification, error) {
	var out []domain.FilterNotification
	for _, list := range f.filterByNotification {
		for _, fn := range list {
			if fn.FilterID == filterID {
				out = append(out, fn)
			}
		}
	}
	return out, nil
}

func (f *fakeNotificationRepo) StoreFilterNotifications(_ context.Context, filterID int, notifications []domain.FilterNotification) error {
	f.removeFilter(filterID)
	for _, fn := range notifications {
		fn.FilterID = filterID
		f.filterByNotification[fn.NotificationID] = append(f.filterByNotification[fn.NotificationID], fn)
	}
	return nil
}

func (f *fakeNotificationRepo) DeleteFilterNotifications(_ context.Context, filterID int) error {
	f.removeFilter(filterID)
	return nil
}

func (f *fakeNotificationRepo) removeFilter(filterID int) {
	for nid, list := range f.filterByNotification {
		kept := list[:0:0]
		for _, fn := range list {
			if fn.FilterID != filterID {
				kept = append(kept, fn)
			}
		}
		f.filterByNotification[nid] = kept
	}
}

func newTestService(repo notificationRepo) *Service {
	svc := &Service{
		log:           zerolog.Nop(),
		repo:          repo,
		notifications: make(map[int]*domain.Notification),
		senders:       make(map[int]Sender),
	}
	svc.registerSenders()
	return svc
}

// Regression for #2235: editing a notification must not drop its per-filter
// mutes/overrides. Before the fix, Update rebuilt the sender from a JSON object
// whose private filters map was nil, so the mute silently disappeared.
func TestService_Update_PreservesFilterMute(t *testing.T) {
	const notifID = 1
	const filterID = 42

	repo := &fakeNotificationRepo{
		notifications: map[int]*domain.Notification{
			notifID: {
				ID:      notifID,
				Name:    "Discord",
				Type:    domain.NotificationTypeDiscord,
				Enabled: true,
				Events:  []string{string(domain.NotificationEventPushApproved)},
				Webhook: "https://discord.example/webhook",
			},
		},
		filterByNotification: map[int][]domain.FilterNotification{
			notifID: {
				{FilterID: filterID, NotificationID: notifID, Events: []string{}}, // muted
			},
		},
	}

	svc := newTestService(repo)

	payload := domain.NotificationPayload{Event: domain.NotificationEventPushApproved, FilterID: filterID}

	sender, ok := svc.senders[notifID]
	if !ok {
		t.Fatal("expected sender to be registered at startup")
	}
	if sender.CanSendPayload(domain.NotificationEventPushApproved, payload) {
		t.Fatal("expected notification to be muted for filter before update")
	}

	// Simulate an HTTP PUT: a fresh object whose unexported filters map is nil.
	updated := &domain.Notification{
		ID:      notifID,
		Name:    "Discord renamed",
		Type:    domain.NotificationTypeDiscord,
		Enabled: true,
		Events:  []string{string(domain.NotificationEventPushApproved)},
		Webhook: "https://discord.example/webhook",
	}
	if err := svc.Update(context.Background(), updated); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	sender, ok = svc.senders[notifID]
	if !ok {
		t.Fatal("expected sender to remain registered after update")
	}
	if sender.CanSendPayload(domain.NotificationEventPushApproved, payload) {
		t.Fatal("mute was lost after editing the notification (regression #2235)")
	}
}

// A freshly created notification must be tracked so a later filter save can
// attach per-filter events to it (previously ok==false silently dropped them).
func TestService_StoreThenMute(t *testing.T) {
	const notifID = 7
	const filterID = 3

	repo := &fakeNotificationRepo{
		notifications:        map[int]*domain.Notification{},
		filterByNotification: map[int][]domain.FilterNotification{},
	}

	svc := newTestService(repo)

	created := &domain.Notification{
		ID:      notifID,
		Name:    "Discord",
		Type:    domain.NotificationTypeDiscord,
		Enabled: true,
		Events:  []string{string(domain.NotificationEventPushApproved)},
		Webhook: "https://discord.example/webhook",
	}
	if err := svc.Store(context.Background(), created); err != nil {
		t.Fatalf("store failed: %v", err)
	}

	payload := domain.NotificationPayload{Event: domain.NotificationEventPushApproved, FilterID: filterID}

	// Global fallback fires before any per-filter config exists.
	if !svc.senders[notifID].CanSendPayload(domain.NotificationEventPushApproved, payload) {
		t.Fatal("expected global fallback to fire for unconfigured filter")
	}

	// Now mute this notification for the filter.
	if err := svc.StoreFilterNotifications(context.Background(), filterID, []domain.FilterNotification{
		{FilterID: filterID, NotificationID: notifID, Events: []string{}},
	}); err != nil {
		t.Fatalf("store filter notifications failed: %v", err)
	}

	if svc.senders[notifID].CanSendPayload(domain.NotificationEventPushApproved, payload) {
		t.Fatal("mute did not take effect for a freshly created notification")
	}
}

// Regression for the unguarded-maps crash: concurrent Send (reads s.senders and
// the shared filters map) and config writes (rebuild both) must be race-free.
// Run with -race. A single writer keeps the in-memory fake repo access serial.
func TestService_ConcurrentSendAndConfig(t *testing.T) {
	const notifID = 1
	const filterID = 42

	newNotif := func() *domain.Notification {
		return &domain.Notification{
			ID:      notifID,
			Name:    "Discord",
			Type:    domain.NotificationTypeDiscord,
			Enabled: true,
			Events:  []string{string(domain.NotificationEventPushApproved)},
			Webhook: "https://discord.example/webhook",
		}
	}
	mutedRow := []domain.FilterNotification{{FilterID: filterID, NotificationID: notifID, Events: []string{}}}

	repo := &fakeNotificationRepo{
		notifications:        map[int]*domain.Notification{notifID: newNotif()},
		filterByNotification: map[int][]domain.FilterNotification{notifID: mutedRow},
	}
	svc := newTestService(repo)

	// PUSH_REJECTED is not enabled globally and the filter is muted, so
	// CanSendPayload returns false and no network Send is attempted - but the
	// selection loop still reads sender state, exercising the race.
	payload := domain.NotificationPayload{Event: domain.NotificationEventPushRejected, FilterID: filterID}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					svc.Send(domain.NotificationEventPushRejected, payload)
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = svc.Update(context.Background(), newNotif())
				_ = svc.StoreFilterNotifications(context.Background(), filterID, mutedRow)
			}
		}
	}()

	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()
}
