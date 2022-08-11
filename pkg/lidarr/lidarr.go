package lidarr

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

type Config struct {
	Hostname string
	APIKey   string

	// basic auth username and password
	BasicAuth bool
	Username  string
	Password  string

	Log *log.Logger
}

type Client interface {
	Test() (*SystemStatusResponse, error)
	Push(release Release) ([]string, error)
}

type client struct {
	config Config
	http   *http.Client

	Log *log.Logger
}

// New create new lidarr client
func New(config Config) Client {

	httpClient := &http.Client{
		Timeout: time.Second * 30,
	}

	c := &client{
		config: config,
		http:   httpClient,
		Log:    config.Log,
	}

	if config.Log == nil {
		c.Log = log.New(io.Discard, "", log.LstdFlags)
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

type BadRequestResponse struct {
	PropertyName   string `json:"propertyName"`
	ErrorMessage   string `json:"errorMessage"`
	AttemptedValue string `json:"attemptedValue"`
	Severity       string `json:"severity"`
}

type SystemStatusResponse struct {
	Version string `json:"version"`
}

func (c *client) Test() (*SystemStatusResponse, error) {
	status, res, err := c.get("system/status")
	if err != nil {
		return nil, errors.Wrap(err, "lidarr client get error")
	}

	if status == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	c.Log.Printf("lidarr system/status response status: %v body: %v", status, string(res))

	response := SystemStatusResponse{}
	err = json.Unmarshal(res, &response)
	if err != nil {
		return nil, errors.Wrap(err, "lidarr client error json unmarshal")
	}

	return &response, nil
}

func (c *client) Push(release Release) ([]string, error) {
	status, res, err := c.postBody("release/push", release)
	if err != nil {
		return nil, errors.Wrap(err, "lidarr client post error")
	}

	c.Log.Printf("lidarr release/push response status: %v body: %v", status, string(res))

	if status == http.StatusBadRequest {
		badreqResponse := make([]*BadRequestResponse, 0)
		err = json.Unmarshal(res, &badreqResponse)
		if err != nil {
			return nil, errors.Wrap(err, "could not unmarshal data")
		}

		if badreqResponse[0] != nil && badreqResponse[0].PropertyName == "Title" && badreqResponse[0].ErrorMessage == "Unable to parse" {
			rejections := []string{fmt.Sprintf("unable to parse: %v", badreqResponse[0].AttemptedValue)}
			return rejections, err
		}
	}

	pushResponse := PushResponse{}
	err = json.Unmarshal(res, &pushResponse)
	if err != nil {
		return nil, errors.Wrap(err, "lidarr client error json unmarshal")
	}

	// log and return if rejected
	if pushResponse.Rejected {
		rejections := strings.Join(pushResponse.Rejections, ", ")

		c.Log.Printf("lidarr release/push rejected %v reasons: %q\n", release.Title, rejections)
		return pushResponse.Rejections, nil
	}

	return nil, nil
}
