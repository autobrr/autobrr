package red

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"golang.org/x/time/rate"
)

type REDClient interface {
	GetTorrentByID(torrentID string) (*domain.TorrentBasic, error)
	TestAPI() (bool, error)
}

type Client struct {
	URL         string
	Timeout     int
	client      *http.Client
	RateLimiter *rate.Limiter
	APIKey      string
}

func NewClient(url string, apiKey string) REDClient {
	if url == "" {
		url = "https://redacted.ch/ajax.php"
	}

	c := &Client{
		APIKey:      apiKey,
		client:      http.DefaultClient,
		URL:         url,
		RateLimiter: rate.NewLimiter(rate.Every(10*time.Second), 10),
	}

	return c
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

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	ctx := context.Background()
	err := c.RateLimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}

	req.Header.Add("Authorization", c.APIKey)
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not make request: %+v", req)
	}

	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	} else if res.StatusCode == http.StatusForbidden {
		return nil, nil
	} else if res.StatusCode == http.StatusBadRequest {
		return nil, errors.New("bad id parameter")
	} else if res.StatusCode == http.StatusTooManyRequests {
		return nil, errors.New("rate-limited")
	}

	return res, nil
}

func (c *Client) GetTorrentByID(torrentID string) (*domain.TorrentBasic, error) {
	if torrentID == "" {
		return nil, errors.New("red client: must have torrentID")
	}

	var r TorrentDetailsResponse

	v := url.Values{}
	v.Add("id", torrentID)
	params := v.Encode()

	reqUrl := fmt.Sprintf("%v?action=torrent&%v", c.URL, params)

	resp, err := c.get(reqUrl)
	if err != nil {
		return nil, errors.Wrap(err, "could not get torrent by id: %v", torrentID)
	}

	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, errors.Wrap(readErr, "could not read body")
	}

	err = json.Unmarshal(body, &r)
	if err != nil {
		return nil, errors.Wrap(readErr, "could not unmarshal body")
	}

	return &domain.TorrentBasic{
		Id:       strconv.Itoa(r.Response.Torrent.Id),
		InfoHash: r.Response.Torrent.InfoHash,
		Size:     strconv.Itoa(r.Response.Torrent.Size),
	}, nil

}

// TestAPI try api access against torrents page
func (c *Client) TestAPI() (bool, error) {
	resp, err := c.get(c.URL + "?action=index")
	if err != nil {
		return false, errors.Wrap(err, "could not run test api")
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, nil
}
