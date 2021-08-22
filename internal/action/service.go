package action

import (
	"io"
	"os"
	"path"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/download_client"
	"github.com/rs/zerolog/log"
)

type Service interface {
	RunActions(torrentFile string, hash string, filter domain.Filter, announce domain.Announce) error
	Store(action domain.Action) (*domain.Action, error)
	Fetch() ([]domain.Action, error)
	Delete(actionID int) error
	ToggleEnabled(actionID int) error
}

type service struct {
	repo      domain.ActionRepo
	clientSvc download_client.Service
}

func NewService(repo domain.ActionRepo, clientSvc download_client.Service) Service {
	return &service{repo: repo, clientSvc: clientSvc}
}

func (s *service) RunActions(torrentFile string, hash string, filter domain.Filter, announce domain.Announce) error {
	for _, action := range filter.Actions {
		if !action.Enabled {
			// only run active actions
			continue
		}

		log.Trace().Msgf("process action: %v", action.Name)

		switch action.Type {
		case domain.ActionTypeTest:
			go s.test(torrentFile)

		case domain.ActionTypeWatchFolder:
			go s.watchFolder(action.WatchFolder, torrentFile)

		case domain.ActionTypeQbittorrent:
			go func() {
				err := s.qbittorrent(action, hash, torrentFile)
				if err != nil {
					log.Error().Err(err).Msg("error sending torrent to client")
				}
			}()

		case domain.ActionTypeExec:
			go s.execCmd(announce, action, torrentFile)

		case domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2:
			go func() {
				err := s.deluge(action, torrentFile)
				if err != nil {
					log.Error().Err(err).Msg("error sending torrent to client")
				}
			}()

		case domain.ActionTypeRadarr:
			go func() {
				err := s.radarr(announce, action)
				if err != nil {
					log.Error().Err(err).Msg("error sending torrent to radarr")
				}
			}()

		case domain.ActionTypeSonarr:
			go func() {
				err := s.sonarr(announce, action)
				if err != nil {
					log.Error().Err(err).Msg("error sending torrent to sonarr")
				}
			}()
		case domain.ActionTypeLidarr:
			go func() {
				err := s.lidarr(announce, action)
				if err != nil {
					log.Error().Err(err).Msg("error sending torrent to lidarr")
				}
			}()

		default:
			log.Warn().Msgf("unsupported action: %v type: %v", action.Name, action.Type)
		}
	}

	return nil
}

func (s *service) Store(action domain.Action) (*domain.Action, error) {
	// validate data

	a, err := s.repo.Store(action)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (s *service) Delete(actionID int) error {
	if err := s.repo.Delete(actionID); err != nil {
		return err
	}

	return nil
}

func (s *service) Fetch() ([]domain.Action, error) {
	actions, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	return actions, nil
}

func (s *service) ToggleEnabled(actionID int) error {
	if err := s.repo.ToggleEnabled(actionID); err != nil {
		return err
	}

	return nil
}

func (s *service) test(torrentFile string) {
	log.Info().Msgf("action TEST: %v", torrentFile)
}

func (s *service) watchFolder(dir string, torrentFile string) {
	log.Trace().Msgf("action WATCH_FOLDER: %v file: %v", dir, torrentFile)

	// Open original file
	original, err := os.Open(torrentFile)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer original.Close()

	_, tmpFileName := path.Split(torrentFile)
	fullFileName := path.Join(dir, tmpFileName)

	// Create new file
	newFile, err := os.Create(fullFileName)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer newFile.Close()

	// Copy file
	_, err = io.Copy(newFile, original)
	if err != nil {
		log.Fatal().Err(err)
	}

	log.Info().Msgf("saved file to watch folder: %v", fullFileName)
}
