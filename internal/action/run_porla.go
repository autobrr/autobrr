// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"encoding/base64"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/porla"

	"github.com/rs/zerolog"
)

func (s *Service) runPorla(ctx context.Context, action *domain.Action, release *domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Debug().Msg("running Porla action")

	instance, err := s.downloaderSvc.GetInstance(ctx, action.ClientID)
	if err != nil {
		return nil, err
	}

	cfg := instance.Config()
	if cfg == nil {
		return nil, errors.New("client %d has no config", action.ClientID)
	}

	if !cfg.Enabled {
		return nil, errors.New("client %s %s not enabled", cfg.Type, cfg.Name)
	}

	client, err := downloader.ClientAs[*porla.Client](instance)
	if err != nil {
		return nil, err
	}

	rejections, err := s.porlaCheckRulesCanDownload(ctx, action, cfg, client)
	if err != nil {
		return nil, errors.Wrap(err, "error checking Porla client rules: %s", action.Name)
	}

	if len(rejections) > 0 {
		return rejections, nil
	}

	opts := &porla.TorrentsAddReq{
		SavePath: action.SavePath,
		//DownloadLimit: downloadLimit,
		//UploadLimit:   uploadLimit,
		//Preset:        preset,
	}

	if action.LimitDownloadSpeed > 0 {
		opts.DownloadLimit = new(action.LimitDownloadSpeed * 1000)
	}

	if action.LimitUploadSpeed > 0 {
		opts.UploadLimit = new(action.LimitUploadSpeed * 1000)
	}

	if action.Label != "" {
		opts.Preset = &action.Label
	}

	switch {
	case release.HasMagnetUri():
		opts.MagnetUri = release.MagnetURI

	default:
		if err := s.rlsDownloadSvc.DownloadRelease(ctx, release); err != nil {
			return nil, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
		}
		opts.Ti = base64.StdEncoding.EncodeToString(release.TorrentDataRawBytes)
	}

	if err := client.TorrentsAdd(ctx, opts); err != nil {
		return nil, errors.Wrap(err, "could not add torrent %s to client: %s", release.TorrentName, cfg.Name)
	}

	l.Info().Str("hash", release.TorrentHash).Str("client", cfg.Name).Msg("release successfully added to client")

	return nil, nil
}

func (s *Service) porlaCheckRulesCanDownload(ctx context.Context, action *domain.Action, cfg *domain.Downloader, client *porla.Client) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Trace().Msg("action porla check rules")

	// check for active downloads and other rules
	if cfg.Settings.Rules.Enabled && !action.IgnoreRules {
		torrents, err := client.TorrentsList(ctx, &porla.TorrentsListFilters{Query: "is:downloading and not is:paused"})
		if err != nil {
			return nil, errors.Wrap(err, "could not fetch active downloads")
		}

		if cfg.Settings.Rules.MaxActiveDownloads > 0 {
			if len(torrents.Torrents) >= cfg.Settings.Rules.MaxActiveDownloads {
				rejection := "max active downloads reached, skipping"

				l.Debug().Msg(rejection)

				return []string{rejection}, nil
			}
		}
	}

	return nil, nil
}
