package action

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
)

func (s *service) RunAction(action *domain.Action, release domain.Release) ([]string, error) {

	var err error
	var rejections []string

	switch action.Type {
	case domain.ActionTypeTest:
		s.test(action.Name)

	case domain.ActionTypeExec:
		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				s.log.Error().Stack().Err(err)
				break
			}
		}

		s.execCmd(release, *action)

	case domain.ActionTypeWatchFolder:
		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				s.log.Error().Stack().Err(err)
				break
			}
		}

		s.watchFolder(*action, release)

	case domain.ActionTypeWebhook:
		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				s.log.Error().Stack().Err(err)
				break
			}
		}

		s.webhook(*action, release)

	case domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2:
		canDownload, err := s.delugeCheckRulesCanDownload(*action)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
			break
		}
		if !canDownload {
			rejections = []string{"max active downloads reached, skipping"}
			break
		}

		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				s.log.Error().Stack().Err(err)
				break
			}
		}

		err = s.deluge(*action, release)
		if err != nil {
			s.log.Error().Stack().Err(err).Msg("error sending torrent to Deluge")
			break
		}

	case domain.ActionTypeQbittorrent:
		canDownload, client, err := s.qbittorrentCheckRulesCanDownload(*action)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
			break
		}
		if !canDownload {
			rejections = []string{"max active downloads reached, skipping"}
			break
		}

		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(); err != nil {
				s.log.Error().Stack().Err(err)
				break
			}
		}

		err = s.qbittorrent(client, *action, release)
		if err != nil {
			s.log.Error().Stack().Err(err).Msg("error sending torrent to qBittorrent")
			break
		}

	case domain.ActionTypeRadarr:
		rejections, err = s.radarr(release, *action)
		if err != nil {
			s.log.Error().Stack().Err(err).Msg("error sending torrent to radarr")
			break
		}

	case domain.ActionTypeSonarr:
		rejections, err = s.sonarr(release, *action)
		if err != nil {
			s.log.Error().Stack().Err(err).Msg("error sending torrent to sonarr")
			break
		}

	case domain.ActionTypeLidarr:
		rejections, err = s.lidarr(release, *action)
		if err != nil {
			s.log.Error().Stack().Err(err).Msg("error sending torrent to lidarr")
			break
		}

	case domain.ActionTypeWhisparr:
		rejections, err = s.whisparr(release, *action)
		if err != nil {
			s.log.Error().Stack().Err(err).Msg("error sending torrent to whisparr")
			break
		}

	default:
		s.log.Warn().Msgf("unsupported action type: %v", action.Type)
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

	payload := &domain.NotificationPayload{
		Subject:        release.TorrentName,
		Message:        "New release!",
		Event:          domain.NotificationEventPushApproved,
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
		s.log.Err(err).Stack().Msgf("process action failed: %v for '%v'", action.Name, release.TorrentName)

		rlsActionStatus.Status = domain.ReleasePushStatusErr
		rlsActionStatus.Rejections = []string{err.Error()}

		payload.Event = domain.NotificationEventPushError
		payload.Status = domain.ReleasePushStatusErr
		payload.Rejections = []string{err.Error()}
	}

	if rejections != nil {
		rlsActionStatus.Status = domain.ReleasePushStatusRejected
		rlsActionStatus.Rejections = rejections

		payload.Event = domain.NotificationEventPushRejected
		payload.Status = domain.ReleasePushStatusRejected
		payload.Rejections = rejections
	}

	// send event for actions
	s.bus.Publish("release:push", rlsActionStatus)

	// send separate event for notifications
	s.bus.Publish("events:notification", payload.Event, payload)

	return rejections, err
}

func (s *service) CheckCanDownload(actions []domain.Action) bool {
	for _, action := range actions {
		if !action.Enabled {
			// only run active actions
			continue
		}

		s.log.Debug().Msgf("action-service: check can download action: %v", action.Name)

		switch action.Type {
		case domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2:
			canDownload, err := s.delugeCheckRulesCanDownload(action)
			if err != nil {
				s.log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
				continue
			}
			if !canDownload {
				continue
			}

			return true

		case domain.ActionTypeQbittorrent:
			canDownload, _, err := s.qbittorrentCheckRulesCanDownload(action)
			if err != nil {
				s.log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
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
	s.log.Info().Msgf("action TEST: %v", name)
}

func (s *service) watchFolder(action domain.Action, release domain.Release) {
	m := NewMacro(release)

	// parse and replace values in argument string before continuing
	watchFolderArgs, err := m.Parse(action.WatchFolder)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("could not parse macro: %v", action.WatchFolder)
	}

	s.log.Trace().Msgf("action WATCH_FOLDER: %v file: %v", watchFolderArgs, release.TorrentTmpFile)

	// Open original file
	original, err := os.Open(release.TorrentTmpFile)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("could not open temp file '%v'", release.TorrentTmpFile)
		return
	}
	defer original.Close()

	_, tmpFileName := path.Split(release.TorrentTmpFile)
	fullFileName := path.Join(watchFolderArgs, tmpFileName+".torrent")

	// Create new file
	newFile, err := os.Create(fullFileName)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("could not create new temp file '%v'", fullFileName)
		return
	}
	defer newFile.Close()

	// Copy file
	_, err = io.Copy(newFile, original)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("could not copy file %v to watch folder", fullFileName)
		return
	}

	s.log.Info().Msgf("saved file to watch folder: %v", fullFileName)
}

func (s *service) webhook(action domain.Action, release domain.Release) {
	m := NewMacro(release)

	// parse and replace values in argument string before continuing
	dataArgs, err := m.Parse(action.WebhookData)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("could not parse macro: %v", action.WebhookData)
		return
	}

	s.log.Trace().Msgf("action WEBHOOK: '%v' file: %v", action.Name, release.TorrentName)
	s.log.Trace().Msgf("webhook action '%v' - host: %v data: %v", action.Name, action.WebhookHost, action.WebhookData)

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := http.Client{Transport: t, Timeout: 15 * time.Second}

	req, err := http.NewRequest(http.MethodPost, action.WebhookHost, bytes.NewBufferString(dataArgs))
	if err != nil {
		s.log.Error().Err(err).Msgf("webhook client request error: %v", action.WebhookHost)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	res, err := client.Do(req)
	if err != nil {
		s.log.Error().Err(err).Msgf("webhook client request error: %v", action.WebhookHost)
		return
	}

	defer res.Body.Close()

	s.log.Info().Msgf("successfully ran webhook action: '%v' to: %v payload: %v", action.Name, action.WebhookHost, dataArgs)

	return
}
