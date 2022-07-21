package notification

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/rs/zerolog"
)

type Service interface {
	Find(ctx context.Context, params domain.NotificationQueryParams) ([]domain.Notification, int, error)
	FindByID(ctx context.Context, id int) (*domain.Notification, error)
	Store(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	Update(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	Delete(ctx context.Context, id int) error
	Send(event domain.NotificationEvent, payload domain.NotificationPayload)
	Test(ctx context.Context, notification domain.Notification) error
}

type service struct {
	log     zerolog.Logger
	repo    domain.NotificationRepo
	senders []domain.NotificationSender
}

func NewService(log logger.Logger, repo domain.NotificationRepo) Service {
	s := &service{
		log:     log.With().Str("module", "notification").Logger(),
		repo:    repo,
		senders: []domain.NotificationSender{},
	}

	s.registerSenders()

	return s
}

func (s *service) Find(ctx context.Context, params domain.NotificationQueryParams) ([]domain.Notification, int, error) {
	n, count, err := s.repo.Find(ctx, params)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find notification with params: %+v", params)
		return nil, 0, err
	}

	return n, count, err
}

func (s *service) FindByID(ctx context.Context, id int) (*domain.Notification, error) {
	n, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find notification by id: %v", id)
		return nil, err
	}

	return n, err
}

func (s *service) Store(ctx context.Context, n domain.Notification) (*domain.Notification, error) {
	_, err := s.repo.Store(ctx, n)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not store notification: %+v", n)
		return nil, err
	}

	// reset senders
	s.senders = []domain.NotificationSender{}

	// re register senders
	s.registerSenders()

	return nil, nil
}

func (s *service) Update(ctx context.Context, n domain.Notification) (*domain.Notification, error) {
	_, err := s.repo.Update(ctx, n)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not update notification: %+v", n)
		return nil, err
	}

	// reset senders
	s.senders = []domain.NotificationSender{}

	// re register senders
	s.registerSenders()

	return nil, nil
}

func (s *service) Delete(ctx context.Context, id int) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not delete notification: %v", id)
		return err
	}

	// reset senders
	s.senders = []domain.NotificationSender{}

	// re register senders
	s.registerSenders()

	return nil
}

func (s *service) registerSenders() {
	senders, err := s.repo.List(context.Background())
	if err != nil {
		s.log.Error().Err(err).Msg("could not find notifications")
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
func (s *service) Send(event domain.NotificationEvent, payload domain.NotificationPayload) {
	if len(s.senders) > 0 {
		s.log.Debug().Msgf("sending notification for %v", string(event))
	}

	go func() {
		for _, sender := range s.senders {
			// check if sender is active and have notification types
			if sender.CanSend(event) {
				sender.Send(event, payload)
			}
		}
	}()

	return
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
