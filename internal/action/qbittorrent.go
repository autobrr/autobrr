package action

import (
	"strconv"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/qbittorrent"
	"github.com/rs/zerolog/log"
)

func (s *service) qbittorrent(action domain.Action, hash string, torrentFile string) error {
	log.Trace().Msgf("action QBITTORRENT: %v", torrentFile)

	// get client for action
	client, err := s.clientSvc.FindByID(action.ClientID)
	if err != nil {
		log.Error().Err(err).Msgf("error finding client: %v", action.ClientID)
		return err
	}

	if client == nil {
		return err
	}

	qbtSettings := qbittorrent.Settings{
		Hostname: client.Host,
		Port:     uint(client.Port),
		Username: client.Username,
		Password: client.Password,
		SSL:      client.SSL,
	}

	qbt := qbittorrent.NewClient(qbtSettings)
	// save cookies?
	err = qbt.Login()
	if err != nil {
		log.Error().Err(err).Msgf("error logging into client: %v", action.ClientID)
		return err
	}

	// TODO check for active downloads and other rules

	options := map[string]string{}

	if action.Paused {
		options["paused"] = "true"
	}
	if action.SavePath != "" {
		options["savepath"] = action.SavePath
		options["autoTMM"] = "false"
	}
	if action.Category != "" {
		options["category"] = action.Category
	}
	if action.Tags != "" {
		options["tags"] = action.Tags
	}
	if action.LimitUploadSpeed > 0 {
		options["upLimit"] = strconv.FormatInt(action.LimitUploadSpeed, 10)
	}
	if action.LimitDownloadSpeed > 0 {
		options["dlLimit"] = strconv.FormatInt(action.LimitDownloadSpeed, 10)
	}

	err = qbt.AddTorrentFromFile(torrentFile, options)
	if err != nil {
		log.Error().Err(err).Msgf("error sending to client: %v", action.ClientID)
		return err
	}

	if !action.Paused && hash != "" {
		err = checkTrackerStatus(*qbt, hash)
		if err != nil {
			log.Error().Err(err).Msgf("could not get tracker status for torrent: %v", hash)
			return err
		}
	}

	log.Debug().Msgf("torrent %v successfully added to: %v", hash, client.Name)

	return nil
}

func checkTrackerStatus(qb qbittorrent.Client, hash string) error {
	announceOK := false
	attempts := 0

	for attempts < REANNOUNCE_MAX_ATTEMPTS {
		log.Debug().Msgf("RE-ANNOUNCE %v attempt: %v", hash, attempts)

		// initial sleep to give tracker a head start
		time.Sleep(REANNOUNCE_INTERVAL * time.Millisecond)

		trackers, err := qb.GetTorrentTrackers(hash)
		if err != nil {
			log.Error().Err(err).Msgf("could not get trackers of torrent: %v", hash)
			return err
		}

		// check if status not working or something else
		_, working := findTrackerStatus(trackers, qbittorrent.TrackerStatusOK)

		if !working {
			err = qb.ReAnnounceTorrents([]string{hash})
			if err != nil {
				log.Error().Err(err).Msgf("could not get re-announce torrent: %v", hash)
				return err
			}

			attempts++
			continue
		} else {
			log.Debug().Msgf("RE-ANNOUNCE %v OK", hash)

			announceOK = true
			break
		}
	}

	if !announceOK {
		log.Debug().Msgf("RE-ANNOUNCE %v took too long, deleting torrent", hash)

		err := qb.DeleteTorrents([]string{hash}, false)
		if err != nil {
			log.Error().Err(err).Msgf("could not delete torrent: %v", hash)
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
func findTrackerStatus(slice []qbittorrent.TorrentTracker, status qbittorrent.TrackerStatus) (int, bool) {
	for i, item := range slice {
		if item.Status == status {
			return i, true
		}
	}
	return -1, false
}
