// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/aria2"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

func (s *Service) aria2(ctx context.Context, action *domain.Action, release *domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Debug().Msg("running aria2 action")

	client, err := s.clientSvc.GetClient(ctx, action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", action.ClientID)
	}
	action.Client = client

	if !client.Enabled {
		return nil, errors.New("client %s %s not enabled", client.Type, client.Name)
	}

	ar := client.Client.(*aria2.Client)

	rejections, err := s.aria2CheckRulesCanDownload(ctx, action, client, ar)
	if err != nil {
		return nil, errors.Wrap(err, "error checking aria2 client rules: %s", action.Name)
	}

	if len(rejections) > 0 {
		return rejections, nil
	}

	options := aria2Options(action)

	l.Trace().Interface("options", options).Msg("action aria2 options")

	var gid string

	if release.HasMagnetUri() {
		gid, err = ar.AddURI(ctx, []string{release.MagnetURI}, options)
		if err != nil {
			return nil, errors.Wrap(err, "could not add torrent from magnet %s to client: %s", release.MagnetURI, client.Name)
		}
	} else {
		if err := s.downloadSvc.DownloadRelease(ctx, release); err != nil {
			return nil, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
		}

		gid, err = ar.AddTorrent(ctx, release.TorrentDataRawBytes, options)
		if err != nil {
			return nil, errors.Wrap(err, "could not add torrent %s to client: %s", release.TorrentName, client.Name)
		}
	}

	l.Info().Str("gid", gid).Str("hash", release.TorrentHash).Str("client", client.Name).Msg("release successfully added to client")

	return nil, nil
}

// aria2Options maps the action onto aria2 download options. Every value has to
// be a string, and the speed limits take the K suffix since autobrr stores them
// as KB/s while aria2 defaults to bytes.
func aria2Options(action *domain.Action) aria2.Options {
	options := aria2.Options{}

	if action.SavePath != "" {
		options["dir"] = action.SavePath
	}

	if action.Paused {
		options["pause"] = "true"
	}

	if action.LimitDownloadSpeed > 0 {
		options["max-download-limit"] = strconv.FormatInt(action.LimitDownloadSpeed, 10) + "K"
	}

	if action.LimitUploadSpeed > 0 {
		options["max-upload-limit"] = strconv.FormatInt(action.LimitUploadSpeed, 10) + "K"
	}

	if action.LimitRatio > 0 {
		options["seed-ratio"] = strconv.FormatFloat(action.LimitRatio, 'f', -1, 64)
	}

	if action.LimitSeedTime > 0 {
		options["seed-time"] = strconv.FormatInt(action.LimitSeedTime, 10)
	}

	return options
}

func (s *Service) aria2CheckRulesCanDownload(ctx context.Context, action *domain.Action, client *domain.DownloadClient, ar *aria2.Client) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Trace().Msg("action aria2 check rules")

	rules := client.Settings.Rules

	if !rules.Enabled || action.IgnoreRules || rules.MaxActiveDownloads <= 0 {
		return nil, nil
	}

	active, err := ar.TellActive(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch active downloads")
	}

	downloading := 0
	for _, download := range active {
		if download.Downloading() {
			downloading++
		}
	}

	if downloading >= rules.MaxActiveDownloads {
		rejection := "max active downloads reached, skipping"

		l.Debug().Int("active_downloads", downloading).Int("max_active_downloads", rules.MaxActiveDownloads).Msg(rejection)

		return []string{rejection}, nil
	}

	return nil, nil
}
