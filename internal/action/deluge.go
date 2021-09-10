package action

import (
	"encoding/base64"
	"errors"
	"io/ioutil"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	delugeClient "github.com/gdm85/go-libdeluge"
	"github.com/rs/zerolog/log"
)

func (s *service) deluge(action domain.Action, torrentFile string) error {
	log.Debug().Msgf("action Deluge: %v", action.Name)

	var err error

	// get client for action
	client, err := s.clientSvc.FindByID(action.ClientID)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error finding client: %v", action.ClientID)
		return err
	}

	if client == nil {
		return errors.New("no client found")
	}

	settings := delugeClient.Settings{
		Hostname:             client.Host,
		Port:                 uint(client.Port),
		Login:                client.Username,
		Password:             client.Password,
		DebugServerResponses: true,
		ReadWriteTimeout:     time.Second * 20,
	}

	switch client.Type {
	case "DELUGE_V1":
		if err = delugeV1(client, settings, action, torrentFile); err != nil {
			return err
		}

	case "DELUGE_V2":
		if err = delugeV2(client, settings, action, torrentFile); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) delugeCheckRulesCanDownload(action domain.Action) (bool, error) {
	log.Trace().Msgf("action Deluge: %v check rules", action.Name)

	// get client for action
	client, err := s.clientSvc.FindByID(action.ClientID)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error finding client: %v ID %v", action.Name, action.ClientID)
		return false, err
	}

	if client == nil {
		return false, errors.New("no client found")
	}

	settings := delugeClient.Settings{
		Hostname:             client.Host,
		Port:                 uint(client.Port),
		Login:                client.Username,
		Password:             client.Password,
		DebugServerResponses: true,
		ReadWriteTimeout:     time.Second * 20,
	}
	var deluge delugeClient.DelugeClient

	switch client.Type {
	case "DELUGE_V1":
		deluge = delugeClient.NewV1(settings)

	case "DELUGE_V2":
		deluge = delugeClient.NewV2(settings)
	}

	// perform connection to Deluge server
	err = deluge.Connect()
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error logging into client: %v %v", client.Name, client.Host)
		return false, err
	}

	defer deluge.Close()

	// check for active downloads and other rules
	if client.Settings.Rules.Enabled && !action.IgnoreRules {
		activeDownloads, err := deluge.TorrentsStatus(delugeClient.StateDownloading, nil)
		if err != nil {
			log.Error().Stack().Err(err).Msg("Deluge - could not fetch downloading torrents")
			return false, err
		}

		// make sure it's not set to 0 by default
		if client.Settings.Rules.MaxActiveDownloads > 0 {

			// if max active downloads reached, check speed and if lower than threshold add anyways
			if len(activeDownloads) >= client.Settings.Rules.MaxActiveDownloads {
				log.Debug().Msg("max active downloads reached, skipping")
				return false, nil

				//	// TODO handle ignore slow torrents
				//if client.Settings.Rules.IgnoreSlowTorrents {
				//
				//	// get session state
				//	// gives type conversion errors
				//	state, err := deluge.GetSessionStatus()
				//	if err != nil {
				//		log.Error().Err(err).Msg("could not get session state")
				//		return err
				//	}
				//
				//	if int64(state.DownloadRate)*1024 >= client.Settings.Rules.DownloadSpeedThreshold {
				//		log.Trace().Msg("max active downloads reached, skip adding")
				//		return nil
				//	}
				//
				//	log.Trace().Msg("active downloads are slower than set limit, lets add it")
				//}
			}
		}
	}

	return true, nil
}

func delugeV1(client *domain.DownloadClient, settings delugeClient.Settings, action domain.Action, torrentFile string) error {

	deluge := delugeClient.NewV1(settings)

	// perform connection to Deluge server
	err := deluge.Connect()
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error logging into client: %v %v", client.Name, client.Host)
		return err
	}

	defer deluge.Close()

	t, err := ioutil.ReadFile(torrentFile)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not read torrent file: %v", torrentFile)
		return err
	}

	// encode file to base64 before sending to deluge
	encodedFile := base64.StdEncoding.EncodeToString(t)
	if encodedFile == "" {
		log.Error().Stack().Err(err).Msgf("could not encode torrent file: %v", torrentFile)
		return err
	}

	// set options
	options := delugeClient.Options{}

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

	log.Trace().Msgf("action Deluge options: %+v", options)

	torrentHash, err := deluge.AddTorrentFile(torrentFile, encodedFile, &options)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not add torrent %v to client: %v", torrentFile, client.Name)
		return err
	}

	if action.Label != "" {
		p, err := deluge.LabelPlugin()
		if err != nil {
			log.Error().Stack().Err(err).Msgf("could not load label plugin: %v", client.Name)
			return err
		}

		if p != nil {
			// TODO first check if label exists, if not, add it, otherwise set
			err = p.SetTorrentLabel(torrentHash, action.Label)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("could not set label: %v on client: %v", action.Label, client.Name)
				return err
			}
		}
	}

	log.Info().Msgf("torrent with hash %v successfully added to client: '%v'", torrentHash, client.Name)

	return nil
}

func delugeV2(client *domain.DownloadClient, settings delugeClient.Settings, action domain.Action, torrentFile string) error {

	deluge := delugeClient.NewV2(settings)

	// perform connection to Deluge server
	err := deluge.Connect()
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error logging into client: %v %v", client.Name, client.Host)
		return err
	}

	defer deluge.Close()

	t, err := ioutil.ReadFile(torrentFile)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not read torrent file: %v", torrentFile)
		return err
	}

	// encode file to base64 before sending to deluge
	encodedFile := base64.StdEncoding.EncodeToString(t)
	if encodedFile == "" {
		log.Error().Stack().Err(err).Msgf("could not encode torrent file: %v", torrentFile)
		return err
	}

	// set options
	options := delugeClient.Options{}

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

	log.Trace().Msgf("action Deluge options: %+v", options)

	torrentHash, err := deluge.AddTorrentFile(torrentFile, encodedFile, &options)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not add torrent %v to client: %v", torrentFile, client.Name)
		return err
	}

	if action.Label != "" {
		p, err := deluge.LabelPlugin()
		if err != nil {
			log.Error().Stack().Err(err).Msgf("could not load label plugin: %v", client.Name)
			return err
		}

		if p != nil {
			// TODO first check if label exists, if not, add it, otherwise set
			err = p.SetTorrentLabel(torrentHash, action.Label)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("could not set label: %v on client: %v", action.Label, client.Name)
				return err
			}
		}
	}

	log.Info().Msgf("torrent with hash %v successfully added to client: '%v'", torrentHash, client.Name)

	return nil
}
