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

	"golang.org/x/time/rate"
)

type PTPClient interface {
	GetTorrentByID(torrentID string) (*domain.TorrentBasic, error)
	TestAPI() (bool, error)
}

type Client struct {
	Url         string
	Timeout     int
	client      *http.Client
	Ratelimiter *rate.Limiter
	APIUser     string
	APIKey      string
	Headers     http.Header
}

func NewClient(url string, apiUser string, apiKey string) PTPClient {
	// set default url
	if url == "" {
		url = "https://passthepopcorn.me/torrents.php"
	}

	c := &Client{
		APIUser:     apiUser,
		APIKey:      apiKey,
		client:      http.DefaultClient,
		Url:         url,
		Ratelimiter: rate.NewLimiter(rate.Every(1*time.Second), 1), // 10 request every 10 seconds
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
	err := c.Ratelimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return nil, errors.Wrap(err, "error waiting for ratelimiter")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "error making request")
	}
	return resp, nil
}

func (c *Client) get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "ptp client request error : %v", url)
	}

	req.Header.Add("ApiUser", c.APIUser)
	req.Header.Add("ApiKey", c.APIKey)
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "ptp client request error : %v", url)
	}

	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	} else if res.StatusCode == http.StatusForbidden {
		return nil, nil
	} else if res.StatusCode == http.StatusTooManyRequests {
		return nil, nil
	}

	return res, nil
}

func (c *Client) GetTorrentByID(torrentID string) (*domain.TorrentBasic, error) {
	if torrentID == "" {
		return nil, errors.New("ptp client: must have torrentID")
	}

	var r TorrentResponse

	v := url.Values{}
	v.Add("torrentid", torrentID)
	params := v.Encode()

	reqUrl := fmt.Sprintf("%v?%v", c.Url, params)

	resp, err := c.get(reqUrl)
	if err != nil {
		return nil, errors.Wrap(err, "error requesting data")
	}

	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, errors.Wrap(readErr, "could not read body")
	}

	err = json.Unmarshal(body, &r)
	if err != nil {
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
func (c *Client) TestAPI() (bool, error) {
	resp, err := c.get(c.Url)
	if err != nil {
		return false, errors.Wrap(err, "error requesting data")
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, nil
}
