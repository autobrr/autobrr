// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package filter

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/utils"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/avast/retry-go/v4"
	"github.com/dustin/go-humanize"
	"github.com/mattn/go-shellwords"
	"github.com/rs/zerolog"
)

type Service interface {
	FindByID(ctx context.Context, filterID int) (*domain.Filter, error)
	FindByIndexerIdentifier(ctx context.Context, indexer string) ([]domain.Filter, error)
	Find(ctx context.Context, params domain.FilterQueryParams) ([]domain.Filter, error)
	CheckFilter(ctx context.Context, f domain.Filter, release *domain.Release) (bool, error)
	ListFilters(ctx context.Context) ([]domain.Filter, error)
	Store(ctx context.Context, filter *domain.Filter) error
	Update(ctx context.Context, filter *domain.Filter) error
	UpdatePartial(ctx context.Context, filter domain.FilterUpdate) error
	Duplicate(ctx context.Context, filterID int) (*domain.Filter, error)
	ToggleEnabled(ctx context.Context, filterID int, enabled bool) error
	Delete(ctx context.Context, filterID int) error
	AdditionalSizeCheck(ctx context.Context, f domain.Filter, release *domain.Release) (bool, error)
	CanDownloadShow(ctx context.Context, release *domain.Release) (bool, error)
	GetDownloadsByFilterId(ctx context.Context, filterID int) (*domain.FilterDownloads, error)
}

type service struct {
	log         zerolog.Logger
	repo        domain.FilterRepo
	actionRepo  domain.ActionRepo
	releaseRepo domain.ReleaseRepo
	indexerSvc  indexer.Service
	apiService  indexer.APIService
}

func NewService(log logger.Logger, repo domain.FilterRepo, actionRepo domain.ActionRepo, releaseRepo domain.ReleaseRepo, apiService indexer.APIService, indexerSvc indexer.Service) Service {
	return &service{
		log:         log.With().Str("module", "filter").Logger(),
		repo:        repo,
		actionRepo:  actionRepo,
		releaseRepo: releaseRepo,
		apiService:  apiService,
		indexerSvc:  indexerSvc,
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
	// find filter
	filter, err := s.repo.FindByID(ctx, filterID)
	if err != nil {
		return nil, err
	}

	// find actions and attach
	actions, err := s.actionRepo.FindByFilterID(ctx, filter.ID)
	if err != nil {
		s.log.Error().Msgf("could not find filter actions for filter id: %v", filter.ID)
	}
	filter.Actions = actions

	// find indexers and attach
	indexers, err := s.indexerSvc.FindByFilterID(ctx, filter.ID)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find indexers for filter: %v", filter.Name)
		return nil, err
	}
	filter.Indexers = indexers

	return filter, nil
}

func (s *service) FindByIndexerIdentifier(ctx context.Context, indexer string) ([]domain.Filter, error) {
	// get filters for indexer
	// we do not load actions here since we do not need it at this stage
	// only load those after filter has matched
	return s.repo.FindByIndexerIdentifier(ctx, indexer)
}

func (s *service) GetDownloadsByFilterId(ctx context.Context, filterID int) (*domain.FilterDownloads, error) {
	return s.GetDownloadsByFilterId(ctx, filterID)
}

func (s *service) Store(ctx context.Context, filter *domain.Filter) error {
	// validate data

	// store
	err := s.repo.Store(ctx, filter)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not store filter: %v", filter)
		return err
	}

	return nil
}

func (s *service) Update(ctx context.Context, filter *domain.Filter) error {
	// validate data
	if filter.Name == "" {
		return errors.New("validation: name can't be empty")
	}

	// replace newline with comma
	filter.Shows = strings.ReplaceAll(filter.Shows, "\n", ",")
	filter.Shows = strings.ReplaceAll(filter.Shows, ",,", ",")

	// update
	if err := s.repo.Update(ctx, filter); err != nil {
		s.log.Error().Err(err).Msgf("could not update filter: %s", filter.Name)
		return err
	}

	// take care of connected indexers
	if err := s.repo.StoreIndexerConnections(ctx, filter.ID, filter.Indexers); err != nil {
		s.log.Error().Err(err).Msgf("could not store filter indexer connections: %s", filter.Name)
		return err
	}

	// take care of connected external filters
	if err := s.repo.StoreFilterExternal(ctx, filter.ID, filter.External); err != nil {
		s.log.Error().Err(err).Msgf("could not store external filters: %s", filter.Name)
		return err
	}

	// take care of filter actions
	actions, err := s.actionRepo.StoreFilterActions(ctx, int64(filter.ID), filter.Actions)
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
		if _, err := s.actionRepo.StoreFilterActions(ctx, int64(filter.ID), filter.Actions); err != nil {
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
	if _, err := s.actionRepo.StoreFilterActions(ctx, int64(filter.ID), filter.Actions); err != nil {
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
	if err := s.actionRepo.DeleteByFilterID(ctx, filterID); err != nil {
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

func (s *service) CheckFilter(ctx context.Context, f domain.Filter, release *domain.Release) (bool, error) {

	s.log.Trace().Msgf("filter.Service.CheckFilter: checking filter: %s %+v", f.Name, f)
	s.log.Trace().Msgf("filter.Service.CheckFilter: checking filter: %s for release: %+v", f.Name, release)

	// do additional fetch to get download counts for filter
	if f.MaxDownloads > 0 {
		downloadCounts, err := s.repo.GetDownloadsByFilterId(ctx, f.ID)
		if err != nil {
			s.log.Error().Err(err).Msg("filter.Service.CheckFilter: error getting download counters for filter")
			return false, nil
		}
		f.Downloads = downloadCounts
	}

	rejections, matchedFilter := f.CheckFilter(release)
	if len(rejections) > 0 {
		s.log.Debug().Msgf("filter.Service.CheckFilter: (%s) for release: %v rejections: (%s)", f.Name, release.TorrentName, release.RejectionsString(true))
		return false, nil
	}

	if matchedFilter {
		// smartEpisode check
		if f.SmartEpisode {
			canDownloadShow, err := s.CanDownloadShow(ctx, release)
			if err != nil {
				s.log.Trace().Msgf("filter.Service.CheckFilter: failed smart episode check: %s", f.Name)
				return false, nil
			}

			if !canDownloadShow {
				s.log.Trace().Msgf("filter.Service.CheckFilter: failed smart episode check: %s", f.Name)
				release.AddRejectionF("smart episode check: not new: (%s) season: %d ep: %d", release.Title, release.Season, release.Episode)
				return false, nil
			}
		}

		// if matched, do additional size check if needed, attach actions and return the filter

		s.log.Debug().Msgf("filter.Service.CheckFilter: found and matched filter: %s", f.Name)

		// Some indexers do not announce the size and if size (min,max) is set in a filter then it will need
		// additional size check. Some indexers have api implemented to fetch this data and for the others
		// it will download the torrent file to parse and make the size check. This is all to minimize the amount of downloads.

		// do additional size check against indexer api or download torrent for size check
		if release.AdditionalSizeCheckRequired {
			s.log.Debug().Msgf("filter.Service.CheckFilter: (%s) additional size check required", f.Name)

			ok, err := s.AdditionalSizeCheck(ctx, f, release)
			if err != nil {
				s.log.Error().Err(err).Msgf("filter.Service.CheckFilter: (%s) additional size check error", f.Name)
				return false, err
			}

			if !ok {
				s.log.Trace().Msgf("filter.Service.CheckFilter: (%s) additional size check not matching what filter wanted", f.Name)
				return false, nil
			}
		}

		// run external filters
		if f.External != nil {
			externalOk, err := s.RunExternalFilters(ctx, f.External, release)
			if err != nil {
				s.log.Error().Err(err).Msgf("filter.Service.CheckFilter: (%s) external filter check error", f.Name)
				return false, err
			}

			if !externalOk {
				s.log.Trace().Msgf("filter.Service.CheckFilter: (%s) additional size check not matching what filter wanted", f.Name)
				return false, nil
			}
		}

		return true, nil
	}

	// if no match, return nil
	return false, nil
}

// AdditionalSizeCheck
// Some indexers do not announce the size and if size (min,max) is set in a filter then it will need
// additional size check. Some indexers have api implemented to fetch this data and for the others
// it will download the torrent file to parse and make the size check. This is all to minimize the amount of downloads.
func (s *service) AdditionalSizeCheck(ctx context.Context, f domain.Filter, release *domain.Release) (bool, error) {
	var err error
	defer func() {
		// try recover panic if anything went wrong with API or size checks
		errors.RecoverPanic(recover(), &err)
	}()

	// do additional size check against indexer api or torrent for size
	s.log.Debug().Msgf("filter.Service.AdditionalSizeCheck: (%s) additional size check required", f.Name)

	switch release.Indexer {
	case "ptp", "btn", "ggn", "redacted", "ops", "mock":
		if release.Size == 0 {
			s.log.Trace().Msgf("filter.Service.AdditionalSizeCheck: (%s) preparing to check via api", f.Name)

			torrentInfo, err := s.apiService.GetTorrentByID(ctx, release.Indexer, release.TorrentID)
			if err != nil || torrentInfo == nil {
				s.log.Error().Stack().Err(err).Msgf("filter.Service.AdditionalSizeCheck: (%s) could not get torrent info from api: '%s' from: %s", f.Name, release.TorrentID, release.Indexer)
				return false, err
			}

			s.log.Debug().Msgf("filter.Service.AdditionalSizeCheck: (%s) got torrent info from api: %+v", f.Name, torrentInfo)

			release.Size = torrentInfo.ReleaseSizeBytes()
		}

	default:
		s.log.Trace().Msgf("filter.Service.AdditionalSizeCheck: (%s) preparing to download torrent metafile", f.Name)

		// if indexer doesn't have api, download torrent and add to tmpPath
		if err := release.DownloadTorrentFileCtx(ctx); err != nil {
			s.log.Error().Stack().Err(err).Msgf("filter.Service.AdditionalSizeCheck: (%s) could not download torrent file with id: '%s' from: %s", f.Name, release.TorrentID, release.Indexer)
			return false, err
		}
	}

	// compare size against filter
	match, err := checkSizeFilter(f.MinSize, f.MaxSize, release.Size)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("filter.Service.AdditionalSizeCheck: (%s) error checking extra size filter", f.Name)
		return false, err
	}
	//no match, lets continue to next filter
	if !match {
		s.log.Debug().Msgf("filter.Service.AdditionalSizeCheck: (%s) filter did not match after additional size check, trying next", f.Name)
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

func (s *service) CanDownloadShow(ctx context.Context, release *domain.Release) (bool, error) {
	return s.releaseRepo.CanDownloadShow(ctx, release.Title, release.Season, release.Episode)
}

func (s *service) RunExternalFilters(ctx context.Context, externalFilters []domain.FilterExternal, release *domain.Release) (bool, error) {
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

		switch external.Type {
		case domain.ExternalFilterTypeExec:
			// run external script
			exitCode, err := s.execCmd(ctx, external, release)
			if err != nil {
				return false, errors.Wrap(err, "error executing external command")
			}

			if exitCode != external.ExecExpectStatus {
				s.log.Trace().Msgf("filter.Service.CheckFilter: external script unexpected exit code. got: %d want: %d", exitCode, external.ExecExpectStatus)
				release.AddRejectionF("external script unexpected exit code. got: %d want: %d", exitCode, external.ExecExpectStatus)
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
				release.AddRejectionF("external webhook unexpected status code. got: %d want: %d", statusCode, external.WebhookExpectStatus)
				return false, nil
			}
		}
	}

	return true, nil
}

func (s *service) execCmd(ctx context.Context, external domain.FilterExternal, release *domain.Release) (int, error) {
	s.log.Trace().Msgf("filter exec release: %s", release.TorrentName)

	if release.TorrentTmpFile == "" && strings.Contains(external.ExecArgs, "TorrentPathName") {
		if err := release.DownloadTorrentFileCtx(ctx); err != nil {
			return 0, errors.Wrap(err, "error downloading torrent file for release: %s", release.TorrentName)
		}
	}

	// read the file into bytes we can then use in the macro
	if len(release.TorrentDataRawBytes) == 0 && release.TorrentTmpFile != "" {
		t, err := os.ReadFile(release.TorrentTmpFile)
		if err != nil {
			return 0, errors.Wrap(err, "could not read torrent file: %s", release.TorrentTmpFile)
		}

		release.TorrentDataRawBytes = t
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

	s.log.Debug().Msgf("executed external script: (%s), args: (%s) for release: (%s) indexer: (%s) total time (%s)", cmd, parsedArgs, release.TorrentName, release.Indexer, duration)

	return 0, nil
}

func (s *service) webhook(ctx context.Context, external domain.FilterExternal, release *domain.Release) (int, error) {
	s.log.Trace().Msgf("preparing to run external webhook filter to: (%s) payload: (%s)", external.WebhookHost, external.WebhookData)

	if external.WebhookHost == "" {
		return 0, errors.New("external filter: missing host for webhook")
	}

	// if webhook data contains TorrentPathName or TorrentDataRawBytes, lets download the torrent file
	if release.TorrentTmpFile == "" && (strings.Contains(external.WebhookData, "TorrentPathName") || strings.Contains(external.WebhookData, "TorrentDataRawBytes")) {
		if err := release.DownloadTorrentFileCtx(ctx); err != nil {
			return 0, errors.Wrap(err, "webhook: could not download torrent file for release: %s", release.TorrentName)
		}
	}

	// if webhook data contains TorrentDataRawBytes, lets read the file into bytes we can then use in the macro
	if len(release.TorrentDataRawBytes) == 0 && strings.Contains(external.WebhookData, "TorrentDataRawBytes") {
		t, err := os.ReadFile(release.TorrentTmpFile)
		if err != nil {
			return 0, errors.Wrap(err, "could not read torrent file: %s", release.TorrentTmpFile)
		}

		release.TorrentDataRawBytes = t
	}

	m := domain.NewMacro(*release)

	// parse and replace values in argument string before continuing
	dataArgs, err := m.Parse(external.WebhookData)
	if err != nil {
		return 0, errors.Wrap(err, "could not parse webhook data macro: %s", external.WebhookData)
	}

	s.log.Trace().Msgf("sending %s to external webhook filter: (%s) payload: (%s)", external.WebhookMethod, external.WebhookHost, external.WebhookData)

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := http.Client{Transport: t, Timeout: 120 * time.Second}

	method := http.MethodPost
	if external.WebhookMethod != "" {
		method = external.WebhookMethod
	}

	var req *http.Request
	if external.WebhookData != "" && dataArgs != "" {
		req, err = http.NewRequestWithContext(ctx, method, external.WebhookHost, bytes.NewBufferString(dataArgs))
		if err != nil {
			return 0, errors.Wrap(err, "could not build request for webhook")
		}

		defer req.Body.Close()
	} else {
		req, err = http.NewRequestWithContext(ctx, method, external.WebhookHost, nil)
		if err != nil {
			return 0, errors.Wrap(err, "could not build request for webhook")
		}
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
			req.Header.Add(http.CanonicalHeaderKey(h[0]), h[1])
		}
	}

	var opts []retry.Option

	if external.WebhookRetryAttempts > 0 {
		option := retry.Attempts(uint(external.WebhookRetryAttempts))
		opts = append(opts, option)
	}
	if external.WebhookRetryDelaySeconds > 0 {
		option := retry.Delay(time.Duration(external.WebhookRetryDelaySeconds) * time.Second)
		opts = append(opts, option)
	}
	if external.WebhookRetryMaxJitterSeconds > 0 {
		option := retry.MaxJitter(time.Duration(external.WebhookRetryMaxJitterSeconds) * time.Second)
		opts = append(opts, option)
	}

	start := time.Now()

	var retryStatusCodes []string
	if external.WebhookRetryStatus != "" {
		retryStatusCodes = strings.Split(strings.ReplaceAll(external.WebhookRetryStatus, " ", ""), ",")
	}

	statusCode, err := retry.DoWithData(
		func() (int, error) {
			clonereq := req.Clone(ctx)
			clonereq.Body = io.NopCloser(req.Body)
			res, err := client.Do(clonereq)
			if err != nil {
				return 0, errors.Wrap(err, "could not make request for webhook")
			}

			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				return 0, errors.Wrap(err, "could not read request body")
			}

			if len(body) > 0 {
				s.log.Debug().Msgf("filter external webhook response status: %d body: %s", res.StatusCode, body)
			}

			if utils.StrSliceContains(retryStatusCodes, strconv.Itoa(res.StatusCode)) {
				return 0, errors.New("retrying webhook request, got status code: %d", res.StatusCode)
			}

			return res.StatusCode, nil
		},
		opts...)

	s.log.Debug().Msgf("successfully ran external webhook filter to: (%s) payload: (%s) finished in %s", external.WebhookHost, dataArgs, time.Since(start))

	return statusCode, err
}
