package release

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/anacrolix/torrent/metainfo"
	"io"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/domain"
)

type Service interface {
	Find(ctx context.Context, query domain.QueryParams) (res []domain.Release, nextCursor int64, count int64, err error)
	Stats(ctx context.Context) (*domain.ReleaseStats, error)
	Store(ctx context.Context, release *domain.Release) error
	StoreReleaseActionStatus(ctx context.Context, actionStatus *domain.ReleaseActionStatus) error
	Process(release domain.Release) error
}

type service struct {
	repo      domain.ReleaseRepo
	actionSvc action.Service
}

func NewService(repo domain.ReleaseRepo, actionService action.Service) Service {
	return &service{
		repo:      repo,
		actionSvc: actionService,
	}
}

func (s *service) Find(ctx context.Context, query domain.QueryParams) (res []domain.Release, nextCursor int64, count int64, err error) {
	res, nextCursor, count, err = s.repo.Find(ctx, query)
	if err != nil {
		return
	}

	return
}

func (s *service) Stats(ctx context.Context) (*domain.ReleaseStats, error) {
	stats, err := s.repo.Stats(ctx)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *service) Store(ctx context.Context, release *domain.Release) error {
	_, err := s.repo.Store(ctx, release)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) StoreReleaseActionStatus(ctx context.Context, actionStatus *domain.ReleaseActionStatus) error {
	err := s.repo.StoreReleaseActionStatus(ctx, actionStatus)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) Process(release domain.Release) error {
	log.Trace().Msgf("start to process release: %+v", release)

	if release.Filter.Actions == nil {
		return fmt.Errorf("no actions for filter: %v", release.Filter.Name)
	}

	// smart episode?

	// run actions (watchFolder, test, exec, qBittorrent, Deluge etc.)
	err := s.actionSvc.RunActions(release.Filter.Actions, release)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error running actions for filter: %v", release.Filter.Name)
		return err
	}

	return nil
}

func (s *service) DownloadTorrentFile(r *domain.Release) (*domain.DownloadTorrentFileResponse, error) {
	if r.TorrentURL == "" {
		return nil, errors.New("download_file: url can't be empty")
	} else if r.TorrentTmpFile != "" {
		// already downloaded
		return nil, nil
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: customTransport}

	// Get the data
	resp, err := client.Get(r.TorrentURL)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error downloading file")
		return nil, err
	}
	defer resp.Body.Close()

	// retry logic

	if resp.StatusCode != http.StatusOK {
		log.Error().Stack().Err(err).Msgf("error downloading file from: %v - bad status: %d", r.TorrentURL, resp.StatusCode)
		return nil, fmt.Errorf("error downloading torrent (%v) file (%v) from '%v' - status code: %d", r.TorrentName, r.TorrentURL, r.Indexer, resp.StatusCode)
	}

	// Create tmp file
	tmpFile, err := os.CreateTemp("", "autobrr-")
	if err != nil {
		log.Error().Stack().Err(err).Msg("error creating temp file")
		return nil, err
	}
	defer tmpFile.Close()

	r.TorrentTmpFile = tmpFile.Name()

	// Write the body to file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error writing downloaded file: %v", tmpFile.Name())
		return nil, err
	}

	meta, err := metainfo.LoadFromFile(tmpFile.Name())
	if err != nil {
		log.Error().Stack().Err(err).Msgf("metainfo could not load file contents: %v", tmpFile.Name())
		return nil, err
	}

	// remove file if fail

	res := domain.DownloadTorrentFileResponse{
		MetaInfo:    meta,
		TmpFileName: tmpFile.Name(),
	}

	if res.TmpFileName == "" || res.MetaInfo == nil {
		log.Error().Stack().Err(err).Msgf("tmp file error - empty body: %v", r.TorrentURL)
		return nil, errors.New("error downloading file, no tmp file")
	}

	log.Debug().Msgf("successfully downloaded file: %v", tmpFile.Name())

	return &res, nil
}
