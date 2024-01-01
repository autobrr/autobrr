// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package ptp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"golang.org/x/time/rate"
)

const DefaultURL = "https://passthepopcorn.me/torrents.php"

var ErrUnauthorized = errors.New("unauthorized: bad credentials")
var ErrForbidden = errors.New("forbidden")
var ErrTooManyRequests = errors.New("too many requests: rate-limit reached")

type ApiClient interface {
	GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error)
	TestAPI(ctx context.Context) (bool, error)
}

type Client struct {
	url         string
	client      *http.Client
	rateLimiter *rate.Limiter
	APIUser     string
	APIKey      string
}

type OptFunc func(*Client)

func WithUrl(url string) OptFunc {
	return func(c *Client) {
		c.url = url
	}
}

func NewClient(apiUser, apiKey string, opts ...OptFunc) ApiClient {
	c := &Client{
		url: DefaultURL,
		client: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
		rateLimiter: rate.NewLimiter(rate.Every(1*time.Second), 1), // 10 request every 10 seconds
		APIUser:     apiUser,
		APIKey:      apiKey,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type TorrentResponse struct {
	Page          string    `json:"Page"`
	Result        string    `json:"Result"`
	GroupId       string    `json:"GroupId"`
	Name          string    `json:"Name"`
	Year          string    `json:"Year"`
	CoverImage    string    `json:"CoverImage"`
	AuthKey       string    `json:"AuthKey"`
	PassKey       string    `json:"PassKey"`
	TorrentId     string    `json:"TorrentId"`
	ImdbId        string    `json:"ImdbId"`
	ImdbRating    string    `json:"ImdbRating"`
	ImdbVoteCount int       `json:"ImdbVoteCount"`
	Torrents      []Torrent `json:"Torrents"`
}
type Torrent struct {
	Id            string  `json:"Id"`
	InfoHash      string  `json:"InfoHash"`
	Quality       string  `json:"Quality"`
	Source        string  `json:"Source"`
	Container     string  `json:"Container"`
	Codec         string  `json:"Codec"`
	Resolution    string  `json:"Resolution"`
	Size          string  `json:"Size"`
	Scene         bool    `json:"Scene"`
	UploadTime    string  `json:"UploadTime"`
	Snatched      string  `json:"Snatched"`
	Seeders       string  `json:"Seeders"`
	Leechers      string  `json:"Leechers"`
	ReleaseName   string  `json:"ReleaseName"`
	ReleaseGroup  *string `json:"ReleaseGroup"`
	Checked       bool    `json:"Checked"`
	GoldenPopcorn bool    `json:"GoldenPopcorn"`
	RemasterTitle string  `json:"RemasterTitle,omitempty"`
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	ctx := context.Background()
	err := c.rateLimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return nil, errors.Wrap(err, "error waiting for ratelimiter")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return resp, errors.Wrap(err, "error making request")
	}
	return resp, nil
}

func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "ptp client request error : %v", url)
	}

	req.Header.Add("ApiUser", c.APIUser)
	req.Header.Add("ApiKey", c.APIKey)
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.Do(req)
	if err != nil {
		return res, errors.Wrap(err, "ptp client request error : %v", url)
	}

	if res.StatusCode == http.StatusUnauthorized {
		return res, ErrUnauthorized
	} else if res.StatusCode == http.StatusForbidden {
		return res, ErrForbidden
	} else if res.StatusCode == http.StatusTooManyRequests {
		return res, ErrTooManyRequests
	}

	return res, nil
}

func (c *Client) GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error) {
	if torrentID == "" {
		return nil, errors.New("ptp client: must have torrentID")
	}

	var r TorrentResponse

	v := url.Values{}
	v.Add("torrentid", torrentID)
	params := v.Encode()

	reqUrl := fmt.Sprintf("%v?%v", c.url, params)

	resp, err := c.get(ctx, reqUrl)
	if err != nil {
		return nil, errors.Wrap(err, "error requesting data")
	}

	defer func() {
		resp.Body.Close()
	}()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, errors.Wrap(readErr, "could not read body")
	}

	if err = json.Unmarshal(body, &r); err != nil {
		return nil, errors.Wrap(readErr, "could not unmarshal body")
	}

	for _, torrent := range r.Torrents {
		if torrent.Id == torrentID {
			return &domain.TorrentBasic{
				Id:       torrent.Id,
				InfoHash: torrent.InfoHash,
				Size:     torrent.Size,
			}, nil
		}
	}

	return nil, nil
}

// TestAPI try api access against torrents page
func (c *Client) TestAPI(ctx context.Context) (bool, error) {
	resp, err := c.get(ctx, c.url)
	if err != nil {
		return false, errors.Wrap(err, "error requesting data")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	return true, nil
}
