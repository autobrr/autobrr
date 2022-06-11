package action

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/qbittorrent"
)

const ReannounceMaxAttempts = 50
const ReannounceInterval = 7000

func (s *service) qbittorrent(qbt *qbittorrent.Client, action domain.Action, release domain.Release) error {
	s.log.Debug().Msgf("action qBittorrent: %v", action.Name)

	// macros handle args and replace vars
	m := NewMacro(release)

	options := map[string]string{}

	if action.Paused {
		options["paused"] = "true"
	}
	if action.SavePath != "" {
		// parse and replace values in argument string before continuing
		actionArgs, err := m.Parse(action.SavePath)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("could not parse macro: %v", action.SavePath)
			return err
		}

		options["savepath"] = actionArgs
		options["autoTMM"] = "false"
	}
	if action.Category != "" {
		// parse and replace values in argument string before continuing
		categoryArgs, err := m.Parse(action.Category)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("could not parse macro: %v", action.Category)
			return err
		}

		options["category"] = categoryArgs
	}
	if action.Tags != "" {
		// parse and replace values in argument string before continuing
		tagsArgs, err := m.Parse(action.Tags)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("could not parse macro: %v", action.Tags)
			return err
		}

		options["tags"] = tagsArgs
	}
	if action.LimitUploadSpeed > 0 {
		options["upLimit"] = strconv.FormatInt(action.LimitUploadSpeed * 1000, 10)
	}
	if action.LimitDownloadSpeed > 0 {
		options["dlLimit"] = strconv.FormatInt(action.LimitDownloadSpeed * 1000, 10)
	}
	if action.LimitRatio > 0 {
		options["ratioLimit"] = strconv.FormatFloat(action.LimitRatio, 'r', 2, 64)
	}
	if action.LimitSeedTime > 0 {
		options["seedingTimeLimit"] = strconv.FormatInt(action.LimitSeedTime, 10)
	}

	s.log.Trace().Msgf("action qBittorrent options: %+v", options)

	err := qbt.AddTorrentFromFile(release.TorrentTmpFile, options)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("could not add torrent %v to client: %v", release.TorrentTmpFile, qbt.Name)
		return err
	}

	if !action.Paused && !action.ReAnnounceSkip && release.TorrentHash != "" {
		if err := s.reannounceTorrent(qbt, action, release.TorrentHash); err != nil {
			s.log.Error().Stack().Err(err).Msgf("could not reannounce torrent: %v", release.TorrentHash)
			return err
		}
	}

	s.log.Info().Msgf("torrent with hash %v successfully added to client: '%v'", release.TorrentHash, qbt.Name)

	return nil
}

func (s *service) qbittorrentCheckRulesCanDownload(action domain.Action) (bool, *qbittorrent.Client, error) {
	s.log.Trace().Msgf("action qBittorrent: %v check rules", action.Name)

	// get client for action
	client, err := s.clientSvc.FindByID(context.TODO(), action.ClientID)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("error finding client: %v", action.ClientID)
		return false, nil, err
	}

	if client == nil {
		return false, nil, err
	}

	qbtSettings := qbittorrent.Settings{
		Hostname:      client.Host,
		Port:          uint(client.Port),
		Username:      client.Username,
		Password:      client.Password,
		TLS:           client.TLS,
		TLSSkipVerify: client.TLSSkipVerify,
	}

	// only set basic auth if enabled
	if client.Settings.Basic.Auth {
		qbtSettings.BasicAuth = client.Settings.Basic.Auth
		qbtSettings.Basic.Username = client.Settings.Basic.Username
		qbtSettings.Basic.Password = client.Settings.Basic.Password
	}

	qbt := qbittorrent.NewClient(qbtSettings)
	qbt.Name = client.Name
	// save cookies?
	err = qbt.Login()
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("error logging into client: %v", client.Host)
		return false, nil, err
	}

	// check for active downloads and other rules
	if client.Settings.Rules.Enabled && !action.IgnoreRules {
		activeDownloads, err := qbt.GetTorrentsActiveDownloads()
		if err != nil {
			s.log.Error().Stack().Err(err).Msg("could not fetch downloading torrents")
			return false, nil, err
		}

		// make sure it's not set to 0 by default
		if client.Settings.Rules.MaxActiveDownloads > 0 {

			// if max active downloads reached, check speed and if lower than threshold add anyways
			if len(activeDownloads) >= client.Settings.Rules.MaxActiveDownloads {
				if client.Settings.Rules.IgnoreSlowTorrents {
					// check speeds of downloads
					info, err := qbt.GetTransferInfo()
					if err != nil {
						s.log.Error().Err(err).Msg("could not get transfer info")
						return false, nil, err
					}

					// if current transfer speed is more than threshold return out and skip
					// DlInfoSpeed is in bytes so lets convert to KB to match DownloadSpeedThreshold
					if info.DlInfoSpeed/1024 >= client.Settings.Rules.DownloadSpeedThreshold {
						s.log.Debug().Msg("max active downloads reached, skipping")
						return false, nil, nil
					}

					s.log.Debug().Msg("active downloads are slower than set limit, lets add it")
				} else {
					s.log.Debug().Msg("max active downloads reached, skipping")
					return false, nil, nil
				}
			}
		}
	}

	return true, qbt, nil
}

func (s *service) reannounceTorrent(qb *qbittorrent.Client, action domain.Action, hash string) error {
	announceOK := false
	attempts := 0

	interval := ReannounceInterval
	if action.ReAnnounceInterval > 0 {
		interval = int(action.ReAnnounceInterval)
	}

	maxAttempts := ReannounceMaxAttempts
	if action.ReAnnounceMaxAttempts > 0 {
		maxAttempts = int(action.ReAnnounceMaxAttempts)
	}

	for attempts < maxAttempts {
		s.log.Debug().Msgf("qBittorrent - run re-announce %v attempt: %v", hash, attempts)

		// add delay for next run
		time.Sleep(time.Duration(interval) * time.Second)

		trackers, err := qb.GetTorrentTrackers(hash)
		if err != nil {
			s.log.Error().Err(err).Msgf("qBittorrent - could not get trackers for torrent: %v", hash)
			return err
		}

		if trackers == nil {
			attempts++
			continue
		}

		s.log.Trace().Msgf("qBittorrent - run re-announce %v attempt: %v trackers (%+v)", hash, attempts, trackers)

		// check if status not working or something else
		working := isTrackerStatusOK(trackers)
		if working {
			s.log.Debug().Msgf("qBittorrent - re-announce for %v OK", hash)

			announceOK = true

			// if working lets return
			return nil
		}

		s.log.Trace().Msgf("qBittorrent - not working yet, lets re-announce %v attempt: %v", hash, attempts)
		err = qb.ReAnnounceTorrents([]string{hash})
		if err != nil {
			s.log.Error().Err(err).Msgf("qBittorrent - could not get re-announce torrent: %v", hash)
			return err
		}

		attempts++
	}

	// delete on failure to reannounce
	if !announceOK && action.ReAnnounceDelete {
		s.log.Debug().Msgf("qBittorrent - re-announce for %v took too long, deleting torrent", hash)

		err := qb.DeleteTorrents([]string{hash}, false)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("qBittorrent - could not delete torrent: %v", hash)
			return err
		}
	}

	return nil
}

// Check if status not working or something else
// https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)#get-torrent-trackers
//  0 Tracker is disabled (used for DHT, PeX, and LSD)
//  1 Tracker has not been contacted yet
//  2 Tracker has been contacted and is working
//  3 Tracker is updating
//  4 Tracker has been contacted, but it is not working (or doesn't send proper replies)
func isTrackerStatusOK(trackers []qbittorrent.TorrentTracker) bool {
	for _, tracker := range trackers {
		if tracker.Status == qbittorrent.TrackerStatusDisabled {
			continue
		}

		// check for certain messages before the tracker status to catch ok status with unreg msg
		if isUnregistered(tracker.Message) {
			return false
		}

		if tracker.Status == qbittorrent.TrackerStatusOK {
			return true
		}
	}

	return false
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
