package steam

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Client struct {
	log        zerolog.Logger
	host       string
	name       string
	headers    []string
	httpClient *http.Client
}

func NewClient(log zerolog.Logger, name, host string, headers ...string) *Client {
	return &Client{
		log:     log.With().Str("list_provider", "steam").Str("list", name).Logger(),
		name:    name,
		host:    host,
		headers: headers,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: sharedhttp.Transport,
		},
	}
}

type Item struct {
	Name string `json:"name"`
}

type ListResponse map[string]Item

func (c *Client) GetList(ctx context.Context, listURL string) (*ListResponse, error) {
	if listURL == "" {
		return nil, errors.New("no URL provided for steam")
	}

	c.log.Debug().Str("url", listURL).Msg("fetching titles")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "could not make new request for URL: %s", listURL)
	}

	for _, header := range c.headers {
		parts := strings.Split(header, "=")
		if len(parts) != 2 {
			continue
		}
		req.Header.Set(parts[0], parts[1])
	}

	//setUserAgent(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch titles from URL: %s", listURL)
	}
	defer sharedhttp.DrainAndClose(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		break
	default:
		return nil, errors.Errorf("failed to fetch titles from URL: %s", listURL)
	}

	var data ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, errors.Wrapf(err, "failed to decode JSON data from URL: %s", listURL)
	}

	return &data, nil
}
