package notification

import (
	"context"
	"fmt"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/containrrr/shoutrrr"
	t "github.com/containrrr/shoutrrr/pkg/types"
)

type Service interface {
	Find(ctx context.Context, params domain.NotificationQueryParams) ([]domain.Notification, int, error)
	FindByID(ctx context.Context, id int) (*domain.Notification, error)
	Store(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	Update(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	Delete(ctx context.Context, id int) error
	Send(event domain.NotificationEvent, msg string) error
	SendEvent(event domain.EventsReleasePushed) error
}

type service struct {
	log  logger.Logger
	repo domain.NotificationRepo
}

func NewService(log logger.Logger, repo domain.NotificationRepo) Service {
	return &service{
		log:  log,
		repo: repo,
	}
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

// Send notifications
func (s *service) Send(event domain.NotificationEvent, msg string) error {
	// find notifications for type X

	notifications, err := s.repo.List(context.Background())
	if err != nil {
		return err
	}

	var urls []string

	for _, n := range notifications {
		if !n.Enabled {
			continue
		}

		switch n.Type {
		case domain.NotificationTypeDiscord:
			urls = append(urls, fmt.Sprintf("discord://%v@%v", n.Token, n.Webhook))
		default:
			return nil
		}
	}

	if len(urls) == 0 {
		return nil
	}

	sender, err := shoutrrr.CreateSender(urls...)
	if err != nil {
		return err
	}

	p := t.Params{"title": "TEST"}
	items := []t.MessageItem{
		{
			Text: "text hello",
			Fields: []t.Field{
				{
					Key:   "eventt",
					Value: "push?",
				},
			},
		},
	}
	//items = append(items, t.MessageItem{
	//	Text: "text hello",
	//	Fields: []t.Field{
	//		{
	//			Key:   "eventt",
	//			Value: "push?",
	//		},
	//	},
	//})

	sender.SendItems(items, p)

	return nil
}

func (s *service) SendEvent(event domain.EventsReleasePushed) error {
	notifications, err := s.repo.List(context.Background())
	if err != nil {
		return err
	}

	return s.send(notifications, event)
}

func (s *service) send(notifications []domain.Notification, event domain.EventsReleasePushed) error {
	// find notifications for type X
	for _, n := range notifications {
		if !n.Enabled {
			continue
		}

		if n.Events == nil {
			continue
		}

		for _, evt := range n.Events {
			if evt == string(event.Status) {
				switch n.Type {
				case domain.NotificationTypeDiscord:
					go s.discordNotification(event, n.Webhook)
				default:
					return nil
				}
			}
		}

	}

	return nil
}
