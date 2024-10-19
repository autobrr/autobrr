// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package ptp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
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
	FreeleechType *string `json:"FreeleechType,omitempty"`
	RemasterTitle *string `json:"RemasterTitle,omitempty"`
	RemasterYear  *string `json:"RemasterYear,omitempty"`
}

// custom unmarshal method for Torrent
func (t *Torrent) UnmarshalJSON(data []byte) error {
	type Alias Torrent

	aux := &struct {
		Id interface{} `json:"Id"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch id := aux.Id.(type) {
	case float64:
		t.Id = fmt.Sprintf("%.0f", id)
	case string:
		t.Id = id
	default:
		return fmt.Errorf("unexpected type for Id: %T", aux.Id)
	}

	return nil
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

func (c *Client) getJSON(ctx context.Context, params url.Values, data any) error {
	reqUrl := fmt.Sprintf("%s?%s", c.url, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, http.NoBody)
	if err != nil {
		return errors.Wrap(err, "ptp client request error : %v", reqUrl)
	}

	req.Header.Add("ApiUser", c.APIUser)
	req.Header.Add("ApiKey", c.APIKey)
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.Do(req)
	if err != nil {
		return errors.Wrap(err, "ptp client request error : %v", reqUrl)
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	} else if res.StatusCode == http.StatusForbidden {
		return ErrForbidden
	} else if res.StatusCode == http.StatusTooManyRequests {
		return ErrTooManyRequests
	}

	body := bufio.NewReader(res.Body)

	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return errors.Wrap(err, "could not unmarshal body")
	}

	return nil
}

func (c *Client) GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error) {
	if torrentID == "" {
		return nil, errors.New("ptp client: must have torrentID")
	}

	var response TorrentResponse

	params := url.Values{}
	params.Add("torrentid", torrentID)

	err := c.getJSON(ctx, params, &response)
	if err != nil {
		return nil, errors.Wrap(err, "error requesting data")
	}

	for _, torrent := range response.Torrents {
		if torrent.Id == torrentID {
			return &domain.TorrentBasic{
				Id:       torrent.Id,
				InfoHash: torrent.InfoHash,
				Size:     torrent.Size,
			}, nil
		}
	}

	return nil, errors.New("could not find torrent with id: %s", torrentID)
}

func (c *Client) GetTorrents(ctx context.Context) (*TorrentListResponse, error) {
	var response TorrentListResponse

	params := url.Values{}

	err := c.getJSON(ctx, params, &response)
	if err != nil {
		return nil, errors.Wrap(err, "error requesting data")
	}

	return &response, nil
}

// TestAPI try api access against torrents page
func (c *Client) TestAPI(ctx context.Context) (bool, error) {
	resp, err := c.GetTorrents(ctx)
	if err != nil {
		return false, errors.Wrap(err, "test api error")
	}

	if resp == nil {
		return false, nil
	}

	return true, nil
}

type TorrentListResponse struct {
	TotalResults string  `json:"TotalResults"`
	Movies       []Movie `json:"Movies"`
	Page         string  `json:"Page"`
}

type Movie struct {
	GroupID        string     `json:"GroupId"`
	Title          string     `json:"Title"`
	Year           string     `json:"Year"`
	Cover          string     `json:"Cover"`
	Tags           []string   `json:"Tags"`
	Directors      []Director `json:"Directors,omitempty"`
	ImdbID         *string    `json:"ImdbId,omitempty"`
	LastUploadTime string     `json:"LastUploadTime"`
	MaxSize        int64      `json:"MaxSize"`
	TotalSnatched  int64      `json:"TotalSnatched"`
	TotalSeeders   int64      `json:"TotalSeeders"`
	TotalLeechers  int64      `json:"TotalLeechers"`
	Torrents       []Torrent  `json:"Torrents"`
}

type Director struct {
	Name string `json:"Name"`
	ID   string `json:"Id"`
}
