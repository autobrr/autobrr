// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/hekmon/transmissionrpc/v3"
	"github.com/rs/zerolog"
)

const (
	ReannounceMaxAttempts = 50
	ReannounceInterval    = 7 // interval in seconds
)

var ErrReannounceTookTooLong = errors.New("ErrReannounceTookTooLong")

func (s *Service) runTransmission(ctx context.Context, action *domain.Action, release *domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Debug().Msg("running Transmission action")

	instance, err := s.downloaderSvc.GetInstance(ctx, action.ClientID)
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

	client, err := downloader.ClientAs[*transmissionrpc.Client](instance)
	if err != nil {
		return nil, err
	}

	rejections, err := s.transmissionCheckRulesCanDownload(ctx, action, cfg, client)
	if err != nil {
		return nil, errors.Wrap(err, "error checking client rules: %s", action.Name)
	}

	if len(rejections) > 0 {
		return rejections, nil
	}

	payload := transmissionrpc.TorrentAddPayload{}

	if action.SavePath != "" {
		payload.DownloadDir = &action.SavePath
	}
	if action.Paused {
		payload.Paused = &action.Paused
	}

	switch {
	case release.HasMagnetUri():
		payload.Filename = &release.MagnetURI
	default:
		if err := s.rlsDownloadSvc.DownloadRelease(ctx, release); err != nil {
			return nil, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
		}

		payload.MetaInfo = new(base64.StdEncoding.EncodeToString(release.TorrentDataRawBytes))
	}

	// Prepare and send payload
	torrent, err := client.TorrentAdd(ctx, payload)
	if err != nil {
		return nil, errors.Wrap(err, "could not add torrent %s to client: %s", release.TorrentName, cfg.Host)
	}

	if action.Label != "" || action.LimitUploadSpeed > 0 || action.LimitDownloadSpeed > 0 || action.LimitRatio > 0 || action.LimitSeedTime > 0 {
		p := transmissionrpc.TorrentSetPayload{
			IDs: []int64{*torrent.ID},
		}

		if action.Label != "" {
			var labels []string
			for label := range strings.SplitSeq(action.Label, ",") {
				label = strings.TrimSpace(label)
				if label != "" {
					labels = append(labels, label)
				}
			}
			p.Labels = labels
		}

		if action.LimitUploadSpeed > 0 {
			p.UploadLimit = new(action.LimitUploadSpeed)
			p.UploadLimited = new(true)
		}
		if action.LimitDownloadSpeed > 0 {
			p.DownloadLimit = new(action.LimitDownloadSpeed)
			p.DownloadLimited = new(true)
		}
		if action.LimitRatio > 0 {
			p.SeedRatioLimit = new(action.LimitRatio)
			p.SeedRatioMode = new(transmissionrpc.SeedRatioModeCustom)
		}
		if action.LimitSeedTime > 0 {
			p.SeedIdleLimit = new(time.Duration(action.LimitSeedTime) * time.Minute)

			// seed idle mode 1
			p.SeedIdleMode = new(int64(1))
		}

		l.Trace().Interface("set_payload", p).Str("hash", *torrent.HashString).Str("client", cfg.Name).Msg("transmission torrent set payload")

		if err := client.TorrentSet(ctx, p); err != nil {
			return nil, errors.Wrap(err, "could not set label for hash %s to client: %s", *torrent.HashString, cfg.Host)
		}

		l.Debug().Str("hash", *torrent.HashString).Str("client", cfg.Name).Msg("set label for torrent successful to client")
	}

	if !action.Paused && !action.ReAnnounceSkip {
		if err := s.transmissionReannounce(ctx, action, client, *torrent.ID); err != nil {
			if errors.Is(err, ErrReannounceTookTooLong) {
				return []string{fmt.Sprintf("reannounce took too long for torrent: %s, deleted", *torrent.HashString)}, nil
			}

			return nil, errors.Wrap(err, "could not reannounce torrent: %s", *torrent.HashString)
		}

		return nil, nil
	}

	l.Info().Str("hash", *torrent.HashString).Str("client", cfg.Name).Msg("release successfully added to client")

	return rejections, nil
}

func (s *Service) transmissionReannounce(ctx context.Context, action *domain.Action, tbt *transmissionrpc.Client, torrentId int64) error {
	l := zerolog.Ctx(ctx)

	interval := ReannounceInterval
	if action.ReAnnounceInterval > 0 {
		interval = int(action.ReAnnounceInterval)
	}

	maxAttempts := ReannounceMaxAttempts
	if action.ReAnnounceMaxAttempts > 0 {
		maxAttempts = int(action.ReAnnounceMaxAttempts)
	}

	attempts := 0

	for attempts <= maxAttempts {
		l.Debug().Int64("client_torrent_id", torrentId).Int("attempt", attempts).Int("max_attempts", maxAttempts).Msg("re-announce attempt")

		// add delay for next run
		time.Sleep(time.Duration(interval) * time.Second)

		t, err := tbt.TorrentGet(ctx, []string{"trackerStats"}, []int64{torrentId})
		if err != nil {
			return errors.Wrap(err, "reannounced, failed to find torrentid")
		}

		if len(t) < 1 {
			return errors.Wrap(err, "reannounced, failed to get torrent from id")
		}

		for _, tracker := range t[0].TrackerStats {
			tracker := tracker

			l.Trace().Interface("tracker", tracker).Msg("transmission tracker")

			if tracker.IsBackup {
				continue
			}

			if isUnregistered(tracker.LastAnnounceResult) {
				continue
			}

			if tracker.SeederCount > 0 {
				return nil
			} else if tracker.LeecherCount > 0 {
				return nil
			}
		}

		l.Debug().Int64("client_torrent_id", torrentId).Int("attempt", attempts).Int("max_attempts", maxAttempts).Msg("transmission re-announce not working yet, re-announce again attempt")

		if err := tbt.TorrentReannounceIDs(ctx, []int64{torrentId}); err != nil {
			return errors.Wrap(err, "failed to reannounce")
		}

		attempts++
	}

	if attempts == maxAttempts && action.ReAnnounceDelete {
		l.Info().Int64("client_torrent_id", torrentId).Msg("re-announce took too long, deleting torrent")

		if err := tbt.TorrentRemove(ctx, transmissionrpc.TorrentRemovePayload{IDs: []int64{torrentId}}); err != nil {
			return errors.Wrap(err, "could not delete torrent: %v from client after max re-announce attempts reached", torrentId)
		}

		return errors.Wrap(ErrReannounceTookTooLong, "transmission re-announce took too long, deleted torrent %v", torrentId)
	}

	return nil
}

func (s *Service) transmissionCheckRulesCanDownload(ctx context.Context, action *domain.Action, cfg *domain.Downloader, client *transmissionrpc.Client) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Trace().Msg("action transmission check rules")

	// check for active downloads and other rules
	if cfg.Settings.Rules.Enabled && !action.IgnoreRules {
		torrents, err := client.TorrentGet(ctx, []string{"status"}, []int64{})
		if err != nil {
			return nil, errors.Wrap(err, "could not fetch active downloads")
		}

		var activeDownloads []transmissionrpc.Torrent

		// there is no way to get torrents by status, so we need to filter ourselves
		for _, torrent := range torrents {
			if *torrent.Status == transmissionrpc.TorrentStatusDownload {
				activeDownloads = append(activeDownloads, torrent)
			}
		}

		// make sure it's not set to 0 by default
		if cfg.Settings.Rules.MaxActiveDownloads > 0 {

			// if max active downloads reached, check speed and if lower than threshold add anyway
			if len(activeDownloads) >= cfg.Settings.Rules.MaxActiveDownloads {
				rejection := "max active downloads reached, skipping"

				l.Debug().Msg(rejection)

				return []string{rejection}, nil
			}
		}
	}

	return nil, nil
}

func isUnregistered(msg string) bool {
	words := []string{"unregistered", "not registered", "not found", "not exist"}

	msg = strings.ToLower(msg)

	for _, v := range words {
		if strings.Contains(msg, v) {
			return true
		}
	}

	return false
}
