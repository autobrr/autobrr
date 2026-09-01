// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/go-rtorrent"
	"github.com/rs/zerolog"
)

func (s *Service) runRTorrent(ctx context.Context, action *domain.Action, release *domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Debug().Msg("running rTorrent action")

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

	client, err := downloader.ClientAs[*rtorrent.Client](instance)
	if err != nil {
		return nil, err
	}

	var rejections []string

	var args []*rtorrent.FieldValue

	if action.Label != "" {
		args = append(args, &rtorrent.FieldValue{
			Field: rtorrent.DLabel,
			Value: action.Label,
		})
	}

	if action.SavePath != "" {
		switch action.ContentLayout {
		case domain.ActionContentLayoutSubfolderNone:
			args = append(args, &rtorrent.FieldValue{
				Field: "d.directory_base",
				Value: action.SavePath,
			})

		default:
			args = append(args, &rtorrent.FieldValue{
				Field: rtorrent.DDirectory,
				Value: action.SavePath,
			})

		}
	}

	if release.HasMagnetUri() {
		var addTorrentMagnet func(context.Context, string, ...*rtorrent.FieldValue) error
		if action.Paused {
			addTorrentMagnet = client.AddStopped
		} else {
			addTorrentMagnet = client.Add
		}

		if err := addTorrentMagnet(ctx, release.MagnetURI, args...); err != nil {
			return nil, errors.Wrap(err, "could not add torrent from magnet: %s", release.MagnetURI)
		}

		l.Info().Str("client", cfg.Name).Msg("release successfully added to client")

		return nil, nil
	}

	if err := s.rlsDownloadSvc.DownloadRelease(ctx, release); err != nil {
		return nil, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
	}

	var addTorrentFile func(context.Context, []byte, ...*rtorrent.FieldValue) error
	if action.Paused {
		addTorrentFile = client.AddTorrentStopped
	} else {
		addTorrentFile = client.AddTorrent
	}

	if err := addTorrentFile(ctx, release.TorrentDataRawBytes, args...); err != nil {
		return nil, errors.Wrap(err, "could not add torrent file: %s", release.TorrentName)
	}

	l.Info().Str("client", cfg.Name).Msg("release successfully added to client")

	return rejections, nil
}
