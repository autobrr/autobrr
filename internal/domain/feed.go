// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/pkg/newznab"
	"github.com/autobrr/autobrr/pkg/torznab"
)

type FeedCacheRepo interface {
	Get(feedId int, key string) ([]byte, error)
	GetByFeed(ctx context.Context, feedId int) ([]FeedCacheItem, error)
	GetCountByFeed(ctx context.Context, feedId int) (int, error)
	Exists(feedId int, key string) (bool, error)
	ExistingItems(ctx context.Context, feedId int, keys []string) (map[string]bool, error)
	Put(feedId int, key string, val []byte, ttl time.Time) error
	PutMany(ctx context.Context, items []FeedCacheItem) error
	Delete(ctx context.Context, feedId int, key string) error
	DeleteByFeed(ctx context.Context, feedId int) error
	DeleteStale(ctx context.Context) error
	DeleteOrphaned(ctx context.Context) error
}

type FeedRepo interface {
	FindOne(ctx context.Context, params FindOneParams) (*Feed, error)
	FindByID(ctx context.Context, id int) (*Feed, error)
	Find(ctx context.Context) ([]Feed, error)
	GetLastRunDataByID(ctx context.Context, id int) (string, error)
	Store(ctx context.Context, feed *Feed) error
	Update(ctx context.Context, feed *Feed) error
	UpdateLastRun(ctx context.Context, feedID int) error
	UpdateLastRunWithData(ctx context.Context, feedID int, data string) error
	UpdateCapabilities(ctx context.Context, feedID int, caps *FeedCapabilities) error
	ToggleEnabled(ctx context.Context, id int, enabled bool) error
	Delete(ctx context.Context, id int) error
}

type Feed struct {
	ID           int               `json:"id"`
	Name         string            `json:"name"`
	Indexer      IndexerMinimal    `json:"indexer"`
	Type         string            `json:"type"`
	Enabled      bool              `json:"enabled"`
	URL          string            `json:"url"`
	Interval     int               `json:"interval"`
	Timeout      int               `json:"timeout"` // seconds
	MaxAge       int               `json:"max_age"` // seconds
	Categories   []int             `json:"categories"`
	Capabilities *FeedCapabilities `json:"capabilities"`
	ApiKey       string            `json:"api_key"`
	Cookie       string            `json:"cookie"`
	Settings     *FeedSettingsJSON `json:"settings"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	IndexerID    int               `json:"indexer_id,omitempty"`
	LastRun      time.Time         `json:"last_run"`
	LastRunData  string            `json:"last_run_data"`
	NextRun      time.Time         `json:"next_run"`

	// belongs to Indexer
	ProxyID  int64  `json:"-"`
	UseProxy bool   `json:"-"`
	Proxy    *Proxy `json:"-"`
}

type FeedSettingsJSON struct {
	DownloadType FeedDownloadType `json:"download_type"`
}

type FeedIndexer struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
}

type FeedType string

const (
	FeedTypeTorznab FeedType = "TORZNAB"
	FeedTypeNewznab FeedType = "NEWZNAB"
	FeedTypeRSS     FeedType = "RSS"
)

type FeedDownloadType string

const (
	FeedDownloadTypeMagnet  FeedDownloadType = "MAGNET"
	FeedDownloadTypeTorrent FeedDownloadType = "TORRENT"
)

type FeedCacheItem struct {
	FeedId string    `json:"feed_id"`
	Key    string    `json:"key"`
	Value  []byte    `json:"value"`
	TTL    time.Time `json:"ttl"`
}

type FindOneParams struct {
	FeedID            int
	IndexerID         int
	IndexerIdentifier string
}

type FeedCapsServer struct {
	Version   string `json:"version"`
	Title     string `json:"title"`
	Strapline string `json:"strapline"`
	Email     string `json:"email"`
	URL       string `json:"url"`
	Image     string `json:"image"`
}

type FeedCapabilitiesLimits struct {
	Max     int `json:"max"`
	Default int `json:"default"`
}

type FeedCapabilitiesSearching struct {
	Search      FeedCapsSearch `json:"search"`
	TvSearch    FeedCapsSearch `json:"tv-search"`
	MovieSearch FeedCapsSearch `json:"movie-search"`
	AudioSearch FeedCapsSearch `json:"audio-search"`
	BookSearch  FeedCapsSearch `json:"book-search"`
}

type FeedCapsSearch struct {
	Available       string `json:"available"`
	SupportedParams string `json:"supportedParams"`
	SearchEngine    string `json:"searchEngine"`
}

type FeedCapabilitiesCategories struct {
	Category []FeedCapabilitiesCategory `json:"category"`
}

type FeedCapabilitiesCategory struct {
	ID            int                        `json:"id"`
	Name          string                     `json:"name"`
	SubCategories []FeedCapabilitiesCategory `json:"subcategories"`
}

type FeedCapabilities struct {
	Server     FeedCapsServer             `json:"server"`
	Limits     FeedCapabilitiesLimits     `json:"limits"`
	Categories []FeedCapabilitiesCategory `json:"categories"`
	Searching  FeedCapabilitiesSearching  `json:"searching"`
}

func NewFeedCapabilitiesFromTorznab(caps *torznab.Caps) *FeedCapabilities {
	c := &FeedCapabilities{
		Server: FeedCapsServer{
			Version:   caps.Server.Version,
			Title:     caps.Server.Title,
			Strapline: caps.Server.Strapline,
			Email:     caps.Server.Email,
			URL:       caps.Server.URL,
			Image:     caps.Server.Image,
		},
		Limits: FeedCapabilitiesLimits{
			Max:     100,
			Default: 50,
		},
		Categories: make([]FeedCapabilitiesCategory, 0),
		Searching:  FeedCapabilitiesSearching{},
	}

	if maxVal := parseFeedCapLimitValue(caps.Limits.Max); maxVal > 0 {
		c.Limits.Max = maxVal
	}
	if def := parseFeedCapLimitValue(caps.Limits.Default); def > 0 {
		c.Limits.Default = def
	}

	for _, cat := range caps.Categories.Categories {
		parentCat := FeedCapabilitiesCategory{
			ID:            cat.ID,
			Name:          cat.Name,
			SubCategories: make([]FeedCapabilitiesCategory, 0),
		}
		for _, j := range cat.SubCategories {
			parentCat.SubCategories = append(parentCat.SubCategories, FeedCapabilitiesCategory{
				ID:   j.ID,
				Name: j.Name,
			})
		}
		c.Categories = append(c.Categories, parentCat)
	}

	c.Searching = FeedCapabilitiesSearching{
		Search:      FeedCapsSearch{Available: caps.Searching.Search.Available, SupportedParams: caps.Searching.Search.SupportedParams},
		TvSearch:    FeedCapsSearch{Available: caps.Searching.TvSearch.Available, SupportedParams: caps.Searching.TvSearch.SupportedParams},
		MovieSearch: FeedCapsSearch{Available: caps.Searching.MovieSearch.Available, SupportedParams: caps.Searching.MovieSearch.SupportedParams},
		AudioSearch: FeedCapsSearch{Available: caps.Searching.AudioSearch.Available, SupportedParams: caps.Searching.AudioSearch.SupportedParams},
		BookSearch:  FeedCapsSearch{Available: caps.Searching.BookSearch.Available, SupportedParams: caps.Searching.BookSearch.SupportedParams},
	}

	return c
}

func NewFeedCapabilitiesFromNewznab(caps *newznab.Caps) *FeedCapabilities {
	c := &FeedCapabilities{
		Server: FeedCapsServer{
			Version:   caps.Server.Version,
			Title:     caps.Server.Title,
			Strapline: caps.Server.Strapline,
			Email:     caps.Server.Email,
			URL:       caps.Server.URL,
			Image:     caps.Server.Image,
		},
		Limits: FeedCapabilitiesLimits{
			Max:     100,
			Default: 50,
		},
		Categories: make([]FeedCapabilitiesCategory, 0),
	}

	if maxVal := parseFeedCapLimitValue(caps.Limits.Max); maxVal > 0 {
		c.Limits.Max = maxVal
	}
	if def := parseFeedCapLimitValue(caps.Limits.Default); def > 0 {
		c.Limits.Default = def
	}

	for _, cat := range caps.Categories.Categories {
		parentCat := FeedCapabilitiesCategory{
			ID:            cat.ID,
			Name:          cat.Name,
			SubCategories: make([]FeedCapabilitiesCategory, 0),
		}
		for _, j := range cat.SubCategories {
			parentCat.SubCategories = append(parentCat.SubCategories, FeedCapabilitiesCategory{
				ID:   j.ID,
				Name: j.Name,
			})
		}
		c.Categories = append(c.Categories, parentCat)
	}

	c.Searching = FeedCapabilitiesSearching{
		Search:      FeedCapsSearch{Available: caps.Searching.Search.Available, SupportedParams: caps.Searching.Search.SupportedParams},
		TvSearch:    FeedCapsSearch{Available: caps.Searching.TvSearch.Available, SupportedParams: caps.Searching.TvSearch.SupportedParams},
		MovieSearch: FeedCapsSearch{Available: caps.Searching.MovieSearch.Available, SupportedParams: caps.Searching.MovieSearch.SupportedParams},
		AudioSearch: FeedCapsSearch{Available: caps.Searching.AudioSearch.Available, SupportedParams: caps.Searching.AudioSearch.SupportedParams},
		BookSearch:  FeedCapsSearch{Available: caps.Searching.BookSearch.Available, SupportedParams: caps.Searching.BookSearch.SupportedParams},
	}

	return c
}

func parseFeedCapLimitValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}
