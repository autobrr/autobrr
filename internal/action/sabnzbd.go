// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sabnzbd"
)

func (s *service) sabnzbd(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Trace().Msg("action Sabnzbd")

	if release.Protocol != domain.ReleaseProtocolNzb {
		return nil, errors.New("action type: %s invalid protocol: %s", action.Type, release.Protocol)
	}

	// get client for action
	client, err := s.clientSvc.FindByID(ctx, action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "sonarr could not find client: %d", action.ClientID)
	}

	// return early if no client found
	if client == nil {
		return nil, errors.New("no sabnzbd client found by id: %d", action.ClientID)
	}

	opts := sabnzbd.Options{
		Addr:   client.Host,
		ApiKey: client.Settings.APIKey,
		Log:    nil,
	}

	if client.Settings.Basic.Auth {
		opts.BasicUser = client.Settings.Basic.Username
		opts.BasicPass = client.Settings.Basic.Password
	}

	sab := sabnzbd.New(opts)

	ids, err := sab.AddFromUrl(ctx, sabnzbd.AddNzbRequest{Url: release.TorrentURL, Category: action.Category})
	if err != nil {
		return nil, errors.Wrap(err, "could not add nzb to sabnzbd")
	}

	s.log.Trace().Msgf("nzb successfully added to client: '%+v'", ids)

	s.log.Info().Msgf("nzb successfully added to client: '%s'", client.Name)

	return nil, nil
}
