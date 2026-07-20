// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sabnzbd"

	"github.com/rs/zerolog"
)

func (s *Service) sabnzbd(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Trace().Msg("running Sabnzbd action")

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

	sab := client.Client.(*sabnzbd.Client)

	ids, err := sab.AddFromUrl(ctx, sabnzbd.AddNzbRequest{Url: release.DownloadURL, Category: action.Category})
	if err != nil {
		return nil, errors.Wrap(err, "could not add nzb to sabnzbd")
	}

	l.Info().Str("client", client.Name).Interface("ids", ids).Msg("release successfully added to client")

	return nil, nil
}
