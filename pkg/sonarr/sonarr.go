package sonarr

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type Config struct {
	Hostname string
	APIKey   string

	// basic auth username and password
	BasicAuth bool
	Username  string
	Password  string
}

type Client interface {
	Test() (*SystemStatusResponse, error)
	Push(release Release) ([]string, error)
}

type client struct {
	config Config
	http   *http.Client
}

// New create new sonarr client
func New(config Config) Client {

	httpClient := &http.Client{
		Timeout: time.Second * 30,
	}

	c := &client{
		config: config,
		http:   httpClient,
	}

	return c
}

type Release struct {
	Title            string `json:"title"`
	DownloadUrl      string `json:"downloadUrl"`
	Size             int64  `json:"size"`
	Indexer          string `json:"indexer"`
	DownloadProtocol string `json:"downloadProtocol"`
	Protocol         string `json:"protocol"`
	PublishDate      string `json:"publishDate"`
}

type PushResponse struct {
	Approved     bool     `json:"approved"`
	Rejected     bool     `json:"rejected"`
	TempRejected bool     `json:"temporarilyRejected"`
	Rejections   []string `json:"rejections"`
}

type SystemStatusResponse struct {
	Version string `json:"version"`
}

func (c *client) Test() (*SystemStatusResponse, error) {
	status, res, err := c.get("system/status")
	if err != nil {
		log.Error().Stack().Err(err).Msg("sonarr client get error")
		return nil, err
	}

	if status == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	log.Trace().Msgf("sonarr system/status response: %v", string(res))

	response := SystemStatusResponse{}
	err = json.Unmarshal(res, &response)
	if err != nil {
		log.Error().Stack().Err(err).Msg("sonarr client error json unmarshal")
		return nil, err
	}

	return &response, nil
}

func (c *client) Push(release Release) ([]string, error) {
	status, res, err := c.postBody("release/push", release)
	if err != nil {
		log.Error().Stack().Err(err).Msg("sonarr client post error")
		return nil, err
	}

	log.Trace().Msgf("sonarr release/push response status: (%v) body: %v", status, string(res))

	pushResponse := make([]PushResponse, 0)
	err = json.Unmarshal(res, &pushResponse)
	if err != nil {
		log.Error().Stack().Err(err).Msg("sonarr client error json unmarshal")
		return nil, err
	}

	// log and return if rejected
	if pushResponse[0].Rejected {
		rejections := strings.Join(pushResponse[0].Rejections, ", ")

		log.Trace().Msgf("sonarr push rejected: %s - reasons: %q", release.Title, rejections)
		return pushResponse[0].Rejections, nil
	}

	// successful push
	return nil, nil
}
