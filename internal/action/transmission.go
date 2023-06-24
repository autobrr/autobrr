// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/avast/retry-go"

	"github.com/hekmon/transmissionrpc/v2"
)

func (s *service) transmission(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action Transmission: %s", action.Name)

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

	tbt, err := transmissionrpc.New(client.Host, client.Username, client.Password, &transmissionrpc.AdvancedConfig{
		HTTPS: client.TLS,
		Port:  uint16(client.Port),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error logging into client: %s", client.Host)
	}

	if release.HasMagnetUri() {
		payload := transmissionrpc.TorrentAddPayload{
			Filename: &release.MagnetURI,
		}
		if action.SavePath != "" {
			payload.DownloadDir = &action.SavePath
		}
		if action.Paused {
			payload.Paused = &action.Paused
		}

		// Prepare and send payload
		torrent, err := tbt.TorrentAdd(ctx, payload)
		if err != nil {
			return nil, errors.Wrap(err, "could not add torrent from magnet %s to client: %s", release.MagnetURI, client.Host)
		}

		s.log.Info().Msgf("torrent from magnet with hash %v successfully added to client: '%s'", torrent.HashString, client.Name)

		return nil, nil

	} else {
		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFileCtx(ctx); err != nil {
				s.log.Error().Err(err).Msgf("could not download torrent file for release: %s", release.TorrentName)
				return nil, err
			}
		}

		b64, err := transmissionrpc.File2Base64(release.TorrentTmpFile)
		if err != nil {
			return nil, errors.Wrap(err, "cant encode file %s into base64", release.TorrentTmpFile)
		}

		payload := transmissionrpc.TorrentAddPayload{
			MetaInfo: &b64,
		}
		if action.SavePath != "" {
			payload.DownloadDir = &action.SavePath
		}
		if action.Paused {
			payload.Paused = &action.Paused
		}

		// Prepare and send payload
		torrent, err := tbt.TorrentAdd(ctx, payload)
		if err != nil {
			return nil, errors.Wrap(err, "could not add torrent %v to client: %v", release.TorrentTmpFile, client.Host)
		}

		s.log.Info().Msgf("torrent with hash %v successfully added to client: '%s'", torrent.HashString, client.Name)
	}

	if !action.ReAnnounceSkip {
		err = retry.Do(func() error {
			f := func() { retry.Delay(time.Second * time.Duration(action.ReAnnounceInterval)) }
			if err := tbt.TorrentReannounceIDs(ctx, []int64{*torrent.ID}); err != nil {
				defer f()
				return errors.Wrap(err, "failed to reannounce")
			}

			t, err := tbt.TorrentGet(ctx, []string{"trackerStats"}, []int64{*torrent.ID})
			if err != nil {
				defer f()
				return errors.Wrap(err, "reannounced, failed to find torrentid")
			}

			if len(t) < 1 {
				defer f()
				return errors.Wrap(err, "reannounced, failed to get torrent from id")
			}

			seeds := int64(0)
			for _, trackers := range t[0].TrackerStats {
				seeds += trackers.SeederCount
			}

			if seeds != 0 {
				return nil
			}

			defer f()
			return errors.New("no seeds yet")
		},
			retry.OnRetry(func(n uint, err error) {
				s.log.Error().Err(err).Msgf("%q: attempt %d - %v\n", err, n, int(action.ReAnnounceMaxAttempts))
			}),
			//retry.Delay(time.Second*3),
			retry.Attempts(uint(action.ReAnnounceMaxAttempts)),
			retry.MaxJitter(time.Second*1),
		)

		if err != nil && action.ReAnnounceDelete {
			tbt.TorrentRemove(ctx, transmissionrpc.TorrentRemovePayload{IDs: []int64{*torrent.ID}})
		}
	}

	return rejections, nil
}
