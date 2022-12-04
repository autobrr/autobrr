package action

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/autobrr/go-qbittorrent"
)

func (s *service) qbittorrent(ctx context.Context, action domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action qBittorrent: %v", action.Name)

	c := s.clientSvc.GetCachedClient(ctx, action.ClientID)

	rejections, err := s.qbittorrentCheckRulesCanDownload(ctx, action, c.Dc, c.Qbt)
	if err != nil {
		return nil, errors.Wrap(err, "error checking client rules: %v", action.Name)
	}

	if len(rejections) > 0 {
		return rejections, nil
	}

	if release.TorrentTmpFile == "" {
		if err := release.DownloadTorrentFileCtx(ctx); err != nil {
			return nil, errors.Wrap(err, "error downloading torrent file for release: %v", release.TorrentName)
		}
	}

	// macros handle args and replace vars
	m := domain.NewMacro(release)

	options, err := s.prepareQbitOptions(action, m)
	if err != nil {
		return nil, errors.Wrap(err, "could not prepare options")
	}

	s.log.Trace().Msgf("action qBittorrent options: %+v", options)

	if err = c.Qbt.AddTorrentFromFileCtx(ctx, release.TorrentTmpFile, options); err != nil {
		return nil, errors.Wrap(err, "could not add torrent %v to client: %v", release.TorrentTmpFile, c.Dc.Name)
	}

	if !action.Paused && !action.ReAnnounceSkip && release.TorrentHash != "" {
		opts := qbittorrent.ReannounceOptions{
			Interval:        int(action.ReAnnounceInterval),
			MaxAttempts:     int(action.ReAnnounceMaxAttempts),
			DeleteOnFailure: action.ReAnnounceDelete,
		}
		if err := c.Qbt.ReannounceTorrentWithRetry(ctx, opts, release.TorrentHash); err != nil {
			return nil, errors.Wrap(err, "could not reannounce torrent: %v", release.TorrentHash)
		}
	}

	s.log.Info().Msgf("torrent with hash %v successfully added to client: '%v'", release.TorrentHash, c.Dc.Name)

	return nil, nil
}

func (s *service) prepareQbitOptions(action domain.Action, m domain.Macro) (map[string]string, error) {
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
		// parse and replace values in argument string before continuing
		actionArgs, err := m.Parse(action.SavePath)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse savepath macro: %v", action.SavePath)
		}

		opts.SavePath = actionArgs
		opts.AutoTMM = false
	}
	if action.Category != "" {
		// parse and replace values in argument string before continuing
		categoryArgs, err := m.Parse(action.Category)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse category macro: %v", action.Category)
		}

		opts.Category = categoryArgs
	}
	if action.Tags != "" {
		// parse and replace values in argument string before continuing
		tagsArgs, err := m.Parse(action.Tags)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse tags macro: %v", action.Tags)
		}

		opts.Tags = tagsArgs
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

func (s *service) qbittorrentCheckRulesCanDownload(ctx context.Context, action domain.Action, client *domain.DownloadClient, qbt *qbittorrent.Client) ([]string, error) {
	s.log.Trace().Msgf("action qBittorrent: %v check rules", action.Name)

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
					// check speeds of downloads
					info, err := qbt.GetTransferInfoCtx(ctx)
					if err != nil {
						return nil, errors.Wrap(err, "could not get transfer info")
					}

					// if current transfer speed is more than threshold return out and skip
					// DlInfoSpeed is in bytes so lets convert to KB to match DownloadSpeedThreshold
					if info.DlInfoSpeed/1024 >= client.Settings.Rules.DownloadSpeedThreshold {
						s.log.Debug().Msg("max active downloads reached, skipping")

						rejections := []string{"max active downloads reached, skipping"}
						return rejections, nil
					}

					s.log.Debug().Msg("active downloads are slower than set limit, lets add it")
				} else {
					s.log.Debug().Msg("max active downloads reached, skipping")

					rejections := []string{"max active downloads reached, skipping"}
					return rejections, nil
				}
			}
		}
	}

	return nil, nil
}
