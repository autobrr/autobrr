// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package red

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"golang.org/x/time/rate"
)

const DefaultURL = "https://redacted.ch/ajax.php"

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
		rateLimiter: rate.NewLimiter(rate.Every(10*time.Second), 10),
		APIKey:      apiKey,
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
		Group   Group   `json:"group"`
		Torrent Torrent `json:"torrent"`
	} `json:"response"`
	Error string `json:"error,omitempty"`
}

type Group struct {
	//WikiBody        string `json:"wikiBody"`
	//WikiImage       string `json:"wikiImage"`
	Id              int    `json:"id"`
	Name            string `json:"name"`
	Year            int    `json:"year"`
	RecordLabel     string `json:"recordLabel"`
	CatalogueNumber string `json:"catalogueNumber"`
	ReleaseType     int    `json:"releaseType"`
	CategoryId      int    `json:"categoryId"`
	CategoryName    string `json:"categoryName"`
	Time            string `json:"time"`
	VanityHouse     bool   `json:"vanityHouse"`
	//MusicInfo       struct {
	//	Composers []interface{} `json:"composers"`
	//	Dj        []interface{} `json:"dj"`
	//	Artists   []struct {
	//		Id   int    `json:"id"`
	//		Name string `json:"name"`
	//	} `json:"artists"`
	//	With []struct {
	//		Id   int    `json:"id"`
	//		Name string `json:"name"`
	//	} `json:"with"`
	//	Conductor []interface{} `json:"conductor"`
	//	RemixedBy []interface{} `json:"remixedBy"`
	//	Producer  []interface{} `json:"producer"`
	//} `json:"musicInfo"`
}

type Torrent struct {
	Id                      int    `json:"id"`
	InfoHash                string `json:"infoHash"`
	Media                   string `json:"media"`
	Format                  string `json:"format"`
	Encoding                string `json:"encoding"`
	Remastered              bool   `json:"remastered"`
	RemasterYear            int    `json:"remasterYear"`
	RemasterTitle           string `json:"remasterTitle"`
	RemasterRecordLabel     string `json:"remasterRecordLabel"`
	RemasterCatalogueNumber string `json:"remasterCatalogueNumber"`
	Scene                   bool   `json:"scene"`
	HasLog                  bool   `json:"hasLog"`
	HasCue                  bool   `json:"hasCue"`
	LogScore                int    `json:"logScore"`
	FileCount               int    `json:"fileCount"`
	Size                    int    `json:"size"`
	Seeders                 int    `json:"seeders"`
	Leechers                int    `json:"leechers"`
	Snatched                int    `json:"snatched"`
	FreeTorrent             bool   `json:"freeTorrent"`
	IsNeutralleech          bool   `json:"isNeutralleech"`
	IsFreeload              bool   `json:"isFreeload"`
	Time                    string `json:"time"`
	Description             string `json:"description"`
	FileList                string `json:"fileList"`
	FilePath                string `json:"filePath"`
	UserId                  int    `json:"userId"`
	Username                string `json:"username"`
}

type IndexResponse struct {
	Status   string `json:"status"`
	Response struct {
		Username      string `json:"username"`
		Id            int    `json:"id"`
		Authkey       string `json:"authkey"`
		Passkey       string `json:"passkey"`
		ApiVersion    string `json:"api_version"`
		Notifications struct {
			Messages        int  `json:"messages"`
			Notifications   int  `json:"notifications"`
			NewAnnouncement bool `json:"newAnnouncement"`
			NewBlog         bool `json:"newBlog"`
		} `json:"notifications"`
		UserStats struct {
			Uploaded      int64   `json:"uploaded"`
			Downloaded    int64   `json:"downloaded"`
			Ratio         float64 `json:"ratio"`
			RequiredRatio float64 `json:"requiredratio"`
			Class         string  `json:"class"`
		} `json:"userstats"`
	} `json:"response"`
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	//ctx := context.Background()
	err := c.rateLimiter.Wait(req.Context()) // This is a blocking call. Honors the rate limit
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (c *Client) getJSON(ctx context.Context, params url.Values, data any) error {
	if c.APIKey == "" {
		return errors.New("RED client missing API key!")
	}

	reqUrl := fmt.Sprintf("%s?%s", c.url, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, http.NoBody)
	if err != nil {
		return errors.Wrap(err, "could not build request")
	}

	req.Header.Add("Authorization", c.APIKey)
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not make request: %+v", req)
	}

	defer res.Body.Close()

	body := bufio.NewReader(res.Body)

	// return early if not OK
	if res.StatusCode != http.StatusOK {
		var errResponse ErrorResponse

		if err := json.NewDecoder(body).Decode(&errResponse); err != nil {
			return errors.Wrap(err, "could not unmarshal body")
		}

		return errors.New("status code: %d status: %s error: %s", res.StatusCode, errResponse.Status, errResponse.Error)
	}

	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return errors.Wrap(err, "could not unmarshal body")
	}

	return nil
}

func (c *Client) GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error) {
	if torrentID == "" {
		return nil, errors.New("red client: must have torrentID")
	}

	var response TorrentDetailsResponse

	params := url.Values{}
	params.Add("action", "torrent")
	params.Add("id", torrentID)

	err := c.getJSON(ctx, params, &response)
	if err != nil {
		return nil, errors.Wrap(err, "could not get torrent by id: %v", torrentID)
	}

	return &domain.TorrentBasic{
		Id:       strconv.Itoa(response.Response.Torrent.Id),
		InfoHash: response.Response.Torrent.InfoHash,
		Size:     strconv.Itoa(response.Response.Torrent.Size),
	}, nil

}

// TestAPI try api access against torrents page
func (c *Client) TestAPI(ctx context.Context) (bool, error) {
	resp, err := c.GetIndex(ctx)
	if err != nil {
		return false, errors.Wrap(err, "test api error")
	}

	if resp == nil {
		return false, nil
	}

	return true, nil
}

// GetIndex get API index
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
