// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/lidarr"
)

func (s *service) lidarr(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Trace().Msg("action LIDARR")

	// TODO validate data

	// get client for action
	client := action.Client

	// return early if no client found
	if client == nil {
		return nil, errors.New("could not find client by id: %v", action.ClientID)
	}

	// initial config
	cfg := lidarr.Config{
		Hostname: client.Host,
		APIKey:   client.Settings.APIKey,
		Log:      s.subLogger,
	}

	// only set basic auth if enabled
	if client.Settings.Basic.Auth {
		cfg.BasicAuth = client.Settings.Basic.Auth
		cfg.Username = client.Settings.Basic.Username
		cfg.Password = client.Settings.Basic.Password
	}

	externalClientId := client.Settings.ExternalDownloadClientId
	if action.ExternalDownloadClientID > 0 {
		externalClientId = int(action.ExternalDownloadClientID)
	}

	externalClient := client.Settings.ExternalDownloadClient
	if action.ExternalDownloadClient != "" {
		externalClient = action.ExternalDownloadClient
	}

	r := lidarr.Release{
		Title:            release.TorrentName,
		InfoUrl:          release.InfoURL,
		DownloadUrl:      release.DownloadURL,
		MagnetUrl:        release.MagnetURI,
		Size:             int64(release.Size),
		Indexer:          release.Indexer.GetExternalIdentifier(),
		DownloadClientId: externalClientId,
		DownloadClient:   externalClient,
		DownloadProtocol: release.Protocol.String(),
		Protocol:         release.Protocol.String(),
		PublishDate:      time.Now().Format(time.RFC3339),
	}

	arr := lidarr.New(cfg)

	rejections, err := arr.Push(ctx, r)
	if err != nil {
		s.log.Error().Err(err).Msgf("lidarr: failed to push release: %v", r)
		return nil, err
	}

	if rejections != nil {
		s.log.Debug().Msgf("lidarr: release push rejected: %v, indexer %v to %v reasons: '%v'", r.Title, r.Indexer, client.Host, rejections)

		return rejections, nil
	}

	s.log.Debug().Msgf("lidarr: successfully pushed release: %v, indexer %v to %v", r.Title, r.Indexer, client.Host)

	return nil, nil
}
