// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/autobrr/go-qbittorrent"
)

func (s *service) qbittorrent(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action qBittorrent: %s", action.Name)

	c := s.clientSvc.GetCachedClient(ctx, action.ClientID)

	if c.Dc.Settings.Rules.Enabled && !action.IgnoreRules {
		// check for active downloads and other rules
		rejections, err := s.qbittorrentCheckRulesCanDownload(ctx, action, c.Dc.Settings.Rules, c.Qbt)
		if err != nil {
			return nil, errors.Wrap(err, "error checking client rules: %s", action.Name)
		}

		if len(rejections) > 0 {
			return rejections, nil
		}
	}

	if release.HasMagnetUri() {
		options, err := s.prepareQbitOptions(action)
		if err != nil {
			return nil, errors.Wrap(err, "could not prepare options")
		}

		s.log.Trace().Msgf("action qBittorrent options: %+v", options)

		if err = c.Qbt.AddTorrentFromUrlCtx(ctx, release.MagnetURI, options); err != nil {
			return nil, errors.Wrap(err, "could not add torrent %s to client: %s", release.MagnetURI, c.Dc.Name)
		}

		s.log.Info().Msgf("torrent from magnet successfully added to client: '%s'", c.Dc.Name)

		return nil, nil
	}

	if release.TorrentTmpFile == "" {
		if err := release.DownloadTorrentFileCtx(ctx); err != nil {
			return nil, errors.Wrap(err, "error downloading torrent file for release: %s", release.TorrentName)
		}
	}

	options, err := s.prepareQbitOptions(action)
	if err != nil {
		return nil, errors.Wrap(err, "could not prepare options")
	}

	s.log.Trace().Msgf("action qBittorrent options: %+v", options)

	if err = c.Qbt.AddTorrentFromFileCtx(ctx, release.TorrentTmpFile, options); err != nil {
		return nil, errors.Wrap(err, "could not add torrent %s to client: %s", release.TorrentTmpFile, c.Dc.Name)
	}

	if !action.Paused && !action.ReAnnounceSkip && release.TorrentHash != "" {
		opts := qbittorrent.ReannounceOptions{
			Interval:        int(action.ReAnnounceInterval),
			MaxAttempts:     int(action.ReAnnounceMaxAttempts),
			DeleteOnFailure: action.ReAnnounceDelete,
		}

		if err := c.Qbt.ReannounceTorrentWithRetry(ctx, release.TorrentHash, &opts); err != nil {
			if errors.Is(err, qbittorrent.ErrReannounceTookTooLong) {
				return []string{fmt.Sprintf("re-announce took too long for hash: %s", release.TorrentHash)}, nil
			}

			return nil, errors.Wrap(err, "could not reannounce torrent: %s", release.TorrentHash)
		}
	}

	s.log.Info().Msgf("torrent with hash %s successfully added to client: '%s'", release.TorrentHash, c.Dc.Name)

	return nil, nil
}

func (s *service) prepareQbitOptions(action *domain.Action) (map[string]string, error) {
	opts := &qbittorrent.TorrentAddOptions{}

	opts.Paused = false
	if action.Paused {
		opts.Paused = true
	}
	if action.SkipHashCheck {
		opts.SkipHashCheck = true
	}
	if action.ContentLayout != "" {
		if action.ContentLayout == domain.ActionContentLayoutSubfolderCreate {
			opts.ContentLayout = qbittorrent.ContentLayoutSubfolderCreate
		} else if action.ContentLayout == domain.ActionContentLayoutSubfolderNone {
			opts.ContentLayout = qbittorrent.ContentLayoutSubfolderNone
		}
		// if ORIGINAL then leave empty
	}
	if action.SavePath != "" {
		opts.SavePath = strings.TrimSpace(action.SavePath)
		opts.AutoTMM = false
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
func (s *service) qbittorrentCheckRulesCanDownload(ctx context.Context, action *domain.Action, rules domain.DownloadClientRules, qbt *qbittorrent.Client) ([]string, error) {
	s.log.Trace().Msgf("action qBittorrent: %s check rules", action.Name)

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

				s.log.Debug().Msg(rejection)

				return []string{rejection}, nil
			}

			if rules.IgnoreSlowTorrentsCondition == domain.IgnoreSlowTorrentsModeMaxReached {
				// get transfer info
				info, err := qbt.GetTransferInfoCtx(ctx)
				if err != nil {
					return nil, errors.Wrap(err, "could not get transfer info")
				}

				rejections := s.qbittorrentCheckIgnoreSlow(rules.DownloadSpeedThreshold, rules.UploadSpeedThreshold, info)
				if len(rejections) > 0 {
					return rejections, nil
				}

				s.log.Debug().Msg("active downloads are slower than set limit, lets add it")

				return nil, nil
			}
		}

		// if less, then we must check if ignore slow always which means we can't return here
	}

	// if max active downloads is unlimited or not reached, lets check if ignore slow always should be checked
	if rules.IgnoreSlowTorrentsCondition == domain.IgnoreSlowTorrentsModeAlways {
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

func (s *service) qbittorrentCheckIgnoreSlow(downloadSpeedThreshold int64, uploadSpeedThreshold int64, info *qbittorrent.TransferInfo) []string {
	s.log.Debug().Msgf("checking client ignore slow torrent rules: %+v", info)

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
