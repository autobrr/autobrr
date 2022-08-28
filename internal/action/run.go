package action

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
	"io/ioutil"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
)

func (s *service) RunAction(action *domain.Action, release domain.Release) ([]string, error) {

	var (
		err        error
		rejections []string
	)

	defer func() {
		if r := recover(); r != nil {
			s.log.Error().Msgf("recovering from panic in run action %v error: %v", action.Name, r)
			err = errors.New("panic in action: %v", action.Name)
			return
		}
	}()

	switch action.Type {
	case domain.ActionTypeTest:
		s.test(action.Name)

	case domain.ActionTypeExec:
		err = s.execCmd(*action, release)

	case domain.ActionTypeWatchFolder:
		err = s.watchFolder(*action, release)

	case domain.ActionTypeWebhook:
		err = s.webhook(*action, release)

	case domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2:
		rejections, err = s.deluge(*action, release)

	case domain.ActionTypeQbittorrent:
		rejections, err = s.qbittorrent(*action, release)

	case domain.ActionTypeRTorrent:
		rejections, err = s.rtorrent(*action, release)

	case domain.ActionTypeTransmission:
		rejections, err = s.transmission(*action, release)

	case domain.ActionTypeRadarr:
		rejections, err = s.radarr(*action, release)

	case domain.ActionTypeSonarr:
		rejections, err = s.sonarr(*action, release)

	case domain.ActionTypeLidarr:
		rejections, err = s.lidarr(*action, release)

	case domain.ActionTypeWhisparr:
		rejections, err = s.whisparr(*action, release)

	default:
		s.log.Warn().Msgf("unsupported action type: %v", action.Type)
		return rejections, err
	}

	rlsActionStatus := &domain.ReleaseActionStatus{
		ReleaseID:  release.ID,
		Status:     domain.ReleasePushStatusApproved,
		Action:     action.Name,
		Type:       action.Type,
		Client:     action.Client.Name,
		Filter:     release.Filter.Name,
		Rejections: []string{},
		Timestamp:  time.Now(),
	}

	payload := &domain.NotificationPayload{
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
		s.log.Error().Err(err).Msgf("process action failed: %v for '%v'", action.Name, release.TorrentName)

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
	s.bus.Publish("events:notification", &payload.Event, payload)

	return rejections, err
}

func (s *service) test(name string) {
	s.log.Info().Msgf("action TEST: %v", name)
}

func (s *service) watchFolder(action domain.Action, release domain.Release) error {
	if release.TorrentTmpFile == "" {
		if err := release.DownloadTorrentFile(); err != nil {
			return errors.Wrap(err, "watch folder: could not download torrent file for release: %v", release.TorrentName)
		}
	}

	if len(release.TorrentDataRawBytes) == 0 && strings.Contains(action.WebhookData, "TorrentDataRawBytes") {
		t, err := ioutil.ReadFile(release.TorrentTmpFile)
		if err != nil {
			return errors.Wrap(err, "could not read torrent file: %v", release.TorrentTmpFile)
		}

		release.TorrentDataRawBytes = t
	}

	m := domain.NewMacro(release)

	// parse and replace values in argument string before continuing
	watchFolderArgs, err := m.Parse(action.WatchFolder)
	if err != nil {
		return errors.Wrap(err, "could not parse watch folder macro: %v", action.WatchFolder)
	}

	s.log.Trace().Msgf("action WATCH_FOLDER: %v file: %v", watchFolderArgs, release.TorrentTmpFile)

	// Open original file
	original, err := os.Open(release.TorrentTmpFile)
	if err != nil {
		return errors.Wrap(err, "could not open temp file: %v", release.TorrentTmpFile)
	}
	defer original.Close()

	_, tmpFileName := path.Split(release.TorrentTmpFile)
	fullFileName := path.Join(watchFolderArgs, tmpFileName+".torrent")

	// Create folder
	err = os.MkdirAll(watchFolderArgs, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "could not create new folders %v", fullFileName)
	}

	// Create new file
	newFile, err := os.Create(fullFileName)
	if err != nil {
		return errors.Wrap(err, "could not create new file %v", fullFileName)
	}
	defer newFile.Close()

	// Copy file
	_, err = io.Copy(newFile, original)
	if err != nil {
		return errors.Wrap(err, "could not copy file %v to watch folder", fullFileName)
	}

	s.log.Info().Msgf("saved file to watch folder: %v", fullFileName)

	return nil
}

func (s *service) webhook(action domain.Action, release domain.Release) error {
	if release.TorrentTmpFile == "" && (strings.Contains(action.WebhookData, "TorrentPathName") || strings.Contains(action.WebhookData, "TorrentDataRawBytes")) {
		if err := release.DownloadTorrentFile(); err != nil {
			return errors.Wrap(err, "webhook: could not download torrent file for release: %v", release.TorrentName)
		}
	}

	if len(release.TorrentDataRawBytes) == 0 && strings.Contains(action.WebhookData, "TorrentDataRawBytes") {
		t, err := ioutil.ReadFile(release.TorrentTmpFile)
		if err != nil {
			return errors.Wrap(err, "could not read torrent file: %v", release.TorrentTmpFile)
		}

		release.TorrentDataRawBytes = t
	}

	m := domain.NewMacro(release)

	// parse and replace values in argument string before continuing
	dataArgs, err := m.Parse(action.WebhookData)
	if err != nil {
		return errors.Wrap(err, "could not parse webhook data macro: %v", action.WebhookData)
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
		return errors.Wrap(err, "could not build request for webhook")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	res, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not make request for webhook")
	}

	defer res.Body.Close()

	s.log.Info().Msgf("successfully ran webhook action: '%v' to: %v payload: %v", action.Name, action.WebhookHost, dataArgs)

	return nil
}
