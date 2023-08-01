// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package filter

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

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
	Store(ctx context.Context, filter domain.Filter) (*domain.Filter, error)
	Update(ctx context.Context, filter domain.Filter) (*domain.Filter, error)
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
	filters, err := s.repo.FindByIndexerIdentifier(ctx, indexer)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find filters for indexer: %v", indexer)
		return nil, err
	}

	return filters, nil
}

func (s *service) GetDownloadsByFilterId(ctx context.Context, filterID int) (*domain.FilterDownloads, error) {
	return s.GetDownloadsByFilterId(ctx, filterID)
}

func (s *service) Store(ctx context.Context, filter domain.Filter) (*domain.Filter, error) {
	// validate data

	// store
	f, err := s.repo.Store(ctx, filter)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not store filter: %v", filter)
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
		s.log.Error().Err(err).Msgf("could not update filter: %v", filter.Name)
		return nil, err
	}

	// take care of connected indexers
	if err = s.repo.StoreIndexerConnections(ctx, f.ID, filter.Indexers); err != nil {
		s.log.Error().Err(err).Msgf("could not store filter indexer connections: %v", filter.Name)
		return nil, err
	}

	// take care of filter actions
	actions, err := s.actionRepo.StoreFilterActions(ctx, filter.Actions, int64(filter.ID))
	if err != nil {
		s.log.Error().Err(err).Msgf("could not store filter actions: %v", filter.Name)
		return nil, err
	}

	f.Actions = actions

	return f, nil
}

func (s *service) UpdatePartial(ctx context.Context, filter domain.FilterUpdate) error {

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

	if filter.Actions != nil {
		// take care of filter actions
		if _, err := s.actionRepo.StoreFilterActions(ctx, filter.Actions, int64(filter.ID)); err != nil {
			s.log.Error().Err(err).Msgf("could not store filter actions: %v", filter.ID)
			return err
		}
	}

	return nil
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
		s.log.Error().Msgf("could not find filter actions: %+v", &filterID)
		return nil, err
	}

	// find indexers and attach
	filterIndexers, err := s.indexerSvc.FindByFilterID(ctx, filterID)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find indexers for filter: %+v", &baseFilter.Name)
		return nil, err
	}

	// update
	filter, err := s.repo.Store(ctx, *baseFilter)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not update filter: %v", baseFilter.Name)
		return nil, err
	}

	// take care of connected indexers
	if err = s.repo.StoreIndexerConnections(ctx, filter.ID, filterIndexers); err != nil {
		s.log.Error().Err(err).Msgf("could not store filter indexer connections: %v", filter.Name)
		return nil, err
	}
	filter.Indexers = filterIndexers

	// reset action id to 0
	for i, a := range filterActions {
		a.ID = 0
		filterActions[i] = a
	}

	// take care of filter actions
	actions, err := s.actionRepo.StoreFilterActions(ctx, filterActions, int64(filter.ID))
	if err != nil {
		s.log.Error().Err(err).Msgf("could not store filter actions: %v", filter.Name)
		return nil, err
	}

	filter.Actions = actions

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

	// delete filter
	if err := s.repo.Delete(ctx, filterID); err != nil {
		s.log.Error().Err(err).Msgf("could not delete filter: %v", filterID)
		return err
	}

	return nil
}

func (s *service) CheckFilter(ctx context.Context, f domain.Filter, release *domain.Release) (bool, error) {

	s.log.Trace().Msgf("filter.Service.CheckFilter: checking filter: %v %+v", f.Name, f)
	s.log.Trace().Msgf("filter.Service.CheckFilter: checking filter: %v for release: %+v", f.Name, release)

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
		s.log.Debug().Msgf("filter.Service.CheckFilter: (%v) for release: %v rejections: (%v)", f.Name, release.TorrentName, release.RejectionsString(true))
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

		s.log.Debug().Msgf("filter.Service.CheckFilter: found and matched filter: %+v", f.Name)

		// Some indexers do not announce the size and if size (min,max) is set in a filter then it will need
		// additional size check. Some indexers have api implemented to fetch this data and for the others
		// it will download the torrent file to parse and make the size check. This is all to minimize the amount of downloads.

		// do additional size check against indexer api or download torrent for size check
		if release.AdditionalSizeCheckRequired {
			s.log.Debug().Msgf("filter.Service.CheckFilter: (%v) additional size check required", f.Name)

			ok, err := s.AdditionalSizeCheck(ctx, f, release)
			if err != nil {
				s.log.Error().Stack().Err(err).Msgf("filter.Service.CheckFilter: (%v) additional size check error", f.Name)
				return false, err
			}

			if !ok {
				s.log.Trace().Msgf("filter.Service.CheckFilter: (%v) additional size check not matching what filter wanted", f.Name)
				return false, nil
			}
		}

		// run external script
		if f.ExternalScriptEnabled && f.ExternalScriptCmd != "" {
			exitCode, err := s.execCmd(ctx, release, f.ExternalScriptCmd, f.ExternalScriptArgs)
			if err != nil {
				s.log.Error().Err(err).Msgf("filter.Service.CheckFilter: error executing external command for filter: %+v", f.Name)
				return false, err
			}

			if exitCode != f.ExternalScriptExpectStatus {
				s.log.Trace().Msgf("filter.Service.CheckFilter: external script unexpected exit code. got: %v want: %v", exitCode, f.ExternalScriptExpectStatus)
				release.AddRejectionF("external script unexpected exit code. got: %v want: %v", exitCode, f.ExternalScriptExpectStatus)
				return false, nil
			}
		}

		// run external webhook
		if f.ExternalWebhookEnabled && f.ExternalWebhookHost != "" && f.ExternalWebhookData != "" {
			// run external scripts
			statusCode, err := s.webhook(ctx, release, f.ExternalWebhookHost, f.ExternalWebhookData)
			if err != nil {
				s.log.Error().Err(err).Msgf("filter.Service.CheckFilter: error executing external webhook for filter: %v", f.Name)
				return false, err
			}

			if statusCode != f.ExternalWebhookExpectStatus {
				s.log.Trace().Msgf("filter.Service.CheckFilter: external webhook unexpected status code. got: %v want: %v", statusCode, f.ExternalWebhookExpectStatus)
				release.AddRejectionF("external webhook unexpected status code. got: %v want: %v", statusCode, f.ExternalWebhookExpectStatus)
				return false, nil
			}
		}

		// found matching filter, lets find the filter actions and attach
		actions, err := s.actionRepo.FindByFilterID(ctx, f.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("filter.Service.CheckFilter: error finding actions for filter: %+v", f.Name)
			return false, err
		}

		// if no actions, continue to next filter
		if len(actions) == 0 {
			s.log.Trace().Msgf("filter.Service.CheckFilter: no actions found for filter '%v', trying next one..", f.Name)
			return false, nil
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

func (s *service) execCmd(ctx context.Context, release *domain.Release, cmd string, args string) (int, error) {
	s.log.Debug().Msgf("filter exec release: %v", release.TorrentName)

	if release.TorrentTmpFile == "" && strings.Contains(args, "TorrentPathName") {
		if err := release.DownloadTorrentFileCtx(ctx); err != nil {
			return 0, errors.Wrap(err, "error downloading torrent file for release: %v", release.TorrentName)
		}
	}

	// read the file into bytes we can then use in the macro
	if len(release.TorrentDataRawBytes) == 0 && release.TorrentTmpFile != "" {
		t, err := os.ReadFile(release.TorrentTmpFile)
		if err != nil {
			return 0, errors.Wrap(err, "could not read torrent file: %v", release.TorrentTmpFile)
		}

		release.TorrentDataRawBytes = t
	}

	// check if program exists
	cmd, err := exec.LookPath(cmd)
	if err != nil {
		return 0, errors.Wrap(err, "exec failed, could not find program: %v", cmd)
	}

	// handle args and replace vars
	m := domain.NewMacro(*release)

	// parse and replace values in argument string before continuing
	parsedArgs, err := m.Parse(args)
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

	err = command.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		s.log.Debug().Msgf("filter script command exited with non zero code: %v", exitErr.ExitCode())
		return exitErr.ExitCode(), nil
	}

	duration := time.Since(start)

	s.log.Debug().Msgf("executed external script: (%v), args: (%v) for release: (%v) indexer: (%v) total time (%v)", cmd, args, release.TorrentName, release.Indexer, duration)

	return 0, nil
}

func (s *service) webhook(ctx context.Context, release *domain.Release, url string, data string) (int, error) {
	s.log.Debug().Msgf("preparing to run external webhook filter to: (%s) payload: (%s)", url, data)

	// if webhook data contains TorrentPathName or TorrentDataRawBytes, lets download the torrent file
	if release.TorrentTmpFile == "" && (strings.Contains(data, "TorrentPathName") || strings.Contains(data, "TorrentDataRawBytes")) {
		if err := release.DownloadTorrentFileCtx(ctx); err != nil {
			return 0, errors.Wrap(err, "webhook: could not download torrent file for release: %s", release.TorrentName)
		}
	}

	// if webhook data contains TorrentDataRawBytes, lets read the file into bytes we can then use in the macro
	if len(release.TorrentDataRawBytes) == 0 && strings.Contains(data, "TorrentDataRawBytes") {
		t, err := os.ReadFile(release.TorrentTmpFile)
		if err != nil {
			return 0, errors.Wrap(err, "could not read torrent file: %s", release.TorrentTmpFile)
		}

		release.TorrentDataRawBytes = t
	}

	m := domain.NewMacro(*release)

	// parse and replace values in argument string before continuing
	dataArgs, err := m.Parse(data)
	if err != nil {
		return 0, errors.Wrap(err, "could not parse webhook data macro: %s", data)
	}

	s.log.Debug().Msgf("sending POST to external webhook filter: (%s) payload: (%s)", url, data)

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := http.Client{Transport: t, Timeout: 120 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(dataArgs))
	if err != nil {
		return 0, errors.Wrap(err, "could not build request for webhook")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	start := time.Now()

	res, err := client.Do(req)
	if err != nil {
		return 0, errors.Wrap(err, "could not make request for webhook")
	}

	defer res.Body.Close()

	if res.StatusCode > 299 {
		return res.StatusCode, nil
	}

	s.log.Debug().Msgf("successfully ran external webhook filter to: (%s) payload: (%s) finished in %s", url, dataArgs, time.Since(start))

	return res.StatusCode, nil
}
