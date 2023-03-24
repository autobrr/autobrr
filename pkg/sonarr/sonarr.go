package sonarr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"log"

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
	Test(ctx context.Context) (*SystemStatusResponse, error)
	Push(ctx context.Context, release Release) ([]string, error)
}

type client struct {
	config Config
	http   *http.Client

	Log *log.Logger
}

// New create new sonarr client
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
		// if no provided logger then use io.Discard
		c.Log = log.New(io.Discard, "", log.LstdFlags)
	}

	return c
}

type Release struct {
	Title            string `json:"title"`
	DownloadUrl      string `json:"downloadUrl,omitempty"`
	MagnetUrl        string `json:"magnetUrl,omitempty"`
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
	ErrorCode      string `json:"errorCode"`
	AttemptedValue string `json:"attemptedValue"`
	Severity       string `json:"severity"`
}

func (r *BadRequestResponse) String() string {
	return fmt.Sprintf("[%v: %v] %v: %v - got value: %v", r.Severity, r.ErrorCode, r.PropertyName, r.ErrorMessage, r.AttemptedValue)
}

type SystemStatusResponse struct {
	Version string `json:"version"`
}

func (c *client) Test(ctx context.Context) (*SystemStatusResponse, error) {
	status, res, err := c.get(ctx, "system/status")
	if err != nil {
		return nil, errors.Wrap(err, "could not make Test")
	}

	if status == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	c.Log.Printf("sonarr system/status status: (%v) response: %v\n", status, string(res))

	response := SystemStatusResponse{}
	if err = json.Unmarshal(res, &response); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	return &response, nil
}

func (c *client) Push(ctx context.Context, release Release) ([]string, error) {
	status, res, err := c.postBody(ctx, "release/push", release)
	if err != nil {
		return nil, errors.Wrap(err, "could not push release to sonarr")
	}

	c.Log.Printf("sonarr release/push status: (%v) response: %v\n", status, string(res))

	if status == http.StatusBadRequest {
		badRequestResponses := make([]*BadRequestResponse, 0)

		if err = json.Unmarshal(res, &badRequestResponses); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal data")
		}

		rejections := []string{}
		for _, response := range badRequestResponses {
			rejections = append(rejections, response.String())
		}

		return rejections, nil
	}

	pushResponse := make([]PushResponse, 0)
	if err = json.Unmarshal(res, &pushResponse); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	// log and return if rejected
	if pushResponse[0].Rejected {
		rejections := strings.Join(pushResponse[0].Rejections, ", ")

		c.Log.Printf("sonarr release/push rejected %v reasons: %q\n", release.Title, rejections)
		return pushResponse[0].Rejections, nil
	}

	// successful push
	return nil, nil
}
