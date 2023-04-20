package action

import (
	"context"
	"fmt"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/autobrr/go-qbittorrent"
)

func (s *service) qbittorrent(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action qBittorrent: %s", action.Name)

	c := s.clientSvc.GetCachedClient(ctx, action.ClientID)

	rejections, err := s.qbittorrentCheckRulesCanDownload(ctx, action, c.Dc, c.Qbt)
	if err != nil {
		return nil, errors.Wrap(err, "error checking client rules: %s", action.Name)
	}

	if len(rejections) > 0 {
		return rejections, nil
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
	} else {
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
			if err := c.Qbt.ReannounceTorrentWithRetry(ctx, opts, release.TorrentHash); err != nil {
				return nil, errors.Wrap(err, "could not reannounce torrent: %s", release.TorrentHash)
			}
		}

		s.log.Info().Msgf("torrent with hash %s successfully added to client: '%s'", release.TorrentHash, c.Dc.Name)
	}

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
		opts.SavePath = action.SavePath
		opts.AutoTMM = false
	}
	if action.Category != "" {
		opts.Category = action.Category
	}
	if action.Tags != "" {
		opts.Tags = action.Tags
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

func (s *service) qbittorrentCheckRulesCanDownload(ctx context.Context, action *domain.Action, client *domain.DownloadClient, qbt *qbittorrent.Client) ([]string, error) {
	s.log.Trace().Msgf("action qBittorrent: %v check rules", action.Name)

	checked := false

	// check for active downloads and other rules
	if client.Settings.Rules.Enabled && !action.IgnoreRules {
		activeDownloads, err := qbt.GetTorrentsActiveDownloadsCtx(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not fetch active downloads")
		}

		// make sure it's not set to 0 by default
		if client.Settings.Rules.MaxActiveDownloads > 0 {
			// if max active downloads reached, check speed and if lower than threshold add anyway
			if len(activeDownloads) >= client.Settings.Rules.MaxActiveDownloads {
				if client.Settings.Rules.IgnoreSlowTorrents {
					if client.Settings.Rules.IgnoreSlowTorrentsCondition == domain.IgnoreSlowTorrentsModeMaxReached {
						rejections, err := s.qbittorrentCheckIgnoreSlow(ctx, client, qbt)
						if err != nil {
							return rejections, err
						}

						s.log.Debug().Msg("active downloads are slower than set limit, lets add it")

						checked = true
					}
				} else {
					rejection := "max active downloads reached, skipping"

					s.log.Debug().Msg(rejection)

					return []string{rejection}, nil
				}
			}
		}

		if !checked && client.Settings.Rules.IgnoreSlowTorrentsCondition == domain.IgnoreSlowTorrentsModeAlways {
			rejections, err := s.qbittorrentCheckIgnoreSlow(ctx, client, qbt)
			if err != nil {
				return rejections, err
			}

			if len(rejections) > 0 {
				return rejections, nil
			}
		}
	}

	return nil, nil
}

func (s *service) qbittorrentCheckIgnoreSlow(ctx context.Context, client *domain.DownloadClient, qbt *qbittorrent.Client) ([]string, error) {
	// get transfer info
	info, err := qbt.GetTransferInfoCtx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get transfer info")
	}

	s.log.Debug().Msgf("checking client ignore slow torrent rules: %+v", info)

	if client.Settings.Rules.DownloadSpeedThreshold > 0 {
		// if current transfer speed is more than threshold return out and skip
		// DlInfoSpeed is in bytes so lets convert to KB to match DownloadSpeedThreshold
		if info.DlInfoSpeed/1024 >= client.Settings.Rules.DownloadSpeedThreshold {
			rejection := fmt.Sprintf("max active downloads reached and total download speed (%d) above threshold: (%d), skipping", info.DlInfoSpeed/1024, client.Settings.Rules.DownloadSpeedThreshold)

			s.log.Debug().Msg(rejection)

			return []string{rejection}, nil
		}
	}

	if client.Settings.Rules.UploadSpeedThreshold > 0 {
		// if current transfer speed is more than threshold return out and skip
		// UpInfoSpeed is in bytes so lets convert to KB to match UploadSpeedThreshold
		if info.UpInfoSpeed/1024 >= client.Settings.Rules.UploadSpeedThreshold {
			rejection := fmt.Sprintf("max active downloads reached and total upload speed (%d) above threshold: (%d), skipping", info.UpInfoSpeed/1024, client.Settings.Rules.UploadSpeedThreshold)

			s.log.Debug().Msg(rejection)

			return []string{rejection}, nil
		}
	}

	s.log.Debug().Msg("active downloads are slower than set limit, lets add it")

	return nil, nil
}
