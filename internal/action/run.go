package action

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
)

func (s *service) RunAction(ctx context.Context, action *domain.Action, release *domain.Release) ([]string, error) {

	var (
		err        error
		rejections []string
	)

	defer func() {
		if r := recover(); r != nil {
			s.log.Error().Msgf("recovering from panic in run action %s error: %v", action.Name, r)
			err = errors.New("panic in action: %s", action.Name)
			return
		}
	}()

	// if set, try to resolve MagnetURI before parsing macros
	// to allow webhook and exec to get the magnet_uri
	if err := release.ResolveMagnetUri(ctx); err != nil {
		return nil, err
	}

	// parse all macros in one go
	if err := action.ParseMacros(release); err != nil {
		return nil, err
	}

	switch action.Type {
	case domain.ActionTypeTest:
		s.test(action.Name)

	case domain.ActionTypeExec:
		err = s.execCmd(ctx, action, *release)

	case domain.ActionTypeWatchFolder:
		err = s.watchFolder(ctx, action, *release)

	case domain.ActionTypeWebhook:
		err = s.webhook(ctx, action, *release)

	case domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2:
		rejections, err = s.deluge(ctx, action, *release)

	case domain.ActionTypeQbittorrent:
		rejections, err = s.qbittorrent(ctx, action, *release)

	case domain.ActionTypeRTorrent:
		rejections, err = s.rtorrent(ctx, action, *release)

	case domain.ActionTypeTransmission:
		rejections, err = s.transmission(ctx, action, *release)

	case domain.ActionTypePorla:
		rejections, err = s.porla(ctx, action, *release)

	case domain.ActionTypeRadarr:
		rejections, err = s.radarr(ctx, action, *release)

	case domain.ActionTypeSonarr:
		rejections, err = s.sonarr(ctx, action, *release)

	case domain.ActionTypeLidarr:
		rejections, err = s.lidarr(ctx, action, *release)

	case domain.ActionTypeWhisparr:
		rejections, err = s.whisparr(ctx, action, *release)

	case domain.ActionTypeReadarr:
		rejections, err = s.readarr(ctx, action, *release)

	case domain.ActionTypeSabnzbd:
		rejections, err = s.sabnzbd(ctx, action, *release)

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
		FilterID:   int64(release.Filter.ID),
		Rejections: []string{},
		Timestamp:  time.Now(),
	}

	payload := &domain.NotificationPayload{
		Event:       domain.NotificationEventPushApproved,
		ReleaseName: release.TorrentName,
		Filter:      release.Filter.Name,
		Indexer:     release.Indexer,
		InfoHash:    release.TorrentHash,

		Size:           release.Size,
		Status:         domain.ReleasePushStatusApproved,
		Action:         action.Name,
		ActionType:     action.Type,
		ActionClient:   action.Client.Name,
		Rejections:     []string{},
		Protocol:       domain.ReleaseProtocolTorrent,
		Implementation: release.Implementation,
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

func (s *service) watchFolder(ctx context.Context, action *domain.Action, release domain.Release) error {
	if release.HasMagnetUri() {
		return fmt.Errorf("action watch folder does not support magnet links: %s", release.TorrentName)
	}

	s.log.Trace().Msgf("action WATCH_FOLDER: %v file: %v", action.WatchFolder, release.TorrentTmpFile)

	// Open original file
	original, err := os.Open(release.TorrentTmpFile)
	if err != nil {
		return errors.Wrap(err, "could not open temp file: %v", release.TorrentTmpFile)
	}
	defer original.Close()

	// default dir to watch folder
	//  /mnt/watch/{{.Indexer}}
	//  /mnt/watch/mock
	//  /mnt/watch/{{.Indexer}}-{{.TorrentName}}.torrent
	//  /mnt/watch/mock-Torrent.Name-GROUP.torrent
	dir := action.WatchFolder
	newFileName := action.WatchFolder

	// if watchFolderArgs does not contain .torrent, create
	if !strings.HasSuffix(action.WatchFolder, ".torrent") {
		_, tmpFileName := filepath.Split(release.TorrentTmpFile)

		newFileName = filepath.Join(action.WatchFolder, tmpFileName+".torrent")
	} else {
		dir, _ = filepath.Split(action.WatchFolder)
	}

	// Create folder
	if err = os.MkdirAll(dir, os.ModePerm); err != nil {
		return errors.Wrap(err, "could not create new folders %v", dir)
	}

	// Create new file
	newFile, err := os.Create(newFileName)
	if err != nil {
		return errors.Wrap(err, "could not create new file %v", newFileName)
	}
	defer newFile.Close()

	// Copy file
	if _, err := io.Copy(newFile, original); err != nil {
		return errors.Wrap(err, "could not copy file %v to watch folder", newFileName)
	}

	s.log.Info().Msgf("saved file to watch folder: %v", newFileName)

	return nil
}

func (s *service) webhook(ctx context.Context, action *domain.Action, release domain.Release) error {
	s.log.Trace().Msgf("action WEBHOOK: '%v' file: %v", action.Name, release.TorrentName)
	if len(action.WebhookData) > 1024 {
		s.log.Trace().Msgf("webhook action '%v' - host: %v data: %v", action.Name, action.WebhookHost, action.WebhookData[:1024])
	} else {
		s.log.Trace().Msgf("webhook action '%v' - host: %v data: %v", action.Name, action.WebhookHost, action.WebhookData)
	}

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := http.Client{Transport: t, Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, action.WebhookHost, bytes.NewBufferString(action.WebhookData))
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

	if len(action.WebhookData) > 256 {
		s.log.Info().Msgf("successfully ran webhook action: '%v' to: %v payload: %v", action.Name, action.WebhookHost, action.WebhookData[:256])
	} else {
		s.log.Info().Msgf("successfully ran webhook action: '%v' to: %v payload: %v", action.Name, action.WebhookHost, action.WebhookData)
	}

	return nil
}
