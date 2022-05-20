package lidarr

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
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
	status, res, err := c.get("system/status")
	if err != nil {
		return nil, fmt.Errorf("lidarr client get error: %w", err)
	}

	if status == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	//log.Trace().Msgf("lidarr system/status response status: %v body: %v", status, string(res))

	response := SystemStatusResponse{}
	err = json.Unmarshal(res, &response)
	if err != nil {
		return nil, fmt.Errorf("lidarr client error json unmarshal: %w", err)
	}

	return &response, nil
}

func (c *client) Push(release Release) ([]string, error) {
	_, res, err := c.postBody("release/push", release)
	if err != nil {
		return nil, fmt.Errorf("lidarr client post error: %w", err)
	}

	//log.Trace().Msgf("lidarr release/push response status: %v body: %v", status, string(res))

	pushResponse := PushResponse{}
	err = json.Unmarshal(res, &pushResponse)
	if err != nil {
		return nil, fmt.Errorf("lidarr client error json unmarshal: %w", err)
	}

	// log and return if rejected
	if pushResponse.Rejected {
		rejections := strings.Join(pushResponse.Rejections, ", ")

		return pushResponse.Rejections, fmt.Errorf("lidarr push rejected: %s - reasons: %q: err %w", release.Title, rejections, err)
	}

	return nil, nil
}
