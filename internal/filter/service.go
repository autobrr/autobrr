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

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/releasedownload"
	"github.com/autobrr/autobrr/internal/utils"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/avast/retry-go/v4"
	"github.com/mattn/go-shellwords"
	"github.com/rs/zerolog"
)

type Service interface {
	FindByID(ctx context.Context, filterID int) (*domain.Filter, error)
	FindByIndexerIdentifier(ctx context.Context, indexer string) ([]*domain.Filter, error)
	Find(ctx context.Context, params domain.FilterQueryParams) ([]*domain.Filter, error)
	CheckFilter(ctx context.Context, f *domain.Filter, release *domain.Release) (bool, error)
	ListFilters(ctx context.Context) ([]domain.Filter, error)
	Store(ctx context.Context, filter *domain.Filter) error
	Update(ctx context.Context, filter *domain.Filter) error
	UpdatePartial(ctx context.Context, filter domain.FilterUpdate) error
	Duplicate(ctx context.Context, filterID int) (*domain.Filter, error)
	ToggleEnabled(ctx context.Context, filterID int, enabled bool) error
	Delete(ctx context.Context, filterID int) error
	AdditionalSizeCheck(ctx context.Context, f *domain.Filter, release *domain.Release) (bool, error)
	AdditionalUploaderCheck(ctx context.Context, f *domain.Filter, release *domain.Release) (bool, error)
	AdditionalRecordLabelCheck(ctx context.Context, f *domain.Filter, release *domain.Release) (bool, error)
	CheckSmartEpisodeCanDownload(ctx context.Context, params *domain.SmartEpisodeParams) (bool, error)
	CheckIsDuplicateRelease(ctx context.Context, profile *domain.DuplicateReleaseProfile, release *domain.Release) (bool, error)
	GetRateLimiter() *RateLimiter
}

type service struct {
	log             zerolog.Logger
	repo            domain.FilterRepo
	actionService   action.Service
	releaseRepo     domain.ReleaseRepo
	indexerSvc      indexer.Service
	apiService      indexer.APIService
	downloadSvc     *releasedownload.DownloadService
	notificationSvc notification.FilterStorer
	rateLimiter     *RateLimiter

	httpClient *http.Client
}

func NewService(log logger.Logger, repo domain.FilterRepo, actionSvc action.Service, releaseRepo domain.ReleaseRepo, apiService indexer.APIService, indexerSvc indexer.Service, downloadSvc *releasedownload.DownloadService, notificationSvc notification.FilterStorer) Service {
	s := &service{
		log:             log.With().Str("module", "filter").Logger(),
		repo:            repo,
		releaseRepo:     releaseRepo,
		actionService:   actionSvc,
		apiService:      apiService,
		indexerSvc:      indexerSvc,
		downloadSvc:     downloadSvc,
		notificationSvc: notificationSvc,
		rateLimiter:     NewRateLimiter(log, repo),
		httpClient: &http.Client{
			Timeout:   time.Second * 120,
			Transport: sharedhttp.TransportTLSInsecure,
		},
	}

	// Initialize rate limiter from database
	s.initFilterRateLimiter()

	return s
}

func (s *service) initFilterRateLimiter() {
	s.log.Debug().Msg("initializing filter rate limiter from database")
	ctx := context.Background()
	filters, err := s.repo.Find(ctx, domain.FilterQueryParams{})
	if err != nil {
		s.log.Error().Err(err).Msg("failed to load filters for rate limiter initialization")
		return
	}

	if err := s.rateLimiter.InitializeFromDB(ctx, filters); err != nil {
		s.log.Error().Err(err).Msg("failed to initialize rate limiter")
	}
}

func (s *service) Find(ctx context.Context, params domain.FilterQueryParams) ([]*domain.Filter, error) {
	// get filters
	filters, err := s.repo.Find(ctx, params)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find list filters")
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
				s.log.Error().Err(err).Msgf("could not get filter downloads for filter: %s", filter.Name)
			}
		}
	}

	return filters, nil
}

func (s *service) ListFilters(ctx context.Context) ([]domain.Filter, error) {
	// get filters
	filters, err := s.repo.ListFilters(ctx)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find list filters")
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

func (s *service) FindByID(ctx context.Context, filterID int) (*domain.Filter, error) {
	filter, err := s.repo.FindByID(ctx, filterID)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find filter for id: %v", filterID)
		return nil, err
	}

	externalFilters, err := s.repo.FindExternalFiltersByID(ctx, filter.ID)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find external filters for filter id: %v", filter.ID)
	}
	filter.External = externalFilters

	actions, err := s.actionService.FindByFilterID(ctx, filter.ID, nil, false)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find filter actions for filter id: %v", filter.ID)
	}
	filter.Actions = actions

	indexers, err := s.indexerSvc.FindByFilterID(ctx, filter.ID)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find indexers for filter: %v", filter.Name)
		return nil, err
	}
	filter.Indexers = indexers

	// Load notifications
	notifications, err := s.notificationSvc.GetFilterNotifications(ctx, filter.ID)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find notifications for filter: %v", filter.Name)
	}
	filter.Notifications = notifications

	return filter, nil
}

func (s *service) FindByIndexerIdentifier(ctx context.Context, indexer string) ([]*domain.Filter, error) {
	// get filters for indexer
	filters, err := s.repo.FindByIndexerIdentifier(ctx, indexer)
	if err != nil {
		return nil, err
	}

	// we do not load actions here since we do not need it at this stage
	// only load those after a filter has matched
	for _, filter := range filters {
		externalFilters, err := s.repo.FindExternalFiltersByID(ctx, filter.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("could not find external filters for filter id: %v", filter.ID)
		}
		filter.External = externalFilters
	}

	return filters, nil
}

func (s *service) Store(ctx context.Context, filter *domain.Filter) error {
	if err := filter.Validate(); err != nil {
		s.log.Error().Err(err).Msgf("invalid filter: %v", filter)
		return err
	}

	if filter.AnnounceTypes == nil || len(filter.AnnounceTypes) == 0 {
		filter.AnnounceTypes = []string{string(domain.AnnounceTypeNew)}
	}

	if err := s.repo.Store(ctx, filter); err != nil {
		s.log.Error().Err(err).Msgf("could not store filter: %v", filter)
		return err
	}

	return nil
}

func (s *service) Update(ctx context.Context, filter *domain.Filter) error {
	err := filter.Validate()
	if err != nil {
		s.log.Error().Err(err).Msgf("validation error filter: %+v", filter)
		return err
	}

	err = filter.Sanitize()
	if err != nil {
		s.log.Error().Err(err).Msgf("could not sanitize filter: %v", filter)
		return err
	}

	// update
	err = s.repo.Update(ctx, filter)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not update filter: %s", filter.Name)
		return err
	}

	// take care of connected indexers
	err = s.repo.StoreIndexerConnections(ctx, filter.ID, filter.Indexers)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not store filter indexer connections: %s", filter.Name)
		return err
	}

	// take care of connected external filters
	err = s.repo.StoreFilterExternal(ctx, filter.ID, filter.External)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not store external filters: %s", filter.Name)
		return err
	}

	// take care of filter actions
	actions, err := s.actionService.StoreFilterActions(ctx, int64(filter.ID), filter.Actions)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not store filter actions: %s", filter.Name)
		return err
	}

	filter.Actions = actions

	// take care of filter notifications
	err = s.notificationSvc.StoreFilterNotifications(ctx, filter.ID, filter.Notifications)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not store filter notifications: %s", filter.Name)
		return err
	}

	// Update rate limiter bucket if max downloads settings changed
	s.rateLimiter.UpdateBucket(filter)

	return nil
}

func (s *service) UpdatePartial(ctx context.Context, filter domain.FilterUpdate) error {
	// cleanup
	if filter.Shows != nil {
		// replace newline with comma
		clean := strings.ReplaceAll(*filter.Shows, "\n", ",")
		clean = strings.ReplaceAll(clean, ",,", ",")

		filter.Shows = &clean
	}

	// update
	if err := s.repo.UpdatePartial(ctx, filter); err != nil {
		s.log.Error().Err(err).Msgf("could not update partial filter: %v", filter.ID)
		return err
	}

	if filter.Indexers != nil {
		// take care of connected indexers
		if err := s.repo.StoreIndexerConnections(ctx, filter.ID, filter.Indexers); err != nil {
			s.log.Error().Err(err).Msgf("could not store filter indexer connections: %v", filter.Name)
			return err
		}
	}

	if filter.External != nil {
		// take care of connected external filters
		if err := s.repo.StoreFilterExternal(ctx, filter.ID, filter.External); err != nil {
			s.log.Error().Err(err).Msgf("could not store external filters: %v", filter.Name)
			return err
		}
	}

	if filter.Actions != nil {
		// take care of filter actions
		if _, err := s.actionService.StoreFilterActions(ctx, int64(filter.ID), filter.Actions); err != nil {
			s.log.Error().Err(err).Msgf("could not store filter actions: %v", filter.ID)
			return err
		}
	}

	if filter.Notifications != nil {
		// take care of filter notifications
		if err := s.notificationSvc.StoreFilterNotifications(ctx, filter.ID, filter.Notifications); err != nil {
			s.log.Error().Err(err).Msgf("could not store filter notifications: %v", filter.ID)
			return err
		}
	}

	// Update rate limiter bucket if max downloads settings may have changed
	if filter.MaxDownloads != nil || filter.MaxDownloadsUnit != nil {
		// Load the full filter to update the bucket
		fullFilter, err := s.FindByID(ctx, filter.ID)
		if err == nil {
			s.rateLimiter.UpdateBucket(fullFilter)
		}
	}

	return nil
}

func (s *service) Duplicate(ctx context.Context, filterID int) (*domain.Filter, error) {
	// find filter with actions, indexers and external filters
	filter, err := s.FindByID(ctx, filterID)
	if err != nil {
		return nil, err
	}

	// reset id and name
	filter.ID = 0
	filter.Name = fmt.Sprintf("%s Copy", filter.Name)
	filter.Enabled = false

	// store new filter
	if err := s.repo.Store(ctx, filter); err != nil {
		s.log.Error().Err(err).Msgf("could not update filter: %s", filter.Name)
		return nil, err
	}

	// take care of connected indexers
	if err := s.repo.StoreIndexerConnections(ctx, filter.ID, filter.Indexers); err != nil {
		s.log.Error().Err(err).Msgf("could not store filter indexer connections: %s", filter.Name)
		return nil, err
	}

	// reset action id to 0
	for i, a := range filter.Actions {
		a := a
		a.ID = 0
		filter.Actions[i] = a
	}

	// take care of filter actions
	if _, err := s.actionService.StoreFilterActions(ctx, int64(filter.ID), filter.Actions); err != nil {
		s.log.Error().Err(err).Msgf("could not store filter actions: %s", filter.Name)
		return nil, err
	}

	// take care of connected external filters
	// the external filters are fetched with FindByID
	if err := s.repo.StoreFilterExternal(ctx, filter.ID, filter.External); err != nil {
		s.log.Error().Err(err).Msgf("could not store external filters: %s", filter.Name)
		return nil, err
	}

	return filter, nil
}

func (s *service) ToggleEnabled(ctx context.Context, filterID int, enabled bool) error {
	if err := s.repo.ToggleEnabled(ctx, filterID, enabled); err != nil {
		s.log.Error().Err(err).Msg("could not update filter enabled")
		return err
	}

	s.log.Debug().Msgf("filter.toggle_enabled: update filter '%v' to '%v'", filterID, enabled)

	return nil
}

func (s *service) Delete(ctx context.Context, filterID int) error {
	if filterID == 0 {
		return nil
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
		s.log.Error().Err(err).Msgf("could not delete filter external: %v", filterID)
		return err
	}

	// delete filter
	if err := s.repo.Delete(ctx, filterID); err != nil {
		s.log.Error().Err(err).Msgf("could not delete filter: %v", filterID)
		return err
	}

	if err := s.notificationSvc.DeleteFilterNotifications(ctx, filterID); err != nil {
		s.log.Error().Err(err).Msgf("could not delete filter notifications: %v", filterID)
		return err
	}

	// Remove rate limiter bucket for deleted filter
	s.rateLimiter.buckets.Delete(filterID)

	return nil
}

func (s *service) CheckFilter(ctx context.Context, f *domain.Filter, release *domain.Release) (bool, error) {
	l := s.log.With().Str("method", "CheckFilter").Logger()

	l.Debug().Msgf("checking filter: %s with release %s", f.Name, release.TorrentName)

	l.Trace().Msgf("checking filter: %s %+v", f.Name, f)
	l.Trace().Msgf("checking filter: %s for release: %+v", f.Name, release)

	releaseIdentifier := fmt.Sprintf("%s-%s", release.Indexer.Identifier, release.TorrentID)

	// Try to acquire rate limit token EARLY before expensive operations
	receipt := s.rateLimiter.TryAcquire(f, releaseIdentifier)
	if receipt == nil && f.IsMaxDownloadsLimitEnabled() {
		// Rate limit reached, reject early
		l.Debug().Msgf("rate limit reached for filter %s, rejecting release %s", f.Name, release.TorrentName)
		if f.RejectReasons == nil {
			f.RejectReasons = domain.NewRejectionReasons()
		}
		f.RejectReasons.Addf("max downloads", fmt.Sprintf("[max downloads] reached %d per %s", f.MaxDownloads, f.MaxDownloadsUnit), "", fmt.Sprintf("reached %d per %s", f.MaxDownloads, f.MaxDownloadsUnit))

		return false, nil
	}

	// Store receipt in release context so it can be released later if needed
	if receipt != nil {
		release.FilterRateLimitReceipt = receipt
	}

	rejections, matchedFilter := f.CheckFilter(release)
	if rejections.Len() > 0 {
		l.Debug().Msgf("(%s) for release: %v rejections: (%s)", f.Name, release.TorrentName, rejections.StringTruncated())
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
			l.Trace().Msgf("failed smart episode check: %s", f.Name)
			return false, nil
		}

		if !canDownloadShow {
			l.Trace().Msgf("failed smart episode check: %s", f.Name)
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
		l.Debug().Msgf("(%s) check is duplicate with profile %s", f.Name, f.DuplicateHandling.Name)

		release.SkipDuplicateProfileID = f.DuplicateHandling.ID
		release.SkipDuplicateProfileName = f.DuplicateHandling.Name

		isDuplicate, err := s.CheckIsDuplicateRelease(ctx, f.DuplicateHandling, release)
		if err != nil {
			return false, errors.Wrap(err, "error finding duplicate handle")
		}

		if isDuplicate {
			l.Debug().Msgf("filter %s rejected release %q as duplicate with profile %q", f.Name, release.TorrentName, f.DuplicateHandling.Name)
			f.RejectReasons.Add("duplicate", "duplicate", "not duplicate")

			// let it continue so external filters can trigger checks
			//return false, nil
			release.IsDuplicate = true
		}
	}

	// if matched, do additional size check if needed, attach actions and return the filter

	l.Debug().Msgf("found and matched filter: %s", f.Name)

	// If size constraints are set in a filter and the indexer did not
	// announce the size, we need to do an additional-out-of-band size check.
	if release.AdditionalSizeCheckRequired {
		l.Debug().Msgf("(%s) additional size check required", f.Name)

		ok, err := s.AdditionalSizeCheck(ctx, f, release)
		if err != nil {
			l.Error().Err(err).Msgf("(%s) additional size check error", f.Name)
			return false, err
		}

		if !ok {
			l.Trace().Msgf("(%s) additional size check not matching what filter wanted", f.Name)
			return false, nil
		}
	}

	// check uploader if the indexer supports check via api
	if release.AdditionalUploaderCheckRequired {
		l.Debug().Msgf("(%s) additional uploader check required", f.Name)

		ok, err := s.AdditionalUploaderCheck(ctx, f, release)
		if err != nil {
			l.Error().Err(err).Msgf("(%s) additional uploader check error", f.Name)
			return false, err
		}

		if !ok {
			l.Trace().Msgf("(%s) additional uploader check not matching what filter wanted", f.Name)
			return false, nil
		}
	}

	if release.AdditionalRecordLabelCheckRequired {
		l.Debug().Msgf("(%s) additional record label check required", f.Name)

		ok, err := s.AdditionalRecordLabelCheck(ctx, f, release)
		if err != nil {
			l.Error().Err(err).Msgf("(%s) additional record label check error", f.Name)
			return false, err
		}

		if !ok {
			l.Trace().Msgf("(%s) additional record label check not matching what filter wanted", f.Name)
			return false, nil
		}
	}

	// run external filters
	if f.External != nil {
		externalOk, err := s.RunExternalFilters(ctx, f, f.External, release)
		if err != nil {
			l.Error().Err(err).Msgf("(%s) external filter check error", f.Name)
			return false, err
		}

		if !externalOk {
			l.Debug().Msgf("(%s) external filter check not matching what filter wanted", f.Name)
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
func (s *service) AdditionalSizeCheck(ctx context.Context, f *domain.Filter, release *domain.Release) (ok bool, err error) {
	defer func() {
		// try recover panic if anything went wrong with API or size checks
		errors.RecoverPanic(recover(), &err)
	}()

	// do additional size check against indexer api or torrent for size
	l := s.log.With().Str("method", "AdditionalSizeCheck").Logger()

	l.Debug().Msgf("(%s) additional api size check required", f.Name)

	switch release.Indexer.Identifier {
	case "btn", "ggn", "redacted", "ops", "mock":
		if (release.Size == 0 && release.AdditionalSizeCheckRequired) || (release.Uploader == "" && release.AdditionalUploaderCheckRequired) || (release.RecordLabel == "" && release.AdditionalRecordLabelCheckRequired) {
			l.Trace().Msgf("(%s) preparing to check size via api", f.Name)

			torrentInfo, err := s.apiService.GetTorrentByID(ctx, release.Indexer.Identifier, release.TorrentID)
			if err != nil || torrentInfo == nil {
				l.Error().Err(err).Msgf("(%s) could not get torrent info from api: '%s' from: %s", f.Name, release.TorrentID, release.Indexer.Identifier)
				return false, err
			}

			l.Debug().Msgf("(%s) got torrent info from api: %+v", f.Name, torrentInfo)

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
			l.Trace().Msgf("(%s) preparing to download torrent metafile", f.Name)

			// if indexer doesn't have api, download torrent and add to tmpPath
			if err := s.downloadSvc.DownloadRelease(ctx, release); err != nil {
				l.Error().Err(err).Msgf("(%s) could not download torrent file with id: '%s' from: %s", f.Name, release.TorrentID, release.Indexer.Identifier)
				return false, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
			}
		}
	}

	sizeOk, err := f.CheckReleaseSize(release.Size)
	if err != nil {
		l.Error().Err(err).Msgf("(%s) error comparing release and filter size", f.Name)
		return false, err
	}

	// reset AdditionalSizeCheckRequired to not re-trigger check
	release.AdditionalSizeCheckRequired = false

	if !sizeOk {
		l.Debug().Msgf("(%s) filter did not match after additional size check, trying next", f.Name)
		return false, nil
	}

	return true, nil
}

func (s *service) AdditionalUploaderCheck(ctx context.Context, f *domain.Filter, release *domain.Release) (ok bool, err error) {
	defer func() {
		// try recover panic if anything went wrong with API or size checks
		errors.RecoverPanic(recover(), &err)
	}()

	// do additional check against indexer api
	l := s.log.With().Str("method", "AdditionalUploaderCheck").Logger()

	// if uploader was fetched before during size check we check it and return early
	if release.Uploader != "" {
		uploaderOk, err := f.CheckUploader(release.Uploader)
		if err != nil {
			l.Error().Err(err).Msgf("(%s) error comparing release and uploaders", f.Name)
			return false, err
		}

		// reset AdditionalUploaderCheckRequired to not re-trigger check
		release.AdditionalUploaderCheckRequired = false

		if !uploaderOk {
			l.Debug().Msgf("(%s) filter did not match after additional uploaders check, trying next", f.Name)
			return false, nil
		}

		return true, nil
	}

	l.Debug().Msgf("(%s) additional api uploader check required", f.Name)

	switch release.Indexer.Identifier {
	case "redacted", "ops", "mock":
		l.Trace().Msgf("(%s) preparing to check via api", f.Name)

		torrentInfo, err := s.apiService.GetTorrentByID(ctx, release.Indexer.Identifier, release.TorrentID)
		if err != nil || torrentInfo == nil {
			l.Error().Err(err).Msgf("(%s) could not get torrent info from api: '%s' from: %s", f.Name, release.TorrentID, release.Indexer.Identifier)
			return false, err
		}

		l.Debug().Msgf("(%s) got torrent info from api: %+v", f.Name, torrentInfo)

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
		l.Error().Err(err).Msgf("(%s) error comparing release and uploaders", f.Name)
		return false, err
	}

	// reset AdditionalUploaderCheckRequired to not re-trigger check
	release.AdditionalUploaderCheckRequired = false

	if !uploaderOk {
		l.Debug().Msgf("(%s) filter did not match after additional uploaders check, trying next", f.Name)
		return false, nil
	}

	return true, nil
}

func (s *service) AdditionalRecordLabelCheck(ctx context.Context, f *domain.Filter, release *domain.Release) (ok bool, err error) {
	defer func() {
		// try recover panic if anything went wrong with API or size checks
		errors.RecoverPanic(recover(), &err)
		if err != nil {
			ok = false
		}
	}()

	// do additional check against indexer api
	l := s.log.With().Str("method", "AdditionalRecordLabelCheck").Logger()

	// if record label was fetched before during size check or uploader check we check it and return early
	if release.RecordLabel != "" {
		recordLabelOk, err := f.CheckRecordLabel(release.RecordLabel)
		if err != nil {
			l.Error().Err(err).Msgf("(%s) error comparing release and record label", f.Name)
			return false, err
		}

		// reset AdditionalRecordLabelCheckRequired to not re-trigger check
		release.AdditionalRecordLabelCheckRequired = false

		if !recordLabelOk {
			l.Debug().Msgf("(%s) filter did not match after additional record label check, trying next", f.Name)
			return false, nil
		}

		return true, nil
	}

	l.Debug().Msgf("(%s) additional api record label check required", f.Name)

	switch release.Indexer.Identifier {
	case "redacted", "ops", "mock":
		l.Trace().Msgf("(%s) preparing to check via api", f.Name)

		torrentInfo, err := s.apiService.GetTorrentByID(ctx, release.Indexer.Identifier, release.TorrentID)
		if err != nil || torrentInfo == nil {
			l.Error().Err(err).Msgf("(%s) could not get torrent info from api: '%s' from: %s", f.Name, release.TorrentID, release.Indexer.Identifier)
			return false, err
		}

		l.Debug().Msgf("(%s) got torrent info from api: %+v", f.Name, torrentInfo)

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
		l.Error().Err(err).Msgf("(%s) error comparing release and record label", f.Name)
		return false, err
	}

	// reset AdditionalRecordLabelCheckRequired to not re-trigger check
	release.AdditionalRecordLabelCheckRequired = false

	if !recordLabelOk {
		l.Debug().Msgf("(%s) filter did not match after additional record label check, trying next", f.Name)
		return false, nil
	}

	return true, nil
}

func (s *service) CheckSmartEpisodeCanDownload(ctx context.Context, params *domain.SmartEpisodeParams) (bool, error) {
	return s.releaseRepo.CheckSmartEpisodeCanDownload(ctx, params)
}

func (s *service) CheckIsDuplicateRelease(ctx context.Context, profile *domain.DuplicateReleaseProfile, release *domain.Release) (bool, error) {
	return s.releaseRepo.CheckIsDuplicateRelease(ctx, profile, release)
}

func (s *service) RunExternalFilters(ctx context.Context, f *domain.Filter, externalFilters []domain.FilterExternal, release *domain.Release) (ok bool, err error) {
	defer func() {
		// try recover panic if anything went wrong with the external filter checks
		errors.RecoverPanic(recover(), &err)
		if err != nil {
			s.log.Error().Err(err).Msgf("filter %s external filter check panic", f.Name)
			ok = false
		}
	}()

	for _, external := range externalFilters {
		l := s.log.With().Str("method", "RunExternalFilters").Str("filter", f.Name).Str("external_filter", external.Name).Logger()

		if !external.Enabled {
			l.Debug().Msgf("external filter not enabled, skipping...")

			continue
		}

		if external.NeedTorrentDownloaded() {
			if err := s.downloadSvc.DownloadRelease(ctx, release); err != nil {
				return false, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
			}
		}

		switch external.Type {
		case domain.ExternalFilterTypeExec:
			// run external script
			exitCode, err := s.execCmd(ctx, external, release)
			if err != nil {
				l.Error().Err(err).Msgf("error executing external script")

				if external.OnError == domain.FilterExternalOnErrorContinue {
					l.Debug().Msgf("external script error, and OnError set to CONTINUE...")
					continue
				}
				return false, errors.Wrap(err, "error executing external command")
			}

			if exitCode != external.ExecExpectStatus {
				l.Debug().Int("expected_status", external.ExecExpectStatus).Int("actual_status", exitCode).Msgf("external script got unexpected exit code")
				f.RejectReasons.Add("external script exit code", exitCode, external.ExecExpectStatus)
				return false, nil
			}

		case domain.ExternalFilterTypeWebhook:
			// run external webhook
			statusCode, err := s.webhook(ctx, external, release)
			if err != nil {
				l.Error().Err(err).Msgf("error executing external webhook")

				// Only continue if the filter is configured to continue on error
				if external.OnError == domain.FilterExternalOnErrorContinue {
					l.Debug().Msgf("external webhook error, and OnError set to CONTINUE...")
					continue
				}
				return false, errors.Wrap(err, "error executing external webhook")
			}

			if statusCode != external.WebhookExpectStatus {
				l.Debug().Int("expected_status", external.WebhookExpectStatus).Int("actual_status", statusCode).Msgf("external webhook got unexpected status code")
				f.RejectReasons.Add("external webhook status code", statusCode, external.WebhookExpectStatus)
				return false, nil
			}
		}
	}

	return true, nil
}

func (s *service) execCmd(_ context.Context, external domain.FilterExternal, release *domain.Release) (int, error) {
	s.log.Trace().Msgf("filter exec release: %s", release.TorrentName)

	// read the file into bytes we can then use in the macro
	if len(release.TorrentDataRawBytes) == 0 && release.TorrentTmpFile != "" {
		if err := release.OpenTorrentFile(); err != nil {
			return 0, errors.Wrap(err, "could not open torrent file for release: %s", release.TorrentName)
		}
	}

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
	p := shellwords.NewParser()
	p.ParseBacktick = true
	commandArgs, err := p.Parse(parsedArgs)
	if err != nil {
		return 0, errors.Wrap(err, "could not parse into shell-words")
	}

	start := time.Now()

	// setup command and args
	command := exec.Command(cmd, commandArgs...)

	s.log.Debug().Msgf("script: %s args: %s", cmd, strings.Join(commandArgs, " "))

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

	execLogger := s.log.With().Str("release", release.TorrentName).Str("filter", release.FilterName).Logger()

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
			s.log.Debug().Msgf("filter script command exited with non zero code: %v", exitErr.ExitCode())
			return exitErr.ExitCode(), nil
		}

		s.log.Error().Err(err).Msg("error waiting for command")
		return 0, err
	}

	s.log.Debug().Msgf("executed external script: (%s), args: (%s) for release: (%s) indexer: (%s) total time (%s)", cmd, parsedArgs, release.TorrentName, release.Indexer.Name, duration)

	return 0, nil
}

func (s *service) webhook(ctx context.Context, external domain.FilterExternal, release *domain.Release) (int, error) {
	l := s.log.With().Str("method", "webhook").Str("external_filter", external.Name).Str("host", external.WebhookHost).Str("http_method", external.WebhookMethod).Logger()

	s.log.Trace().Msgf("preparing to run external webhook filter to: (%s) payload: (%s)", external.WebhookHost, external.WebhookData)

	if external.WebhookHost == "" {
		return 0, errors.New("external filter: missing host for webhook")
	}

	// if webhook data contains TorrentDataRawBytes, lets read the file into bytes we can then use in the macro
	if len(release.TorrentDataRawBytes) == 0 && strings.Contains(external.WebhookData, "TorrentDataRawBytes") {
		if err := release.OpenTorrentFile(); err != nil {
			return 0, errors.Wrap(err, "could not open torrent file for release: %s", release.TorrentName)
		}
	}

	m := domain.NewMacro(*release)

	// parse and replace values in argument string before continuing
	dataArgs, err := m.Parse(external.WebhookData)
	if err != nil {
		return 0, errors.Wrap(err, "could not parse webhook data macro: %s", external.WebhookData)
	}

	s.log.Trace().Msgf("sending %s to external webhook filter: (%s) payload: (%s)", external.WebhookMethod, external.WebhookHost, external.WebhookData)

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
		headers := strings.Split(external.WebhookHeaders, ";")

		for _, header := range headers {
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

			defer res.Body.Close()

			l.Debug().Int("status_code", res.StatusCode).Msg("filter external webhook response")

			if s.log.Debug().Enabled() {
				body, err := io.ReadAll(res.Body)
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

func (s *service) GetRateLimiter() *RateLimiter {
	return s.rateLimiter
}
