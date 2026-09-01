// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

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
		s.runTest(ctx)

	case domain.ActionTypeExec:
		err = s.runExecCmd(ctx, action, release)

	case domain.ActionTypeWatchFolder:
		err = s.runWatchFolder(ctx, action, release)

	case domain.ActionTypeWebhook:
		err = s.runWebhook(ctx, action)

	case domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2:
		rejections, err = s.runDeluge(ctx, action, release)

	case domain.ActionTypeQbittorrent:
		rejections, err = s.runQbittorrent(ctx, action, release)

	case domain.ActionTypeRTorrent:
		rejections, err = s.runRTorrent(ctx, action, release)

	case domain.ActionTypeTransmission:
		rejections, err = s.runTransmission(ctx, action, release)

	case domain.ActionTypePorla:
		rejections, err = s.runPorla(ctx, action, release)

	case domain.ActionTypeAria2:
		rejections, err = s.runAria2(ctx, action, release)

	case domain.ActionTypeRadarr:
		rejections, err = s.runRadarr(ctx, action, release)

	case domain.ActionTypeSonarr:
		rejections, err = s.runSonarr(ctx, action, release)

	case domain.ActionTypeLidarr:
		rejections, err = s.runLidarr(ctx, action, release)

	case domain.ActionTypeWhisparr, domain.ActionTypeWhisparrV3:
		rejections, err = s.runWhisparr(ctx, action, release)

	case domain.ActionTypeReadarr:
		rejections, err = s.runReadarr(ctx, action, release)

	case domain.ActionTypeSportarr:
		rejections, err = s.runSportarr(ctx, action, release)

	case domain.ActionTypeSabnzbd:
		rejections, err = s.runSabnzbd(ctx, action, release)

	case domain.ActionTypeNzbget:
		rejections, err = s.runNzbget(ctx, action, release)

	default:
		return nil, errors.New("unsupported action type: %s", action.Type)
	}

	return rejections, err
}

func (s *Service) CheckActionPreconditions(ctx context.Context, action *domain.Action, release *domain.Release) error {
	if err := s.rlsDownloadSvc.ResolveMagnetURI(ctx, release); err != nil {
		return errors.Wrap(err, "could not resolve magnet uri: %s", release.MagnetURI)
	}

	// Fetch the torrent once, here, onto the shared release. The action executors
	// take the release by value, so a download started inside one of them lands
	// on a copy and the next action would fetch the same torrent again.
	// Magnets carry no torrent to fetch, the clients get the URI instead.
	if !release.HasMagnetUri() && (action.NeedsTorrentDownloaded() || action.CheckMacrosNeedRawDataBytes(release)) {
		if err := s.rlsDownloadSvc.DownloadRelease(ctx, release); err != nil {
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

func (s *Service) runTest(ctx context.Context) {
	l := zerolog.Ctx(ctx)

	l.Debug().Msg("running Test action")

	l.Info().Msg("test action success")
}
