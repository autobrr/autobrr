// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"encoding/base64"
	"os"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/autobrr/go-deluge"
)

func (s *service) deluge(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action Deluge: %s", action.Name)

	var err error

	client, err := s.clientSvc.GetClient(ctx, action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", action.ClientID)
	}
	action.Client = client

	if !client.Enabled {
		return nil, errors.New("client %s %s not enabled", client.Type, client.Name)
	}

	var rejections []string

	switch client.Type {
	case "DELUGE_V1":
		rejections, err = s.delugeV1(ctx, client, action, release)

	case "DELUGE_V2":
		rejections, err = s.delugeV2(ctx, client, action, release)
	}

	return rejections, err
}

func (s *service) delugeCheckRulesCanDownload(ctx context.Context, del deluge.DelugeClient, client *domain.DownloadClient, action *domain.Action) ([]string, error) {
	s.log.Trace().Msgf("action Deluge: %v check rules", action.Name)

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
				s.log.Debug().Msg("max active downloads reached, skipping")

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

func (s *service) delugeV1(ctx context.Context, client *domain.DownloadClient, action *domain.Action, release domain.Release) ([]string, error) {
	//downloadClient := client.Client.(*deluge.Client)
	downloadClient := deluge.NewV1(deluge.Settings{
		Hostname:             client.Host,
		Port:                 uint(client.Port),
		Login:                client.Username,
		Password:             client.Password,
		DebugServerResponses: true,
		ReadWriteTimeout:     time.Second * 60,
	})

	// perform connection to Deluge server
	err := downloadClient.Connect(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to client %s at %s", client.Name, client.Host)
	}

	defer downloadClient.Close()

	// perform connection to Deluge server
	rejections, err := s.delugeCheckRulesCanDownload(ctx, downloadClient, client, action)
	if err != nil {
		s.log.Error().Err(err).Msgf("error checking client rules: %s", action.Name)
		return nil, err
	}
	if rejections != nil {
		return rejections, nil
	}

	if release.HasMagnetUri() {
		options, err := s.prepareDelugeOptions(action)
		if err != nil {
			return nil, errors.Wrap(err, "could not prepare options")
		}

		s.log.Trace().Msgf("action Deluge options: %+v", options)

		torrentHash, err := downloadClient.AddTorrentMagnet(ctx, release.MagnetURI, &options)
		if err != nil {
			return nil, errors.Wrap(err, "could not add torrent magnet %s to client: %s", release.MagnetURI, client.Name)
		}

		if action.Label != "" {
			labelPluginActive, err := downloadClient.LabelPlugin(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "could not load label plugin for client: %s", client.Name)
			}

			if labelPluginActive != nil {
				err = labelPluginActive.SetTorrentLabel(ctx, torrentHash, action.Label)
				if err != nil {
					if rpcErr, ok := err.(deluge.RPCError); ok && rpcErr.ExceptionMessage == "Unknown Label" {
						if addErr := labelPluginActive.AddLabel(ctx, action.Label); addErr != nil {
							return nil, errors.Wrap(addErr, "could not add label: %s on client: %s", action.Label, client.Name)
						}
						err = labelPluginActive.SetTorrentLabel(ctx, torrentHash, action.Label)
					}
					if err != nil {
						return nil, errors.Wrap(err, "could not set label: %s on client: %s", action.Label, client.Name)
					}
				}
			}
		}

		s.log.Info().Msgf("torrent from magnet with hash %s successfully added to client: '%s'", torrentHash, client.Name)

		return nil, nil
	} else {
		if release.TorrentTmpFile == "" {
			if err := s.downloadSvc.DownloadRelease(ctx, &release); err != nil {
				return nil, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
			}
		}

		t, err := os.ReadFile(release.TorrentTmpFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not read torrent file: %s", release.TorrentTmpFile)
		}

		// encode file to base64 before sending to deluge
		encodedFile := base64.StdEncoding.EncodeToString(t)
		if encodedFile == "" {
			return nil, errors.Wrap(err, "could not encode torrent file: %s", release.TorrentTmpFile)
		}

		options, err := s.prepareDelugeOptions(action)
		if err != nil {
			return nil, errors.Wrap(err, "could not prepare options")
		}

		s.log.Trace().Msgf("action Deluge options: %+v", options)

		torrentHash, err := downloadClient.AddTorrentFile(ctx, release.TorrentTmpFile, encodedFile, &options)
		if err != nil {
			return nil, errors.Wrap(err, "could not add torrent %v to client: %v", release.TorrentTmpFile, client.Name)
		}

		if action.Label != "" {
			labelPluginActive, err := downloadClient.LabelPlugin(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "could not load label plugin for client: %s", client.Name)
			}

			if labelPluginActive != nil {
				err = labelPluginActive.SetTorrentLabel(ctx, torrentHash, action.Label)
				if err != nil {
					if rpcErr, ok := err.(deluge.RPCError); ok && rpcErr.ExceptionMessage == "Unknown Label" {
						if addErr := labelPluginActive.AddLabel(ctx, action.Label); addErr != nil {
							return nil, errors.Wrap(addErr, "could not add label: %s on client: %s", action.Label, client.Name)
						}
						err = labelPluginActive.SetTorrentLabel(ctx, torrentHash, action.Label)
					}
					if err != nil {
						return nil, errors.Wrap(err, "could not set label: %s on client: %s", action.Label, client.Name)
					}
				}
			}
		}

		s.log.Info().Msgf("torrent with hash %s successfully added to client: '%s'", torrentHash, client.Name)
	}

	return nil, nil
}

func (s *service) delugeV2(ctx context.Context, client *domain.DownloadClient, action *domain.Action, release domain.Release) ([]string, error) {
	//downloadClient := client.Client.(*deluge.ClientV2)
	downloadClient := deluge.NewV2(deluge.Settings{
		Hostname:             client.Host,
		Port:                 uint(client.Port),
		Login:                client.Username,
		Password:             client.Password,
		DebugServerResponses: true,
		ReadWriteTimeout:     time.Second * 60,
	})

	// perform connection to Deluge server
	err := downloadClient.Connect(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to client %s at %s", client.Name, client.Host)
	}

	defer downloadClient.Close()

	// perform connection to Deluge server
	rejections, err := s.delugeCheckRulesCanDownload(ctx, downloadClient, client, action)
	if err != nil {
		s.log.Error().Err(err).Msgf("error checking client rules: %s", action.Name)
		return nil, err
	}
	if rejections != nil {
		return rejections, nil
	}

	if release.HasMagnetUri() {
		options, err := s.prepareDelugeOptions(action)
		if err != nil {
			return nil, errors.Wrap(err, "could not prepare options")
		}

		s.log.Trace().Msgf("action Deluge options: %+v", options)

		torrentHash, err := downloadClient.AddTorrentMagnet(ctx, release.MagnetURI, &options)
		if err != nil {
			return nil, errors.Wrap(err, "could not add torrent magnet %s to client: %s", release.MagnetURI, client.Name)
		}

		if action.Label != "" {
			labelPluginActive, err := downloadClient.LabelPlugin(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "could not load label plugin for client: %s", client.Name)
			}

			if labelPluginActive != nil {
				err = labelPluginActive.SetTorrentLabel(ctx, torrentHash, action.Label)
				if err != nil {
					if rpcErr, ok := err.(deluge.RPCError); ok && rpcErr.ExceptionMessage == "Unknown Label" {
						if addErr := labelPluginActive.AddLabel(ctx, action.Label); addErr != nil {
							return nil, errors.Wrap(addErr, "could not add label: %s on client: %s", action.Label, client.Name)
						}
						err = labelPluginActive.SetTorrentLabel(ctx, torrentHash, action.Label)
					}
					if err != nil {
						return nil, errors.Wrap(err, "could not set label: %s on client: %s", action.Label, client.Name)
					}
				}
			}
		}

		s.log.Info().Msgf("torrent with hash %s successfully added to client: '%s'", torrentHash, client.Name)

		return nil, nil
	} else {
		if err := s.downloadSvc.DownloadRelease(ctx, &release); err != nil {
			return nil, errors.Wrap(err, "could not download torrent file for release: %s", release.TorrentName)
		}

		t, err := os.ReadFile(release.TorrentTmpFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not read torrent file: %s", release.TorrentTmpFile)
		}

		// encode file to base64 before sending to deluge
		encodedFile := base64.StdEncoding.EncodeToString(t)
		if encodedFile == "" {
			return nil, errors.Wrap(err, "could not encode torrent file: %s", release.TorrentTmpFile)
		}

		// set options
		options, err := s.prepareDelugeOptions(action)
		if err != nil {
			return nil, errors.Wrap(err, "could not prepare options")
		}

		s.log.Trace().Msgf("action Deluge options: %+v", options)

		torrentHash, err := downloadClient.AddTorrentFile(ctx, release.TorrentTmpFile, encodedFile, &options)
		if err != nil {
			return nil, errors.Wrap(err, "could not add torrent %s to client: %s", release.TorrentTmpFile, client.Name)
		}

		if action.Label != "" {
			labelPluginActive, err := downloadClient.LabelPlugin(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "could not load label plugin for client: %s", client.Name)
			}

			if labelPluginActive != nil {
				err = labelPluginActive.SetTorrentLabel(ctx, torrentHash, action.Label)
				if err != nil {
					if rpcErr, ok := err.(deluge.RPCError); ok && rpcErr.ExceptionMessage == "Unknown Label" {
						if addErr := labelPluginActive.AddLabel(ctx, action.Label); addErr != nil {
							return nil, errors.Wrap(addErr, "could not add label: %s on client: %s", action.Label, client.Name)
						}
						err = labelPluginActive.SetTorrentLabel(ctx, torrentHash, action.Label)
					}
					if err != nil {
						return nil, errors.Wrap(err, "could not set label: %s on client: %s", action.Label, client.Name)
					}
				}
			}
		}

		s.log.Info().Msgf("torrent with hash %s successfully added to client: '%s'", torrentHash, client.Name)
	}

	return nil, nil
}

func (s *service) prepareDelugeOptions(action *domain.Action) (deluge.Options, error) {

	// set options
	options := deluge.Options{}

	if action.Paused {
		options.AddPaused = &action.Paused
	}
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
