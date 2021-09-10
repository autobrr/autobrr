package filter

import (
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/pkg/wildcard"
)

type Service interface {
	//FindFilter(announce domain.Announce) (*domain.Filter, error)

	FindByID(filterID int) (*domain.Filter, error)
	FindByIndexerIdentifier(announce domain.Announce) (*domain.Filter, error)
	ListFilters() ([]domain.Filter, error)
	Store(filter domain.Filter) (*domain.Filter, error)
	Update(filter domain.Filter) (*domain.Filter, error)
	Delete(filterID int) error
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
	//actions, err := s.actionRepo.FindFilterActions(filter.ID)
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

	//log.Debug().Msgf("found filter: %+v", filter)

	return filter, nil
}

func (s *service) FindByIndexerIdentifier(announce domain.Announce) (*domain.Filter, error) {
	// get filter for tracker
	filters, err := s.repo.FindByIndexerIdentifier(announce.Site)
	if err != nil {
		log.Error().Err(err).Msgf("could not find filters for indexer: %v", announce.Site)
		return nil, err
	}

	// match against announce/releaseInfo
	for _, filter := range filters {
		// if match, return the filter
		matchedFilter := s.checkFilter(filter, announce)
		if matchedFilter {
			log.Trace().Msgf("found matching filter: %+v", &filter)
			log.Debug().Msgf("found matching filter: %v", &filter.Name)

			// find actions and attach
			actions, err := s.actionRepo.FindByFilterID(filter.ID)
			if err != nil {
				log.Error().Err(err).Msgf("could not find filter actions: %+v", &filter.ID)
				return nil, err
			}

			// if no actions found, check next filter
			if actions == nil {
				continue
			}

			filter.Actions = actions

			return &filter, nil
		}
	}

	// if no match, return nil
	return nil, nil
}

//func (s *service) FindFilter(announce domain.Announce) (*domain.Filter, error) {
//	// get filter for tracker
//	filters, err := s.repo.FindFiltersForSite(announce.Site)
//	if err != nil {
//		return nil, err
//	}
//
//	// match against announce/releaseInfo
//	for _, filter := range filters {
//		// if match, return the filter
//		matchedFilter := s.checkFilter(filter, announce)
//		if matchedFilter {
//
//			log.Debug().Msgf("found filter: %+v", &filter)
//
//			// find actions and attach
//			actions, err := s.actionRepo.FindByFilterID(filter.ID)
//			if err != nil {
//				log.Error().Msgf("could not find filter actions: %+v", &filter.ID)
//			}
//			filter.Actions = actions
//
//			return &filter, nil
//		}
//	}
//
//	// if no match, return nil
//	return nil, nil
//}

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

func (s *service) Update(filter domain.Filter) (*domain.Filter, error) {
	// validate data

	// store
	f, err := s.repo.Update(filter)
	if err != nil {
		log.Error().Err(err).Msgf("could not update filter: %v", filter.Name)
		return nil, err
	}

	// take care of connected indexers
	if err = s.repo.DeleteIndexerConnections(f.ID); err != nil {
		log.Error().Err(err).Msgf("could not delete filter indexer connections: %v", filter.Name)
		return nil, err
	}

	for _, i := range filter.Indexers {
		if err = s.repo.StoreIndexerConnection(f.ID, int(i.ID)); err != nil {
			log.Error().Err(err).Msgf("could not store filter indexer connections: %v", filter.Name)
			return nil, err
		}
	}

	return f, nil
}

func (s *service) Delete(filterID int) error {
	if filterID == 0 {
		return nil
	}

	// delete
	if err := s.repo.Delete(filterID); err != nil {
		log.Error().Err(err).Msgf("could not delete filter: %v", filterID)
		return err
	}

	return nil
}

// checkFilter tries to match filter against announce
func (s *service) checkFilter(filter domain.Filter, announce domain.Announce) bool {

	if !filter.Enabled {
		return false
	}

	if filter.Scene && announce.Scene != filter.Scene {
		return false
	}

	if filter.Freeleech && announce.Freeleech != filter.Freeleech {
		return false
	}

	if filter.Shows != "" && !checkFilterStrings(announce.TorrentName, filter.Shows) {
		return false
	}

	//if filter.Seasons != "" && !checkFilterStrings(announce.TorrentName, filter.Seasons) {
	//	return false
	//}
	//
	//if filter.Episodes != "" && !checkFilterStrings(announce.TorrentName, filter.Episodes) {
	//	return false
	//}

	// matchRelease
	if filter.MatchReleases != "" && !checkFilterStrings(announce.TorrentName, filter.MatchReleases) {
		return false
	}

	if filter.MatchReleaseGroups != "" && !checkFilterStrings(announce.TorrentName, filter.MatchReleaseGroups) {
		return false
	}

	if filter.ExceptReleaseGroups != "" && checkFilterStrings(announce.TorrentName, filter.ExceptReleaseGroups) {
		return false
	}

	if filter.MatchUploaders != "" && !checkFilterStrings(announce.Uploader, filter.MatchUploaders) {
		return false
	}

	if filter.ExceptUploaders != "" && checkFilterStrings(announce.Uploader, filter.ExceptUploaders) {
		return false
	}

	if len(filter.Resolutions) > 0 && !checkFilterSlice(announce.TorrentName, filter.Resolutions) {
		return false
	}

	if len(filter.Codecs) > 0 && !checkFilterSlice(announce.TorrentName, filter.Codecs) {
		return false
	}

	if len(filter.Sources) > 0 && !checkFilterSlice(announce.TorrentName, filter.Sources) {
		return false
	}

	if len(filter.Containers) > 0 && !checkFilterSlice(announce.TorrentName, filter.Containers) {
		return false
	}

	if filter.Years != "" && !checkFilterStrings(announce.TorrentName, filter.Years) {
		return false
	}

	if filter.MatchCategories != "" && !checkFilterStrings(announce.Category, filter.MatchCategories) {
		return false
	}

	if filter.ExceptCategories != "" && checkFilterStrings(announce.Category, filter.ExceptCategories) {
		return false
	}

	if filter.Tags != "" && !checkFilterStrings(announce.Tags, filter.Tags) {
		return false
	}

	if filter.ExceptTags != "" && checkFilterStrings(announce.Tags, filter.ExceptTags) {
		return false
	}

	return true
}

func checkFilterSlice(name string, filterList []string) bool {
	name = strings.ToLower(name)

	for _, filter := range filterList {
		filter = strings.ToLower(filter)
		// check if line contains * or ?, if so try wildcard match, otherwise try substring match
		a := strings.ContainsAny(filter, "?|*")
		if a {
			match := wildcard.Match(filter, name)
			if match {
				return true
			}
		} else {
			b := strings.Contains(name, filter)
			if b {
				return true
			}
		}
	}

	return false
}

func checkFilterStrings(name string, filterList string) bool {
	filterSplit := strings.Split(filterList, ",")
	name = strings.ToLower(name)

	for _, s := range filterSplit {
		s = strings.ToLower(s)
		// check if line contains * or ?, if so try wildcard match, otherwise try substring match
		a := strings.ContainsAny(s, "?|*")
		if a {
			match := wildcard.Match(s, name)
			if match {
				return true
			}
		} else {
			b := strings.Contains(name, s)
			if b {
				return true
			}
		}

	}

	return false
}
