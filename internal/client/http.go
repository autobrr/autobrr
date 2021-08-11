package client

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

type DownloadFileResponse struct {
	Body     *io.ReadCloser
	FileName string
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
		return nil, nil
	}

	// create md5 hash of url for tmp file
	hash := md5.Sum([]byte(url))
	hashString := hex.EncodeToString(hash[:])
	tmpFileName := fmt.Sprintf("/tmp/%v", hashString)

	log.Debug().Msgf("tmpFileName: %v", tmpFileName)

	// Create the file
	out, err := os.Create(tmpFileName)
	if err != nil {
		return nil, err
	}

	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		// TODO better error message
		return nil, err
	}
	defer resp.Body.Close()

	// retry logic

	if resp.StatusCode != 200 {
		return nil, err
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, err
	}

	// remove file if fail

	res := DownloadFileResponse{
		Body:     &resp.Body,
		FileName: tmpFileName,
	}

	return &res, nil
}
