package notification

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
)

type Service interface {
	Find(ctx context.Context, params domain.NotificationQueryParams) ([]domain.Notification, int, error)
	FindByID(ctx context.Context, id int) (*domain.Notification, error)
	Store(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	Update(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	Delete(ctx context.Context, id int) error
	Send(event domain.NotificationEvent, payload domain.NotificationPayload) error
	Test(ctx context.Context, notification domain.Notification) error
}

type service struct {
	log     logger.Logger
	repo    domain.NotificationRepo
	senders []domain.NotificationSender
}

func NewService(log logger.Logger, repo domain.NotificationRepo) Service {
	s := &service{
		log:     log,
		repo:    repo,
		senders: []domain.NotificationSender{},
	}

	s.registerSenders()

	return s
}

func (s *service) Find(ctx context.Context, params domain.NotificationQueryParams) ([]domain.Notification, int, error) {
	return s.repo.Find(ctx, params)
}

func (s *service) FindByID(ctx context.Context, id int) (*domain.Notification, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Store(ctx context.Context, n domain.Notification) (*domain.Notification, error) {
	return s.repo.Store(ctx, n)
}

func (s *service) Update(ctx context.Context, n domain.Notification) (*domain.Notification, error) {
	return s.repo.Update(ctx, n)
}

func (s *service) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) registerSenders() {
	senders, err := s.repo.List(context.Background())
	if err != nil {
		return
	}

	for _, n := range senders {
		if n.Enabled {
			switch n.Type {
			case domain.NotificationTypeDiscord:
				s.senders = append(s.senders, NewDiscordSender(s.log, n))
			case domain.NotificationTypeTelegram:
				s.senders = append(s.senders, NewTelegramSender(s.log, n))
			}
		}
	}

	return
}

// Send notifications
func (s *service) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	s.log.Debug().Msgf("sending notification for %v", string(event))

	for _, sender := range s.senders {
		// check if sender is active and have notification types
		if sender.CanSend(event) {
			sender.Send(event, payload)
		}
	}

	return nil
}

func (s *service) Test(ctx context.Context, notification domain.Notification) error {
	var agent domain.NotificationSender

	switch notification.Type {
	case domain.NotificationTypeDiscord:
		agent = NewDiscordSender(s.log, notification)
	case domain.NotificationTypeTelegram:
		agent = NewTelegramSender(s.log, notification)
	}

	return agent.Send(domain.NotificationEventTest, domain.NotificationPayload{
		Subject: "Test Notification",
		Message: "autobrr goes brr!!",
	})
}
