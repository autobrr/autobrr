package ggn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

type Client interface {
	GetTorrentByID(torrentID string) (*domain.TorrentBasic, error)
	TestAPI() (bool, error)
}

type client struct {
	Url         string
	Timeout     int
	client      *http.Client
	Ratelimiter *rate.Limiter
	APIKey      string
	Headers     http.Header
}

func NewClient(url string, apiKey string) Client {
	// set default url
	if url == "" {
		url = "https://gazellegames.net/api.php"
	}

	c := &client{
		APIKey:      apiKey,
		client:      http.DefaultClient,
		Url:         url,
		Ratelimiter: rate.NewLimiter(rate.Every(5*time.Second), 1), // 5 request every 10 seconds
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
	GameInfo     struct {
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
	} `json:"gameInfo"`
	Tags     []string `json:"tags"`
	Platform string   `json:"platform"`
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

func (c *client) Do(req *http.Request) (*http.Response, error) {
	ctx := context.Background()
	err := c.Ratelimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *client) get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		log.Error().Err(err).Msgf("ggn client request error : %v", url)
		return nil, err
	}

	req.Header.Add("X-API-Key", c.APIKey)
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("ggn client request error : %v", url)
		return nil, err
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

func (c *client) GetTorrentByID(torrentID string) (*domain.TorrentBasic, error) {
	if torrentID == "" {
		return nil, fmt.Errorf("ggn client: must have torrentID")
	}

	var r Response

	v := url.Values{}
	v.Add("id", torrentID)
	params := v.Encode()

	resp, err := c.get(c.Url + "?request=torrent" + params)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}

	err = json.Unmarshal(body, &r)
	if err != nil {
		return nil, err
	}

	if r.Status != "success" {
		return nil, fmt.Errorf("bad status: %v", r.Status)
	}

	t := &domain.TorrentBasic{
		Id:       strconv.Itoa(r.Response.Torrent.Id),
		InfoHash: r.Response.Torrent.InfoHash,
		Size:     strconv.FormatUint(r.Response.Torrent.Size, 10),
	}

	return t, nil

}

// TestAPI try api access against torrents page
func (c *client) TestAPI() (bool, error) {
	resp, err := c.get(c.Url)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, nil
}
