// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"os"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/porla"
)

func (s *service) porla(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action Porla: %s", action.Name)

	client, err := s.clientSvc.GetClient(ctx, action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", action.ClientID)
	}
	action.Client = client

	if !client.Enabled {
		return nil, errors.New("client %s %s not enabled", client.Type, client.Name)
	}

	prl := client.Client.(*porla.Client)

	rejections, err := s.porlaCheckRulesCanDownload(ctx, action, client, prl)
	if err != nil {
		return nil, errors.Wrap(err, "error checking Porla client rules: %s", action.Name)
	}

	if len(rejections) > 0 {
		return rejections, nil
	}

	var downloadLimit *int64 = nil
	var uploadLimit *int64 = nil

	if action.LimitDownloadSpeed > 0 {
		dlValue := action.LimitDownloadSpeed * 1000
		downloadLimit = &dlValue
	}

	if action.LimitUploadSpeed > 0 {
		ulValue := action.LimitUploadSpeed * 1000
		uploadLimit = &ulValue
	}

	var preset *string = nil

	if action.Label != "" {
		preset = &action.Label
	}

	if release.HasMagnetUri() {
		opts := &porla.TorrentsAddReq{
			DownloadLimit: downloadLimit,
			MagnetUri:     release.MagnetURI,
			SavePath:      action.SavePath,
			UploadLimit:   uploadLimit,
			Preset:        preset,
		}

		if err = prl.TorrentsAdd(ctx, opts); err != nil {
			return nil, errors.Wrap(err, "could not add torrent from magnet %s to client: %s", release.MagnetURI, client.Name)
		}

		s.log.Info().Msgf("torrent with hash %s successfully added to client: '%s'", release.TorrentHash, client.Name)

		return nil, nil
	} else {
		if err := s.downloadSvc.DownloadRelease(ctx, &release); err != nil {
			return nil, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
		}

		file, err := os.Open(release.TorrentTmpFile)
		if err != nil {
			return nil, errors.Wrap(err, "error opening file %s", release.TorrentTmpFile)
		}
		defer file.Close()

		content, err := io.ReadAll(bufio.NewReader(file))
		if err != nil {
			return nil, errors.Wrap(err, "failed to read file: %s", release.TorrentTmpFile)
		}

		opts := &porla.TorrentsAddReq{
			DownloadLimit: downloadLimit,
			SavePath:      action.SavePath,
			Ti:            base64.StdEncoding.EncodeToString(content),
			UploadLimit:   uploadLimit,
			Preset:        preset,
		}

		if err = prl.TorrentsAdd(ctx, opts); err != nil {
			return nil, errors.Wrap(err, "could not add torrent %s to client: %s", release.TorrentTmpFile, client.Name)
		}

		s.log.Info().Msgf("torrent with hash %s successfully added to client: '%s'", release.TorrentHash, client.Name)
	}

	return nil, nil
}

func (s *service) porlaCheckRulesCanDownload(ctx context.Context, action *domain.Action, client *domain.DownloadClient, prla *porla.Client) ([]string, error) {
	s.log.Trace().Msgf("action Porla: %s check rules", action.Name)

	// check for active downloads and other rules
	if client.Settings.Rules.Enabled && !action.IgnoreRules {
		torrents, err := prla.TorrentsList(ctx, &porla.TorrentsListFilters{Query: "is:downloading and not is:paused"})
		if err != nil {
			return nil, errors.Wrap(err, "could not fetch active downloads")
		}

		if client.Settings.Rules.MaxActiveDownloads > 0 {
			if len(torrents.Torrents) >= client.Settings.Rules.MaxActiveDownloads {
				rejection := "max active downloads reached, skipping"

				s.log.Debug().Msg(rejection)

				return []string{rejection}, nil
			}
		}
	}

	return nil, nil
}
