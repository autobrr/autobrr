package trakt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
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

const (
	// maxPages bounds how many pages GetList follows, in case a misbehaving server keeps advertising more.
	maxPages = 100

	// pageLimit is the largest page size Trakt serves; it clamps larger values and reports the applied one in the pagination headers.
	pageLimit = 250

	maxRateLimitRetries = 3
	defaultRetryAfter   = 10 * time.Second
	maxRetryAfter       = 60 * time.Second
)

// apiHost is the host that gets an explicit page limit; other hosts, like autobrr hosted lists, are left untouched.
var apiHost = "api.trakt.tv"

type rateLimitError struct {
	retryAfter time.Duration
}

func (e *rateLimitError) Error() string {
	return fmt.Sprintf("rate limited, retry after: %s", e.retryAfter)
}

// GetList fetches every item of the list, following Trakt pagination when the response carries pagination headers.
func (c *Client) GetList(ctx context.Context, listURL string) ([]Item, error) {
	if listURL == "" {
		return nil, errors.Errorf("no URL provided for trakt")
	}

	reqURL := transformTraktURL(listURL)

	baseURL, err := url.Parse(reqURL)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse URL: %s", reqURL)
	}

	query := baseURL.Query()
	// A pasted API URL may carry a page param; drop it so the walk starts at page 1.
	query.Del("page")
	if strings.EqualFold(baseURL.Hostname(), apiHost) && !query.Has("limit") {
		query.Set("limit", strconv.Itoa(pageLimit))
	}
	baseURL.RawQuery = query.Encode()

	items, pageCount, err := c.getPage(ctx, baseURL.String())
	if err != nil {
		return nil, err
	}

	if pageCount <= 1 {
		return items, nil
	}

	// Fail instead of truncating, a partial list would overwrite the filter and drop titles.
	if pageCount > maxPages {
		return nil, errors.Errorf("list has %d pages, exceeding the maximum of %d", pageCount, maxPages)
	}

	for page := 2; page <= pageCount; page++ {
		pageURL := *baseURL
		query := pageURL.Query()
		query.Set("page", strconv.Itoa(page))
		pageURL.RawQuery = query.Encode()

		pageItems, _, err := c.getPage(ctx, pageURL.String())
		if err != nil {
			return nil, errors.Wrapf(err, "could not fetch page %d of %d", page, pageCount)
		}

		items = append(items, pageItems...)
	}

	return items, nil
}

func (c *Client) getPage(ctx context.Context, reqURL string) ([]Item, int, error) {
	for attempt := 1; ; attempt++ {
		items, pageCount, err := c.fetchPage(ctx, reqURL)

		var rateLimitErr *rateLimitError
		if !errors.As(err, &rateLimitErr) || attempt > maxRateLimitRetries {
			return items, pageCount, err
		}

		c.log.Debug().Str("url", reqURL).Dur("retry_after", rateLimitErr.retryAfter).Int("attempt", attempt).Msg("rate limited, waiting before retry")

		timer := time.NewTimer(rateLimitErr.retryAfter)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, 0, ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) fetchPage(ctx context.Context, reqURL string) ([]Item, int, error) {
	c.log.Debug().Str("url", reqURL).Msg("fetching titles")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "could not make new request for URL: %s", reqURL)
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
		return nil, 0, errors.Wrapf(err, "failed to fetch titles from URL: %s", reqURL)
	}
	defer sharedhttp.DrainAndClose(resp)

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusTooManyRequests:
		return nil, 0, &rateLimitError{retryAfter: parseRetryAfter(resp.Header.Get("Retry-After"))}
	default:
		return nil, 0, errors.Errorf("failed to fetch titles from URL: %s status: %d", reqURL, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return nil, 0, errors.Errorf("invalid content type for URL: %s, expected application/json got: %s", reqURL, contentType)
	}

	var data []Item
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, 0, errors.Wrapf(err, "failed to decode JSON data from URL: %s", reqURL)
	}

	// Trakt serves at most one page per request since June 2026, even for /all, see #2812.
	// Endpoints without pagination (like autobrr hosted lists) omit the header.
	header := resp.Header.Get("X-Pagination-Page-Count")
	if header == "" {
		return data, 0, nil
	}

	pageCount, err := strconv.Atoi(header)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "invalid X-Pagination-Page-Count header for URL: %s", reqURL)
	}

	return data, pageCount, nil
}

// parseRetryAfter reads the delay in seconds from a Retry-After header and caps it so a bogus value cannot stall the refresh job.
func parseRetryAfter(value string) time.Duration {
	seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || seconds < 0 {
		return defaultRetryAfter
	}

	if seconds >= int64(maxRetryAfter/time.Second) {
		return maxRetryAfter
	}

	return time.Duration(seconds) * time.Second
}
