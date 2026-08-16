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
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

func (s *Service) RunAction(ctx context.Context, action *domain.Action, release *domain.Release) (rejections []string, err error) {
	l := s.log.With().Str("trace_id", release.TraceID).Str("action", action.Name).Str("action_type", string(action.Type)).Str("release", release.TorrentName).Logger()

	// executors are only invoked from here, so they pick this logger up with zerolog.Ctx
	ctx = l.WithContext(ctx)

	defer func() {
		errors.RecoverPanic(recover(), &err)
		if err != nil {
			l.Error().Err(err).Msg("recovering from panic in run action")
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
		s.test(ctx)

	case domain.ActionTypeExec:
		err = s.execCmd(ctx, action, release)

	case domain.ActionTypeWatchFolder:
		err = s.watchFolder(ctx, action, release)

	case domain.ActionTypeWebhook:
		err = s.webhook(ctx, action)

	case domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2:
		rejections, err = s.deluge(ctx, action, release)

	case domain.ActionTypeQbittorrent:
		rejections, err = s.qbittorrent(ctx, action, release)

	case domain.ActionTypeRTorrent:
		rejections, err = s.rtorrent(ctx, action, release)

	case domain.ActionTypeTransmission:
		rejections, err = s.transmission(ctx, action, release)

	case domain.ActionTypePorla:
		rejections, err = s.porla(ctx, action, release)

	case domain.ActionTypeAria2:
		rejections, err = s.aria2(ctx, action, release)

	case domain.ActionTypeRadarr:
		rejections, err = s.radarr(ctx, action, release)

	case domain.ActionTypeSonarr:
		rejections, err = s.sonarr(ctx, action, release)

	case domain.ActionTypeLidarr:
		rejections, err = s.lidarr(ctx, action, release)

	case domain.ActionTypeWhisparr, domain.ActionTypeWhisparrV3:
		rejections, err = s.whisparr(ctx, action, release)

	case domain.ActionTypeReadarr:
		rejections, err = s.readarr(ctx, action, release)

	case domain.ActionTypeSportarr:
		rejections, err = s.sportarr(ctx, action, release)

	case domain.ActionTypeSabnzbd:
		rejections, err = s.sabnzbd(ctx, action, release)

	case domain.ActionTypeNzbget:
		rejections, err = s.nzbget(ctx, action, release)

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
		l.Error().Err(err).Msg("process action failed")

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

func (s *Service) CheckActionPreconditions(ctx context.Context, action *domain.Action, release *domain.Release) error {
	if err := s.downloadSvc.ResolveMagnetURI(ctx, release); err != nil {
		return errors.Wrap(err, "could not resolve magnet uri: %s", release.MagnetURI)
	}

	// Fetch the torrent once, here, onto the shared release. The action executors
	// take the release by value, so a download started inside one of them lands
	// on a copy and the next action would fetch the same torrent again.
	// Magnets carry no torrent to fetch, the clients get the URI instead.
	if !release.HasMagnetUri() && (action.NeedsTorrentDownloaded() || action.CheckMacrosNeedRawDataBytes(release)) {
		if err := s.downloadSvc.DownloadRelease(ctx, release); err != nil {
			return errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
		}
	}

	// the path macros hand a file to an external script or webhook, so those
	// need the in-memory torrent written to disk first
	if action.CheckMacrosNeedTorrentTmpFile(release) {
		if err := release.WriteTemporaryFile(); err != nil {
			return errors.Wrap(err, "could not write torrent file for release: %s", release.TorrentName)
		}
	}

	return nil
}

func (s *Service) test(ctx context.Context) {
	l := zerolog.Ctx(ctx)

	l.Debug().Msg("running Test action")

	l.Info().Msg("test action success")
}

func (s *Service) watchFolder(ctx context.Context, action *domain.Action, release *domain.Release) error {
	l := zerolog.Ctx(ctx)

	if release.HasMagnetUri() {
		return fmt.Errorf("action watch folder does not support magnet links: %s", release.TorrentName)
	}

	l.Debug().Str("watch_folder", action.WatchFolder).Str("release", release.TorrentName).Msg("running Watch Folder action")

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
		// The torrent is no longer backed by a tmp file, so name it after the
		// infohash. It is unique, safe to use as a file name as it stands, and
		// set alongside the raw bytes. Anything richer belongs in a client or
		// one of our other tools rather than in the file name.
		newFileName = filepath.Join(action.WatchFolder, "autobrr-"+release.TorrentHash+".torrent")
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
	if _, err := io.Copy(newFile, release.TorrentReader()); err != nil {
		return errors.Wrap(err, "could not copy file %v to watch folder", newFileName)
	}

	l.Info().Str("file", newFileName).Msg("saved file to watch folder")

	return nil
}

func (s *Service) webhook(ctx context.Context, action *domain.Action) error {
	l := zerolog.Ctx(ctx)

	l.Debug().Msg("running Webhook action")

	l.Trace().Str("host", action.WebhookHost).Str("payload", truncString(action.WebhookData, 1024)).Msg("running Webhook action")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, action.WebhookHost, bytes.NewBufferString(action.WebhookData))
	if err != nil {
		return errors.Wrap(err, "could not build request for webhook")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	start := time.Now()
	res, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not make request for webhook")
	}

	defer sharedhttp.DrainAndClose(res)

	l.Info().Str("host", action.WebhookHost).Str("payload", truncString(action.WebhookData, 256)).Dur("duration", time.Since(start)).Msg("webhook action executed")

	return nil
}

func truncString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
