// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package whisparr

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/arr"
)

type ReleasePushRequest struct {
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

// ReleasePushResponse only covers the fields both versions share. The rest of
// the response is Sonarr shaped on v2 and Radarr shaped on v3.
type ReleasePushResponse struct {
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

// MajorVersion returns the leading component of the reported version, 3 for
// "3.0.0.742". It returns 0 when the version can not be parsed.
func (r SystemStatusResponse) MajorVersion() int {
	major, _, _ := strings.Cut(r.Version, ".")

	v, err := strconv.Atoi(major)
	if err != nil {
		return 0
	}

	return v
}

// Series is a studio as managed by Whisparr v2, where scenes are episodes and
// seasons are years. The resource has no alternate titles.
type Series struct {
	ID         int64        `json:"id"`
	Title      string       `json:"title,omitempty"`
	SortTitle  string       `json:"sortTitle,omitempty"`
	CleanTitle string       `json:"cleanTitle,omitempty"`
	TitleSlug  string       `json:"titleSlug,omitempty"`
	Network    string       `json:"network,omitempty"`
	Status     string       `json:"status,omitempty"`
	SeriesType string       `json:"seriesType,omitempty"`
	Overview   string       `json:"overview,omitempty"`
	Year       int          `json:"year,omitempty"`
	Path       string       `json:"path,omitempty"`
	TvdbID     int64        `json:"tvdbId,omitempty"`
	Runtime    int          `json:"runtime,omitempty"`
	Genres     []string     `json:"genres,omitempty"`
	Tags       []int        `json:"tags,omitempty"`
	Images     []*arr.Image `json:"images,omitempty"`
	Ratings    *arr.Ratings `json:"ratings,omitempty"`
	Added      time.Time    `json:"added"`
	Ended      bool         `json:"ended,omitempty"`
	Monitored  bool         `json:"monitored"`
}

// Movie is a movie or a scene as managed by Whisparr v3, discriminated by
// ItemType. Studios and performers are separate resources.
type Movie struct {
	ID              int64               `json:"id"`
	Title           string              `json:"title,omitempty"`
	AlternateTitles []*AlternativeTitle `json:"alternateTitles,omitempty"`
	SortTitle       string              `json:"sortTitle,omitempty"`
	CleanTitle      string              `json:"cleanTitle,omitempty"`
	TitleSlug       string              `json:"titleSlug,omitempty"`
	ItemType        string              `json:"itemType,omitempty"`
	StudioTitle     string              `json:"studioTitle,omitempty"`
	StudioForeignID string              `json:"studioForeignId,omitempty"`
	PerformerNames  []string            `json:"performerNames,omitempty"`
	Overview        string              `json:"overview,omitempty"`
	Year            int                 `json:"year,omitempty"`
	Path            string              `json:"path,omitempty"`
	Status          string              `json:"status,omitempty"`
	ForeignID       string              `json:"foreignId,omitempty"`
	TmdbID          int64               `json:"tmdbId,omitempty"`
	ReleaseDate     string              `json:"releaseDate,omitempty"`
	Runtime         int                 `json:"runtime,omitempty"`
	Genres          []string            `json:"genres,omitempty"`
	Tags            []int               `json:"tags,omitempty"`
	Images          []*arr.Image        `json:"images,omitempty"`
	Ratings         *arr.Ratings        `json:"ratings,omitempty"`
	Added           time.Time           `json:"added"`
	HasFile         bool                `json:"hasFile,omitempty"`
	IsAvailable     bool                `json:"isAvailable,omitempty"`
	Monitored       bool                `json:"monitored"`
}

type AlternativeTitle struct {
	ID              int    `json:"id"`
	MovieMetadataID int    `json:"movieMetadataId,omitempty"`
	Title           string `json:"title"`
	CleanTitle      string `json:"cleanTitle,omitempty"`
	SourceType      string `json:"sourceType,omitempty"`
}
