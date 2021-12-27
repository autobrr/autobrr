package filter

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"
)

type Service interface {
	FindByID(filterID int) (*domain.Filter, error)
	FindByIndexerIdentifier(indexer string) ([]domain.Filter, error)
	FindAndCheckFilters(release *domain.Release) (bool, *domain.Filter, error)
	ListFilters() ([]domain.Filter, error)
	Store(filter domain.Filter) (*domain.Filter, error)
	Update(ctx context.Context, filter domain.Filter) (*domain.Filter, error)
	Delete(ctx context.Context, filterID int) error
}

type service struct {
	repo       domain.FilterRepo
	actionRepo domain.ActionRepo
	indexerSvc indexer.Service
}

func NewService(repo domain.FilterRepo, actionRepo domain.ActionRepo, indexerSvc indexer.Service) Service {
	return &service{
		repo:       repo,
		actionRepo: actionRepo,
		indexerSvc: indexerSvc,
	}
}

func (s *service) ListFilters() ([]domain.Filter, error) {
	// get filters
	filters, err := s.repo.ListFilters()
	if err != nil {
		return nil, err
	}

	var ret []domain.Filter

	for _, filter := range filters {
		indexers, err := s.indexerSvc.FindByFilterID(filter.ID)
		if err != nil {
			return nil, err
		}
		filter.Indexers = indexers

		ret = append(ret, filter)
	}

	return ret, nil
}

func (s *service) FindByID(filterID int) (*domain.Filter, error) {
	// find filter
	filter, err := s.repo.FindByID(filterID)
	if err != nil {
		return nil, err
	}

	// find actions and attach
	actions, err := s.actionRepo.FindByFilterID(filter.ID)
	if err != nil {
		log.Error().Msgf("could not find filter actions: %+v", &filter.ID)
	}
	filter.Actions = actions

	// find indexers and attach
	indexers, err := s.indexerSvc.FindByFilterID(filter.ID)
	if err != nil {
		log.Error().Err(err).Msgf("could not find indexers for filter: %+v", &filter.Name)
		return nil, err
	}
	filter.Indexers = indexers

	return filter, nil
}

func (s *service) FindByIndexerIdentifier(indexer string) ([]domain.Filter, error) {
	// get filters for indexer
	filters, err := s.repo.FindByIndexerIdentifier(indexer)
	if err != nil {
		log.Error().Err(err).Msgf("could not find filters for indexer: %v", indexer)
		return nil, err
	}

	return filters, nil
}

func (s *service) Store(filter domain.Filter) (*domain.Filter, error) {
	// validate data

	// store
	f, err := s.repo.Store(filter)
	if err != nil {
		log.Error().Err(err).Msgf("could not store filter: %v", filter)
		return nil, err
	}

	return f, nil
}

func (s *service) Update(ctx context.Context, filter domain.Filter) (*domain.Filter, error) {
	// validate data
	if filter.Name == "" {
		return nil, errors.New("validation: name can't be empty")
	}

	// update
	f, err := s.repo.Update(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msgf("could not update filter: %v", filter.Name)
		return nil, err
	}

	// take care of connected indexers
	if err = s.repo.StoreIndexerConnections(ctx, f.ID, filter.Indexers); err != nil {
		log.Error().Err(err).Msgf("could not store filter indexer connections: %v", filter.Name)
		return nil, err
	}

	// take care of filter actions
	actions, err := s.actionRepo.StoreFilterActions(ctx, filter.Actions, int64(filter.ID))
	if err != nil {
		log.Error().Err(err).Msgf("could not store filter actions: %v", filter.Name)
		return nil, err
	}

	f.Actions = actions

	return f, nil
}

func (s *service) Delete(ctx context.Context, filterID int) error {
	if filterID == 0 {
		return nil
	}

	// take care of filter actions
	if err := s.actionRepo.DeleteByFilterID(ctx, filterID); err != nil {
		log.Error().Err(err).Msg("could not delete filter actions")
		return err
	}

	// take care of filter indexers
	if err := s.repo.DeleteIndexerConnections(ctx, filterID); err != nil {
		log.Error().Err(err).Msg("could not delete filter indexers")
		return err
	}

	// delete filter
	if err := s.repo.Delete(ctx, filterID); err != nil {
		log.Error().Err(err).Msgf("could not delete filter: %v", filterID)
		return err
	}

	return nil
}

func (s *service) FindAndCheckFilters(release *domain.Release) (bool, *domain.Filter, error) {

	filters, err := s.repo.FindByIndexerIdentifier(release.Indexer)
	if err != nil {
		log.Error().Err(err).Msgf("could not find filters for indexer: %v", release.Indexer)
		return false, nil, err
	}

	// loop and check release to filter until match
	for _, f := range filters {
		log.Trace().Msgf("checking filter: %+v", f.Name)

		matchedFilter := release.CheckFilter(f)
		// if matched, attach actions and return the f
		if matchedFilter {
			//release.Filter = &f
			//release.FilterID = f.ID
			//release.FilterName = f.Name

			log.Debug().Msgf("found and matched filter: %+v", f.Name)

			// TODO do additional size check against indexer api or torrent for size
			if release.AdditionalSizeCheckRequired {
				log.Debug().Msgf("additional size check required for: %+v", f.Name)
				// check if indexer = btn,ptp,ggn,red
				// fetch api for data
				// else download torrent and add to tmpPath
				// if size != response.size
				// r.RecheckSizeFilter(f)
				//continue
			}

			// find actions and attach
			actions, err := s.actionRepo.FindByFilterID(f.ID)
			if err != nil {
				log.Error().Err(err).Msgf("could not find actions for filter: %+v", f.Name)
			}
			f.Actions = actions

			return true, &f, nil
		}
	}

	// if no match, return nil
	return false, nil, nil
}
