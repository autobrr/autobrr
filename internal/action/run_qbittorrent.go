// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/autobrr/go-qbittorrent"
	"github.com/rs/zerolog"
)

func (s *Service) runQbittorrent(ctx context.Context, action *domain.Action, release *domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Debug().Msg("running qBittorrent action")

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

	client, err := downloader.ClientAs[*qbittorrent.Client](instance)
	if err != nil {
		return nil, err
	}

	if cfg.Settings.Rules.Enabled && !action.IgnoreRules {
		// check for active downloads and other rules
		rejections, err := s.qbittorrentCheckRulesCanDownload(ctx, action, cfg.Settings.Rules, client)
		if err != nil {
			return nil, errors.Wrap(err, "error checking client rules: %s", action.Name)
		}

		if len(rejections) > 0 {
			return rejections, nil
		}
	}

	options, err := s.prepareQbitOptions(action)
	if err != nil {
		return nil, errors.Wrap(err, "could not prepare options")
	}

	l.Trace().Interface("options", options).Msg("action qbittorrent options")

	if release.HasMagnetUri() {
		if _, err = client.AddTorrentFromUrlCtx(ctx, release.MagnetURI, options); err != nil {
			return nil, errors.Wrap(err, "could not add torrent %s to client: %s", release.MagnetURI, cfg.Name)
		}

		l.Info().Str("client", cfg.Name).Msg("release successfully added to client")

		return nil, nil
	}

	if err := s.rlsDownloadSvc.DownloadRelease(ctx, release); err != nil {
		return nil, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
	}

	if _, err = client.AddTorrentFromMemoryCtx(ctx, release.TorrentDataRawBytes, options); err != nil {
		return nil, errors.Wrap(err, "could not add torrent %s to client: %s", release.TorrentName, cfg.Name)
	}

	if release.TorrentHash != "" {
		// check if torrent queueing is enabled if priority is set
		switch action.PriorityLayout {
		case domain.PriorityLayoutMax, domain.PriorityLayoutMin:
			prefs, err := client.GetAppPreferencesCtx(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "could not get application preferences from client: '%s'", cfg.Name)
			}
			// enable queueing if it's disabled
			if !prefs.QueueingEnabled {
				if err := client.SetPreferencesQueueingEnabled(true); err != nil {
					return nil, errors.Wrap(err, "could not enable torrent queueing")
				}
				l.Trace().Str("client", cfg.Name).Msg("torrent queueing was disabled, now enabled in client")
			}
			// set priority if queueing is enabled
			if action.PriorityLayout == domain.PriorityLayoutMax {
				if err := client.SetMaxPriorityCtx(ctx, []string{release.TorrentHash}); err != nil {
					return nil, errors.Wrap(err, "could not set torrent %s to max priority", release.TorrentHash)
				}
				l.Debug().Str("hash", release.TorrentHash).Str("client", cfg.Name).Msg("torrent set to max priority in client")

			} else { // domain.PriorityLayoutMin
				if err := client.SetMinPriorityCtx(ctx, []string{release.TorrentHash}); err != nil {
					return nil, errors.Wrap(err, "could not set torrent %s to min priority", release.TorrentHash)
				}
				l.Debug().Str("hash", release.TorrentHash).Str("client", cfg.Name).Msg("torrent set to min priority in client")
			}

		case domain.PriorityLayoutDefault:
			// do nothing as it's disabled or unset
		default:
			l.Warn().Interface("priority_layout", action.PriorityLayout).Msg("unknown priority setting, no priority changes made")
		}
	} else {
		// add anyway if no hash
		l.Trace().Msg("no torrent hash provided, skipping priority setting")
	}

	if !action.Paused && !action.ReAnnounceSkip && release.TorrentHash != "" {
		opts := qbittorrent.ReannounceOptions{
			Interval:        int(action.ReAnnounceInterval),
			MaxAttempts:     int(action.ReAnnounceMaxAttempts),
			DeleteOnFailure: action.ReAnnounceDelete,
		}

		if err := client.ReannounceTorrentWithRetry(ctx, release.TorrentHash, &opts); err != nil {
			if errors.Is(err, qbittorrent.ErrReannounceTookTooLong) {
				return []string{fmt.Sprintf("re-announce took too long for hash: %s", release.TorrentHash)}, nil
			}

			return nil, errors.Wrap(err, "could not reannounce torrent: %s", release.TorrentHash)
		}
	}

	l.Info().Str("hash", release.TorrentHash).Str("client", cfg.Name).Msg("release successfully added to client")

	return nil, nil
}

func (s *Service) prepareQbitOptions(action *domain.Action) (map[string]string, error) {
	opts := &qbittorrent.TorrentAddOptions{}

	opts.Paused = false
	if action.Paused {
		opts.Paused = true
	}
	if action.SkipHashCheck {
		opts.SkipHashCheck = true
	}
	if action.FirstLastPiecePrio {
		opts.FirstLastPiecePrio = true
	}
	switch action.ContentLayout {
	case domain.ActionContentLayoutSubfolderCreate:
		opts.ContentLayout = qbittorrent.ContentLayoutSubfolderCreate
	case domain.ActionContentLayoutSubfolderNone:
		opts.ContentLayout = qbittorrent.ContentLayoutSubfolderNone
	}
	if action.SavePath != "" {
		opts.SavePath = strings.TrimSpace(action.SavePath)
	}
	if action.DownloadPath != "" {
		opts.DownloadPath = strings.TrimSpace(action.DownloadPath)
	}
	if action.Category != "" {
		opts.Category = strings.TrimSpace(action.Category)
	}
	if action.Tags != "" {
		// Split the action.Tags string by comma
		tags := strings.Split(action.Tags, ",")

		// Create a new slice to store the trimmed tags
		trimmedTags := make([]string, 0, len(tags))

		// Iterate over the tags and trim each one
		for _, tag := range tags {
			trimmedTag := strings.TrimSpace(tag)
			trimmedTags = append(trimmedTags, trimmedTag)
		}

		// Join the trimmed tags back together with commas
		opts.Tags = strings.Join(trimmedTags, ",")
	}
	if action.LimitUploadSpeed > 0 {
		opts.LimitUploadSpeed = action.LimitUploadSpeed
	}
	if action.LimitDownloadSpeed > 0 {
		opts.LimitDownloadSpeed = action.LimitDownloadSpeed
	}
	if action.LimitRatio > 0 {
		opts.LimitRatio = action.LimitRatio
	}
	if action.LimitSeedTime > 0 {
		opts.LimitSeedTime = action.LimitSeedTime
	}

	return opts.Prepare(), nil
}

// qbittorrentCheckRulesCanDownload
func (s *Service) qbittorrentCheckRulesCanDownload(ctx context.Context, action *domain.Action, rules domain.DownloaderRules, qbt *qbittorrent.Client) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Trace().Msg("action qbittorrent check rules")

	// make sure it's not set to 0 by default
	if rules.MaxActiveDownloads > 0 {

		// get active downloads
		activeDownloads, err := qbt.GetTorrentsActiveDownloadsCtx(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not fetch active downloads")
		}

		// if max active downloads reached, check speed and if lower than threshold add anyway
		if len(activeDownloads) >= rules.MaxActiveDownloads {
			// if we do not care about slow torrents then return early
			if !rules.IgnoreSlowTorrents {
				rejection := "max active downloads reached, skipping"

				l.Debug().Msg(rejection)

				return []string{rejection}, nil
			}

			if rules.IgnoreSlowTorrents && rules.IgnoreSlowTorrentsCondition == domain.IgnoreSlowTorrentsModeMaxReached {
				// get transfer info
				info, err := qbt.GetTransferInfoCtx(ctx)
				if err != nil {
					return nil, errors.Wrap(err, "could not get transfer info")
				}

				rejections := s.qbittorrentCheckIgnoreSlow(rules.DownloadSpeedThreshold, rules.UploadSpeedThreshold, info)
				if len(rejections) > 0 {
					return rejections, nil
				}

				l.Debug().Msg("active downloads are slower than set limit, lets add it")

				return nil, nil
			}
		}

		// if less, then we must check if ignore slow always which means we can't return here
	}

	// if max active downloads is unlimited or not reached, lets check if ignore slow always should be checked
	if rules.IgnoreSlowTorrents && rules.IgnoreSlowTorrentsCondition == domain.IgnoreSlowTorrentsModeAlways {
		// get transfer info
		info, err := qbt.GetTransferInfoCtx(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get transfer info")
		}

		rejections := s.qbittorrentCheckIgnoreSlow(rules.DownloadSpeedThreshold, rules.UploadSpeedThreshold, info)
		if len(rejections) > 0 {
			return rejections, nil
		}

		return nil, nil
	}

	return nil, nil
}

func (s *Service) qbittorrentCheckIgnoreSlow(downloadSpeedThreshold int64, uploadSpeedThreshold int64, info *qbittorrent.TransferInfo) []string {
	s.log.Debug().Interface("info", info).Msg("checking client ignore slow torrent rules")

	rejections := make([]string, 0)

	if downloadSpeedThreshold > 0 {
		// if current transfer speed is more than threshold return out and skip
		// DlInfoSpeed is in bytes so lets convert to KB to match DownloadSpeedThreshold
		if info.DlInfoSpeed/1024 >= downloadSpeedThreshold {
			rejection := fmt.Sprintf("total download speed (%d) above threshold: (%d), skipping", info.DlInfoSpeed/1024, downloadSpeedThreshold)

			s.log.Debug().Msg(rejection)

			rejections = append(rejections, rejection)
		}
	}

	if uploadSpeedThreshold > 0 {
		// if current transfer speed is more than threshold return out and skip
		// UpInfoSpeed is in bytes so lets convert to KB to match UploadSpeedThreshold
		if info.UpInfoSpeed/1024 >= uploadSpeedThreshold {
			rejection := fmt.Sprintf("total upload speed (%d) above threshold: (%d), skipping", info.UpInfoSpeed/1024, uploadSpeedThreshold)

			s.log.Debug().Msg(rejection)

			rejections = append(rejections, rejection)
		}
	}

	s.log.Debug().Msg("active downloads are slower than set limit, lets add it")

	return rejections
}
