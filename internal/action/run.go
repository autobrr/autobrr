package action

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/domain"
)

func (s *service) RunActions(actions []domain.Action, release domain.Release) error {

	for _, action := range actions {
		// only run active actions
		if !action.Enabled {
			continue
		}

		log.Debug().Msgf("process action: %v for '%v'", action.Name, release.TorrentName)

		err := s.runAction(action, release)
		if err != nil {
			log.Err(err).Stack().Msgf("process action failed: %v for '%v'", action.Name, release.TorrentName)

			s.bus.Publish("release:store-action-status", &domain.ReleaseActionStatus{
				ReleaseID:  release.ID,
				Status:     domain.ReleasePushStatusErr,
				Action:     action.Name,
				Type:       action.Type,
				Rejections: []string{err.Error()},
				Timestamp:  time.Now(),
			})

			s.bus.Publish("events:release:push", &domain.EventsReleasePushed{
				ReleaseName:    release.TorrentName,
				Filter:         release.Filter.Name,
				Indexer:        release.Indexer,
				InfoHash:       release.TorrentHash,
				Size:           release.Size,
				Status:         domain.ReleasePushStatusErr,
				Action:         action.Name,
				ActionType:     action.Type,
				Rejections:     []string{err.Error()},
				Protocol:       domain.ReleaseProtocolTorrent,
				Implementation: domain.ReleaseImplementationIRC,
				Timestamp:      time.Now(),
			})
		}
	}

	// safe to delete tmp file

	return nil
}

func (s *service) RunAction(action *domain.Action, release domain.Release) ([]string, error) {

	var err error
	var rejections []string

	switch action.Type {
	case domain.ActionTypeTest:
		s.test(action.Name)

	case domain.ActionTypeExec:
		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				log.Error().Stack().Err(err)
				break
			}
		}

		s.execCmd(release, *action)

	case domain.ActionTypeWatchFolder:
		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				log.Error().Stack().Err(err)
				break
			}
		}

		s.watchFolder(*action, release)

	case domain.ActionTypeWebhook:
		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				log.Error().Stack().Err(err)
				break
			}
		}

		s.webhook(*action, release)

	case domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2:
		canDownload, err := s.delugeCheckRulesCanDownload(*action)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
			break
		}
		if !canDownload {
			rejections = []string{"max active downloads reached, skipping"}
			break
		}

		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				log.Error().Stack().Err(err)
				break
			}
		}

		err = s.deluge(*action, release)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error sending torrent to Deluge")
			break
		}

	case domain.ActionTypeQbittorrent:
		canDownload, client, err := s.qbittorrentCheckRulesCanDownload(*action)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
			break
		}
		if !canDownload {
			rejections = []string{"max active downloads reached, skipping"}
			break
		}

		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				log.Error().Stack().Err(err)
				break
			}
		}

		err = s.qbittorrent(client, *action, release)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error sending torrent to qBittorrent")
			break
		}

	case domain.ActionTypeRadarr:
		rejections, err = s.radarr(release, *action)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error sending torrent to radarr")
			break
		}

	case domain.ActionTypeSonarr:
		rejections, err = s.sonarr(release, *action)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error sending torrent to sonarr")
			break
		}

	case domain.ActionTypeLidarr:
		rejections, err = s.lidarr(release, *action)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error sending torrent to lidarr")
			break
		}

	case domain.ActionTypeWhisparr:
		rejections, err = s.whisparr(release, *action)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error sending torrent to whisparr")
			break
		}

	default:
		log.Warn().Msgf("unsupported action type: %v", action.Type)
		return rejections, err
	}

	rlsActionStatus := &domain.ReleaseActionStatus{
		ReleaseID:  release.ID,
		Status:     domain.ReleasePushStatusApproved,
		Action:     action.Name,
		Type:       action.Type,
		Rejections: []string{},
		Timestamp:  time.Now(),
	}

	notificationEvent := &domain.EventsReleasePushed{
		ReleaseName:    release.TorrentName,
		Filter:         release.Filter.Name,
		Indexer:        release.Indexer,
		InfoHash:       release.TorrentHash,
		Size:           release.Size,
		Status:         domain.ReleasePushStatusApproved,
		Action:         action.Name,
		ActionType:     action.Type,
		ActionClient:   action.Client.Name,
		Rejections:     []string{},
		Protocol:       domain.ReleaseProtocolTorrent,
		Implementation: domain.ReleaseImplementationIRC,
		Timestamp:      time.Now(),
	}

	if err != nil {
		log.Err(err).Stack().Msgf("process action failed: %v for '%v'", action.Name, release.TorrentName)

		rlsActionStatus.Status = domain.ReleasePushStatusErr
		rlsActionStatus.Rejections = []string{err.Error()}

		notificationEvent.Status = domain.ReleasePushStatusErr
		notificationEvent.Rejections = []string{err.Error()}
	}

	if rejections != nil {
		rlsActionStatus.Status = domain.ReleasePushStatusRejected
		rlsActionStatus.Rejections = rejections

		notificationEvent.Status = domain.ReleasePushStatusRejected
		notificationEvent.Rejections = rejections
	}

	// send event for actions
	s.bus.Publish("release:push", rlsActionStatus)

	// send separate event for notifications
	s.bus.Publish("events:release:push", notificationEvent)

	return rejections, err
}

func (s *service) runAction(action domain.Action, release domain.Release) error {

	var err error
	var rejections []string

	switch action.Type {
	case domain.ActionTypeTest:
		s.test(action.Name)

	case domain.ActionTypeExec:
		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				log.Error().Stack().Err(err)
				return err
			}
		}

		s.execCmd(release, action)

	case domain.ActionTypeWatchFolder:
		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				log.Error().Stack().Err(err)
				return err
			}
		}

		s.watchFolder(action, release)

	case domain.ActionTypeWebhook:
		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				log.Error().Stack().Err(err)
				return err
			}
		}

		s.webhook(action, release)

	case domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2:
		canDownload, err := s.delugeCheckRulesCanDownload(action)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
			return err
		}
		if !canDownload {
			rejections = []string{"max active downloads reached, skipping"}
			break
		}

		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				log.Error().Stack().Err(err)
				return err
			}
		}

		err = s.deluge(action, release)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error sending torrent to Deluge")
			return err
		}

	case domain.ActionTypeQbittorrent:
		canDownload, client, err := s.qbittorrentCheckRulesCanDownload(action)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
			return err
		}
		if !canDownload {
			rejections = []string{"max active downloads reached, skipping"}
			break
		}

		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				log.Error().Stack().Err(err)
				return err
			}
		}

		err = s.qbittorrent(client, action, release)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error sending torrent to qBittorrent")
			return err
		}

	case domain.ActionTypeRadarr:
		rejections, err = s.radarr(release, action)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error sending torrent to radarr")
			return err
		}

	case domain.ActionTypeSonarr:
		rejections, err = s.sonarr(release, action)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error sending torrent to sonarr")
			return err
		}

	case domain.ActionTypeLidarr:
		rejections, err = s.lidarr(release, action)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error sending torrent to lidarr")
			return err
		}

	case domain.ActionTypeWhisparr:
		rejections, err = s.whisparr(release, action)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error sending torrent to whisparr")
			return err
		}

	default:
		log.Warn().Msgf("unsupported action: %v type: %v", action.Name, action.Type)
		return nil
	}

	rlsActionStatus := &domain.ReleaseActionStatus{
		ReleaseID:  release.ID,
		Status:     domain.ReleasePushStatusApproved,
		Action:     action.Name,
		Type:       action.Type,
		Rejections: []string{},
		Timestamp:  time.Now(),
	}

	notificationEvent := &domain.EventsReleasePushed{
		ReleaseName:    release.TorrentName,
		Filter:         release.Filter.Name,
		Indexer:        release.Indexer,
		InfoHash:       release.TorrentHash,
		Size:           release.Size,
		Status:         domain.ReleasePushStatusApproved,
		Action:         action.Name,
		ActionType:     action.Type,
		Rejections:     []string{},
		Protocol:       domain.ReleaseProtocolTorrent,
		Implementation: domain.ReleaseImplementationIRC,
		Timestamp:      time.Now(),
	}

	if rejections != nil {
		rlsActionStatus.Status = domain.ReleasePushStatusRejected
		rlsActionStatus.Rejections = rejections

		notificationEvent.Status = domain.ReleasePushStatusRejected
		notificationEvent.Rejections = rejections
	}

	// send event for actions
	s.bus.Publish("release:push", rlsActionStatus)

	// send separate event for notifications
	s.bus.Publish("events:release:push", notificationEvent)

	return nil
}

func (s *service) CheckCanDownload(actions []domain.Action) bool {
	for _, action := range actions {
		if !action.Enabled {
			// only run active actions
			continue
		}

		log.Debug().Msgf("action-service: check can download action: %v", action.Name)

		switch action.Type {
		case domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2:
			canDownload, err := s.delugeCheckRulesCanDownload(action)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
				continue
			}
			if !canDownload {
				continue
			}

			return true

		case domain.ActionTypeQbittorrent:
			canDownload, _, err := s.qbittorrentCheckRulesCanDownload(action)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
				continue
			}
			if !canDownload {
				continue
			}

			return true
		}
	}

	return false
}

func (s *service) test(name string) {
	log.Info().Msgf("action TEST: %v", name)
}

func (s *service) watchFolder(action domain.Action, release domain.Release) {
	m := NewMacro(release)

	// parse and replace values in argument string before continuing
	watchFolderArgs, err := m.Parse(action.WatchFolder)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not parse macro: %v", action.WatchFolder)
	}

	log.Trace().Msgf("action WATCH_FOLDER: %v file: %v", watchFolderArgs, release.TorrentTmpFile)

	// Open original file
	original, err := os.Open(release.TorrentTmpFile)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not open temp file '%v'", release.TorrentTmpFile)
		return
	}
	defer original.Close()

	_, tmpFileName := path.Split(release.TorrentTmpFile)
	fullFileName := path.Join(watchFolderArgs, tmpFileName+".torrent")

	// Create new file
	newFile, err := os.Create(fullFileName)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not create new temp file '%v'", fullFileName)
		return
	}
	defer newFile.Close()

	// Copy file
	_, err = io.Copy(newFile, original)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not copy file %v to watch folder", fullFileName)
		return
	}

	log.Info().Msgf("saved file to watch folder: %v", fullFileName)
}

func (s *service) webhook(action domain.Action, release domain.Release) {
	m := NewMacro(release)

	// parse and replace values in argument string before continuing
	dataArgs, err := m.Parse(action.WebhookData)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not parse macro: %v", action.WebhookData)
		return
	}

	log.Trace().Msgf("action WEBHOOK: '%v' file: %v", action.Name, release.TorrentName)
	log.Trace().Msgf("webhook action '%v' - host: %v data: %v", action.Name, action.WebhookHost, action.WebhookData)

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := http.Client{Transport: t, Timeout: 15 * time.Second}

	req, err := http.NewRequest(http.MethodPost, action.WebhookHost, bytes.NewBufferString(dataArgs))
	if err != nil {
		log.Error().Err(err).Msgf("webhook client request error: %v", action.WebhookHost)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	res, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("webhook client request error: %v", action.WebhookHost)
		return
	}

	defer res.Body.Close()

	log.Info().Msgf("successfully ran webhook action: '%v' to: %v payload: %v", action.Name, action.WebhookHost, dataArgs)

	return
}
