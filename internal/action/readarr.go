// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/readarr"
)

func (s *service) readarr(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Trace().Msg("action READARR")

	// TODO validate data

	// get client for action
	client, err := s.clientSvc.FindByID(ctx, action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "readarr could not find client: %v", action.ClientID)
	}

	// return early if no client found
	if client == nil {
		return nil, errors.New("no client found")
	}

	// initial config
	cfg := readarr.Config{
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

	externalId := 0
	if client.Settings.ExternalDownloadClientId > 0 {
		externalId = client.Settings.ExternalDownloadClientId
	} else if action.ExternalDownloadClientID > 0 {
		externalId = int(action.ExternalDownloadClientID)
	}

	r := readarr.Release{
		Title:            release.TorrentName,
		InfoUrl:          release.InfoURL,
		DownloadUrl:      release.DownloadURL,
		MagnetUrl:        release.MagnetURI,
		Size:             int64(release.Size),
		Indexer:          release.Indexer,
		DownloadClientId: externalId,
		DownloadProtocol: string(release.Protocol),
		Protocol:         string(release.Protocol),
		PublishDate:      time.Now().Format(time.RFC3339),
	}

	arr := readarr.New(cfg)

	rejections, err := arr.Push(ctx, r)
	if err != nil {
		return nil, errors.Wrap(err, "readarr: failed to push release: %v", r)
	}

	if rejections != nil {
		s.log.Debug().Msgf("readarr: release push rejected: %v, indexer %v to %v reasons: '%v'", r.Title, r.Indexer, client.Host, rejections)

		return rejections, nil
	}

	s.log.Debug().Msgf("readarr: successfully pushed release: %v, indexer %v to %v", r.Title, r.Indexer, client.Host)

	return nil, nil
}
