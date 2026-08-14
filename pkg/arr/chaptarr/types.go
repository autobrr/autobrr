// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package chaptarr

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

// BadRequestResponse is one entry of the validation error array Chaptarr returns
// with a 400. Unlike Readarr, which serializes the whole FluentValidation failure,
// Chaptarr projects it down to the property name and message only.
type BadRequestResponse struct {
	PropertyName string `json:"propertyName"`
	ErrorMessage string `json:"errorMessage"`
}

func (r *BadRequestResponse) String() string {
	return fmt.Sprintf("%s: %s", r.PropertyName, r.ErrorMessage)
}

type SystemStatusResponse struct {
	AppName string `json:"appName"`
	Version string `json:"version"`
}

// Book omits genres: Chaptarr has served it both as a string and as an array
// (autobrr/autobrr#2413), and a single mismatched field fails the whole decode.
type Book struct {
	ID                 int64        `json:"id"`
	Title              string       `json:"title"`
	AuthorTitle        string       `json:"authorTitle"`
	SeriesTitle        string       `json:"seriesTitle"`
	Disambiguation     string       `json:"disambiguation,omitempty"`
	Overview           string       `json:"overview,omitempty"`
	AuthorID           int64        `json:"authorId"`
	ForeignBookID      string       `json:"foreignBookId"`
	TitleSlug          string       `json:"titleSlug"`
	Asin               string       `json:"asin,omitempty"`
	AudibleAsin        string       `json:"audibleASIN,omitempty"`
	Monitored          bool         `json:"monitored"`
	AudiobookMonitored bool         `json:"audiobookMonitored"`
	EbookMonitored     bool         `json:"ebookMonitored"`
	AnyEditionOk       bool         `json:"anyEditionOk"`
	MediaType          string       `json:"mediaType"`
	Narrator           string       `json:"narrator,omitempty"`
	PageCount          int          `json:"pageCount"`
	Ratings            *arr.Ratings `json:"ratings"`
	ReleaseDate        time.Time    `json:"releaseDate"`
	Added              time.Time    `json:"added"`
	Author             *BookAuthor  `json:"author,omitempty"`
	Images             []*arr.Image `json:"images"`
	Links              []*arr.Link  `json:"links"`
	Statistics         *Statistics  `json:"statistics,omitempty"`
	Editions           []*Edition   `json:"editions"`
}

// Statistics for a Book, or maybe an author.
type Statistics struct {
	BookCount      int     `json:"bookCount"`
	BookFileCount  int     `json:"bookFileCount"`
	TotalBookCount int     `json:"totalBookCount"`
	SizeOnDisk     int64   `json:"sizeOnDisk"`
	PercentOfBooks float64 `json:"percentOfBooks"`
}

// BookAuthor of a Book.
type BookAuthor struct {
	ID              int64        `json:"id"`
	Status          string       `json:"status"`
	AuthorName      string       `json:"authorName"`
	ForeignAuthorID string       `json:"foreignAuthorId"`
	TitleSlug       string       `json:"titleSlug"`
	Overview        string       `json:"overview,omitempty"`
	Path            string       `json:"path"`
	CleanName       string       `json:"cleanName"`
	SortName        string       `json:"sortName"`
	Tags            []int        `json:"tags"`
	AudiobookTags   []int        `json:"audiobookTags"`
	EbookTags       []int        `json:"ebookTags"`
	Links           []*arr.Link  `json:"links"`
	Images          []*arr.Image `json:"images"`
	Ratings         *arr.Ratings `json:"ratings"`
	Statistics      *Statistics  `json:"statistics"`
	Added           time.Time    `json:"added"`
	Monitored       bool         `json:"monitored"`
}

// Edition is more Book meta data.
type Edition struct {
	ID               int64        `json:"id"`
	BookID           int64        `json:"bookId"`
	ForeignEditionID string       `json:"foreignEditionId"`
	TitleSlug        string       `json:"titleSlug"`
	Isbn13           string       `json:"isbn13"`
	Isbn10           string       `json:"isbn10"`
	Asin             string       `json:"asin"`
	AudibleAsin      string       `json:"audibleASIN,omitempty"`
	Title            string       `json:"title"`
	Subtitle         string       `json:"subtitle,omitempty"`
	Language         string       `json:"language,omitempty"`
	Overview         string       `json:"overview,omitempty"`
	Format           string       `json:"format"`
	EditionFormat    string       `json:"editionFormat,omitempty"`
	Publisher        string       `json:"publisher"`
	Narrator         string       `json:"narrator,omitempty"`
	PageCount        int          `json:"pageCount"`
	DurationSeconds  int          `json:"durationSeconds,omitempty"`
	ReleaseDate      time.Time    `json:"releaseDate"`
	Images           []*arr.Image `json:"images"`
	Links            []*arr.Link  `json:"links"`
	Ratings          *arr.Ratings `json:"ratings"`
	Monitored        bool         `json:"monitored"`
	ManualAdd        bool         `json:"manualAdd"`
	IsEbook          bool         `json:"isEbook"`
}
