// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"
	"github.com/autobrr/autobrr/pkg/arr/readarr"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

func (s *Service) runReadarr(ctx context.Context, action *domain.Action, release *domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Trace().Msg("running Readarr action")

	instance, err := s.clientSvc.GetInstance(ctx, action.ClientID)
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

	client, err := downloader.ClientAs[*readarr.Client](instance)
	if err != nil {
		return nil, err
	}

	req := readarr.Release{
		Title:            release.TorrentName,
		InfoUrl:          release.InfoURL,
		DownloadUrl:      release.DownloadURL,
		MagnetUrl:        release.MagnetURI,
		Size:             release.Size,
		Indexer:          release.Indexer.GetExternalIdentifier(),
		DownloadProtocol: release.Protocol.String(),
		Protocol:         release.Protocol.String(),
		PublishDate:      time.Now().Format(time.RFC3339),
		DownloadClientId: cfg.Settings.ExternalDownloadClientId,
		DownloadClient:   cfg.Settings.ExternalDownloadClient,
	}

	if action.ExternalDownloadClientID > 0 {
		req.DownloadClientId = int(action.ExternalDownloadClientID)
	}

	if action.ExternalDownloadClient != "" {
		req.DownloadClient = action.ExternalDownloadClient
	}

	rejections, err := client.Push(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "readarr: failed to push release: %s", req.Title)
	}

	if rejections != nil {
		l.Debug().Str("indexer", req.Indexer).Str("host", cfg.Host).Strs("rejections", rejections).Msg("client rejected the release")

		return rejections, nil
	}

	l.Info().Str("indexer", req.Indexer).Str("host", cfg.Host).Msg("release successfully added to client")

	return nil, nil
}
