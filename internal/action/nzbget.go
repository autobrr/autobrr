// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/nzbget"
)

func (s *service) nzbget(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Trace().Msg("action NZBGet")

	if release.Protocol != domain.ReleaseProtocolNzb {
		return nil, errors.New("action type: %s invalid protocol: %s", action.Type, release.Protocol)
	}

	client, err := s.clientSvc.GetClient(ctx, action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", action.ClientID)
	}
	action.Client = client

	if !client.Enabled {
		return nil, errors.New("client %s %s not enabled", client.Type, client.Name)
	}

	nzb := client.Client.(*nzbget.Client)

	resp, err := nzb.AddFromURL(ctx, nzbget.AddNzbRequest{URL: release.DownloadURL, Category: action.Category})
	if err != nil {
		return nil, errors.Wrap(err, "could not add nzb to nzbget")
	}

	s.log.Trace().Msgf("nzb successfully added to client: '%+v'", resp)

	s.log.Info().Msgf("nzb successfully added to client: '%s'", client.Name)

	return nil, nil
}
