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

	var err error
	var tmpFile string
	var hash string

	for _, action := range actions {
		if !action.Enabled {
			// only run active actions
			continue
		}

		log.Debug().Msgf("process action: %v for '%v'", action.Name, release.TorrentName)

		actionStatus := domain.ReleaseActionStatus{
			ReleaseID:  release.ID,
			Status:     domain.ReleasePushStatusPending,
			Action:     action.Name,
			Type:       action.Type,
			Rejections: []string{},
			Timestamp:  time.Now(),
		}

		s.bus.Publish("release:store-action-status", &actionStatus)

		switch action.Type {
		case domain.ActionTypeTest:
			s.test(action.Name)

		case domain.ActionTypeExec:
			if release.TorrentTmpFile == "" {
				t, err := release.DownloadTorrentFile(nil)
				if err != nil {
					log.Error().Stack().Err(err)
					return err
				}

				tmpFile = t.TmpFileName
			}

			go func(release domain.Release, action domain.Action, tmpFile string) {
				s.execCmd(release, action, tmpFile)
			}(release, action, tmpFile)

		case domain.ActionTypeWatchFolder:
			if release.TorrentTmpFile == "" {
				t, err := release.DownloadTorrentFile(nil)
				if err != nil {
					log.Error().Stack().Err(err)
					return err
				}

				tmpFile = t.TmpFileName
			}
			s.watchFolder(action.WatchFolder, tmpFile)

		case domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2:
			canDownload, err := s.delugeCheckRulesCanDownload(action)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
				continue
			}
			if !canDownload {
				s.bus.Publish("release:store-action-status", &domain.ReleaseActionStatus{
					ID:         actionStatus.ID,
					ReleaseID:  release.ID,
					Status:     domain.ReleasePushStatusRejected,
					Action:     action.Name,
					Type:       action.Type,
					Rejections: []string{"deluge busy"},
				})
				continue
			}
			if release.TorrentTmpFile == "" {
				t, err := release.DownloadTorrentFile(nil)
				if err != nil {
					log.Error().Stack().Err(err)
					return err
				}

				tmpFile = t.TmpFileName
			}

			go func(action domain.Action, tmpFile string) {
				err = s.deluge(action, tmpFile)
				if err != nil {
					log.Error().Stack().Err(err).Msg("error sending torrent to Deluge")
				}
			}(action, tmpFile)

		case domain.ActionTypeQbittorrent:
			canDownload, err := s.qbittorrentCheckRulesCanDownload(action)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
				continue
			}
			if !canDownload {
				s.bus.Publish("release:store-action-status", &domain.ReleaseActionStatus{
					ID:         actionStatus.ID,
					ReleaseID:  release.ID,
					Status:     domain.ReleasePushStatusRejected,
					Action:     action.Name,
					Type:       action.Type,
					Rejections: []string{"qbittorrent busy"},
				})
				continue
			}

			if release.TorrentTmpFile == "" {
				t, err := release.DownloadTorrentFile(nil)
				if err != nil {
					log.Error().Stack().Err(err)
					return err
				}

				tmpFile = t.TmpFileName
				hash = t.MetaInfo.HashInfoBytes().String()
			}

			go func(action domain.Action, hash string, tmpFile string) {
				err = s.qbittorrent(action, hash, tmpFile)
				if err != nil {
					log.Error().Stack().Err(err).Msg("error sending torrent to qBittorrent")
				}
			}(action, hash, tmpFile)

		case domain.ActionTypeRadarr:
			go func(release domain.Release, action domain.Action) {
				err = s.radarr(release, action)
				if err != nil {
					log.Error().Stack().Err(err).Msg("error sending torrent to radarr")
					//continue
				}
			}(release, action)

		case domain.ActionTypeSonarr:
			go func(release domain.Release, action domain.Action) {
				err = s.sonarr(release, action)
				if err != nil {
					log.Error().Stack().Err(err).Msg("error sending torrent to sonarr")
					//continue
				}
			}(release, action)

		case domain.ActionTypeLidarr:
			go func(release domain.Release, action domain.Action) {
				err = s.lidarr(release, action)
				if err != nil {
					log.Error().Stack().Err(err).Msg("error sending torrent to lidarr")
					//continue
				}
			}(release, action)

		default:
			log.Warn().Msgf("unsupported action: %v type: %v", action.Name, action.Type)
		}

		s.bus.Publish("release:store-action-status", &domain.ReleaseActionStatus{
			ID:         actionStatus.ID,
			ReleaseID:  release.ID,
			Status:     domain.ReleasePushStatusApproved,
			Action:     action.Name,
			Type:       action.Type,
			Rejections: []string{},
		})
	}

	// safe to delete tmp file

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
			canDownload, err := s.qbittorrentCheckRulesCanDownload(action)
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
