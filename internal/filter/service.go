// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package filter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/logger"
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
	Find(ctx context.Context, params domain.FilterQueryParams) ([]domain.Filter, error)
	CheckFilter(ctx context.Context, f *domain.Filter, release *domain.Release) (bool, error)
	ListFilters(ctx context.Context) ([]domain.Filter, error)
	Store(ctx context.Context, filter *domain.Filter) error
	Update(ctx context.Context, filter *domain.Filter) error
	UpdatePartial(ctx context.Context, filter domain.FilterUpdate) error
	Duplicate(ctx context.Context, filterID int) (*domain.Filter, error)
	ToggleEnabled(ctx context.Context, filterID int, enabled bool) error
	Delete(ctx context.Context, filterID int) error
	AdditionalSizeCheck(ctx context.Context, f *domain.Filter, release *domain.Release) (bool, error)
	CheckSmartEpisodeCanDownload(ctx context.Context, params *domain.SmartEpisodeParams) (bool, error)
	GetDownloadsByFilterId(ctx context.Context, filterID int) (*domain.FilterDownloads, error)
}

type service struct {
	log           zerolog.Logger
	repo          domain.FilterRepo
	actionService action.Service
	releaseRepo   domain.ReleaseRepo
	indexerSvc    indexer.Service
	apiService    indexer.APIService
	downloadSvc   *releasedownload.DownloadService

	httpClient *http.Client
}

func NewService(log logger.Logger, repo domain.FilterRepo, actionSvc action.Service, releaseRepo domain.ReleaseRepo, apiService indexer.APIService, indexerSvc indexer.Service, downloadSvc *releasedownload.DownloadService) Service {
	return &service{
		log:           log.With().Str("module", "filter").Logger(),
		repo:          repo,
		releaseRepo:   releaseRepo,
		actionService: actionSvc,
		apiService:    apiService,
		indexerSvc:    indexerSvc,
		downloadSvc:   downloadSvc,
		httpClient: &http.Client{
			Timeout:   time.Second * 120,
			Transport: sharedhttp.TransportTLSInsecure,
		},
	}
}

func (s *service) Find(ctx context.Context, params domain.FilterQueryParams) ([]domain.Filter, error) {
	// get filters
	filters, err := s.repo.Find(ctx, params)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find list filters")
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

func (s *service) ListFilters(ctx context.Context) ([]domain.Filter, error) {
	// get filters
	filters, err := s.repo.ListFilters(ctx)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find list filters")
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

	return filter, nil
}

func (s *service) FindByIndexerIdentifier(ctx context.Context, indexer string) ([]*domain.Filter, error) {
	// get filters for indexer
	filters, err := s.repo.FindByIndexerIdentifier(ctx, indexer)
	if err != nil {
		return nil, err
	}

	// we do not load actions here since we do not need it at this stage
	// only load those after filter has matched
	for _, filter := range filters {
		filter := filter

		externalFilters, err := s.repo.FindExternalFiltersByID(ctx, filter.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("could not find external filters for filter id: %v", filter.ID)
		}
		filter.External = externalFilters

	}

	return filters, nil
}

func (s *service) GetDownloadsByFilterId(ctx context.Context, filterID int) (*domain.FilterDownloads, error) {
	return s.GetDownloadsByFilterId(ctx, filterID)
}

func (s *service) Store(ctx context.Context, filter *domain.Filter) error {
	if err := filter.Validate(); err != nil {
		s.log.Error().Err(err).Msgf("invalid filter: %v", filter)
		return err
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

	return nil
}

func (s *service) CheckFilter(ctx context.Context, f *domain.Filter, release *domain.Release) (bool, error) {
	l := s.log.With().Str("method", "CheckFilter").Logger()

	l.Trace().Msgf("checking filter: %s %+v", f.Name, f)
	l.Trace().Msgf("checking filter: %s for release: %+v", f.Name, release)

	// do additional fetch to get download counts for filter
	if f.MaxDownloads > 0 {
		downloadCounts, err := s.repo.GetDownloadsByFilterId(ctx, f.ID)
		if err != nil {
			l.Error().Err(err).Msg("error getting download counters for filter")
			return false, nil
		}
		f.Downloads = downloadCounts
	}

	rejections, matchedFilter := f.CheckFilter(release)
	if len(rejections) > 0 {
		l.Debug().Msgf("(%s) for release: %v rejections: (%s)", f.Name, release.TorrentName, f.RejectionsString(true))
		return false, nil
	}

	if matchedFilter {
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
					f.AddRejectionF("smart episode check: not new: (%s) Daily: %d-%d-%d", release.Title, release.Year, release.Month, release.Day)
				} else {
					f.AddRejectionF("smart episode check: not new: (%s) season: %d ep: %d", release.Title, release.Season, release.Episode)
				}

				return false, nil
			}
		}

		// if matched, do additional size check if needed, attach actions and return the filter

		l.Debug().Msgf("found and matched filter: %s", f.Name)

		// If size constraints are set in a filter and the indexer did not
		// announce the size, we need to do an additional out of band size
		// check.
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

	// if no match, return nil
	return false, nil
}

// AdditionalSizeCheck performs additional out of band checks to determine the
// size of a torrent. Some indexers do not announce torrent size, so it is
// necessary to determine the size of the torrent in some other way. Some
// indexers have an API implemented to fetch this data. For those which don't,
// it is necessary to download the torrent file and parse it to make the size
// check. We use the API where available to minimize the number of torrents we
// need to download.
func (s *service) AdditionalSizeCheck(ctx context.Context, f *domain.Filter, release *domain.Release) (bool, error) {
	var err error
	defer func() {
		// try recover panic if anything went wrong with API or size checks
		errors.RecoverPanic(recover(), &err)
	}()

	// do additional size check against indexer api or torrent for size
	l := s.log.With().Str("method", "AdditionalSizeCheck").Logger()

	l.Debug().Msgf("(%s) additional size check required", f.Name)

	switch release.Indexer.Identifier {
	case "ptp", "btn", "ggn", "redacted", "ops", "mock":
		if release.Size == 0 {
			l.Trace().Msgf("(%s) preparing to check via api", f.Name)

			torrentInfo, err := s.apiService.GetTorrentByID(ctx, release.Indexer.Identifier, release.TorrentID)
			if err != nil || torrentInfo == nil {
				l.Error().Err(err).Msgf("(%s) could not get torrent info from api: '%s' from: %s", f.Name, release.TorrentID, release.Indexer.Identifier)
				return false, err
			}

			l.Debug().Msgf("(%s) got torrent info from api: %+v", f.Name, torrentInfo)

			release.Size = torrentInfo.ReleaseSizeBytes()
		}

	default:
		l.Trace().Msgf("(%s) preparing to download torrent metafile", f.Name)

		// if indexer doesn't have api, download torrent and add to tmpPath
		if err := s.downloadSvc.DownloadRelease(ctx, release); err != nil {
			l.Error().Err(err).Msgf("(%s) could not download torrent file with id: '%s' from: %s", f.Name, release.TorrentID, release.Indexer.Identifier)
			return false, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
		}
	}

	sizeOk, err := f.CheckReleaseSize(release.Size)
	if err != nil {
		l.Error().Err(err).Msgf("(%s) error comparing release and filter size", f.Name)
		return false, err
	}

	if !sizeOk {
		l.Debug().Msgf("(%s) filter did not match after additional size check, trying next", f.Name)
		return false, err
	}

	return true, nil
}

func (s *service) CheckSmartEpisodeCanDownload(ctx context.Context, params *domain.SmartEpisodeParams) (bool, error) {
	return s.releaseRepo.CheckSmartEpisodeCanDownload(ctx, params)
}

func (s *service) RunExternalFilters(ctx context.Context, f *domain.Filter, externalFilters []domain.FilterExternal, release *domain.Release) (bool, error) {
	var err error

	defer func() {
		// try recover panic if anything went wrong with the external filter checks
		errors.RecoverPanic(recover(), &err)
	}()

	// sort filters by index
	sort.Slice(externalFilters, func(i, j int) bool {
		return externalFilters[i].Index < externalFilters[j].Index
	})

	for _, external := range externalFilters {
		if !external.Enabled {
			s.log.Debug().Msgf("external filter %s not enabled, skipping...", external.Name)

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
				return false, errors.Wrap(err, "error executing external command")
			}

			if exitCode != external.ExecExpectStatus {
				s.log.Trace().Msgf("filter.Service.CheckFilter: external script unexpected exit code. got: %d want: %d", exitCode, external.ExecExpectStatus)
				f.AddRejectionF("external script unexpected exit code. got: %d want: %d", exitCode, external.ExecExpectStatus)
				return false, nil
			}

		case domain.ExternalFilterTypeWebhook:
			// run external webhook
			statusCode, err := s.webhook(ctx, external, release)
			if err != nil {
				return false, errors.Wrap(err, "error executing external webhook")
			}

			if statusCode != external.WebhookExpectStatus {
				s.log.Trace().Msgf("filter.Service.CheckFilter: external webhook unexpected status code. got: %d want: %d", statusCode, external.WebhookExpectStatus)
				f.AddRejectionF("external webhook unexpected status code. got: %d want: %d", statusCode, external.WebhookExpectStatus)
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

	var opts []retry.Option

	opts = append(opts, retry.DelayType(retry.FixedDelay))
	opts = append(opts, retry.LastErrorOnly(true))

	if external.WebhookRetryAttempts > 0 {
		opts = append(opts, retry.Attempts(uint(external.WebhookRetryAttempts)))
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
			res, err := s.httpClient.Do(clonereq)
			if err != nil {
				return 0, errors.Wrap(err, "could not make request for webhook")
			}

			defer res.Body.Close()

			s.log.Debug().Msgf("filter external webhook response status: %d", res.StatusCode)

			if s.log.Debug().Enabled() {
				body, err := io.ReadAll(res.Body)
				if err != nil {
					return res.StatusCode, errors.Wrap(err, "could not read request body")
				}

				if len(body) > 0 {
					s.log.Debug().Msgf("filter external webhook response status: %d body: %s", res.StatusCode, body)
				}
			}

			if utils.StrSliceContains(retryStatusCodes, strconv.Itoa(res.StatusCode)) {
				return 0, errors.New("webhook got unwanted status code: %d", res.StatusCode)
			}

			return res.StatusCode, nil
		},
		opts...)

	s.log.Debug().Msgf("successfully ran external webhook filter to: (%s) payload: (%s) finished in %s", external.WebhookHost, dataArgs, time.Since(start))

	return statusCode, err
}
