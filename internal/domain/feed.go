// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"time"
)

type FeedCacheRepo interface {
	Get(feedId int, key string) ([]byte, error)
	GetByFeed(ctx context.Context, feedId int) ([]FeedCacheItem, error)
	GetCountByFeed(ctx context.Context, feedId int) (int, error)
	Exists(feedId int, key string) (bool, error)
	Put(feedId int, key string, val []byte, ttl time.Time) error
	PutMany(ctx context.Context, items []FeedCacheItem) error
	Delete(ctx context.Context, feedId int, key string) error
	DeleteByFeed(ctx context.Context, feedId int) error
	DeleteStale(ctx context.Context) error
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
	ToggleEnabled(ctx context.Context, id int, enabled bool) error
	Delete(ctx context.Context, id int) error
}

type Feed struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastRun   time.Time `json:"last_run"`
	NextRun   time.Time `json:"next_run"`

	Settings     *FeedSettingsJSON `json:"settings"`
	Proxy        *Proxy
	Indexer      IndexerMinimal `json:"indexer"`
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	URL          string         `json:"url"`
	ApiKey       string         `json:"api_key"`
	Cookie       string         `json:"cookie"`
	LastRunData  string         `json:"last_run_data"`
	Capabilities []string       `json:"capabilities"`
	ID           int            `json:"id"`
	Interval     int            `json:"interval"`
	Timeout      int            `json:"timeout"` // seconds
	MaxAge       int            `json:"max_age"` // seconds
	IndexerID    int            `json:"indexer_id,omitempty"`

	// belongs to Indexer
	ProxyID  int64
	Enabled  bool `json:"enabled"`
	UseProxy bool
}

type FeedSettingsJSON struct {
	DownloadType FeedDownloadType `json:"download_type"`
}

type FeedIndexer struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	ID         int    `json:"id"`
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
	TTL    time.Time `json:"ttl"`
	FeedId string    `json:"feed_id"`
	Key    string    `json:"key"`
	Value  []byte    `json:"value"`
}

type FindOneParams struct {
	IndexerIdentifier string
	FeedID            int
	IndexerID         int
}
