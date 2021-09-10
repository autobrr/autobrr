package action

import (
	"io"
	"os"
	"path"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/client"
	"github.com/autobrr/autobrr/internal/domain"
)

func (s *service) RunActions(actions []domain.Action, announce domain.Announce) error {

	var err error
	var tmpFile string
	var hash string

	for _, action := range actions {
		if !action.Enabled {
			// only run active actions
			continue
		}

		log.Debug().Msgf("process action: %v", action.Name)

		switch action.Type {
		case domain.ActionTypeTest:
			s.test(action.Name)

		case domain.ActionTypeExec:
			if tmpFile == "" {
				tmpFile, hash, err = downloadFile(announce.TorrentUrl)
				if err != nil {
					log.Error().Stack().Err(err)
					return err
				}
			}
			go func(announce domain.Announce, action domain.Action, tmpFile string) {
				s.execCmd(announce, action, tmpFile)
			}(announce, action, tmpFile)

		case domain.ActionTypeWatchFolder:
			if tmpFile == "" {
				tmpFile, hash, err = downloadFile(announce.TorrentUrl)
				if err != nil {
					log.Error().Stack().Err(err)
					return err
				}
			}
			s.watchFolder(action.WatchFolder, tmpFile)

		case domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2:
			canDownload, err := s.delugeCheckRulesCanDownload(action)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
				continue
			}
			if canDownload {
				if tmpFile == "" {
					tmpFile, hash, err = downloadFile(announce.TorrentUrl)
					if err != nil {
						log.Error().Stack().Err(err)
						return err
					}
				}

				go func(action domain.Action, tmpFile string) {
					err = s.deluge(action, tmpFile)
					if err != nil {
						log.Error().Stack().Err(err).Msg("error sending torrent to Deluge")
					}
				}(action, tmpFile)
			}

		case domain.ActionTypeQbittorrent:
			canDownload, err := s.qbittorrentCheckRulesCanDownload(action)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("error checking client rules: %v", action.Name)
				continue
			}
			if canDownload {
				if tmpFile == "" {
					tmpFile, hash, err = downloadFile(announce.TorrentUrl)
					if err != nil {
						log.Error().Stack().Err(err)
						return err
					}
				}

				go func(action domain.Action, hash string, tmpFile string) {
					err = s.qbittorrent(action, hash, tmpFile)
					if err != nil {
						log.Error().Stack().Err(err).Msg("error sending torrent to qBittorrent")
					}
				}(action, hash, tmpFile)
			}

		case domain.ActionTypeRadarr:
			go func(announce domain.Announce, action domain.Action) {
				err = s.radarr(announce, action)
				if err != nil {
					log.Error().Stack().Err(err).Msg("error sending torrent to radarr")
					//continue
				}
			}(announce, action)

		case domain.ActionTypeSonarr:
			go func(announce domain.Announce, action domain.Action) {
				err = s.sonarr(announce, action)
				if err != nil {
					log.Error().Stack().Err(err).Msg("error sending torrent to sonarr")
					//continue
				}
			}(announce, action)

		case domain.ActionTypeLidarr:
			go func(announce domain.Announce, action domain.Action) {
				err = s.lidarr(announce, action)
				if err != nil {
					log.Error().Stack().Err(err).Msg("error sending torrent to lidarr")
					//continue
				}
			}(announce, action)

		default:
			log.Warn().Msgf("unsupported action: %v type: %v", action.Name, action.Type)
		}
	}

	// safe to delete tmp file

	return nil
}

// downloadFile returns tmpFile, hash, error
func downloadFile(url string) (string, string, error) {
	// create http client
	c := client.NewHttpClient()

	// download torrent file
	// TODO check extra headers, cookie
	res, err := c.DownloadFile(url, nil)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not download file: %v", url)
		return "", "", err
	}

	// match more filters like torrent size

	// Get meta info from file to find out the hash for later use
	meta, err := metainfo.LoadFromFile(res.FileName)
	//meta, err := metainfo.Load(res.Body)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("metainfo could not open file: %v", res.FileName)
		return "", "", err
	}

	// torrent info hash used for re-announce
	hash := meta.HashInfoBytes().String()

	return res.FileName, hash, nil
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
	fullFileName := path.Join(dir, tmpFileName)

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
