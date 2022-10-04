package action

import (
	"context"
	"encoding/base64"
	"os"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	delugeClient "github.com/gdm85/go-libdeluge"
)

func (s *service) deluge(action domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action Deluge: %v", action.Name)

	var err error

	// get client for action
	client, err := s.clientSvc.FindByID(context.TODO(), action.ClientID)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("error finding client: %v", action.ClientID)
		return nil, err
	}

	if client == nil {
		return nil, errors.New("could not find client by id: %v", action.ClientID)
	}

	var rejections []string

	switch client.Type {
	case "DELUGE_V1":
		rejections, err = s.delugeV1(client, action, release)

	case "DELUGE_V2":
		rejections, err = s.delugeV2(client, action, release)
	}

	return rejections, err
}

func (s *service) delugeCheckRulesCanDownload(deluge delugeClient.DelugeClient, client *domain.DownloadClient, action domain.Action) ([]string, error) {
	s.log.Trace().Msgf("action Deluge: %v check rules", action.Name)

	// check for active downloads and other rules
	if client.Settings.Rules.Enabled && !action.IgnoreRules {
		activeDownloads, err := deluge.TorrentsStatus(delugeClient.StateDownloading, nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not fetch downloading torrents")
		}

		// make sure it's not set to 0 by default
		if client.Settings.Rules.MaxActiveDownloads > 0 {

			// if max active downloads reached, check speed and if lower than threshold add anyways
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

func (s *service) delugeV1(client *domain.DownloadClient, action domain.Action, release domain.Release) ([]string, error) {
	settings := delugeClient.Settings{
		Hostname:             client.Host,
		Port:                 uint(client.Port),
		Login:                client.Username,
		Password:             client.Password,
		DebugServerResponses: true,
		ReadWriteTimeout:     time.Second * 20,
	}

	deluge := delugeClient.NewV1(settings)

	// perform connection to Deluge server
	err := deluge.Connect()
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to client %v at %v", client.Name, client.Host)
	}

	defer deluge.Close()

	// perform connection to Deluge server
	rejections, err := s.delugeCheckRulesCanDownload(deluge, client, action)
	if err != nil {
		s.log.Error().Err(err).Msgf("error checking client rules: %v", action.Name)
		return nil, err
	}
	if rejections != nil {
		return rejections, nil
	}

	if release.TorrentTmpFile == "" {
		if err = release.DownloadTorrentFile(); err != nil {
			s.log.Error().Err(err).Msgf("could not download torrent file for release: %v", release.TorrentName)
			return nil, err
		}
	}

	t, err := os.ReadFile(release.TorrentTmpFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not read torrent file: %v", release.TorrentTmpFile)
	}

	// encode file to base64 before sending to deluge
	encodedFile := base64.StdEncoding.EncodeToString(t)
	if encodedFile == "" {
		return nil, errors.Wrap(err, "could not encode torrent file: %v", release.TorrentTmpFile)
	}

	// macros handle args and replace vars
	m := domain.NewMacro(release)

	options, err := s.prepareDelugeOptions(action, m)
	if err != nil {
		return nil, errors.Wrap(err, "could not prepare options")
	}

	s.log.Trace().Msgf("action Deluge options: %+v", options)

	torrentHash, err := deluge.AddTorrentFile(release.TorrentTmpFile, encodedFile, &options)
	if err != nil {
		return nil, errors.Wrap(err, "could not add torrent %v to client: %v", release.TorrentTmpFile, client.Name)
	}

	if action.Label != "" {
		labelPluginActive, err := deluge.LabelPlugin()
		if err != nil {
			return nil, errors.Wrap(err, "could not load label plugin for client: %v", client.Name)
		}

		// parse and replace values in argument string before continuing
		labelArgs, err := m.Parse(action.Label)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse macro label: %v", action.Label)
		}

		if labelPluginActive != nil {
			// TODO first check if label exists, if not, add it, otherwise set
			err = labelPluginActive.SetTorrentLabel(torrentHash, labelArgs)
			if err != nil {
				return nil, errors.Wrap(err, "could not set label: %v on client: %v", action.Label, client.Name)
			}
		}
	}

	s.log.Info().Msgf("torrent with hash %v successfully added to client: '%v'", torrentHash, client.Name)

	return nil, nil
}

func (s *service) delugeV2(client *domain.DownloadClient, action domain.Action, release domain.Release) ([]string, error) {
	settings := delugeClient.Settings{
		Hostname:             client.Host,
		Port:                 uint(client.Port),
		Login:                client.Username,
		Password:             client.Password,
		DebugServerResponses: true,
		ReadWriteTimeout:     time.Second * 20,
	}

	deluge := delugeClient.NewV2(settings)

	// perform connection to Deluge server
	err := deluge.Connect()
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to client %v at %v", client.Name, client.Host)
	}

	defer deluge.Close()

	// perform connection to Deluge server
	rejections, err := s.delugeCheckRulesCanDownload(deluge, client, action)
	if err != nil {
		s.log.Error().Err(err).Msgf("error checking client rules: %v", action.Name)
		return nil, err
	}
	if rejections != nil {
		return rejections, nil
	}

	if release.TorrentTmpFile == "" {
		if err = release.DownloadTorrentFile(); err != nil {
			s.log.Error().Err(err).Msgf("could not download torrent file for release: %v", release.TorrentName)
			return nil, err
		}
	}

	t, err := os.ReadFile(release.TorrentTmpFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not read torrent file: %v", release.TorrentTmpFile)
	}

	// encode file to base64 before sending to deluge
	encodedFile := base64.StdEncoding.EncodeToString(t)
	if encodedFile == "" {
		return nil, errors.Wrap(err, "could not encode torrent file: %v", release.TorrentTmpFile)
	}

	// macros handle args and replace vars
	m := domain.NewMacro(release)

	// set options
	options, err := s.prepareDelugeOptions(action, m)
	if err != nil {
		return nil, errors.Wrap(err, "could not prepare options")
	}

	s.log.Trace().Msgf("action Deluge options: %+v", options)

	torrentHash, err := deluge.AddTorrentFile(release.TorrentTmpFile, encodedFile, &options)
	if err != nil {
		return nil, errors.Wrap(err, "could not add torrent %v to client: %v", release.TorrentTmpFile, client.Name)
	}

	if action.Label != "" {
		labelPluginActive, err := deluge.LabelPlugin()
		if err != nil {
			return nil, errors.Wrap(err, "could not load label plugin for client: %v", client.Name)
		}

		// parse and replace values in argument string before continuing
		labelArgs, err := m.Parse(action.Label)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse macro label: %v", action.Label)
		}

		if labelPluginActive != nil {
			// TODO first check if label exists, if not, add it, otherwise set
			err = labelPluginActive.SetTorrentLabel(torrentHash, labelArgs)
			if err != nil {
				return nil, errors.Wrap(err, "could not set label: %v on client: %v", action.Label, client.Name)
			}
		}
	}

	s.log.Info().Msgf("torrent with hash %v successfully added to client: '%v'", torrentHash, client.Name)

	return nil, nil
}

func (s *service) prepareDelugeOptions(action domain.Action, m domain.Macro) (delugeClient.Options, error) {

	// set options
	options := delugeClient.Options{}

	if action.Paused {
		options.AddPaused = &action.Paused
	}
	if action.SavePath != "" {
		// parse and replace values in argument string before continuing
		savePathArgs, err := m.Parse(action.SavePath)
		if err != nil {
			return options, errors.Wrap(err, "could not parse save path macro: %v", action.SavePath)
		}

		options.DownloadLocation = &savePathArgs
	}
	if action.LimitDownloadSpeed > 0 {
		maxDL := int(action.LimitDownloadSpeed)
		options.MaxDownloadSpeed = &maxDL
	}
	if action.LimitUploadSpeed > 0 {
		maxUL := int(action.LimitUploadSpeed)
		options.MaxUploadSpeed = &maxUL
	}

	return options, nil
}
