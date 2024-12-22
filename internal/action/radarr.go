// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/arr/radarr"
	"github.com/autobrr/autobrr/pkg/errors"
)

func (s *service) radarr(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Trace().Msg("action RADARR")

	// TODO validate data

	client, err := s.clientSvc.GetClient(ctx, action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", action.ClientID)
	}
	action.Client = client

	if !client.Enabled {
		return nil, errors.New("client %s %s not enabled", client.Type, client.Name)
	}

	arr := client.Client.(*radarr.Client)

	r := radarr.Release{
		Title:            release.TorrentName,
		InfoUrl:          release.InfoURL,
		DownloadUrl:      release.DownloadURL,
		MagnetUrl:        release.MagnetURI,
		Size:             release.Size,
		Indexer:          release.Indexer.GetExternalIdentifier(),
		DownloadClientId: client.Settings.ExternalDownloadClientId,
		DownloadClient:   client.Settings.ExternalDownloadClient,
		DownloadProtocol: release.Protocol.String(),
		Protocol:         release.Protocol.String(),
		PublishDate:      time.Now().Format(time.RFC3339),
	}

	if action.ExternalDownloadClientID > 0 {
		r.DownloadClientId = int(action.ExternalDownloadClientID)
	}

	if action.ExternalDownloadClient != "" {
		r.DownloadClient = action.ExternalDownloadClient
	}

	rejections, err := arr.Push(ctx, r)
	if err != nil {
		return nil, errors.Wrap(err, "radarr failed to push release: %v", r)
	}

	if rejections != nil {
		s.log.Debug().Msgf("radarr: release push rejected: %v, indexer %v to %v reasons: '%v'", r.Title, r.Indexer, client.Host, rejections)

		return rejections, nil
	}

	s.log.Debug().Msgf("radarr: successfully pushed release: %v, indexer %v to %v", r.Title, r.Indexer, client.Host)

	return nil, nil
}
