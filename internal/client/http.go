package client

import (
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/anacrolix/torrent/metainfo"

	"github.com/rs/zerolog/log"
)

type DownloadFileResponse struct {
	Body     *io.ReadCloser
	FileName string
}

type DownloadTorrentFileResponse struct {
	MetaInfo    *metainfo.MetaInfo
	TmpFileName string
}

type HttpClient struct {
	http *http.Client
}

func NewHttpClient() *HttpClient {
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	return &HttpClient{
		http: httpClient,
	}
}

func (c *HttpClient) DownloadFile(url string, opts map[string]string) (*DownloadFileResponse, error) {
	if url == "" {
		return nil, errors.New("download_file: url can't be empty")
	}

	// Create tmp file
	tmpFile, err := os.CreateTemp("", "autobrr-")
	if err != nil {
		log.Error().Stack().Err(err).Msg("error creating temp file")
		return nil, err
	}
	defer tmpFile.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error downloading file from %v", url)
		return nil, err
	}
	defer resp.Body.Close()

	// retry logic

	if resp.StatusCode != http.StatusOK {
		log.Error().Stack().Err(err).Msgf("error downloading file from: %v - bad status: %d", url, resp.StatusCode)
		return nil, err
	}

	// Write the body to file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error writing downloaded file: %v", tmpFile.Name())
		return nil, err
	}

	// remove file if fail

	res := DownloadFileResponse{
		Body:     &resp.Body,
		FileName: tmpFile.Name(),
	}

	if res.FileName == "" || res.Body == nil {
		log.Error().Stack().Err(err).Msgf("tmp file error - empty body: %v", url)
		return nil, errors.New("error downloading file, no tmp file")
	}

	log.Debug().Msgf("successfully downloaded file: %v", tmpFile.Name())

	return &res, nil
}

func (c *HttpClient) DownloadTorrentFile(url string, opts map[string]string) (*DownloadTorrentFileResponse, error) {
	if url == "" {
		return nil, errors.New("download_file: url can't be empty")
	}

	// Create tmp file
	tmpFile, err := os.CreateTemp("", "autobrr-")
	if err != nil {
		log.Error().Stack().Err(err).Msg("error creating temp file")
		return nil, err
	}
	defer tmpFile.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error downloading file from %v", url)
		return nil, err
	}
	defer resp.Body.Close()

	// retry logic

	if resp.StatusCode != http.StatusOK {
		log.Error().Stack().Err(err).Msgf("error downloading file from: %v - bad status: %d", url, resp.StatusCode)
		return nil, err
	}

	// Write the body to file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error writing downloaded file: %v", tmpFile.Name())
		return nil, err
	}

	meta, err := metainfo.Load(resp.Body)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("metainfo could not load file contents: %v", tmpFile.Name())
		return nil, err
	}

	// remove file if fail

	res := DownloadTorrentFileResponse{
		MetaInfo:    meta,
		TmpFileName: tmpFile.Name(),
	}

	if res.TmpFileName == "" || res.MetaInfo == nil {
		log.Error().Stack().Err(err).Msgf("tmp file error - empty body: %v", url)
		return nil, errors.New("error downloading file, no tmp file")
	}

	log.Debug().Msgf("successfully downloaded file: %v", tmpFile.Name())

	return &res, nil
}
