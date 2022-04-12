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
	CheckFilter(f domain.Filter, release *domain.Release) (bool, error)
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

	ret := make([]domain.Filter, 0)

	for _, filter := range filters {
		indexers, err := s.indexerSvc.FindByFilterID(ctx, filter.ID)
		if err != nil {
			return ret, err
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

func (s *service) CheckFilter(f domain.Filter, release *domain.Release) (bool, error) {

	log.Trace().Msgf("filter.Service.CheckFilter: checking filter: %v %+v", f.Name, f)
	log.Trace().Msgf("filter.Service.CheckFilter: checking filter: %v for release: %+v", f.Name, release)

	rejections, matchedFilter := release.CheckFilter(f)
	if len(rejections) > 0 {
		log.Trace().Msgf("filter.Service.CheckFilter: (%v) for release: %v rejections: (%v)", f.Name, release.TorrentName, release.RejectionsString())
		return false, nil
	}

	if matchedFilter {
		// if matched, do additional size check if needed, attach actions and return the filter

		log.Debug().Msgf("filter.Service.CheckFilter: found and matched filter: %+v", f.Name)

		// Some indexers do not announce the size and if size (min,max) is set in a filter then it will need
		// additional size check. Some indexers have api implemented to fetch this data and for the others
		// it will download the torrent file to parse and make the size check. This is all to minimize the amount of downloads.

		// do additional size check against indexer api or download torrent for size check
		if release.AdditionalSizeCheckRequired {
			log.Debug().Msgf("filter.Service.CheckFilter: (%v) additional size check required", f.Name)

			ok, err := s.AdditionalSizeCheck(f, release)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("filter.Service.CheckFilter: (%v) additional size check error", f.Name)
				return false, err
			}

			if !ok {
				log.Trace().Msgf("filter.Service.CheckFilter: (%v) additional size check not matching what filter wanted", f.Name)
				return false, nil
			}
		}

		// found matching filter, lets find the filter actions and attach
		actions, err := s.actionRepo.FindByFilterID(context.TODO(), f.ID)
		if err != nil {
			log.Error().Err(err).Msgf("filter.Service.CheckFilter: error finding actions for filter: %+v", f.Name)
			return false, err
		}

		// if no actions, continue to next filter
		if len(actions) == 0 {
			log.Trace().Msgf("filter.Service.CheckFilter: no actions found for filter '%v', trying next one..", f.Name)
			return false, err
		}
		release.Filter.Actions = actions

		return true, nil
	}

	// if no match, return nil
	return false, nil
}

// AdditionalSizeCheck
// Some indexers do not announce the size and if size (min,max) is set in a filter then it will need
// additional size check. Some indexers have api implemented to fetch this data and for the others
// it will download the torrent file to parse and make the size check. This is all to minimize the amount of downloads.
func (s *service) AdditionalSizeCheck(f domain.Filter, release *domain.Release) (bool, error) {

	// do additional size check against indexer api or torrent for size
	log.Debug().Msgf("filter.Service.AdditionalSizeCheck: (%v) additional size check required", f.Name)

	switch release.Indexer {
	case "ptp", "btn", "ggn", "redacted", "mock":
		if release.Size == 0 {
			log.Trace().Msgf("filter.Service.AdditionalSizeCheck: (%v) preparing to check via api", f.Name)
			torrentInfo, err := s.apiService.GetTorrentByID(release.Indexer, release.TorrentID)
			if err != nil || torrentInfo == nil {
				log.Error().Stack().Err(err).Msgf("filter.Service.AdditionalSizeCheck: (%v) could not get torrent info from api: '%v' from: %v", f.Name, release.TorrentID, release.Indexer)
				return false, err
			}

			log.Debug().Msgf("filter.Service.AdditionalSizeCheck: (%v) got torrent info from api: %+v", f.Name, torrentInfo)

			release.Size = torrentInfo.ReleaseSizeBytes()
		}

	default:
		log.Trace().Msgf("filter.Service.AdditionalSizeCheck: (%v) preparing to download torrent metafile", f.Name)

		// if indexer doesn't have api, download torrent and add to tmpPath
		err := release.DownloadTorrentFile()
		if err != nil {
			log.Error().Stack().Err(err).Msgf("filter.Service.AdditionalSizeCheck: (%v) could not download torrent file with id: '%v' from: %v", f.Name, release.TorrentID, release.Indexer)
			return false, err
		}
	}

	// compare size against filter
	match, err := checkSizeFilter(f.MinSize, f.MaxSize, release.Size)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("filter.Service.AdditionalSizeCheck: (%v) error checking extra size filter", f.Name)
		return false, err
	}
	//no match, lets continue to next filter
	if !match {
		log.Debug().Msgf("filter.Service.AdditionalSizeCheck: (%v) filter did not match after additional size check, trying next", f.Name)
		return false, nil
	}

	return true, nil
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
