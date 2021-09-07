package release

import (
	"errors"
	"fmt"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/client"
	"github.com/autobrr/autobrr/internal/domain"
)

type Service interface {
	Process(announce domain.Announce) error
}

type service struct {
	actionSvc action.Service
}

func NewService(actionService action.Service) Service {
	return &service{actionSvc: actionService}
}

func (s *service) Process(announce domain.Announce) error {
	log.Trace().Msgf("start to process release: %+v", announce)

	if announce.Filter.Actions == nil {
		return fmt.Errorf("no actions for filter: %v", announce.Filter.Name)
	}

	// check can download
	// smart episode?
	// check against rules like active downloading torrents

	var err error
	var hash string
	var res *client.DownloadFileResponse

	// download torrent only if action is not any of radarr, sonarr or lidarr
	if !actionIsArr(announce.Filter.Actions) {
	// create http client
	c := client.NewHttpClient()

	// download torrent file
	// TODO check extra headers, cookie
		res, err = c.DownloadFile(announce.TorrentUrl, nil)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not download file: %v", announce.TorrentName)
		return err
	}

	if res.FileName == "" {
		return errors.New("error downloading file, no tmp file")
	}

	if res.Body == nil {
		log.Error().Stack().Err(err).Msgf("tmp file error - empty body: %v", announce.TorrentName)
		return errors.New("empty body")
	}

	// onTorrentDownloaded
		log.Trace().Msgf("downloaded torrent file: %v", res.FileName)

	// match more filters like torrent size

	// Get meta info from file to find out the hash for later use
	meta, err := metainfo.LoadFromFile(res.FileName)
	if err != nil {
		log.Error().Err(err).Msgf("metainfo could not open file: %v", res.FileName)
		return err
	}

	// torrent info hash used for re-announce
		hash = meta.HashInfoBytes().String()
	}

	// take action (watchFolder, test, runProgram, qBittorrent, Deluge etc)
	err = s.actionSvc.RunActions(res.FileName, hash, *announce.Filter, announce)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error running actions for filter: %v", announce.Filter.Name)
		return err
	}

	// safe to delete tmp file

	return nil
}

func actionIsArr(actions []domain.Action) bool {
	arrs := []domain.ActionType{domain.ActionTypeRadarr, domain.ActionTypeSonarr, domain.ActionTypeLidarr}
	for _, a := range actions {
		for _, arr := range arrs {
			if a.Type == arr {
				return true
			}
		}
	}

	return false
}

func actionIsTorrentClient(actions []domain.Action) bool {
	bt := []domain.ActionType{domain.ActionTypeQbittorrent, domain.ActionTypeDelugeV1, domain.ActionTypeDelugeV2}
	for _, a := range actions {
		for _, c := range bt {
			if a.Type == c {
				return true
			}
		}
	}

	return false
}
