// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package hebits

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

const DefaultURL = "https://hebits.net/ajax.php"

type ApiClient interface {
	GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error)
	TestAPI(ctx context.Context) (bool, error)
}

type Client struct {
	url         string
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	cookie      string

	log zerolog.Logger
}

type OptFunc func(*Client)

func WithUrl(url string) OptFunc {
	return func(c *Client) {
		c.url = url
	}
}

func WithHTTPClient(httpClient *http.Client) OptFunc {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func WithLog(log zerolog.Logger) OptFunc {
	return func(c *Client) {
		c.log = log
	}
}

func (c *Client) logger(ctx context.Context) *zerolog.Logger {
	if l := zerolog.Ctx(ctx); l.GetLevel() != zerolog.Disabled {
		return l
	}
	return &c.log
}

func NewClient(cookie string, opts ...OptFunc) *Client {
	c := &Client{
		url: DefaultURL,
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if isLoginPath(req.URL.Path) {
					return http.ErrUseLastResponse
				}
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		rateLimiter: rate.NewLimiter(rate.Every(10*time.Second), 10),
		cookie:      cookie,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type ErrorResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type TorrentDetailsResponse struct {
	Status   string `json:"status"`
	Response struct {
		Torrent Torrent `json:"torrent"`
	} `json:"response"`
	Error string `json:"error,omitempty"`
}

type Torrent struct {
	Id          int  `json:"id"`
	Size        int  `json:"size"`
	Seeders     int  `json:"seeders"`
	Leechers    int  `json:"leechers"`
	Snatched    int  `json:"snatched"`
	FreeTorrent bool `json:"freeTorrent"`
}

type IndexResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	start := time.Now()
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	if waited := time.Since(start); waited > time.Second {
		c.logger(ctx).Debug().Dur("waited", waited).Msg("rate limiter delayed request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (c *Client) getJSON(ctx context.Context, params url.Values, data any) error {
	if c.cookie == "" {
		return errors.New("hebits client missing cookie")
	}

	reqUrl := fmt.Sprintf("%s?%s", c.url, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, http.NoBody)
	if err != nil {
		return errors.Wrap(err, "could not build request")
	}

	req.Header.Set("Cookie", c.cookie)
	req.Header.Set("User-Agent", "autobrr")
	req.Header.Set("Accept", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not make request")
	}

	defer sharedhttp.DrainAndClose(res)

	c.logger(ctx).Trace().Str("url", reqUrl).Int("status", res.StatusCode).Msg("hebits api response")

	if isLoginRedirect(res) {
		return errors.New("authentication failed: redirected to login")
	}

	if res.StatusCode != http.StatusOK {
		return errors.New("status code: %d", res.StatusCode)
	}

	contentType := res.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "json") {
		return errors.New("unexpected content type: %s", contentType)
	}

	body := bufio.NewReader(res.Body)

	peek, _ := body.Peek(16)
	trimmed := bytes.TrimSpace(peek)
	if len(trimmed) > 0 && trimmed[0] != '{' && trimmed[0] != '[' {
		return errors.New("response is not json")
	}

	if err := json.NewDecoder(body).Decode(data); err != nil {
		return errors.Wrap(err, "could not unmarshal body")
	}

	return nil
}

func (c *Client) GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error) {
	if torrentID == "" {
		return nil, errors.New("hebits client: must have torrentID")
	}

	var response TorrentDetailsResponse

	params := url.Values{}
	params.Add("action", "torrent")
	params.Add("id", torrentID)

	err := c.getJSON(ctx, params, &response)
	if err != nil {
		return nil, errors.Wrap(err, "could not get torrent by id: %v", torrentID)
	}

	if !strings.EqualFold(response.Status, "success") {
		errMsg := response.Status
		if response.Error != "" {
			errMsg = fmt.Sprintf("%s error: %s", response.Status, response.Error)
		}
		return nil, errors.New("hebits api status: %s", errMsg)
	}

	t := response.Response.Torrent
	if t.Id != 0 && strconv.Itoa(t.Id) != torrentID {
		return nil, errors.New("hebits api torrent id mismatch: requested %s got %d", torrentID, t.Id)
	}

	tb := &domain.TorrentBasic{
		Id:       strconv.Itoa(t.Id),
		Size:     strconv.Itoa(t.Size),
		Seeders:  t.Seeders,
		Leechers: t.Leechers,
	}

	if t.FreeTorrent {
		tb.Freeleech = true
		tb.FreeleechPercent = 100
	}

	return tb, nil
}

func (c *Client) TestAPI(ctx context.Context) (bool, error) {
	start := time.Now()

	resp, err := c.GetIndex(ctx)
	if err != nil {
		return false, errors.Wrap(err, "test api error")
	}

	if resp == nil || !strings.EqualFold(resp.Status, "success") {
		return false, nil
	}

	c.logger(ctx).Debug().Dur("duration", time.Since(start)).Msg("hebits api test completed")

	return true, nil
}

func (c *Client) GetIndex(ctx context.Context) (*IndexResponse, error) {
	var response IndexResponse

	params := url.Values{}
	params.Add("action", "index")

	err := c.getJSON(ctx, params, &response)
	if err != nil {
		return nil, errors.Wrap(err, "test api error")
	}

	return &response, nil
}

func isLoginPath(path string) bool {
	return strings.Contains(strings.ToLower(path), "login.php")
}

func isLoginRedirect(res *http.Response) bool {
	if res.StatusCode < 300 || res.StatusCode >= 400 {
		return false
	}

	return isLoginPath(res.Header.Get("Location"))
}
