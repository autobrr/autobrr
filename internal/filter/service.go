// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package filter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/utils"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/Hellseher/go-shellquote"
	"github.com/avast/retry-go/v4"
	"github.com/rs/zerolog"
)

type filterRepo interface {
	ListFilters(ctx context.Context) ([]domain.Filter, error)
	Find(ctx context.Context, params domain.FilterQueryParams) ([]*domain.Filter, error)
	FindByID(ctx context.Context, filterID int) (*domain.Filter, error)
	FindByIndexerIdentifier(ctx context.Context, indexer string) ([]*domain.Filter, error)
	FindExternalFiltersByID(ctx context.Context, filterId int) ([]domain.FilterExternal, error)
	FindExternalFiltersByFilterIDs(ctx context.Context, filterIDs []int) (map[int][]domain.FilterExternal, error)
	Store(ctx context.Context, filter *domain.Filter) error
	Update(ctx context.Context, filter *domain.Filter) error
	UpdatePartial(ctx context.Context, filter domain.FilterUpdate) error
	ToggleEnabled(ctx context.Context, filterID int, enabled bool) error
	Delete(ctx context.Context, filterID int) error
	StoreIndexerConnection(ctx context.Context, filterID int, indexerID int) error
	StoreIndexerConnections(ctx context.Context, filterID int, indexers []domain.Indexer) error
	StoreFilterExternal(ctx context.Context, filterID int, externalFilters []domain.FilterExternal) error
	DeleteIndexerConnections(ctx context.Context, filterID int) error
	DeleteFilterExternal(ctx context.Context, filterID int) error
	GetFilterDownloadCount(ctx context.Context, filter *domain.Filter) error
	GetFilterNotifications(ctx context.Context, filterID int) ([]domain.FilterNotification, error)
	StoreFilterNotifications(ctx context.Context, filterID int, notifications []domain.FilterNotification) error
	DeleteFilterNotifications(ctx context.Context, filterID int) error
}

type actionService interface {
	StoreFilterActions(ctx context.Context, filterID int64, actions []*domain.Action) ([]*domain.Action, error)
	FindByFilterID(ctx context.Context, filterID int, active *bool, withClient bool) ([]*domain.Action, error)
	DeleteByFilterID(ctx context.Context, filterID int) error
}

type indexerService interface {
	FindByFilterID(ctx context.Context, filterID int) ([]domain.Indexer, error)
	List(ctx context.Context) ([]domain.Indexer, error)
}

type indexerAPIService interface {
	GetTorrentByID(ctx context.Context, indexer string, torrentID string) (*domain.TorrentBasic, error)
}

type downloadService interface {
	DownloadRelease(ctx context.Context, rls *domain.Release) error
}

type notificationService interface {
	GetFilterNotifications(ctx context.Context, filterID int) ([]domain.FilterNotification, error)
	StoreFilterNotifications(ctx context.Context, filterID int, notifications []domain.FilterNotification) error
	DeleteFilterNotifications(ctx context.Context, filterID int) error
}

type releaseRepo interface {
	CheckSmartEpisodeCanDownload(ctx context.Context, p *domain.SmartEpisodeParams) (bool, error)
	CheckIsDuplicateRelease(ctx context.Context, profile *domain.DuplicateReleaseProfile, release *domain.Release) (bool, error)
}

type Service struct {
	log             zerolog.Logger
	repo            filterRepo
	actionService   actionService
	releaseRepo     releaseRepo
	indexerSvc      indexerService
	apiService      indexerAPIService
	downloadSvc     downloadService
	notificationSvc notificationService

	httpClient *http.Client
}

func NewService(log zerolog.Logger, repo filterRepo, actionSvc actionService, releaseRepo releaseRepo, apiService indexerAPIService, indexerSvc indexerService, downloadSvc downloadService, notificationSvc notificationService) *Service {
	return &Service{
		log:             log.With().Str("module", "filter").Logger(),
		repo:            repo,
		releaseRepo:     releaseRepo,
		actionService:   actionSvc,
		apiService:      apiService,
		indexerSvc:      indexerSvc,
		downloadSvc:     downloadSvc,
		notificationSvc: notificationSvc,
		httpClient: &http.Client{
			Timeout:   time.Second * 120,
			Transport: sharedhttp.TransportTLSInsecure,
		},
	}
}

func (s *Service) Find(ctx context.Context, params domain.FilterQueryParams) ([]*domain.Filter, error) {
	// get filters
	filters, err := s.repo.Find(ctx, params)
	if err != nil {
		s.log.Error().Err(err).Msg("could not find list filters")
		return nil, err
	}

	for _, filter := range filters {
		indexers, err := s.indexerSvc.FindByFilterID(ctx, filter.ID)
		if err != nil {
			return filters, err
		}
		filter.Indexers = indexers

		if filter.IsMaxDownloadsLimitEnabled() {
			if err := s.repo.GetFilterDownloadCount(ctx, filter); err != nil {
				s.log.Error().Err(err).Str("filter", filter.Name).Msg("could not get filter downloads")
			}
		}
	}

	return filters, nil
}

func (s *Service) ListFilters(ctx context.Context) ([]domain.Filter, error) {
	// get filters
	filters, err := s.repo.ListFilters(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("could not find list filters")
		return nil, err
	}

	for idx, filter := range filters {
		indexers, err := s.indexerSvc.FindByFilterID(ctx, filter.ID)
		if err != nil {
			return filters, err
		}
		filters[idx].Indexers = indexers
	}

	return filters, nil
}

func (s *Service) FindByID(ctx context.Context, filterID int) (*domain.Filter, error) {
	filter, err := s.repo.FindByID(ctx, filterID)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filterID).Msg("could not find filter")
		return nil, err
	}

	externalFilters, err := s.repo.FindExternalFiltersByID(ctx, filter.ID)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not find external filters")
	}
	filter.External = externalFilters

	actions, err := s.actionService.FindByFilterID(ctx, filter.ID, nil, false)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not find filter actions")
	}
	filter.Actions = actions

	indexers, err := s.indexerSvc.FindByFilterID(ctx, filter.ID)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not find indexers")
		return nil, err
	}
	filter.Indexers = indexers

	// Load notifications
	notifications, err := s.notificationSvc.GetFilterNotifications(ctx, filter.ID)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not find notifications")
	}
	filter.Notifications = notifications

	return filter, nil
}

func (s *Service) FindByIndexerIdentifier(ctx context.Context, indexer string) ([]*domain.Filter, error) {
	// get filters for indexer
	filters, err := s.repo.FindByIndexerIdentifier(ctx, indexer)
	if err != nil {
		return nil, err
	}

	// we do not load actions here since we do not need it at this stage
	// only load those after a filter has matched
	filterIDs := make([]int, 0, len(filters))
	for _, filter := range filters {
		filterIDs = append(filterIDs, filter.ID)
	}

	externalFilters, err := s.repo.FindExternalFiltersByFilterIDs(ctx, filterIDs)
	if err != nil {
		s.log.Error().Err(err).Str("indexer", indexer).Msg("could not find external filters")
	}

	for _, filter := range filters {
		filter.External = externalFilters[filter.ID]
	}

	return filters, nil
}

func (s *Service) Store(ctx context.Context, filter *domain.Filter) error {
	if err := filter.Validate(); err != nil {
		s.log.Error().Err(err).Interface("filter_data", filter).Msg("invalid filter")
		return err
	}

	if filter.AnnounceTypes == nil || len(filter.AnnounceTypes) == 0 {
		filter.AnnounceTypes = []string{string(domain.AnnounceTypeNew)}
	}

	if err := s.repo.Store(ctx, filter); err != nil {
		s.log.Error().Err(err).Interface("filter_data", filter).Msg("could not store filter")
		return err
	}

	return nil
}

func (s *Service) Update(ctx context.Context, filter *domain.Filter) error {
	err := filter.Validate()
	if err != nil {
		s.log.Error().Err(err).Interface("filter_data", filter).Msg("validation error")
		return err
	}

	err = filter.Sanitize()
	if err != nil {
		s.log.Error().Err(err).Interface("filter_data", filter).Msg("could not sanitize filter")
		return err
	}

	if err := s.validateIndexers(ctx, filter.Indexers); err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("invalid indexers for filter")
		return err
	}

	if _, err := s.FindByID(ctx, filter.ID); err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not find filter")
		return err
	}

	// update
	err = s.repo.Update(ctx, filter)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not update filter")
		return err
	}

	// take care of connected indexers
	err = s.repo.StoreIndexerConnections(ctx, filter.ID, filter.Indexers)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not store filter indexer connections")
		return err
	}

	// take care of connected external filters
	err = s.repo.StoreFilterExternal(ctx, filter.ID, filter.External)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not store external filters")
		return err
	}

	// take care of filter actions
	actions, err := s.actionService.StoreFilterActions(ctx, int64(filter.ID), filter.Actions)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not store filter actions")
		return err
	}

	filter.Actions = actions

	// take care of filter notifications
	err = s.notificationSvc.StoreFilterNotifications(ctx, filter.ID, filter.Notifications)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not store filter notifications")
		return err
	}

	return nil
}

// validateIndexers rejects indexers that do not exist. The database cannot be
// relied on for this: SQLite runs without foreign key enforcement for legacy
// reasons, so unknown ids would otherwise be stored as orphaned connections.
func (s *Service) validateIndexers(ctx context.Context, indexers []domain.Indexer) error {
	if len(indexers) == 0 {
		return nil
	}

	existing, err := s.indexerSvc.List(ctx)
	if err != nil {
		return err
	}

	existingIDs := make(map[int64]struct{}, len(existing))
	for _, indexer := range existing {
		existingIDs[indexer.ID] = struct{}{}
	}

	for _, indexer := range indexers {
		if _, ok := existingIDs[indexer.ID]; !ok {
			return errors.Wrap(domain.ErrIndexerNotFound, "indexer with id %d does not exist", indexer.ID)
		}
	}

	return nil
}

func (s *Service) UpdatePartial(ctx context.Context, filter domain.FilterUpdate) error {
	// cleanup
	if filter.Shows != nil {
		// replace newline with comma
		clean := strings.ReplaceAll(*filter.Shows, "\n", ",")
		clean = strings.ReplaceAll(clean, ",,", ",")

		filter.Shows = &clean
	}

	if filter.Indexers != nil {
		if err := s.validateIndexers(ctx, filter.Indexers); err != nil {
			s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("invalid indexers for filter")
			return err
		}
	}

	if _, err := s.FindByID(ctx, filter.ID); err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not find filter")
		return err
	}

	// update
	if err := s.repo.UpdatePartial(ctx, filter); err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not update partial filter")
		return err
	}

	if filter.Indexers != nil {
		// take care of connected indexers
		if err := s.repo.StoreIndexerConnections(ctx, filter.ID, filter.Indexers); err != nil {
			s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not store filter indexer connections")
			return err
		}
	}

	if filter.External != nil {
		// take care of connected external filters
		if err := s.repo.StoreFilterExternal(ctx, filter.ID, filter.External); err != nil {
			s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not store external filters")
			return err
		}
	}

	if filter.Actions != nil {
		// take care of filter actions
		if _, err := s.actionService.StoreFilterActions(ctx, int64(filter.ID), filter.Actions); err != nil {
			s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not store filter actions")
			return err
		}
	}

	if filter.Notifications != nil {
		// take care of filter notifications
		if err := s.notificationSvc.StoreFilterNotifications(ctx, filter.ID, filter.Notifications); err != nil {
			s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not store filter notifications")
			return err
		}
	}

	return nil
}

func (s *Service) Duplicate(ctx context.Context, filterID int) (*domain.Filter, error) {
	// find filter with actions, indexers and external filters
	filter, err := s.FindByID(ctx, filterID)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filterID).Msg("could not find filter")
		return nil, err
	}

	// reset id and name
	filter.ID = 0
	filter.Name = fmt.Sprintf("%s Copy", filter.Name)
	filter.Enabled = false

	// store new filter
	if err := s.repo.Store(ctx, filter); err != nil {
		s.log.Error().Err(err).Str("filter", filter.Name).Msg("could not store filter")
		return nil, err
	}

	// take care of connected indexers
	if err := s.repo.StoreIndexerConnections(ctx, filter.ID, filter.Indexers); err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not store filter indexer connections")
		return nil, err
	}

	// reset action id to 0
	for i, a := range filter.Actions {
		a.ID = 0
		filter.Actions[i] = a
	}

	// take care of filter actions
	if _, err := s.actionService.StoreFilterActions(ctx, int64(filter.ID), filter.Actions); err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not store filter actions")
		return nil, err
	}

	// take care of connected external filters
	// the external filters are fetched with FindByID
	if err := s.repo.StoreFilterExternal(ctx, filter.ID, filter.External); err != nil {
		s.log.Error().Err(err).Int("filter_id", filter.ID).Msg("could not store external filters")
		return nil, err
	}

	return filter, nil
}

func (s *Service) ToggleEnabled(ctx context.Context, filterID int, enabled bool) error {
	_, err := s.FindByID(ctx, filterID)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filterID).Msg("could not find filter")
		return err
	}

	if err := s.repo.ToggleEnabled(ctx, filterID, enabled); err != nil {
		s.log.Error().Err(err).Msg("could not update filter enabled")
		return err
	}

	s.log.Debug().Int("filter_id", filterID).Bool("enabled", enabled).Msg("updated filter")

	return nil
}

func (s *Service) Delete(ctx context.Context, filterID int) error {
	if filterID == 0 {
		return nil
	}

	_, err := s.FindByID(ctx, filterID)
	if err != nil {
		s.log.Error().Err(err).Int("filter_id", filterID).Msg("could not find filter")
		return err
	}

	// take care of filter actions
	if err := s.actionService.DeleteByFilterID(ctx, filterID); err != nil {
		s.log.Error().Err(err).Msg("could not delete filter actions")
		return err
	}

	// take care of filter indexers
	if err := s.repo.DeleteIndexerConnections(ctx, filterID); err != nil {
		s.log.Error().Err(err).Msg("could not delete filter indexers")
		return err
	}

	// delete filter external
	if err := s.repo.DeleteFilterExternal(ctx, filterID); err != nil {
		s.log.Error().Err(err).Int("filter_id", filterID).Msg("could not delete filter external")
		return err
	}

	// delete filter
	if err := s.repo.Delete(ctx, filterID); err != nil {
		s.log.Error().Err(err).Int("filter_id", filterID).Msg("could not delete filter")
		return err
	}

	if err := s.notificationSvc.DeleteFilterNotifications(ctx, filterID); err != nil {
		s.log.Error().Err(err).Int("filter_id", filterID).Msg("could not delete filter notifications")
		return err
	}

	return nil
}

func (s *Service) CheckFilter(ctx context.Context, f *domain.Filter, release *domain.Release) (bool, error) {
	l := s.log.With().Str("method", "CheckFilter").Str("trace_id", release.TraceID).Str("filter", f.Name).Str("release", release.TorrentName).Logger()

	l.Debug().Msg("checking filter with release")

	l.Trace().Interface("filter_data", f).Msg("checking filter")
	l.Trace().Interface("release_data", release).Msg("checking filter for release")

	// do additional fetch to get download counts for filter
	if f.IsMaxDownloadsLimitEnabled() {
		if err := s.repo.GetFilterDownloadCount(ctx, f); err != nil {
			l.Error().Err(err).Msg("error getting download counters for filter")
			return false, nil
		}
	}

	rejections, matchedFilter := f.CheckFilter(release)
	if rejections.Len() > 0 {
		l.Debug().Str("rejections", rejections.StringTruncated()).Msg("rejections for release")
		return false, nil
	}

	if !matchedFilter {
		// if no match, return nil
		return false, nil
	}

	// smartEpisode check
	if f.SmartEpisode {
		params := &domain.SmartEpisodeParams{
			Title:   release.Title,
			Season:  release.Season,
			Episode: release.Episode,
			Year:    release.Year,
			Month:   release.Month,
			Day:     release.Day,
			Repack:  release.Repack,
			Proper:  release.Proper,
			Group:   release.Group,
		}
		canDownloadShow, err := s.CheckSmartEpisodeCanDownload(ctx, params)
		if err != nil {
			l.Trace().Msg("failed smart episode check")
			return false, nil
		}

		if !canDownloadShow {
			l.Trace().Msg("failed smart episode check")
			if params.IsDailyEpisode() {
				f.RejectReasons.Add("smart episode", fmt.Sprintf("not new (%s) daily: %d-%d-%d", release.Title, release.Year, release.Month, release.Day), fmt.Sprintf("expected newer than (%s) daily: %d-%d-%d", release.Title, release.Year, release.Month, release.Day))
			} else {
				f.RejectReasons.Add("smart episode", fmt.Sprintf("not new (%s) season: %d ep: %d", release.Title, release.Season, release.Episode), fmt.Sprintf("expected newer than (%s) season: %d ep: %d", release.Title, release.Season, release.Episode))
			}
			return false, nil
		}
	}

	// check duplicates
	if f.DuplicateHandling != nil {
		l.Debug().Str("profile", f.DuplicateHandling.Name).Msg("check is duplicate with profile")

		release.SkipDuplicateProfileID = f.DuplicateHandling.ID
		release.SkipDuplicateProfileName = f.DuplicateHandling.Name

		isDuplicate, err := s.CheckIsDuplicateRelease(ctx, f.DuplicateHandling, release)
		if err != nil {
			return false, errors.Wrap(err, "error finding duplicate handle")
		}

		if isDuplicate {
			l.Debug().Str("profile", f.DuplicateHandling.Name).Msg("rejected release as duplicate with profile")
			f.RejectReasons.Add("duplicate", "duplicate", "not duplicate")

			// let it continue so external filters can trigger checks
			//return false, nil
			release.IsDuplicate = true
		}
	}

	// if matched, do additional size check if needed, attach actions and return the filter

	l.Debug().Msg("found and matched filter")

	// If size constraints are set in a filter and the indexer did not
	// announce the size, we need to do an additional out of band size check.
	if release.AdditionalSizeCheckRequired {
		l.Debug().Msg("additional size check required")

		ok, err := s.AdditionalSizeCheck(ctx, f, release)
		if err != nil {
			l.Error().Err(err).Msg("additional size check error")
			return false, err
		}

		if !ok {
			l.Trace().Msg("additional size check not matching what filter wanted")
			return false, nil
		}
	}

	// check uploader if the indexer supports check via api
	if release.AdditionalUploaderCheckRequired {
		l.Debug().Msg("additional uploader check required")

		ok, err := s.AdditionalUploaderCheck(ctx, f, release)
		if err != nil {
			l.Error().Err(err).Msg("additional uploader check error")
			return false, err
		}

		if !ok {
			l.Trace().Msg("additional uploader check not matching what filter wanted")
			return false, nil
		}
	}

	if release.AdditionalRecordLabelCheckRequired {
		l.Debug().Msg("additional record label check required")

		ok, err := s.AdditionalRecordLabelCheck(ctx, f, release)
		if err != nil {
			l.Error().Err(err).Msg("additional record label check error")
			return false, err
		}

		if !ok {
			l.Trace().Msg("additional record label check not matching what filter wanted")
			return false, nil
		}
	}

	// run external filters
	if f.External != nil {
		externalOk, err := s.RunExternalFilters(ctx, f, f.External, release)
		if err != nil {
			l.Error().Err(err).Msg("external filter check error")
			return false, err
		}

		if !externalOk {
			l.Debug().Msg("external filter check not matching what filter wanted")
			return false, nil
		}
	}

	return true, nil
}

// AdditionalSizeCheck performs additional out-of-band checks to determine the
// values of a torrent. Some indexers do not announce torrent size, so it is
// necessary to determine the size of the torrent in some other way. Some
// indexers have an API implemented to fetch this data. For those which don't,
// it is necessary to download the torrent file and parse it to make the size
// check. We use the API where available to minimize the number of torrents we
// need to download.
func (s *Service) AdditionalSizeCheck(ctx context.Context, f *domain.Filter, release *domain.Release) (ok bool, err error) {
	defer func() {
		// try recover panic if anything went wrong with API or size checks
		errors.RecoverPanic(recover(), &err)
	}()

	// do additional size check against indexer api or torrent for size
	l := s.log.With().Str("method", "AdditionalSizeCheck").Str("trace_id", release.TraceID).Str("filter", f.Name).Str("release", release.TorrentName).Logger()

	l.Debug().Msg("additional api size check required")

	switch release.Indexer.Identifier {
	case "btn", "ggn", "redacted", "ops", "mock":
		if (release.Size == 0 && release.AdditionalSizeCheckRequired) || (release.Uploader == "" && release.AdditionalUploaderCheckRequired) || (release.RecordLabel == "" && release.AdditionalRecordLabelCheckRequired) {
			l.Trace().Str("filter", f.Name).Msg("preparing to check size via api")

			torrentInfo, err := s.apiService.GetTorrentByID(ctx, release.Indexer.Identifier, release.TorrentID)
			if err != nil || torrentInfo == nil {
				l.Error().Err(err).Str("torrent_id", release.TorrentID).Str("indexer", release.Indexer.Identifier).Msg("could not get torrent info from api")
				return false, err
			}

			l.Debug().Interface("torrent_info", torrentInfo).Msg("got torrent info from api")

			torrentSize := torrentInfo.ReleaseSizeBytes()
			if release.Size == 0 && torrentSize > 0 {
				release.Size = torrentSize
			}

			if release.Uploader == "" {
				release.Uploader = torrentInfo.Uploader
			}

			if release.RecordLabel == "" {
				release.RecordLabel = torrentInfo.RecordLabel
			}
		}

	default:
		if release.Size == 0 && release.AdditionalSizeCheckRequired {
			l.Trace().Msg("preparing to download torrent metafile")

			// if indexer doesn't have api, download torrent and add to tmpPath
			if err := s.downloadSvc.DownloadRelease(ctx, release); err != nil {
				l.Error().Err(err).Str("torrent_id", release.TorrentID).Str("indexer", release.Indexer.Identifier).Msg("could not download torrent file")
				return false, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
			}
		}
	}

	sizeOk, err := f.CheckReleaseSize(release.Size)
	if err != nil {
		l.Error().Err(err).Msg("error comparing release and filter size")
		return false, err
	}

	// reset AdditionalSizeCheckRequired to not re-trigger check
	release.AdditionalSizeCheckRequired = false

	if !sizeOk {
		l.Debug().Msg("filter did not match after additional size check, trying next")
		return false, nil
	}

	return true, nil
}

func (s *Service) AdditionalUploaderCheck(ctx context.Context, f *domain.Filter, release *domain.Release) (ok bool, err error) {
	defer func() {
		// try recover panic if anything went wrong with API or size checks
		errors.RecoverPanic(recover(), &err)
	}()

	// do additional check against indexer api
	l := s.log.With().Str("method", "AdditionalUploaderCheck").Str("trace_id", release.TraceID).Str("filter", f.Name).Str("release", release.TorrentName).Logger()

	// if uploader was fetched before during size check we check it and return early
	if release.Uploader != "" {
		uploaderOk, err := f.CheckUploader(release.Uploader)
		if err != nil {
			l.Error().Err(err).Msg("error comparing release and uploaders")
			return false, err
		}

		// reset AdditionalUploaderCheckRequired to not re-trigger check
		release.AdditionalUploaderCheckRequired = false

		if !uploaderOk {
			l.Debug().Msg("filter did not match after additional uploaders check, trying next")
			return false, nil
		}

		return true, nil
	}

	l.Debug().Msg("additional api uploader check required")

	switch release.Indexer.Identifier {
	case "redacted", "ops", "mock":
		l.Trace().Msg("preparing to check via api")

		torrentInfo, err := s.apiService.GetTorrentByID(ctx, release.Indexer.Identifier, release.TorrentID)
		if err != nil || torrentInfo == nil {
			l.Error().Err(err).Str("torrent_id", release.TorrentID).Str("indexer", release.Indexer.Identifier).Msg("could not get torrent info from api")
			return false, err
		}

		l.Debug().Interface("torrent_info", torrentInfo).Msg("got torrent info from api")

		torrentSize := torrentInfo.ReleaseSizeBytes()
		if release.Size == 0 && torrentSize > 0 {
			release.Size = torrentSize
		}

		if release.RecordLabel == "" {
			release.RecordLabel = torrentInfo.RecordLabel
		}

		if release.Uploader == "" {
			release.Uploader = torrentInfo.Uploader
		}

	default:
		return false, errors.New("additional uploader check not supported for this indexer: %s", release.Indexer.Identifier)
	}

	uploaderOk, err := f.CheckUploader(release.Uploader)
	if err != nil {
		l.Error().Err(err).Msg("error comparing release and uploaders")
		return false, err
	}

	// reset AdditionalUploaderCheckRequired to not re-trigger check
	release.AdditionalUploaderCheckRequired = false

	if !uploaderOk {
		l.Debug().Msg("filter did not match after additional uploaders check, trying next")
		return false, nil
	}

	return true, nil
}

func (s *Service) AdditionalRecordLabelCheck(ctx context.Context, f *domain.Filter, release *domain.Release) (ok bool, err error) {
	defer func() {
		// try recover panic if anything went wrong with API or size checks
		errors.RecoverPanic(recover(), &err)
		if err != nil {
			ok = false
		}
	}()

	// do additional check against indexer api
	l := s.log.With().Str("method", "AdditionalRecordLabelCheck").Str("trace_id", release.TraceID).Str("filter", f.Name).Str("release", release.TorrentName).Logger()

	// if record label was fetched before during size check or uploader check we check it and return early
	if release.RecordLabel != "" {
		recordLabelOk, err := f.CheckRecordLabel(release.RecordLabel)
		if err != nil {
			l.Error().Err(err).Msg("error comparing release and record label")
			return false, err
		}

		// reset AdditionalRecordLabelCheckRequired to not re-trigger check
		release.AdditionalRecordLabelCheckRequired = false

		if !recordLabelOk {
			l.Debug().Msg("filter did not match after additional record label check, trying next")
			return false, nil
		}

		return true, nil
	}

	l.Debug().Msg("additional api record label check required")

	switch release.Indexer.Identifier {
	case "redacted", "ops", "mock":
		l.Trace().Msg("preparing to check via api")

		torrentInfo, err := s.apiService.GetTorrentByID(ctx, release.Indexer.Identifier, release.TorrentID)
		if err != nil || torrentInfo == nil {
			l.Error().Err(err).Str("torrent_id", release.TorrentID).Str("indexer", release.Indexer.Identifier).Msg("could not get torrent info from api")
			return false, err
		}

		l.Debug().Interface("torrent_info", torrentInfo).Msg("got torrent info from api")

		torrentSize := torrentInfo.ReleaseSizeBytes()
		if release.Size == 0 && torrentSize > 0 {
			release.Size = torrentSize
		}

		if release.Uploader == "" {
			release.Uploader = torrentInfo.Uploader
		}

		if release.RecordLabel == "" {
			release.RecordLabel = torrentInfo.RecordLabel
		}

	default:
		return false, errors.New("additional record label check not supported for this indexer: %s", release.Indexer.Identifier)
	}

	recordLabelOk, err := f.CheckRecordLabel(release.RecordLabel)
	if err != nil {
		l.Error().Err(err).Msg("error comparing release and record label")
		return false, err
	}

	// reset AdditionalRecordLabelCheckRequired to not re-trigger check
	release.AdditionalRecordLabelCheckRequired = false

	if !recordLabelOk {
		l.Debug().Msg("filter did not match after additional record label check, trying next")
		return false, nil
	}

	return true, nil
}

func (s *Service) CheckSmartEpisodeCanDownload(ctx context.Context, params *domain.SmartEpisodeParams) (bool, error) {
	return s.releaseRepo.CheckSmartEpisodeCanDownload(ctx, params)
}

func (s *Service) CheckIsDuplicateRelease(ctx context.Context, profile *domain.DuplicateReleaseProfile, release *domain.Release) (bool, error) {
	return s.releaseRepo.CheckIsDuplicateRelease(ctx, profile, release)
}

func (s *Service) RunExternalFilters(ctx context.Context, f *domain.Filter, externalFilters []domain.FilterExternal, release *domain.Release) (ok bool, err error) {
	defer func() {
		// try recover panic if anything went wrong with the external filter checks
		errors.RecoverPanic(recover(), &err)
		if err != nil {
			s.log.Error().Err(err).Str("filter", f.Name).Msg("external filter check panic")
			ok = false
		}
	}()

	for _, external := range externalFilters {
		l := s.log.With().Str("method", "RunExternalFilters").Str("trace_id", release.TraceID).Str("filter", f.Name).Str("external_filter", external.Name).Logger()

		if !external.Enabled {
			l.Debug().Msg("external filter not enabled, skipping")

			continue
		}

		if external.NeedTorrentDownloaded() {
			if err := s.downloadSvc.DownloadRelease(ctx, release); err != nil {
				return false, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
			}

			// the path macro hands a file to the script or webhook, so the
			// in-memory torrent has to be written to disk first
			if external.NeedTorrentTmpFile() {
				if err := release.WriteTemporaryFile(); err != nil {
					return false, errors.Wrap(err, "could not write torrent file for release: %s", release.TorrentName)
				}
			}
		}

		switch external.Type {
		case domain.ExternalFilterTypeExec:
			// run external script
			exitCode, err := s.execCmd(ctx, external, release)
			if err != nil {
				l.Error().Err(err).Msg("error executing external script")

				if external.OnError == domain.FilterExternalOnErrorContinue {
					l.Debug().Msg("external script error, and OnError set to continue")
					continue
				}
				return false, errors.Wrap(err, "error executing external command")
			}

			if exitCode != external.ExecExpectStatus {
				l.Debug().Int("expected_status", external.ExecExpectStatus).Int("actual_status", exitCode).Msg("external script got unexpected exit code")
				f.RejectReasons.Add("external script exit code", exitCode, external.ExecExpectStatus)
				return false, nil
			}

		case domain.ExternalFilterTypeWebhook:
			// run external webhook
			statusCode, err := s.webhook(ctx, external, release)
			if err != nil {
				l.Error().Err(err).Msg("error executing external webhook")

				// Only continue if the filter is configured to continue on error
				if external.OnError == domain.FilterExternalOnErrorContinue {
					l.Debug().Msg("external webhook error, and OnError set to continue")
					continue
				}
				return false, errors.Wrap(err, "error executing external webhook")
			}

			if statusCode != external.WebhookExpectStatus {
				l.Debug().Int("expected_status", external.WebhookExpectStatus).Int("actual_status", statusCode).Msg("external webhook got unexpected status code")
				f.RejectReasons.Add("external webhook status code", statusCode, external.WebhookExpectStatus)
				return false, nil
			}
		}
	}

	return true, nil
}

func (s *Service) execCmd(_ context.Context, external domain.FilterExternal, release *domain.Release) (int, error) {
	s.log.Trace().Str("release", release.TorrentName).Msg("filter exec release")

	// check if program exists
	cmd, err := exec.LookPath(external.ExecCmd)
	if err != nil {
		return 0, errors.Wrap(err, "exec failed, could not find program: %s", cmd)
	}

	// handle args and replace vars
	m := domain.NewMacro(*release)

	// parse and replace values in argument string before continuing
	parsedArgs, err := m.Parse(external.ExecArgs)
	if err != nil {
		return 0, errors.Wrap(err, "could not parse macro")
	}

	// we need to split on space into a string slice, so we can spread the args into exec
	commandArgs, err := shellquote.Split(parsedArgs)
	if err != nil {
		return 0, errors.Wrap(err, "could not parse into shell-words")
	}

	start := time.Now()

	// setup command and args
	command := exec.Command(cmd, commandArgs...)

	s.log.Debug().Str("script", cmd).Str("args", strings.Join(commandArgs, " ")).Msg("executing script")

	// Create a pipe to capture the standard output of the command
	cmdOutput, err := command.StdoutPipe()
	if err != nil {
		s.log.Error().Err(err).Msg("could not create stdout pipe")
		return 0, err
	}

	duration := time.Since(start)

	// Start the command
	if err := command.Start(); err != nil {
		s.log.Error().Err(err).Msg("error starting command")
		return 0, err
	}

	// Create a buffer to store the output
	outputBuffer := make([]byte, 4096)

	execLogger := s.log.With().Str("trace_id", release.TraceID).Str("release", release.TorrentName).Str("filter", release.FilterName).Logger()

	for {
		// Read the output into the buffer
		n, err := cmdOutput.Read(outputBuffer)
		if err != nil {
			break
		}

		// Write the output to the logger
		execLogger.Trace().Msg(string(outputBuffer[:n]))
	}

	// Wait for the command to finish and check for any errors
	if err := command.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			s.log.Debug().Int("exit_code", exitErr.ExitCode()).Msg("filter script exited with non-zero code")
			return exitErr.ExitCode(), nil
		}

		s.log.Error().Err(err).Msg("error waiting for command")
		return 0, err
	}

	s.log.Debug().Str("script", cmd).Str("args", parsedArgs).Str("release", release.TorrentName).Str("indexer", release.Indexer.Identifier).Dur("duration", duration).Msg("executed external script")

	return 0, nil
}

func (s *Service) webhook(ctx context.Context, external domain.FilterExternal, release *domain.Release) (int, error) {
	l := s.log.With().Str("method", "webhook").Str("trace_id", release.TraceID).Str("external_filter", external.Name).Str("host", external.WebhookHost).Str("http_method", external.WebhookMethod).Logger()

	l.Trace().Str("payload", external.WebhookData).Msg("preparing to run external webhook filter")

	if external.WebhookHost == "" {
		return 0, errors.New("external filter: missing host for webhook")
	}

	m := domain.NewMacro(*release)

	// parse and replace values in argument string before continuing
	dataArgs, err := m.Parse(external.WebhookData)
	if err != nil {
		return 0, errors.Wrap(err, "could not parse webhook data macro: %s", external.WebhookData)
	}

	l.Debug().Str("payload", external.WebhookData).Msg("sending external webhook filter request")

	method := http.MethodPost
	if external.WebhookMethod != "" {
		method = external.WebhookMethod
	}

	req, err := http.NewRequestWithContext(ctx, method, external.WebhookHost, nil)
	if err != nil {
		return 0, errors.Wrap(err, "could not build request for webhook")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	if external.WebhookHeaders != "" {
		for header := range strings.SplitSeq(external.WebhookHeaders, ";") {
			h := strings.Split(header, "=")

			if len(h) != 2 {
				continue
			}

			// add header to req
			req.Header.Add(h[0], h[1]) // go already canonicalizes the provided header key.
		}
	}

	retryAttempts := external.WebhookRetryAttempts
	if retryAttempts == 0 {
		retryAttempts = 1
	}

	opts := []retry.Option{
		retry.DelayType(retry.FixedDelay),
		retry.LastErrorOnly(true),
		retry.Attempts(uint(retryAttempts)),
	}

	if external.WebhookRetryDelaySeconds > 0 {
		opts = append(opts, retry.Delay(time.Duration(external.WebhookRetryDelaySeconds)*time.Second))
	}

	var retryStatusCodes []string
	if external.WebhookRetryStatus != "" {
		retryStatusCodes = strings.Split(strings.ReplaceAll(external.WebhookRetryStatus, " ", ""), ",")
	}

	start := time.Now()

	statusCode, err := retry.DoWithData(
		func() (int, error) {
			clonereq := req.Clone(ctx)
			if external.WebhookData != "" && dataArgs != "" {
				clonereq.Body = io.NopCloser(bytes.NewBufferString(dataArgs))
			}

			l.Trace().Msg("making filter external webhook request..")

			res, err := s.httpClient.Do(clonereq)
			if err != nil {
				return 0, errors.Wrap(err, "could not make request for webhook")
			}

			defer sharedhttp.DrainAndClose(res)

			l.Debug().Int("status_code", res.StatusCode).Msg("filter external webhook response")

			if s.log.Debug().Enabled() {
				body, err := io.ReadAll(io.LimitReader(res.Body, 4096)) // 4KB limit for debug logging
				if err != nil {
					return res.StatusCode, errors.Wrap(err, "could not read request body")
				}

				if len(body) > 0 {
					l.Debug().Int("status_code", res.StatusCode).Str("body", string(body)).Msg("filter external webhook response body")
				}
			}

			if utils.StrSliceContains(retryStatusCodes, strconv.Itoa(res.StatusCode)) {
				return 0, errors.New("webhook got unwanted status code: %d", res.StatusCode)
			}

			return res.StatusCode, nil
		},
		opts...)

	if err != nil {
		l.Error().Err(err).Msg("error sending webhook")

		return statusCode, errors.Wrap(err, "could not make request for webhook")
	}

	l.Debug().Str("args", dataArgs).TimeDiff("duration", time.Now(), start).Msg("successfully ran external webhook filter")

	return statusCode, nil
}
