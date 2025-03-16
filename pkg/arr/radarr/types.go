// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package radarr

import (
	"fmt"
	"time"

	"github.com/autobrr/autobrr/pkg/arr"
)

type Movie struct {
	InCinemas             time.Time           `json:"inCinemas,omitempty"`
	PhysicalRelease       time.Time           `json:"physicalRelease,omitempty"`
	DigitalRelease        time.Time           `json:"digitalRelease,omitempty"`
	Added                 time.Time           `json:"added,omitempty"`
	Ratings               *arr.Ratings        `json:"ratings,omitempty"`
	MovieFile             *MovieFile          `json:"movieFile,omitempty"`
	Collection            *Collection         `json:"collection,omitempty"`
	Title                 string              `json:"title,omitempty"`
	Path                  string              `json:"path,omitempty"`
	MinimumAvailability   string              `json:"minimumAvailability,omitempty"`
	OriginalTitle         string              `json:"originalTitle,omitempty"`
	SortTitle             string              `json:"sortTitle,omitempty"`
	Status                string              `json:"status,omitempty"`
	Overview              string              `json:"overview,omitempty"`
	Website               string              `json:"website,omitempty"`
	YouTubeTrailerID      string              `json:"youTubeTrailerId,omitempty"`
	Studio                string              `json:"studio,omitempty"`
	FolderName            string              `json:"folderName,omitempty"`
	CleanTitle            string              `json:"cleanTitle,omitempty"`
	ImdbID                string              `json:"imdbId,omitempty"`
	TitleSlug             string              `json:"titleSlug,omitempty"`
	Certification         string              `json:"certification,omitempty"`
	AlternateTitles       []*AlternativeTitle `json:"alternateTitles,omitempty"`
	Images                []*arr.Image        `json:"images,omitempty"`
	Genres                []string            `json:"genres,omitempty"`
	Tags                  []int               `json:"tags,omitempty"`
	ID                    int64               `json:"id"`
	QualityProfileID      int64               `json:"qualityProfileId,omitempty"`
	TmdbID                int64               `json:"tmdbId,omitempty"`
	SecondaryYearSourceID int                 `json:"secondaryYearSourceId,omitempty"`
	SizeOnDisk            int64               `json:"sizeOnDisk,omitempty"`
	Year                  int                 `json:"year,omitempty"`
	Runtime               int                 `json:"runtime,omitempty"`
	HasFile               bool                `json:"hasFile,omitempty"`
	IsAvailable           bool                `json:"isAvailable,omitempty"`
	Monitored             bool                `json:"monitored"`
}

type AlternativeTitle struct {
	Language   *arr.Value `json:"language"`
	Title      string     `json:"title"`
	SourceType string     `json:"sourceType"`
	MovieID    int        `json:"movieId"`
	SourceID   int        `json:"sourceId"`
	Votes      int        `json:"votes"`
	VoteCount  int        `json:"voteCount"`
	ID         int        `json:"id"`
}

type MovieFile struct {
	DateAdded           time.Time    `json:"dateAdded"`
	Quality             *arr.Quality `json:"quality"`
	MediaInfo           *MediaInfo   `json:"mediaInfo"`
	RelativePath        string       `json:"relativePath"`
	Path                string       `json:"path"`
	SceneName           string       `json:"sceneName"`
	ReleaseGroup        string       `json:"releaseGroup"`
	Edition             string       `json:"edition"`
	Languages           []*arr.Value `json:"languages"`
	ID                  int64        `json:"id"`
	MovieID             int64        `json:"movieId"`
	Size                int64        `json:"size"`
	IndexerFlags        int64        `json:"indexerFlags"`
	QualityCutoffNotMet bool         `json:"qualityCutoffNotMet"`
}

type MediaInfo struct {
	AudioAdditionalFeatures string  `json:"audioAdditionalFeatures"`
	AudioCodec              string  `json:"audioCodec"`
	AudioLanguages          string  `json:"audioLanguages"`
	VideoCodec              string  `json:"videoCodec"`
	Resolution              string  `json:"resolution"`
	RunTime                 string  `json:"runTime"`
	ScanType                string  `json:"scanType"`
	Subtitles               string  `json:"subtitles"`
	AudioBitrate            int     `json:"audioBitrate"`
	AudioChannels           float64 `json:"audioChannels"`
	AudioStreamCount        int     `json:"audioStreamCount"`
	VideoBitDepth           int     `json:"videoBitDepth"`
	VideoBitrate            int     `json:"videoBitrate"`
	VideoFps                float64 `json:"videoFps"`
}

type Collection struct {
	Name   string       `json:"name"`
	Images []*arr.Image `json:"images"`
	TmdbID int64        `json:"tmdbId"`
}

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

type SystemStatusResponse struct {
	Version string `json:"version"`
}

type BadRequestResponse struct {
	Severity       string `json:"severity"`
	ErrorCode      string `json:"errorCode"`
	ErrorMessage   string `json:"errorMessage"`
	PropertyName   string `json:"propertyName"`
	AttemptedValue string `json:"attemptedValue"`
}

func (r *BadRequestResponse) String() string {
	return fmt.Sprintf("[%s: %s] %s: %s - got value: %s", r.Severity, r.ErrorCode, r.PropertyName, r.ErrorMessage, r.AttemptedValue)
}
