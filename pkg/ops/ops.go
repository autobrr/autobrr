package ops

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

	"golang.org/x/time/rate"
)

type ApiClient interface {
	GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error)
	TestAPI(ctx context.Context) (bool, error)
	UseURL(url string)
}

type Client struct {
	Url         string
	client      *http.Client
	RateLimiter *rate.Limiter
	APIKey      string
}

func NewClient(apiKey string) ApiClient {
	c := &Client{
		Url: "https://orpheus.network/ajax.php",
		client: &http.Client{
			Timeout: time.Second * 30,
		},
		RateLimiter: rate.NewLimiter(rate.Every(10*time.Second), 5),
		APIKey:      apiKey,
	}

	return c
}

func (c *Client) UseURL(url string) {
	c.Url = url
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
	FreeTorrent             string `json:"freeTorrent"`
	Time                    string `json:"time"`
	Description             string `json:"description"`
	FileList                string `json:"fileList"`
	FilePath                string `json:"filePath"`
	UserId                  int    `json:"userId"`
	Username                string `json:"username"`
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	//ctx := context.Background()
	err := c.RateLimiter.Wait(req.Context()) // This is a blocking call. Honors the rate limit
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	if c.APIKey == "" {
		return nil, errors.New("orpheus client missing API key!")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}

	req.Header.Add("Authorization", fmt.Sprintf("token %s", c.APIKey))
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not make request: %+v", req)
	}

	// return early if not OK
	if res.StatusCode != http.StatusOK {
		var r ErrorResponse

		body, readErr := io.ReadAll(res.Body)
		if readErr != nil {
			return nil, errors.Wrap(readErr, "could not read body")
		}

		if err = json.Unmarshal(body, &r); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal body")
		}

		res.Body.Close()

		return nil, errors.New("status code: %d status: %s error: %s", res.StatusCode, r.Status, r.Error)
	}

	return res, nil
}

func (c *Client) GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error) {
	if torrentID == "" {
		return nil, errors.New("orpheus client: must have torrentID")
	}

	var r TorrentDetailsResponse

	v := url.Values{}
	v.Add("id", torrentID)
	params := v.Encode()

	reqUrl := fmt.Sprintf("%s?action=torrent&%s", c.Url, params)

	resp, err := c.get(ctx, reqUrl)
	if err != nil {
		return nil, errors.Wrap(err, "could not get torrent by id: %v", torrentID)
	}

	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, errors.Wrap(readErr, "could not read body")
	}

	if err := json.Unmarshal(body, &r); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal body")
	}

	res := &domain.TorrentBasic{
		Id:          strconv.Itoa(r.Response.Torrent.Id),
		InfoHash:    r.Response.Torrent.InfoHash,
		Size:        strconv.Itoa(r.Response.Torrent.Size),
		Uploader:    r.Response.Torrent.Username,
		RecordLabel: r.Response.Group.RecordLabel,
	}

	return res, nil
}

// TestAPI try api access against torrents page
func (c *Client) TestAPI(ctx context.Context) (bool, error) {
	resp, err := c.get(ctx, c.Url+"?action=index")
	if err != nil {
		return false, errors.Wrap(err, "test api error")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	return true, nil
}
