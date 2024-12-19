// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package sonarr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"log"

	"github.com/autobrr/autobrr/pkg/arr"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"
)

type Config struct {
	Hostname string
	APIKey   string

	// basic auth username and password
	BasicAuth bool
	Username  string
	Password  string

	Log *log.Logger
}

type ClientInterface interface {
	Test(ctx context.Context) (*SystemStatusResponse, error)
	Push(ctx context.Context, release Release) ([]string, error)
}

type Client struct {
	config Config
	http   *http.Client

	Log *log.Logger
}

// New create new sonarr Client
func New(config Config) *Client {
	httpClient := &http.Client{
		Timeout:   time.Second * 120,
		Transport: sharedhttp.Transport,
	}

	c := &Client{
		config: config,
		http:   httpClient,
		Log:    log.New(io.Discard, "", log.LstdFlags),
	}

	if config.Log != nil {
		c.Log = config.Log
	}

	return c
}

type Release struct {
	Title            string `json:"title"`
	InfoUrl          string `json:"infoUrl,omitempty"`
	DownloadUrl      string `json:"downloadUrl,omitempty"`
	MagnetUrl        string `json:"magnetUrl,omitempty"`
	Size             uint64 `json:"size"`
	Indexer          string `json:"indexer"`
	DownloadProtocol string `json:"downloadProtocol"`
	Protocol         string `json:"protocol"`
	PublishDate      string `json:"publishDate"`
	DownloadClientId int    `json:"downloadClientId,omitempty"`
	DownloadClient   string `json:"downloadClient,omitempty"`
}

type PushResponse struct {
	Approved     bool     `json:"approved"`
	Rejected     bool     `json:"rejected"`
	TempRejected bool     `json:"temporarilyRejected"`
	Rejections   []string `json:"rejections"`
}

type BadRequestResponse struct {
	PropertyName   string `json:"propertyName"`
	ErrorMessage   string `json:"errorMessage"`
	ErrorCode      string `json:"errorCode"`
	AttemptedValue string `json:"attemptedValue"`
	Severity       string `json:"severity"`
}

func (r *BadRequestResponse) String() string {
	return fmt.Sprintf("[%s: %s] %s: %s - got value: %s", r.Severity, r.ErrorCode, r.PropertyName, r.ErrorMessage, r.AttemptedValue)
}

type SystemStatusResponse struct {
	Version string `json:"version"`
}

func (c *Client) Test(ctx context.Context) (*SystemStatusResponse, error) {
	status, res, err := c.get(ctx, "system/status")
	if err != nil {
		return nil, errors.Wrap(err, "could not make Test")
	}

	if status == http.StatusUnauthorized {
		return nil, errors.New("unauthorized: bad credentials")
	}

	c.Log.Printf("sonarr system/status status: (%v) response: %v\n", status, string(res))

	response := SystemStatusResponse{}
	if err = json.Unmarshal(res, &response); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	return &response, nil
}

func (c *Client) Push(ctx context.Context, release Release) ([]string, error) {
	status, res, err := c.postBody(ctx, "release/push", release)
	if err != nil {
		return nil, errors.Wrap(err, "could not push release to sonarr")
	}

	c.Log.Printf("sonarr release/push status: (%v) response: %v\n", status, string(res))

	if status == http.StatusBadRequest {
		badRequestResponses := make([]*BadRequestResponse, 0)

		if err = json.Unmarshal(res, &badRequestResponses); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal data")
		}

		rejections := []string{}
		for _, response := range badRequestResponses {
			rejections = append(rejections, response.String())
		}

		return rejections, nil
	}

	pushResponse := make([]PushResponse, 0)
	if err = json.Unmarshal(res, &pushResponse); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	// log and return if rejected
	if pushResponse[0].Rejected {
		rejections := strings.Join(pushResponse[0].Rejections, ", ")

		c.Log.Printf("sonarr release/push rejected %v reasons: %q\n", release.Title, rejections)
		return pushResponse[0].Rejections, nil
	}

	// successful push
	return nil, nil
}

func (c *Client) GetAllSeries(ctx context.Context) ([]Series, error) {
	return c.GetSeries(ctx, 0)
}

func (c *Client) GetSeries(ctx context.Context, tvdbId int64) ([]Series, error) {
	status, res, err := c.get(ctx, "series")
	if err != nil {
		return nil, errors.Wrap(err, "could not get series")
	}

	c.Log.Printf("sonarr series status: (%v) response: %v\n", status, string(res))

	if status == http.StatusBadRequest {
		badRequestResponses := make([]*BadRequestResponse, 0)

		if err = json.Unmarshal(res, &badRequestResponses); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal data")
		}

		return nil, nil
	}

	data := make([]Series, 0)
	if err = json.Unmarshal(res, &data); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal data")
	}

	return data, nil
}

//type Tag struct {
//	ID    int
//	Label string
//}

func (c *Client) GetTags(ctx context.Context) ([]*arr.Tag, error) {
	data := make([]*arr.Tag, 0)
	err := c.getJSON(ctx, "tag", nil, data)
	if err != nil {
		return nil, errors.Wrap(err, "could not get tags")
	}

	//c.Log.Printf("sonarr tag status: (%v) response: %v\n", status, string(res))

	//if status == http.StatusBadRequest {
	//	badRequestResponses := make([]*BadRequestResponse, 0)
	//
	//	if err = json.Unmarshal(res, &badRequestResponses); err != nil {
	//		return nil, errors.Wrap(err, "could not unmarshal data")
	//	}
	//
	//	return nil, nil
	//}
	//
	//if err = json.Unmarshal(res, &data); err != nil {
	//	return nil, errors.Wrap(err, "could not unmarshal data")
	//}

	return data, nil
}

type AlternateTitle struct {
	Title        string `json:"title"`
	SeasonNumber int    `json:"seasonNumber"`
}

type Season struct {
	SeasonNumber int         `json:"seasonNumber"`
	Monitored    bool        `json:"monitored"`
	Statistics   *Statistics `json:"statistics,omitempty"`
}

type Statistics struct {
	SeasonCount       int       `json:"seasonCount"`
	PreviousAiring    time.Time `json:"previousAiring"`
	EpisodeFileCount  int       `json:"episodeFileCount"`
	EpisodeCount      int       `json:"episodeCount"`
	TotalEpisodeCount int       `json:"totalEpisodeCount"`
	SizeOnDisk        int64     `json:"sizeOnDisk"`
	PercentOfEpisodes float64   `json:"percentOfEpisodes"`
}

type Series struct {
	ID                int64             `json:"id"`
	Title             string            `json:"title,omitempty"`
	AlternateTitles   []*AlternateTitle `json:"alternateTitles,omitempty"`
	SortTitle         string            `json:"sortTitle,omitempty"`
	Status            string            `json:"status,omitempty"`
	Overview          string            `json:"overview,omitempty"`
	PreviousAiring    time.Time         `json:"previousAiring,omitempty"`
	Network           string            `json:"network,omitempty"`
	Images            []*arr.Image      `json:"images,omitempty"`
	Seasons           []*Season         `json:"seasons,omitempty"`
	Year              int               `json:"year,omitempty"`
	Path              string            `json:"path,omitempty"`
	QualityProfileID  int64             `json:"qualityProfileId,omitempty"`
	LanguageProfileID int64             `json:"languageProfileId,omitempty"`
	Runtime           int               `json:"runtime,omitempty"`
	TvdbID            int64             `json:"tvdbId,omitempty"`
	TvRageID          int64             `json:"tvRageId,omitempty"`
	TvMazeID          int64             `json:"tvMazeId,omitempty"`
	FirstAired        time.Time         `json:"firstAired,omitempty"`
	SeriesType        string            `json:"seriesType,omitempty"`
	CleanTitle        string            `json:"cleanTitle,omitempty"`
	ImdbID            string            `json:"imdbId,omitempty"`
	TitleSlug         string            `json:"titleSlug,omitempty"`
	RootFolderPath    string            `json:"rootFolderPath,omitempty"`
	Certification     string            `json:"certification,omitempty"`
	Genres            []string          `json:"genres,omitempty"`
	Tags              []int             `json:"tags,omitempty"`
	Added             time.Time         `json:"added,omitempty"`
	Ratings           *arr.Ratings      `json:"ratings,omitempty"`
	Statistics        *Statistics       `json:"statistics,omitempty"`
	NextAiring        time.Time         `json:"nextAiring,omitempty"`
	AirTime           string            `json:"airTime,omitempty"`
	Ended             bool              `json:"ended,omitempty"`
	SeasonFolder      bool              `json:"seasonFolder,omitempty"`
	Monitored         bool              `json:"monitored"`
	UseSceneNumbering bool              `json:"useSceneNumbering,omitempty"`
}
