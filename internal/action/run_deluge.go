// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"encoding/base64"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/autobrr/go-deluge"
	"github.com/rs/zerolog"
)

func (s *Service) runDeluge(ctx context.Context, action *domain.Action, release *domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Debug().Msg("running Deluge action")

	var err error

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

	var rejections []string

	switch cfg.Type {
	case domain.DownloaderTypeDelugeV1:
		client, convErr := downloader.ClientAs[*deluge.Client](instance)
		if convErr != nil {
			return nil, convErr
		}
		rejections, err = s.delugeV1(ctx, cfg, client, action, release)

	case domain.DownloaderTypeDelugeV2:
		client, convErr := downloader.ClientAs[*deluge.ClientV2](instance)
		if convErr != nil {
			return nil, convErr
		}
		rejections, err = s.delugeV2(ctx, cfg, client, action, release)
	}

	return rejections, err
}

func (s *Service) delugeCheckRulesCanDownload(ctx context.Context, del deluge.DelugeClient, client *domain.Downloader, action *domain.Action) ([]string, error) {
	l := zerolog.Ctx(ctx)

	l.Trace().Msg("check rules")

	// check for active downloads and other rules
	if client.Settings.Rules.Enabled && !action.IgnoreRules {
		activeDownloads, err := del.TorrentsStatus(ctx, deluge.StateDownloading, nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not fetch downloading torrents")
		}

		// make sure it's not set to 0 by default
		if client.Settings.Rules.MaxActiveDownloads > 0 {

			// if max active downloads reached, check speed and if lower than threshold add anyway
			if len(activeDownloads) >= client.Settings.Rules.MaxActiveDownloads {
				l.Debug().Msg("max active downloads reached, skipping")

				rejections := []string{"max active downloads reached, skipping"}
				return rejections, nil

				//	// TODO handle ignore slow torrents
				//if client.Settings.Rules.IgnoreSlowTorrents {
				//
				//	// get session state
				//	// gives type conversion errors
				//	state, err := deluge.GetSessionStatus()
				//	if err != nil {
				//		s.log.Error().Err(err).Msg("could not get session state")
				//		return err
				//	}
				//
				//	if int64(state.DownloadRate)*1024 >= client.Settings.Rules.DownloadSpeedThreshold {
				//		s.log.Trace().Msg("max active downloads reached, skip adding")
				//		return nil
				//	}
				//
				//	s.log.Trace().Msg("active downloads are slower than set limit, lets add it")
				//}
			}
		}
	}

	return nil, nil
}

func (s *Service) delugeV1(ctx context.Context, cfg *domain.Downloader, client *deluge.Client, action *domain.Action, release *domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	//client := deluge.NewV1(deluge.Settings{
	//	Hostname:             cfg.Host,
	//	Port:                 uint(cfg.Port),
	//	Login:                cfg.Username,
	//	Password:             cfg.Password,
	//	DebugServerResponses: true,
	//	ReadWriteTimeout:     time.Second * 60,
	//})

	// perform connection to Deluge server
	err := client.Connect(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to client %s at %s", cfg.Name, cfg.Host)
	}

	defer client.Close()

	// perform connection to Deluge server
	rejections, err := s.delugeCheckRulesCanDownload(ctx, client, cfg, action)
	if err != nil {
		l.Error().Err(err).Msg("error checking client rules")
		return nil, err
	}
	if rejections != nil {
		return rejections, nil
	}

	options, err := s.prepareDelugeOptions(action)
	if err != nil {
		return nil, errors.Wrap(err, "could not prepare options")
	}

	l.Trace().Interface("options", options).Msg("action deluge options")

	var torrentHash string

	switch {
	case release.HasMagnetUri():
		torrentHash, err = client.AddTorrentMagnet(ctx, release.MagnetURI, &options)
		if err != nil {
			return nil, errors.Wrap(err, "could not add torrent magnet %s to client: %s", release.MagnetURI, cfg.Name)
		}
		break

	default:
		if err := s.rlsDownloadSvc.DownloadRelease(ctx, release); err != nil {
			return nil, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
		}

		// encode file to base64 before sending to deluge
		encodedFile := base64.StdEncoding.EncodeToString(release.TorrentDataRawBytes)
		if encodedFile == "" {
			return nil, errors.New("could not encode torrent file for release: %s", release.TorrentName)
		}

		torrentHash, err = client.AddTorrentFile(ctx, release.TorrentHash+".torrent", encodedFile, &options)
		if err != nil {
			return nil, errors.Wrap(err, "could not add torrent %v to client: %v", release.TorrentName, cfg.Name)
		}
	}

	if action.Label != "" {
		labelPluginActive, err := client.LabelPlugin(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not load label plugin for client: %s", cfg.Name)
		}

		if labelPluginActive != nil {
			if err := delugeSetOrCreateTorrentLabel(ctx, labelPluginActive, cfg.Name, torrentHash, action.Label); err != nil {
				return nil, errors.Wrap(err, "could not set label: %s on client: %s", action.Label, cfg.Name)
			}
		}
	}

	l.Info().Str("hash", torrentHash).Str("client", cfg.Name).Msg("release successfully added to client")

	return nil, nil
}

// delugeSetOrCreateTorrentLabel set torrent label if it exists or create label if it does not
func delugeSetOrCreateTorrentLabel(ctx context.Context, plugin *deluge.LabelPlugin, clientName string, hash string, label string) error {
	err := plugin.SetTorrentLabel(ctx, hash, label)
	if err != nil {
		// if label does not exist the client will throw an RPC error.
		// We can parse that and check for specific error for Unknown Label and then create the label
		var rpcErr deluge.RPCError
		switch {
		case errors.As(err, &rpcErr) && rpcErr.ExceptionMessage == "Unknown Label":
			if addErr := plugin.AddLabel(ctx, label); addErr != nil {
				return errors.Wrap(addErr, "could not add label: %s on client: %s", label, clientName)
			}

			if err = plugin.SetTorrentLabel(ctx, hash, label); err != nil {
				return errors.Wrap(err, "could not set label: %s on client: %s", label, clientName)
			}

		default:
			return errors.Wrap(err, "could not set label: %s on client: %s", label, clientName)
		}
	}

	return nil
}

func (s *Service) delugeV2(ctx context.Context, cfg *domain.Downloader, client *deluge.ClientV2, action *domain.Action, release *domain.Release) ([]string, error) {
	l := zerolog.Ctx(ctx)

	//client := deluge.NewV2(deluge.Settings{
	//	Hostname:             cfg.Host,
	//	Port:                 uint(cfg.Port),
	//	Login:                cfg.Username,
	//	Password:             cfg.Password,
	//	DebugServerResponses: true,
	//	ReadWriteTimeout:     time.Second * 60,
	//})

	// perform connection to Deluge server
	err := client.Connect(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to client %s at %s", cfg.Name, cfg.Host)
	}

	defer client.Close()

	// perform connection to Deluge server
	rejections, err := s.delugeCheckRulesCanDownload(ctx, client, cfg, action)
	if err != nil {
		l.Error().Err(err).Msg("error checking client rules")
		return nil, err
	}
	if rejections != nil {
		return rejections, nil
	}

	options, err := s.prepareDelugeOptions(action)
	if err != nil {
		return nil, errors.Wrap(err, "could not prepare options")
	}

	l.Trace().Interface("options", options).Msg("action deluge options")

	var torrentHash string

	switch {
	case release.HasMagnetUri():
		torrentHash, err = client.AddTorrentMagnet(ctx, release.MagnetURI, &options)
		if err != nil {
			return nil, errors.Wrap(err, "could not add torrent magnet %s to client: %s", release.MagnetURI, cfg.Name)
		}
	default:
		if err := s.rlsDownloadSvc.DownloadRelease(ctx, release); err != nil {
			return nil, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
		}

		// encode file to base64 before sending to deluge
		encodedFile := base64.StdEncoding.EncodeToString(release.TorrentDataRawBytes)
		if encodedFile == "" {
			return nil, errors.New("could not encode torrent file for release: %s", release.TorrentName)
		}

		torrentHash, err = client.AddTorrentFile(ctx, release.TorrentHash+".torrent", encodedFile, &options)
		if err != nil {
			return nil, errors.Wrap(err, "could not add torrent %s to client: %s", release.TorrentName, cfg.Name)
		}

	}

	if action.Label != "" {
		labelPluginActive, err := client.LabelPlugin(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not load label plugin for client: %s", cfg.Name)
		}

		if labelPluginActive != nil {
			if err := delugeSetOrCreateTorrentLabel(ctx, labelPluginActive, cfg.Name, torrentHash, action.Label); err != nil {
				return nil, errors.Wrap(err, "could not set label: %s on client: %s", action.Label, cfg.Name)
			}
		}
	}

	l.Info().Str("hash", torrentHash).Str("client", cfg.Name).Msg("torrent successfully added to client")

	return nil, nil
}

func (s *Service) prepareDelugeOptions(action *domain.Action) (deluge.Options, error) {
	// set options
	options := deluge.Options{}

	// always set; to override client default
	options.AddPaused = &action.Paused

	if action.SavePath != "" {
		options.DownloadLocation = &action.SavePath
	}
	if action.LimitDownloadSpeed > 0 {
		maxDL := int(action.LimitDownloadSpeed)
		options.MaxDownloadSpeed = &maxDL
	}
	if action.LimitUploadSpeed > 0 {
		maxUL := int(action.LimitUploadSpeed)
		options.MaxUploadSpeed = &maxUL
	}
	if action.SkipHashCheck {
		options.V2.SeedMode = &action.SkipHashCheck
	}

	return options, nil
}
