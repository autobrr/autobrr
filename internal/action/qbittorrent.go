package action

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/rs/zerolog"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/qbittorrent"
)

const ReannounceMaxAttempts = 50
const ReannounceInterval = 7000

func (s *service) qbittorrent(action domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action qBittorrent: %v", action.Name)

	// get client for action
	client, err := s.clientSvc.FindByID(context.TODO(), action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "error finding client: %v", action.ClientID)
	}

	if client == nil {
		return nil, errors.New("could not find client by id: %v", action.ClientID)
	}

	qbtSettings := qbittorrent.Settings{
		Name:          client.Name,
		Hostname:      client.Host,
		Port:          uint(client.Port),
		Username:      client.Username,
		Password:      client.Password,
		TLS:           client.TLS,
		TLSSkipVerify: client.TLSSkipVerify,
	}

	// setup sub logger adapter which is compatible with *log.Logger
	qbtSettings.Log = zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "qBittorrent").Str("client", client.Name).Logger(), zerolog.TraceLevel)

	// only set basic auth if enabled
	if client.Settings.Basic.Auth {
		qbtSettings.BasicAuth = client.Settings.Basic.Auth
		qbtSettings.Basic.Username = client.Settings.Basic.Username
		qbtSettings.Basic.Password = client.Settings.Basic.Password
	}

	qbt := qbittorrent.NewClient(qbtSettings)

	// only login if we have a password
	if qbtSettings.Password != "" {
		if err = qbt.Login(); err != nil {
			return nil, errors.Wrap(err, "could not log into client: %v at %v", client.Name, client.Host)
		}
	}

	rejections, err := s.qbittorrentCheckRulesCanDownload(action, client, qbt)
	if err != nil {
		return nil, errors.Wrap(err, "error checking client rules: %v", action.Name)
	}

	if rejections != nil {
		return rejections, nil
	}

	if release.TorrentTmpFile == "" {
		err = release.DownloadTorrentFile()
		if err != nil {
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

	if err = qbt.AddTorrentFromFile(release.TorrentTmpFile, options); err != nil {
		return nil, errors.Wrap(err, "could not add torrent %v to client: %v", release.TorrentTmpFile, client.Name)
	}

	if !action.Paused && !action.ReAnnounceSkip && release.TorrentHash != "" {
		if err := s.reannounceTorrent(qbt, action, release.TorrentHash); err != nil {
			return nil, errors.Wrap(err, "could not reannounce torrent: %v", release.TorrentHash)
		}
	}

	s.log.Info().Msgf("torrent with hash %v successfully added to client: '%v'", release.TorrentHash, client.Name)

	return nil, nil
}

func (s *service) prepareQbitOptions(action domain.Action, m domain.Macro) (map[string]string, error) {

	options := map[string]string{}

	if action.Paused {
		options["paused"] = "true"
	}
	if action.SkipHashCheck {
		options["skip_checking"] = "true"
	}
	if action.ContentLayout != "" {
		if action.ContentLayout == domain.ActionContentLayoutSubfolderCreate {
			options["root_folder"] = "true"
		} else if action.ContentLayout == domain.ActionContentLayoutSubfolderNone {
			options["root_folder"] = "false"
		}
		// if ORIGINAL then leave empty
	}
	if action.SavePath != "" {
		// parse and replace values in argument string before continuing
		actionArgs, err := m.Parse(action.SavePath)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse savepath macro: %v", action.SavePath)
		}

		options["savepath"] = actionArgs
		options["autoTMM"] = "false"
	}
	if action.Category != "" {
		// parse and replace values in argument string before continuing
		categoryArgs, err := m.Parse(action.Category)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse category macro: %v", action.Category)
		}

		options["category"] = categoryArgs
	}
	if action.Tags != "" {
		// parse and replace values in argument string before continuing
		tagsArgs, err := m.Parse(action.Tags)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse tags macro: %v", action.Tags)
		}

		options["tags"] = tagsArgs
	}
	if action.LimitUploadSpeed > 0 {
		options["upLimit"] = strconv.FormatInt(action.LimitUploadSpeed*1000, 10)
	}
	if action.LimitDownloadSpeed > 0 {
		options["dlLimit"] = strconv.FormatInt(action.LimitDownloadSpeed*1000, 10)
	}
	if action.LimitRatio > 0 {
		options["ratioLimit"] = strconv.FormatFloat(action.LimitRatio, 'r', 2, 64)
	}
	if action.LimitSeedTime > 0 {
		options["seedingTimeLimit"] = strconv.FormatInt(action.LimitSeedTime, 10)
	}

	return options, nil
}

func (s *service) qbittorrentCheckRulesCanDownload(action domain.Action, client *domain.DownloadClient, qbt *qbittorrent.Client) ([]string, error) {
	s.log.Trace().Msgf("action qBittorrent: %v check rules", action.Name)

	// check for active downloads and other rules
	if client.Settings.Rules.Enabled && !action.IgnoreRules {
		activeDownloads, err := qbt.GetTorrentsActiveDownloads()
		if err != nil {
			return nil, errors.Wrap(err, "could not fetch active downloads")
		}

		// make sure it's not set to 0 by default
		if client.Settings.Rules.MaxActiveDownloads > 0 {

			// if max active downloads reached, check speed and if lower than threshold add anyways
			if len(activeDownloads) >= client.Settings.Rules.MaxActiveDownloads {
				if client.Settings.Rules.IgnoreSlowTorrents {
					// check speeds of downloads
					info, err := qbt.GetTransferInfo()
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
			return errors.Wrap(err, "could not get trackers for torrent with hash: %v", hash)
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
			return errors.Wrap(err, "could not re-announce torrent with hash: %v", hash)
		}

		attempts++
	}

	// delete on failure to reannounce
	if !announceOK && action.ReAnnounceDelete {
		s.log.Debug().Msgf("qBittorrent - re-announce for %v took too long, deleting torrent", hash)

		err := qb.DeleteTorrents([]string{hash}, false)
		if err != nil {
			return errors.Wrap(err, "could not delete torrent with hash: %v", hash)
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
