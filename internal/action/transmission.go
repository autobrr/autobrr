// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/hekmon/transmissionrpc/v3"
	"github.com/rs/zerolog"
)

const (
	ReannounceMaxAttempts = 50
	ReannounceInterval    = 7 // interval in seconds
)

var ErrReannounceTookTooLong = errors.New("ErrReannounceTookTooLong")
var TrTrue = true

func (s *Service) transmission(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Debug().Msg("running Transmission action")

	client, err := s.clientSvc.GetClient(ctx, action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", action.ClientID)
	}
	action.Client = client

	if !client.Enabled {
		return nil, errors.New("client %s %s not enabled", client.Type, client.Name)
	}

	tbt := client.Client.(*transmissionrpc.Client)

	rejections, err := s.transmissionCheckRulesCanDownload(ctx, action, client, tbt)
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

	if release.HasMagnetUri() {
		payload.Filename = &release.MagnetURI

		// Prepare and send payload
		torrent, err := tbt.TorrentAdd(ctx, payload)
		if err != nil {
			return nil, errors.Wrap(err, "could not add torrent from magnet %s to client: %s", release.MagnetURI, client.Host)
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
				p.UploadLimit = &action.LimitUploadSpeed
				p.UploadLimited = &TrTrue
			}
			if action.LimitDownloadSpeed > 0 {
				p.DownloadLimit = &action.LimitDownloadSpeed
				p.DownloadLimited = &TrTrue
			}
			if action.LimitRatio > 0 {
				p.SeedRatioLimit = &action.LimitRatio
				ratioMode := transmissionrpc.SeedRatioModeCustom
				p.SeedRatioMode = &ratioMode
			}
			if action.LimitSeedTime > 0 {
				t := time.Duration(action.LimitSeedTime) * time.Minute
				//p.SeedIdleLimit = &action.LimitSeedTime
				p.SeedIdleLimit = &t

				// seed idle mode 1
				seedIdleMode := int64(1)
				p.SeedIdleMode = &seedIdleMode
			}

			if err := tbt.TorrentSet(ctx, p); err != nil {
				return nil, errors.Wrap(err, "could not set label for hash %s to client: %s", *torrent.HashString, client.Host)
			}

			l.Debug().Str("hash", *torrent.HashString).Str("client", client.Name).Msg("set label for torrent successful to client")
		}

		l.Info().Str("hash", *torrent.HashString).Str("client", client.Name).Msg("release successfully added to client")

		return nil, nil
	}

	if err := s.downloadSvc.DownloadRelease(ctx, &release); err != nil {
		return nil, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
	}

	b64, err := transmissionrpc.File2Base64(release.TorrentTmpFile)
	if err != nil {
		return nil, errors.Wrap(err, "cant encode file %s into base64", release.TorrentTmpFile)
	}

	payload.MetaInfo = &b64

	// Prepare and send payload
	torrent, err := tbt.TorrentAdd(ctx, payload)
	if err != nil {
		return nil, errors.Wrap(err, "could not add torrent %s to client: %s", release.TorrentTmpFile, client.Host)
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
			p.UploadLimit = &action.LimitUploadSpeed
			p.UploadLimited = &TrTrue
		}
		if action.LimitDownloadSpeed > 0 {
			p.DownloadLimit = &action.LimitDownloadSpeed
			p.DownloadLimited = &TrTrue
		}
		if action.LimitRatio > 0 {
			p.SeedRatioLimit = &action.LimitRatio
			ratioMode := transmissionrpc.SeedRatioModeCustom
			p.SeedRatioMode = &ratioMode
		}
		if action.LimitSeedTime > 0 {
			t := time.Duration(action.LimitSeedTime) * time.Minute
			p.SeedIdleLimit = &t

			// seed idle mode 1
			seedIdleMode := int64(1)
			p.SeedIdleMode = &seedIdleMode
		}

		l.Trace().Interface("set_payload", p).Str("hash", *torrent.HashString).Str("client", client.Name).Msg("transmission torrent set payload")

		if err := tbt.TorrentSet(ctx, p); err != nil {
			return nil, errors.Wrap(err, "could not set label for hash %s to client: %s", *torrent.HashString, client.Host)
		}

		l.Debug().Str("hash", *torrent.HashString).Str("client", client.Name).Msg("set label for torrent successful to client")
	}

	if !action.Paused && !action.ReAnnounceSkip {
		if err := s.transmissionReannounce(ctx, action, tbt, *torrent.ID); err != nil {
			if errors.Is(err, ErrReannounceTookTooLong) {
				return []string{fmt.Sprintf("reannounce took too long for torrent: %s, deleted", *torrent.HashString)}, nil
			}

			return nil, errors.Wrap(err, "could not reannounce torrent: %s", *torrent.HashString)
		}

		return nil, nil
	}

	l.Info().Str("hash", *torrent.HashString).Str("client", client.Name).Msg("release successfully added to client")

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

func (s *Service) transmissionCheckRulesCanDownload(ctx context.Context, action *domain.Action, client *domain.DownloadClient, tbt *transmissionrpc.Client) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Trace().Msg("action transmission check rules")

	// check for active downloads and other rules
	if client.Settings.Rules.Enabled && !action.IgnoreRules {
		torrents, err := tbt.TorrentGet(ctx, []string{"status"}, []int64{})
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
		if client.Settings.Rules.MaxActiveDownloads > 0 {

			// if max active downloads reached, check speed and if lower than threshold add anyway
			if len(activeDownloads) >= client.Settings.Rules.MaxActiveDownloads {
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
