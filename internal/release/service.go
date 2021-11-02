package release

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/domain"
)

type Service interface {
	Find(ctx context.Context, query domain.QueryParams) (res []domain.Release, nextCursor int64, err error)
	Store(release domain.Release) error
	Process(announce domain.Announce) error
}

type service struct {
	repo      domain.ReleaseRepo
	actionSvc action.Service
}

func NewService(repo domain.ReleaseRepo, actionService action.Service) Service {
	return &service{
		repo:      repo,
		actionSvc: actionService,
	}
}

func (s *service) Find(ctx context.Context, query domain.QueryParams) (res []domain.Release, nextCursor int64, err error) {
	//releases, err := s.repo.Find(ctx, query)
	res, nextCursor, err = s.repo.Find(ctx, query)
	if err != nil {
		//return nil, err
		return
	}
	return

	//return releases, nil
}

func (s *service) Store(release domain.Release) error {

	return nil
}

func (s *service) Process(announce domain.Announce) error {
	log.Trace().Msgf("start to process release: %+v", announce)

	if announce.Filter.Actions == nil {
		return fmt.Errorf("no actions for filter: %v", announce.Filter.Name)
	}

	// smart episode?

	// run actions (watchFolder, test, exec, qBittorrent, Deluge etc.)
	err := s.actionSvc.RunActions(announce.Filter.Actions, announce)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error running actions for filter: %v", announce.Filter.Name)
		return err
	}

	return nil
}
