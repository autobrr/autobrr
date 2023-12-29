// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package ggn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"golang.org/x/time/rate"
)

const DefaultURL = "https://gazellegames.net/api.php"

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
	APIKey      string
}

type OptFunc func(*Client)

func WithUrl(url string) OptFunc {
	return func(c *Client) {
		c.url = url
	}
}

func NewClient(apiKey string, opts ...OptFunc) ApiClient {
	c := &Client{
		url: DefaultURL,
		client: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
		rateLimiter: rate.NewLimiter(rate.Every(5*time.Second), 1), // 5 request every 10 seconds
		APIKey:      apiKey,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type Group struct {
	BbWikiBody   string   `json:"bbWikiBody"`
	WikiBody     string   `json:"wikiBody"`
	WikiImage    string   `json:"wikiImage"`
	Id           int      `json:"id"`
	Name         string   `json:"name"`
	Aliases      []string `json:"aliases"`
	Year         int      `json:"year"`
	CategoryId   int      `json:"categoryId"`
	CategoryName string   `json:"categoryName"`
	MasterGroup  int      `json:"masterGroup"`
	Time         string   `json:"time"`
	Tags         []string `json:"tags"`
	Platform     string   `json:"platform"`
}

type GameInfo struct {
	Screenshots []string `json:"screenshots"`
	Trailer     string   `json:"trailer"`
	Rating      string   `json:"rating"`
	MetaRating  struct {
		Score   string `json:"score"`
		Percent string `json:"percent"`
		Link    string `json:"link"`
	} `json:"metaRating"`
	IgnRating struct {
		Score   string `json:"score"`
		Percent string `json:"percent"`
		Link    string `json:"link"`
	} `json:"ignRating"`
	GamespotRating struct {
		Score   string `json:"score"`
		Percent string `json:"percent"`
		Link    string `json:"link"`
	} `json:"gamespotRating"`
	Weblinks struct {
		GamesWebsite  string `json:"GamesWebsite"`
		Wikipedia     string `json:"Wikipedia"`
		Giantbomb     string `json:"Giantbomb"`
		GameFAQs      string `json:"GameFAQs"`
		PCGamingWiki  string `json:"PCGamingWiki"`
		Steam         string `json:"Steam"`
		Amazon        string `json:"Amazon"`
		GOG           string `json:"GOG"`
		HowLongToBeat string `json:"HowLongToBeat"`
	} `json:"weblinks"`
	//`json:"gameInfo"`
}

type Torrent struct {
	Id             int    `json:"id"`
	InfoHash       string `json:"infoHash"`
	Type           string `json:"type"`
	Link           string `json:"link"`
	Format         string `json:"format"`
	Encoding       string `json:"encoding"`
	Region         string `json:"region"`
	Language       string `json:"language"`
	Remastered     bool   `json:"remastered"`
	RemasterYear   int    `json:"remasterYear"`
	RemasterTitle  string `json:"remasterTitle"`
	Scene          bool   `json:"scene"`
	HasCue         bool   `json:"hasCue"`
	ReleaseTitle   string `json:"releaseTitle"`
	ReleaseType    string `json:"releaseType"`
	GameDOXType    string `json:"gameDOXType"`
	GameDOXVersion string `json:"gameDOXVersion"`
	FileCount      int    `json:"fileCount"`
	Size           uint64 `json:"size"`
	Seeders        int    `json:"seeders"`
	Leechers       int    `json:"leechers"`
	Snatched       int    `json:"snatched"`
	FreeTorrent    bool   `json:"freeTorrent"`
	NeutralTorrent bool   `json:"neutralTorrent"`
	Reported       bool   `json:"reported"`
	Time           string `json:"time"`
	BbDescription  string `json:"bbDescription"`
	Description    string `json:"description"`
	FileList       []struct {
		Ext  string `json:"ext"`
		Size string `json:"size"`
		Name string `json:"name"`
	} `json:"fileList"`
	FilePath string `json:"filePath"`
	UserId   int    `json:"userId"`
	Username string `json:"username"`
}

type TorrentResponse struct {
	Group   Group   `json:"group"`
	Torrent Torrent `json:"torrent"`
}

type Response struct {
	Status   string          `json:"status"`
	Response TorrentResponse `json:"response,omitempty"`
	Error    string          `json:"error,omitempty"`
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
		return nil, errors.Wrap(err, "ggn client request error : %s", url)
	}

	req.Header.Add("X-API-Key", c.APIKey)
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.Do(req)
	if err != nil {
		return res, errors.Wrap(err, "ggn client request error : %s", url)
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
		return nil, errors.New("ggn client: must have torrentID")
	}

	var r Response

	v := url.Values{}
	v.Add("id", torrentID)
	params := v.Encode()

	reqUrl := fmt.Sprintf("%s?%s&%s", c.url, "request=torrent", params)

	resp, err := c.get(ctx, reqUrl)
	if err != nil {
		return nil, errors.Wrap(err, "error getting data")
	}

	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, errors.Wrap(readErr, "error reading body")
	}

	if err = json.Unmarshal(body, &r); err != nil {
		return nil, errors.Wrap(err, "error unmarshal body")
	}

	if r.Status != "success" {
		return nil, errors.New("bad status: %s", r.Status)
	}

	t := &domain.TorrentBasic{
		Id:       strconv.Itoa(r.Response.Torrent.Id),
		InfoHash: r.Response.Torrent.InfoHash,
		Size:     strconv.FormatUint(r.Response.Torrent.Size, 10),
	}

	return t, nil

}

// TestAPI try api access against torrents page
func (c *Client) TestAPI(ctx context.Context) (bool, error) {
	resp, err := c.get(ctx, c.url)
	if err != nil {
		return false, errors.Wrap(err, "error getting data")
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, nil
}
