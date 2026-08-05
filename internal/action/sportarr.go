// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/arr/sportarr"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

func (s *Service) sportarr(ctx context.Context, action *domain.Action, release *domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Trace().Msg("running Sportarr action")

	client, err := s.clientSvc.GetClient(ctx, action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", action.ClientID)
	}
	action.Client = client

	if !client.Enabled {
		return nil, errors.New("client %s %s not enabled", client.Type, client.Name)
	}

	arrClient, ok := client.Client.(*sportarr.Client)
	if !ok {
		return nil, errors.New("invalid client type")
	}

	r := sportarr.ReleasePushRequest{
		Title:            release.TorrentName,
		InfoUrl:          release.InfoURL,
		DownloadUrl:      release.DownloadURL,
		MagnetUrl:        release.MagnetURI,
		Size:             release.Size,
		Indexer:          release.Indexer.GetExternalIdentifier(),
		DownloadProtocol: release.Protocol.String(),
		Protocol:         release.Protocol.String(),
		PublishDate:      time.Now().Format(time.RFC3339),
	}

	indexerFlags := sportarr.BuildIndexerFlags(sportarr.ReleaseMeta{
		FreeleechPercent: release.FreeleechPercent,
		Origin:           release.Origin,
	})
	r.IndexerFlags = int(indexerFlags)

	rejections, err := arrClient.Push(ctx, r)
	if err != nil {
		return nil, errors.Wrap(err, "sportarr: failed to push release: %v", r)
	}

	if rejections != nil {
		l.Debug().Str("indexer", r.Indexer).Str("host", client.Host).Strs("rejections", rejections).Msg("client rejected the release")

		return rejections, nil
	}

	l.Info().Str("indexer", r.Indexer).Str("host", client.Host).Msg("release successfully added to client")

	return nil, nil
}
