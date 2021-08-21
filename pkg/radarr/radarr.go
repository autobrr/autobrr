package radarr

import (
	"net/http"
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

type Client struct {
	config Config
	http   *http.Client
}

func New(config Config) *Client {

	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}

	c := &Client{
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

func (c *Client) Test() error {
	_, err := c.get("system/status")
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Push(release Release) error {
	err := c.post("release/push", release)
	if err != nil {
		return err
	}

	return nil
}
