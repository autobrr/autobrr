// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package lidarr

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

func (r BadRequestResponse) String() string {
	return fmt.Sprintf("[%s: %s] %s: %s - got value: %s", r.Severity, r.ErrorCode, r.PropertyName, r.ErrorMessage, r.AttemptedValue)
}

type SystemStatusResponse struct {
	Version string `json:"version"`
}

type Statistics struct {
	AlbumCount      int     `json:"albumCount,omitempty"`
	TrackFileCount  int     `json:"trackFileCount"`
	TrackCount      int     `json:"trackCount"`
	TotalTrackCount int     `json:"totalTrackCount"`
	SizeOnDisk      int     `json:"sizeOnDisk"`
	PercentOfTracks float64 `json:"percentOfTracks"`
}

type Artist struct {
	LastInfoSync      time.Time         `json:"lastInfoSync,omitempty"`
	Added             time.Time         `json:"added,omitempty"`
	Ratings           *arr.Ratings      `json:"ratings,omitempty"`
	Statistics        *Statistics       `json:"statistics,omitempty"`
	LastAlbum         *Album            `json:"lastAlbum,omitempty"`
	NextAlbum         *Album            `json:"nextAlbum,omitempty"`
	AddOptions        *ArtistAddOptions `json:"addOptions,omitempty"`
	Status            string            `json:"status,omitempty"`
	ArtistName        string            `json:"artistName,omitempty"`
	ForeignArtistID   string            `json:"foreignArtistId,omitempty"`
	Overview          string            `json:"overview,omitempty"`
	ArtistType        string            `json:"artistType,omitempty"`
	Disambiguation    string            `json:"disambiguation,omitempty"`
	RootFolderPath    string            `json:"rootFolderPath,omitempty"`
	Path              string            `json:"path,omitempty"`
	CleanName         string            `json:"cleanName,omitempty"`
	SortName          string            `json:"sortName,omitempty"`
	Links             []*arr.Link       `json:"links,omitempty"`
	Images            []*arr.Image      `json:"images,omitempty"`
	Genres            []string          `json:"genres,omitempty"`
	Tags              []int             `json:"tags,omitempty"`
	ID                int64             `json:"id"`
	TadbID            int64             `json:"tadbId,omitempty"`
	DiscogsID         int64             `json:"discogsId,omitempty"`
	QualityProfileID  int64             `json:"qualityProfileId,omitempty"`
	MetadataProfileID int64             `json:"metadataProfileId,omitempty"`
	AlbumFolder       bool              `json:"albumFolder,omitempty"`
	Monitored         bool              `json:"monitored"`
	Ended             bool              `json:"ended,omitempty"`
}

type Album struct {
	ReleaseDate    time.Time        `json:"releaseDate"`
	Ratings        *arr.Ratings     `json:"ratings"`
	Artist         *Artist          `json:"artist"`
	Statistics     *Statistics      `json:"statistics"`
	AddOptions     *AlbumAddOptions `json:"addOptions,omitempty"`
	Title          string           `json:"title"`
	Disambiguation string           `json:"disambiguation"`
	Overview       string           `json:"overview"`
	ForeignAlbumID string           `json:"foreignAlbumId"`
	AlbumType      string           `json:"albumType"`
	RemoteCover    string           `json:"remoteCover,omitempty"`
	SecondaryTypes []interface{}    `json:"secondaryTypes"`
	Releases       []*AlbumRelease  `json:"releases"`
	Genres         []interface{}    `json:"genres"`
	Media          []*Media         `json:"media"`
	Links          []*arr.Link      `json:"links"`
	Images         []*arr.Image     `json:"images"`
	ID             int64            `json:"id,omitempty"`
	ArtistID       int64            `json:"artistId"`
	ProfileID      int64            `json:"profileId"`
	Duration       int              `json:"duration"`
	MediumCount    int              `json:"mediumCount"`
	Monitored      bool             `json:"monitored"`
	AnyReleaseOk   bool             `json:"anyReleaseOk"`
	Grabbed        bool             `json:"grabbed"`
}

// Release is part of an Album.
type AlbumRelease struct {
	ForeignReleaseID string   `json:"foreignReleaseId"`
	Title            string   `json:"title"`
	Status           string   `json:"status"`
	Disambiguation   string   `json:"disambiguation"`
	Format           string   `json:"format"`
	Media            []*Media `json:"media"`
	Country          []string `json:"country"`
	Label            []string `json:"label"`
	ID               int64    `json:"id"`
	AlbumID          int64    `json:"albumId"`
	Duration         int      `json:"duration"`
	TrackCount       int      `json:"trackCount"`
	MediumCount      int      `json:"mediumCount"`
	Monitored        bool     `json:"monitored"`
}

// Media is part of an Album.
type Media struct {
	MediumName   string `json:"mediumName"`
	MediumFormat string `json:"mediumFormat"`
	MediumNumber int64  `json:"mediumNumber"`
}

// ArtistAddOptions is part of an artist and an album.
type ArtistAddOptions struct {
	Monitor                string `json:"monitor,omitempty"`
	Monitored              bool   `json:"monitored,omitempty"`
	SearchForMissingAlbums bool   `json:"searchForMissingAlbums,omitempty"`
}

type AlbumAddOptions struct {
	SearchForNewAlbum bool `json:"searchForNewAlbum,omitempty"`
}
