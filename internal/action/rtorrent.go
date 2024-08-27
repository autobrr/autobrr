// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/autobrr/go-rtorrent"
	"github.com/icholy/digest"
)

func (s *service) rtorrent(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action rTorrent: %s", action.Name)

	client, err := s.clientSvc.GetClient(ctx, action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", action.ClientID)
	}

	if !client.Enabled {
		return nil, errors.New("client %s %s not enabled", client.Type, client.Name)
	}

	rt := client.Client.(*rtorrent.Client)

	// create config
	cfg := rtorrent.Config{
		Addr:          client.Host,
		TLSSkipVerify: client.TLSSkipVerify,
		BasicUser:     client.Settings.Basic.Username,
		BasicPass:     client.Settings.Basic.Password,
	}

	httpClient := &http.Client{
		Transport: &digest.Transport{
			Username: client.Settings.Basic.Username,
			Password: client.Settings.Basic.Password,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}

	// override client
	rt = rtorrent.NewClientWithOpts(cfg, rtorrent.WithCustomClient(httpClient))

	var rejections []string


	if release.HasMagnetUri() {
		var args []*rtorrent.FieldValue

		if action.Label != "" {
			args = append(args, &rtorrent.FieldValue{
				Field: rtorrent.DLabel,
				Value: action.Label,
			})
		}
		if action.SavePath != "" {
			if action.ContentLayout == domain.ActionContentLayoutSubfolderNone {
				args = append(args, &rtorrent.FieldValue{
					Field: "d.directory_base",
					Value: action.SavePath,
				})
			} else {
				args = append(args, &rtorrent.FieldValue{
					Field: rtorrent.DDirectory,
					Value: action.SavePath,
				})
			}
		}

		var addTorrentMagnet func(context.Context, string, ...*rtorrent.FieldValue) error
		if action.Paused {
			addTorrentMagnet = rt.AddStopped
		} else {
			addTorrentMagnet = rt.Add
		}

		if err := addTorrentMagnet(ctx, release.MagnetURI, args...); err != nil {
			return nil, errors.Wrap(err, "could not add torrent from magnet: %s", release.MagnetURI)
		}

		s.log.Info().Msgf("torrent from magnet successfully added to client: '%s'", client.Name)

		return nil, nil
	}

	if release.TorrentTmpFile == "" {
		if err := release.DownloadTorrentFileCtx(ctx); err != nil {
			s.log.Error().Err(err).Msgf("could not download torrent file for release: %s", release.TorrentName)
			return nil, err
		}
	}

	tmpFile, err := os.ReadFile(release.TorrentTmpFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not read torrent file: %s", release.TorrentTmpFile)
	}

	var args []*rtorrent.FieldValue

	if action.Label != "" {
		args = append(args, &rtorrent.FieldValue{
			Field: rtorrent.DLabel,
			Value: action.Label,
		})
	}
	if action.SavePath != "" {
		if action.ContentLayout == domain.ActionContentLayoutSubfolderNone {
			args = append(args, &rtorrent.FieldValue{
				Field: "d.directory_base",
				Value: action.SavePath,
			})
		} else {
			args = append(args, &rtorrent.FieldValue{
				Field: rtorrent.DDirectory,
				Value: action.SavePath,
			})
		}
	}

	var addTorrentFile func(context.Context, []byte, ...*rtorrent.FieldValue) error
	if action.Paused {
		addTorrentFile = rt.AddTorrentStopped
	} else {
		addTorrentFile = rt.AddTorrent
	}

	if err := addTorrentFile(ctx, tmpFile, args...); err != nil {
		return nil, errors.Wrap(err, "could not add torrent file: %s", release.TorrentTmpFile)
	}

	s.log.Info().Msgf("torrent successfully added to client: '%s'", client.Name)

	return rejections, nil
}
