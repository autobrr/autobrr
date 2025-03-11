// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package sonarr

import (
	"fmt"
	"time"

	"github.com/autobrr/autobrr/pkg/arr"
)

type Release struct {
	Title            string `json:"title"`
	InfoUrl          string `json:"infoUrl,omitempty"`
	DownloadUrl      string `json:"downloadUrl,omitempty"`
	MagnetUrl        string `json:"magnetUrl,omitempty"`
	Indexer          string `json:"indexer"`
	DownloadProtocol string `json:"downloadProtocol"`
	Protocol         string `json:"protocol"`
	PublishDate      string `json:"publishDate"`
	DownloadClient   string `json:"downloadClient,omitempty"`
	Size             uint64 `json:"size"`
	DownloadClientId int    `json:"downloadClientId,omitempty"`
}

type PushResponse struct {
	Rejections   []string `json:"rejections"`
	Approved     bool     `json:"approved"`
	Rejected     bool     `json:"rejected"`
	TempRejected bool     `json:"temporarilyRejected"`
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

type AlternateTitle struct {
	Title        string `json:"title"`
	SeasonNumber int    `json:"seasonNumber"`
}

type Season struct {
	Statistics   *Statistics `json:"statistics,omitempty"`
	SeasonNumber int         `json:"seasonNumber"`
	Monitored    bool        `json:"monitored"`
}

type Statistics struct {
	PreviousAiring    time.Time `json:"previousAiring"`
	SeasonCount       int       `json:"seasonCount"`
	EpisodeFileCount  int       `json:"episodeFileCount"`
	EpisodeCount      int       `json:"episodeCount"`
	TotalEpisodeCount int       `json:"totalEpisodeCount"`
	SizeOnDisk        int64     `json:"sizeOnDisk"`
	PercentOfEpisodes float64   `json:"percentOfEpisodes"`
}

type Series struct {
	PreviousAiring    time.Time         `json:"previousAiring,omitempty"`
	FirstAired        time.Time         `json:"firstAired,omitempty"`
	Added             time.Time         `json:"added,omitempty"`
	NextAiring        time.Time         `json:"nextAiring,omitempty"`
	Ratings           *arr.Ratings      `json:"ratings,omitempty"`
	Statistics        *Statistics       `json:"statistics,omitempty"`
	Title             string            `json:"title,omitempty"`
	SortTitle         string            `json:"sortTitle,omitempty"`
	Status            string            `json:"status,omitempty"`
	Overview          string            `json:"overview,omitempty"`
	Network           string            `json:"network,omitempty"`
	Path              string            `json:"path,omitempty"`
	SeriesType        string            `json:"seriesType,omitempty"`
	CleanTitle        string            `json:"cleanTitle,omitempty"`
	ImdbID            string            `json:"imdbId,omitempty"`
	TitleSlug         string            `json:"titleSlug,omitempty"`
	RootFolderPath    string            `json:"rootFolderPath,omitempty"`
	Certification     string            `json:"certification,omitempty"`
	AirTime           string            `json:"airTime,omitempty"`
	AlternateTitles   []*AlternateTitle `json:"alternateTitles,omitempty"`
	Images            []*arr.Image      `json:"images,omitempty"`
	Seasons           []*Season         `json:"seasons,omitempty"`
	Genres            []string          `json:"genres,omitempty"`
	Tags              []int             `json:"tags,omitempty"`
	ID                int64             `json:"id"`
	Year              int               `json:"year,omitempty"`
	QualityProfileID  int64             `json:"qualityProfileId,omitempty"`
	LanguageProfileID int64             `json:"languageProfileId,omitempty"`
	Runtime           int               `json:"runtime,omitempty"`
	TvdbID            int64             `json:"tvdbId,omitempty"`
	TvRageID          int64             `json:"tvRageId,omitempty"`
	TvMazeID          int64             `json:"tvMazeId,omitempty"`
	Ended             bool              `json:"ended,omitempty"`
	SeasonFolder      bool              `json:"seasonFolder,omitempty"`
	Monitored         bool              `json:"monitored"`
	UseSceneNumbering bool              `json:"useSceneNumbering,omitempty"`
}
