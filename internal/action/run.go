package action

import (
	"io"
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

		go func(release domain.Release, action domain.Action) {
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
				return
			}
		}(release, action)
	}

	// safe to delete tmp file

	return nil
}

func (s *service) runAction(action domain.Action, release domain.Release) error {

	var err error
	var rejections []string

	switch action.Type {
	case domain.ActionTypeTest:
		s.test(action.Name)

	case domain.ActionTypeExec:
		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(nil); err != nil {
				log.Error().Stack().Err(err)
				return err
			}
		}

		s.execCmd(release, action, release.TorrentTmpFile)

	case domain.ActionTypeWatchFolder:
		if release.TorrentTmpFile == "" {
			if err := release.DownloadTorrentFile(nil); err != nil {
				log.Error().Stack().Err(err)
				return err
			}
		}

		s.watchFolder(action.WatchFolder, release.TorrentTmpFile)

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
			if err := release.DownloadTorrentFile(nil); err != nil {
				log.Error().Stack().Err(err)
				return err
			}
		}

		err = s.deluge(action, release.TorrentTmpFile)
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
			if err := release.DownloadTorrentFile(nil); err != nil {
				log.Error().Stack().Err(err)
				return err
			}
		}

		err = s.qbittorrent(client, action, release.TorrentTmpFile, release.TorrentHash)
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

	default:
		log.Warn().Msgf("unsupported action: %v type: %v", action.Name, action.Type)
		return nil
	}

	if rejections != nil {
		s.bus.Publish("release:push-rejected", &domain.ReleaseActionStatus{
			ReleaseID:  release.ID,
			Status:     domain.ReleasePushStatusRejected,
			Action:     action.Name,
			Type:       action.Type,
			Rejections: rejections,
			Timestamp:  time.Now(),
		})

		return nil
	}

	s.bus.Publish("release:push-approved", &domain.ReleaseActionStatus{
		ReleaseID:  release.ID,
		Status:     domain.ReleasePushStatusApproved,
		Action:     action.Name,
		Type:       action.Type,
		Rejections: []string{},
		Timestamp:  time.Now(),
	})

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

func (s *service) watchFolder(dir string, torrentFile string) {
	log.Trace().Msgf("action WATCH_FOLDER: %v file: %v", dir, torrentFile)

	// Open original file
	original, err := os.Open(torrentFile)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not open temp file '%v'", torrentFile)
		return
	}
	defer original.Close()

	_, tmpFileName := path.Split(torrentFile)
	fullFileName := path.Join(dir, tmpFileName+".torrent")

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
