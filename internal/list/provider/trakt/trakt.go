package trakt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
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
	apikey     string
	headers    []string
	httpClient *http.Client
}

func NewClient(log zerolog.Logger, name, host, apikey string, headers ...string) *Client {
	return &Client{
		log:     log.With().Str("list_provider", "trakt").Str("list", name).Logger(),
		name:    name,
		host:    host,
		apikey:  apikey,
		headers: headers,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: sharedhttp.Transport,
		},
	}
}

type Item struct {
	Title string `json:"title"`
	Movie struct {
		Title string `json:"title"`
	} `json:"movie"`
	Show struct {
		Title string `json:"title"`
	} `json:"show"`
}

var (
	traktSmartListWebRegex  = regexp.MustCompile(`(?i)^(?:https?://)?(?:app\.|www\.)?trakt\.tv/lists/smart/view/([^/?#]+)`)
	traktUserListWebRegex   = regexp.MustCompile(`(?i)^(?:https?://)?(?:app\.|www\.)?trakt\.tv/users/([^/]+)/lists/([^/?#]+)`)
	traktUserWatchlistRegex = regexp.MustCompile(`(?i)^(?:https?://)?(?:app\.|www\.)?trakt\.tv/users/([^/]+)/watchlist`)
)

func transformTraktURL(rawURL string) string {
	if matches := traktSmartListWebRegex.FindStringSubmatch(rawURL); len(matches) == 2 {
		return fmt.Sprintf("https://api.trakt.tv/smart-lists/%s/items", matches[1])
	}

	if matches := traktUserListWebRegex.FindStringSubmatch(rawURL); len(matches) == 3 {
		return fmt.Sprintf("https://api.trakt.tv/users/%s/lists/%s/items", matches[1], matches[2])
	}

	if matches := traktUserWatchlistRegex.FindStringSubmatch(rawURL); len(matches) == 2 {
		return fmt.Sprintf("https://api.trakt.tv/users/%s/watchlist/items", matches[1])
	}

	return rawURL
}

func (c *Client) GetList(ctx context.Context, listURL string) ([]Item, error) {
	if listURL == "" {
		return nil, errors.Errorf("no URL provided for trakt")
	}

	reqURL := transformTraktURL(listURL)

	c.log.Debug().Str("url", reqURL).Msg("fetching titles")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "could not make new request for URL: %s", reqURL)
	}

	req.Header.Set("trakt-api-version", "2")
	if c.apikey != "" {
		req.Header.Set("trakt-api-key", c.apikey)
	}

	for _, header := range c.headers {
		parts := strings.Split(header, "=")
		if len(parts) != 2 {
			continue
		}
		req.Header.Set(parts[0], parts[1])
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch titles from URL: %s", reqURL)
	}
	defer sharedhttp.DrainAndClose(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		break
	//case http.StatusNotFound:
	//	return nil, errors.Wrapf(ErrNotFound, "list or endpoint not found: %s", listURL)
	default:
		return nil, errors.Errorf("failed to fetch titles from URL: %s", listURL)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return nil, errors.Errorf("invalid content type for URL: %s, expected application/json got: %s", reqURL, contentType)
	}

	var data []Item
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, errors.Wrapf(err, "failed to decode JSON data from URL: %s", reqURL)
	}

	return data, nil
}
