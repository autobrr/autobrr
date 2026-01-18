package notification

import (
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
			senders: map[int]domain.NotificationSender{
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
			senders: map[int]domain.NotificationSender{
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
}
