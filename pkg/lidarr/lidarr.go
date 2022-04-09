package lidarr

import (
	"encoding/json"
	"io"
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

// New create new lidarr client
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
	res, err := c.get("system/status")
	if err != nil {
		log.Error().Stack().Err(err).Msg("lidarr client get error")
		return nil, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Stack().Err(err).Msg("lidarr client error reading body")
		return nil, err
	}

	response := SystemStatusResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Error().Stack().Err(err).Msg("lidarr client error json unmarshal")
		return nil, err
	}

	log.Trace().Msgf("lidarr system/status response: %+v", response)

	return &response, nil
}

func (c *client) Push(release Release) ([]string, error) {
	res, err := c.post("release/push", release)
	if err != nil {
		log.Error().Stack().Err(err).Msg("lidarr client post error")
		return nil, err
	}

	if res == nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Stack().Err(err).Msg("lidarr client error reading body")
		return nil, err
	}

	pushResponse := PushResponse{}
	err = json.Unmarshal(body, &pushResponse)
	if err != nil {
		log.Error().Stack().Err(err).Msg("lidarr client error json unmarshal")
		return nil, err
	}

	log.Trace().Msgf("lidarr release/push response body: %+v", string(body))

	// log and return if rejected
	if pushResponse.Rejected {
		rejections := strings.Join(pushResponse.Rejections, ", ")

		log.Trace().Msgf("lidarr push rejected: %s - reasons: %q", release.Title, rejections)
		return pushResponse.Rejections, nil
	}

	return nil, nil
}
