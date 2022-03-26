package filter

import (
	"context"
	"errors"
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"
)

type Service interface {
	FindByID(ctx context.Context, filterID int) (*domain.Filter, error)
	FindByIndexerIdentifier(indexer string) ([]domain.Filter, error)
	FindAndCheckFilters(release *domain.Release) (bool, *domain.Filter, error)
	ListFilters(ctx context.Context) ([]domain.Filter, error)
	Store(ctx context.Context, filter domain.Filter) (*domain.Filter, error)
	Update(ctx context.Context, filter domain.Filter) (*domain.Filter, error)
	Duplicate(ctx context.Context, filterID int) (*domain.Filter, error)
	ToggleEnabled(ctx context.Context, filterID int, enabled bool) error
	Delete(ctx context.Context, filterID int) error
}

type service struct {
	repo       domain.FilterRepo
	actionRepo domain.ActionRepo
	indexerSvc indexer.Service
	apiService indexer.APIService
}

func NewService(repo domain.FilterRepo, actionRepo domain.ActionRepo, apiService indexer.APIService, indexerSvc indexer.Service) Service {
	return &service{
		repo:       repo,
		actionRepo: actionRepo,
		apiService: apiService,
		indexerSvc: indexerSvc,
	}
}

func (s *service) ListFilters(ctx context.Context) ([]domain.Filter, error) {
	// get filters
	filters, err := s.repo.ListFilters(ctx)
	if err != nil {
		return nil, err
	}

	var ret []domain.Filter

	for _, filter := range filters {
		indexers, err := s.indexerSvc.FindByFilterID(ctx, filter.ID)
		if err != nil {
			return nil, err
		}
		filter.Indexers = indexers

		ret = append(ret, filter)
	}

	return ret, nil
}

func (s *service) FindByID(ctx context.Context, filterID int) (*domain.Filter, error) {
	// find filter
	filter, err := s.repo.FindByID(ctx, filterID)
	if err != nil {
		return nil, err
	}

	// find actions and attach
	actions, err := s.actionRepo.FindByFilterID(ctx, filter.ID)
	if err != nil {
		log.Error().Msgf("could not find filter actions: %+v", &filter.ID)
	}
	filter.Actions = actions

	// find indexers and attach
	indexers, err := s.indexerSvc.FindByFilterID(ctx, filter.ID)
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

func (s *service) Store(ctx context.Context, filter domain.Filter) (*domain.Filter, error) {
	// validate data

	// store
	f, err := s.repo.Store(ctx, filter)
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

func (s *service) Duplicate(ctx context.Context, filterID int) (*domain.Filter, error) {
	// find filter
	baseFilter, err := s.repo.FindByID(ctx, filterID)
	if err != nil {
		return nil, err
	}
	baseFilter.ID = 0
	baseFilter.Name = fmt.Sprintf("%v Copy", baseFilter.Name)
	baseFilter.Enabled = false

	// find actions and attach
	filterActions, err := s.actionRepo.FindByFilterID(ctx, filterID)
	if err != nil {
		log.Error().Msgf("could not find filter actions: %+v", &filterID)
		return nil, err
	}

	// find indexers and attach
	filterIndexers, err := s.indexerSvc.FindByFilterID(ctx, filterID)
	if err != nil {
		log.Error().Err(err).Msgf("could not find indexers for filter: %+v", &baseFilter.Name)
		return nil, err
	}

	// update
	filter, err := s.repo.Store(ctx, *baseFilter)
	if err != nil {
		log.Error().Err(err).Msgf("could not update filter: %v", baseFilter.Name)
		return nil, err
	}

	// take care of connected indexers
	if err = s.repo.StoreIndexerConnections(ctx, filter.ID, filterIndexers); err != nil {
		log.Error().Err(err).Msgf("could not store filter indexer connections: %v", filter.Name)
		return nil, err
	}
	filter.Indexers = filterIndexers

	// take care of filter actions
	actions, err := s.actionRepo.StoreFilterActions(ctx, filterActions, int64(filter.ID))
	if err != nil {
		log.Error().Err(err).Msgf("could not store filter actions: %v", filter.Name)
		return nil, err
	}

	filter.Actions = actions

	return filter, nil
}

func (s *service) ToggleEnabled(ctx context.Context, filterID int, enabled bool) error {
	if err := s.repo.ToggleEnabled(ctx, filterID, enabled); err != nil {
		log.Error().Err(err).Msg("could not update filter enabled")
		return err
	}

	log.Debug().Msgf("filter.toggle_enabled: update filter '%v' to '%v'", filterID, enabled)

	return nil
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
	// find all enabled filters for indexer
	filters, err := s.repo.FindByIndexerIdentifier(release.Indexer)
	if err != nil {
		log.Error().Err(err).Msgf("filter-service.find_and_check_filters: could not find filters for indexer: %v", release.Indexer)
		return false, nil, err
	}

	log.Trace().Msgf("filter-service.find_and_check_filters: found (%d) active filters to check for indexer '%v'", len(filters), release.Indexer)

	// save outside of loop to check multiple filters with only one fetch
	var torrentInfo *domain.TorrentBasic

	// loop and check release to filter until match
	for _, f := range filters {
		log.Trace().Msgf("filter-service.find_and_check_filters: checking filter: %+v", f.Name)

		matchedFilter := release.CheckFilter(f)
		if matchedFilter {
			// if matched, do additional size check if needed, attach actions and return the filter

			log.Debug().Msgf("filter-service.find_and_check_filters: found and matched filter: %+v", f.Name)

			// Some indexers do not announce the size and if size (min,max) is set in a filter then it will need
			// additional size check. Some indexers have api implemented to fetch this data and for the others
			// it will download the torrent file to parse and make the size check. This is all to minimize the amount of downloads.

			// do additional size check against indexer api or torrent for size
			if release.AdditionalSizeCheckRequired {
				log.Debug().Msgf("filter-service.find_and_check_filters: (%v) additional size check required", f.Name)

				// check if indexer = btn, ptp, ggn or red
				if release.Indexer == "ptp" || release.Indexer == "btn" || release.Indexer == "ggn" || release.Indexer == "redacted" {
					// fetch torrent info from api
					// save outside of loop to check multiple filters with only one fetch
					if torrentInfo == nil {
						torrentInfo, err = s.apiService.GetTorrentByID(release.Indexer, release.TorrentID)
						if err != nil || torrentInfo == nil {
							log.Error().Stack().Err(err).Msgf("filter-service.find_and_check_filters: (%v) could not get torrent: '%v' from: %v", f.Name, release.TorrentID, release.Indexer)
							continue
						}

						log.Debug().Msgf("filter-service.find_and_check_filters: (%v) got torrent info: %+v", f.Name, torrentInfo)
					}

					// compare size against filters
					match, err := checkSizeFilter(f.MinSize, f.MaxSize, torrentInfo.ReleaseSizeBytes())
					if err != nil {
						log.Error().Stack().Err(err).Msgf("filter-service.find_and_check_filters: (%v) could not check size filter", f.Name)
						continue
					}

					// no match, lets continue to next filter
					if !match {
						log.Debug().Msgf("filter-service.find_and_check_filters: (%v) filter did not match after additional size check, trying next", f.Name)
						continue
					}

					// store size on the release
					release.Size = torrentInfo.ReleaseSizeBytes()
				} else {
					log.Trace().Msgf("filter-service.find_and_check_filters: (%v) additional size check required: preparing to download metafile", f.Name)

					// if indexer doesn't have api, download torrent and add to tmpPath
					err = release.DownloadTorrentFile()
					if err != nil {
						log.Error().Stack().Err(err).Msgf("filter-service.find_and_check_filters: (%v) could not download torrent file with id: '%v' from: %v", f.Name, release.TorrentID, release.Indexer)
						return false, nil, err
					}

					// compare size against filter
					match, err := checkSizeFilter(f.MinSize, f.MaxSize, release.Size)
					if err != nil {
						log.Error().Stack().Err(err).Msgf("filter-service.find_and_check_filters: (%v) could not check size filter", f.Name)
						continue
					}

					// no match, lets continue to next filter
					if !match {
						log.Debug().Msgf("filter-service.find_and_check_filters: (%v) filter did not match after additional size check, trying next", f.Name)
						continue
					}
				}
			}

			// found matching filter, lets find the filter actions and attach
			actions, err := s.actionRepo.FindByFilterID(context.TODO(), f.ID)
			if err != nil {
				log.Error().Err(err).Msgf("could not find actions for filter: %+v", f.Name)
			}

			// if no actions, continue to next filter
			if len(actions) == 0 {
				log.Trace().Msgf("filter-service.find_and_check_filters: no actions found for filter '%v', trying next one..", f.Name)
				continue
			}
			f.Actions = actions

			return true, &f, nil
		}
	}

	// if no match, return nil
	return false, nil, nil
}

func checkSizeFilter(minSize string, maxSize string, releaseSize uint64) (bool, error) {
	// handle both min and max
	if minSize != "" {
		// string to bytes
		minSizeBytes, err := humanize.ParseBytes(minSize)
		if err != nil {
			// log could not parse into bytes
		}

		if releaseSize <= minSizeBytes {
			//r.addRejection("size: smaller than min size")
			return false, nil
		}

	}

	if maxSize != "" {
		// string to bytes
		maxSizeBytes, err := humanize.ParseBytes(maxSize)
		if err != nil {
			// log could not parse into bytes
		}

		if releaseSize >= maxSizeBytes {
			//r.addRejection("size: larger than max size")
			return false, nil
		}
	}

	return true, nil
}
