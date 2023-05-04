// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/mrobinsn/go-rtorrent/rtorrent"
)

func (s *service) rtorrent(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action rTorrent: %s", action.Name)

	var err error

	// get client for action
	client, err := s.clientSvc.FindByID(ctx, action.ClientID)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("error finding client: %d", action.ClientID)
		return nil, err
	}

	if client == nil {
		return nil, errors.New("could not find client by id: %d", action.ClientID)
	}

	var rejections []string

	httpClient := http.Client{
		Timeout: 60 * time.Second,
		//Transport: &http.Transport{
		//	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		//},
		Transport: &authRoundTripper{
			BasicUser: client.Settings.Basic.Username,
			BasicPass: client.Settings.Basic.Password,
		},
	}

	// only set basic auth if enabled
	if client.Settings.Basic.Auth {
		httpClient.Transport = &authRoundTripper{
			BasicUser: client.Settings.Basic.Username,
			BasicPass: client.Settings.Basic.Password,
		}
	}

	// create client
	rt := rtorrent.New(client.Host, true)

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

		var addTorrentMagnet func(string, ...*rtorrent.FieldValue) error
		if action.Paused {
			addTorrentMagnet = rt.AddStopped
		} else {
			addTorrentMagnet = rt.Add
		}

		if err := addTorrentMagnet(release.MagnetURI, args...); err != nil {
			return nil, errors.Wrap(err, "could not add torrent from magnet: %s", release.MagnetURI)
		}

		s.log.Info().Msgf("torrent from magnet successfully added to client: '%s'", client.Name)

		return nil, nil

	} else {
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

		var addTorrentFile func([]byte, ...*rtorrent.FieldValue) error
		if action.Paused {
			addTorrentFile = rt.AddTorrentStopped
		} else {
			addTorrentFile = rt.AddTorrent
		}

		if err := addTorrentFile(tmpFile, args...); err != nil {
			return nil, errors.Wrap(err, "could not add torrent file: %s", release.TorrentTmpFile)
		}

		s.log.Info().Msgf("torrent successfully added to client: '%s'", client.Name)
	}

	return rejections, nil
}

type authRoundTripper struct {
	http.Transport
	BasicUser string
	BasicPass string
}

func (rt *authRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	rt.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	if rt.BasicUser != "" && rt.BasicPass != "" {
		r.SetBasicAuth(rt.BasicUser, rt.BasicPass)
	}

	return rt.Transport.RoundTrip(r)
}
