package client

import (
	"io"
	"net/http"
	"os"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/avast/retry-go"
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

	res := &DownloadTorrentFileResponse{}
	// try request and if fail run 3 retries
	err = retry.Do(func() error {
		resp, err := http.Get(url)
		if err != nil {
			return errors.New("error downloading file: %q", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errors.New("error downloading file bad status: %d", resp.StatusCode)
		}

		nuke := func() {
			tmpFile.Seek(0, io.SeekStart)
			tmpFile.Truncate(0)
		}

		// Write the body to file
		_, err = io.Copy(tmpFile, resp.Body)
		if err != nil {
			nuke()
			return errors.New("error writing downloaded file: %v | %q", tmpFile.Name(), err)
		}

		meta, err := metainfo.Load(resp.Body)
		if err != nil {
			nuke()
			return errors.New("metainfo could not load file contents: %v | %q", tmpFile.Name(), err)
		}

		res = &DownloadTorrentFileResponse{
			MetaInfo:    meta,
			TmpFileName: tmpFile.Name(),
		}

		if res.TmpFileName == "" || res.MetaInfo == nil {
			nuke()
			return errors.New("tmp file error - empty body")
		}

		if len(res.MetaInfo.InfoBytes) < 1 {
			nuke()
			return errors.New("could not read infohash")
		}

		log.Debug().Msgf("successfully downloaded file: %v", tmpFile.Name())
		return nil
	},
		// retry.OnRetry(func(n uint, err error) { c.log.Printf("%q: attempt %d - %v\n", err, n, url) }),
		retry.Delay(time.Second*5),
		retry.Attempts(3),
		retry.MaxJitter(time.Second*1))

	if err != nil {
		res = nil
	}

	return res, err
}
