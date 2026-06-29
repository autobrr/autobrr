// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/utils"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/avast/retry-go/v4"
)

func (s *service) RunAction(ctx context.Context, action *domain.Action, release *domain.Release) (rejections []string, err error) {
	defer func() {
		errors.RecoverPanic(recover(), &err)
		if err != nil {
			s.log.Error().Err(err).Msgf("recovering from panic in run action %s", action.Name)
		}
	}()

	// Check preconditions: download torrent file if needed
	if err := s.CheckActionPreconditions(ctx, action, release); err != nil {
		return nil, err
	}

	// parse all macros in one go
	if err := action.ParseMacros(release); err != nil {
		return nil, err
	}

	switch action.Type {
	case domain.ActionTypeTest:
		s.test(action.Name)

	case domain.ActionTypeExec:
		err = s.execCmd(ctx, action, *release)

	case domain.ActionTypeWatchFolder:
		err = s.watchFolder(ctx, action, *release)

	case domain.ActionTypeWebhook:
		err = s.webhook(ctx, action, *release)

	case domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2:
		rejections, err = s.deluge(ctx, action, *release)

	case domain.ActionTypeQbittorrent:
		rejections, err = s.qbittorrent(ctx, action, *release)

	case domain.ActionTypeRTorrent:
		rejections, err = s.rtorrent(ctx, action, *release)

	case domain.ActionTypeTransmission:
		rejections, err = s.transmission(ctx, action, *release)

	case domain.ActionTypePorla:
		rejections, err = s.porla(ctx, action, *release)

	case domain.ActionTypeRadarr:
		rejections, err = s.radarr(ctx, action, *release)

	case domain.ActionTypeSonarr:
		rejections, err = s.sonarr(ctx, action, *release)

	case domain.ActionTypeLidarr:
		rejections, err = s.lidarr(ctx, action, *release)

	case domain.ActionTypeWhisparr:
		rejections, err = s.whisparr(ctx, action, *release)

	case domain.ActionTypeReadarr:
		rejections, err = s.readarr(ctx, action, *release)

	case domain.ActionTypeSabnzbd:
		rejections, err = s.sabnzbd(ctx, action, *release)

	case domain.ActionTypeNzbget:
		rejections, err = s.nzbget(ctx, action, *release)

	default:
		return nil, errors.New("unsupported action type: %s", action.Type)
	}

	payload := &domain.NotificationPayload{
		Event:          domain.NotificationEventPushApproved,
		ReleaseName:    release.TorrentName,
		Filter:         release.FilterName,
		FilterID:       release.FilterID,
		Indexer:        release.Indexer.Name,
		InfoHash:       release.TorrentHash,
		Size:           release.Size,
		Status:         domain.ReleasePushStatusApproved,
		Action:         action.Name,
		ActionType:     action.Type,
		Rejections:     []string{},
		Protocol:       release.Protocol,
		Implementation: release.Implementation,
		Timestamp:      time.Now(),
		Release:        release,
	}

	if action.Client != nil {
		payload.ActionClient = action.Client.Name
	}

	if err != nil {
		s.log.Error().Err(err).Msgf("process action failed: %v for '%v'", action.Name, release.TorrentName)

		payload.Event = domain.NotificationEventPushError
		payload.Status = domain.ReleasePushStatusErr
		payload.Rejections = []string{err.Error()}
	}

	if rejections != nil {
		payload.Event = domain.NotificationEventPushRejected
		payload.Status = domain.ReleasePushStatusRejected
		payload.Rejections = rejections
	}

	// send separate event for notifications
	s.bus.Publish(domain.EventNotificationSend, &payload.Event, payload)

	return rejections, err
}

func (s *service) CheckActionPreconditions(ctx context.Context, action *domain.Action, release *domain.Release) error {
	if err := s.downloadSvc.ResolveMagnetURI(ctx, release); err != nil {
		return errors.Wrap(err, "could not resolve magnet uri: %s", release.MagnetURI)
	}

	// parse all macros in one go
	if action.CheckMacrosNeedTorrentTmpFile(release) {
		if err := s.downloadSvc.DownloadRelease(ctx, release); err != nil {
			return errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
		}
	}

	if action.CheckMacrosNeedRawDataBytes(release) {
		if err := release.OpenTorrentFile(); err != nil {
			return errors.Wrap(err, "could not open torrent file for release: %s", release.TorrentName)
		}
	}

	return nil
}

func (s *service) test(name string) {
	s.log.Info().Msgf("action TEST: %v", name)
}

func (s *service) watchFolder(ctx context.Context, action *domain.Action, release domain.Release) error {
	if release.HasMagnetUri() {
		return fmt.Errorf("action watch folder does not support magnet links: %s", release.TorrentName)
	}

	s.log.Trace().Msgf("action WATCH_FOLDER: %v file: %v", action.WatchFolder, release.TorrentTmpFile)

	if len(release.TorrentDataRawBytes) < 1 {
		return fmt.Errorf("watch_folder: missing torrent %s", release.TorrentName)
	}

	// default dir to watch folder
	//  /mnt/watch/{{.Indexer}}
	//  /mnt/watch/mock
	//  /mnt/watch/{{.Indexer}}-{{.TorrentName}}.torrent
	//  /mnt/watch/mock-Torrent.Name-GROUP.torrent
	dir := action.WatchFolder
	newFileName := action.WatchFolder

	// if watchFolderArgs does not contain .torrent, create
	if !strings.HasSuffix(action.WatchFolder, ".torrent") {
		_, tmpFileName := filepath.Split(release.TorrentTmpFile)

		newFileName = filepath.Join(action.WatchFolder, tmpFileName+".torrent")
	} else {
		dir, _ = filepath.Split(action.WatchFolder)
	}

	// Create folder
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return errors.Wrap(err, "could not create new folders %v", dir)
	}

	// Create new file
	newFile, err := os.Create(newFileName)
	if err != nil {
		return errors.Wrap(err, "could not create new file %v", newFileName)
	}
	defer newFile.Close()

	// Copy file
	if _, err := io.Copy(newFile, bytes.NewReader(release.TorrentDataRawBytes)); err != nil {
		return errors.Wrap(err, "could not copy file %v to watch folder", newFileName)
	}

	s.log.Info().Msgf("saved file to watch folder: %v", newFileName)

	return nil
}

func (s *service) webhook(ctx context.Context, action *domain.Action, release domain.Release) error {
	s.log.Trace().Msgf("action WEBHOOK: '%s' file: %s", action.Name, release.TorrentName)
	if len(action.WebhookData) > 1024 {
		s.log.Trace().Msgf("webhook action '%s' - host: %s data: %s", action.Name, action.WebhookHost, action.WebhookData[:1024])
	} else {
		s.log.Trace().Msgf("webhook action '%s' - host: %s data: %s", action.Name, action.WebhookHost, action.WebhookData)
	}

	if action.WebhookHost == "" {
		return errors.New("webhook action: missing host for webhook")
	}

	// default to POST but allow the method to be configured per action
	method := http.MethodPost
	if action.WebhookMethod != "" {
		method = action.WebhookMethod
	}

	req, err := http.NewRequestWithContext(ctx, method, action.WebhookHost, nil)
	if err != nil {
		return errors.Wrap(err, "could not build request for webhook")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	// custom headers in the form of "Key=Value"
	for _, header := range action.WebhookHeaders {
		h := strings.SplitN(header, "=", 2)
		if len(h) != 2 {
			continue
		}

		// go already canonicalizes the provided header key.
		req.Header.Set(strings.TrimSpace(h[0]), strings.TrimSpace(h[1]))
	}

	retryAttempts := action.WebhookRetryAttempts
	if retryAttempts == 0 {
		retryAttempts = 1
	}

	opts := []retry.Option{
		retry.DelayType(retry.FixedDelay),
		retry.LastErrorOnly(true),
		retry.Attempts(uint(retryAttempts)),
	}

	if action.WebhookRetryDelaySeconds > 0 {
		opts = append(opts, retry.Delay(time.Duration(action.WebhookRetryDelaySeconds)*time.Second))
	}

	var retryStatusCodes []string
	if action.WebhookRetryStatus != "" {
		retryStatusCodes = strings.Split(strings.ReplaceAll(action.WebhookRetryStatus, " ", ""), ",")
	}

	start := time.Now()

	statusCode, err := retry.DoWithData(
		func() (int, error) {
			clonereq := req.Clone(ctx)
			if action.WebhookData != "" {
				// set Body, ContentLength and GetBody explicitly so the request is
				// sent with a Content-Length header (not chunked) and can be replayed.
				clonereq.Body = io.NopCloser(bytes.NewBufferString(action.WebhookData))
				clonereq.ContentLength = int64(len(action.WebhookData))
				clonereq.GetBody = func() (io.ReadCloser, error) {
					return io.NopCloser(bytes.NewBufferString(action.WebhookData)), nil
				}
			}

			res, err := s.httpClient.Do(clonereq)
			if err != nil {
				return 0, errors.Wrap(err, "could not make request for webhook")
			}

			defer sharedhttp.DrainAndClose(res)

			// if the status code is in the retry list, return an error to trigger a retry
			if utils.StrSliceContains(retryStatusCodes, strconv.Itoa(res.StatusCode)) {
				return 0, errors.New("webhook got unwanted status code: %d", res.StatusCode)
			}

			return res.StatusCode, nil
		},
		opts...)
	if err != nil {
		return errors.Wrap(err, "could not make request for webhook")
	}

	// if an expected status code is configured, treat a mismatch as a failure
	if action.WebhookExpectStatus > 0 && statusCode != action.WebhookExpectStatus {
		return errors.New("webhook action '%s' got unexpected status code: %d (expected %d)", action.Name, statusCode, action.WebhookExpectStatus)
	}

	if len(action.WebhookData) > 256 {
		s.log.Info().Msgf("successfully ran webhook action: '%s' to: %s payload: %s finished in %s", action.Name, action.WebhookHost, action.WebhookData[:256], time.Since(start))
	} else {
		s.log.Info().Msgf("successfully ran webhook action: '%s' to: %s payload: %s finished in %s", action.Name, action.WebhookHost, action.WebhookData, time.Since(start))
	}

	return nil
}
