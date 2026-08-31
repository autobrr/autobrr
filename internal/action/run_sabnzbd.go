// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sabnzbd"

	"github.com/rs/zerolog"
)

func (s *Service) runSabnzbd(ctx context.Context, action *domain.Action, release *domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Trace().Msg("running Sabnzbd action")

	if release.Protocol != domain.ReleaseProtocolNzb {
		return nil, errors.New("action type: %s invalid protocol: %s", action.Type, release.Protocol)
	}

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

	client, err := downloader.ClientAs[*sabnzbd.Client](instance)
	if err != nil {
		return nil, err
	}

	ids, err := client.AddFromUrl(ctx, sabnzbd.AddNzbRequest{Url: release.DownloadURL, Category: action.Category})
	if err != nil {
		return nil, errors.Wrap(err, "could not add nzb to sabnzbd")
	}

	l.Info().Str("client", cfg.Name).Interface("ids", ids).Msg("release successfully added to client")

	return nil, nil
}
