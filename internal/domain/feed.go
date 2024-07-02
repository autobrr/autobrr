// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"time"
)

type FeedCacheRepo interface {
	Get(feedId int64, key string) ([]byte, error)
	GetByFeed(ctx context.Context, feedId int64) ([]FeedCacheItem, error)
	GetCountByFeed(ctx context.Context, feedId int64) (int64, error)
	Exists(feedId int64, key string) (bool, error)
	Put(feedId int64, key string, val []byte, ttl time.Time) error
	PutMany(ctx context.Context, items []FeedCacheItem) error
	Delete(ctx context.Context, feedId int64, key string) error
	DeleteByFeed(ctx context.Context, feedId int64) error
	DeleteStale(ctx context.Context) error
}

type FeedRepo interface {
	FindByID(ctx context.Context, id int64) (*Feed, error)
	FindByIndexerIdentifier(ctx context.Context, indexer string) (*Feed, error)
	Find(ctx context.Context) ([]Feed, error)
	GetLastRunDataByID(ctx context.Context, id int64) (string, error)
	Store(ctx context.Context, feed *Feed) error
	Update(ctx context.Context, feed *Feed) error
	UpdateLastRun(ctx context.Context, feedID int64) error
	UpdateLastRunWithData(ctx context.Context, feedID int64, data string) error
	ToggleEnabled(ctx context.Context, id int64, enabled bool) error
	Delete(ctx context.Context, id int64) error
}

type Feed struct {
	ID           int64             `json:"id"`
	Name         string            `json:"name"`
	Indexer      IndexerMinimal    `json:"indexer"`
	Type         string            `json:"type"`
	Enabled      bool              `json:"enabled"`
	URL          string            `json:"url"`
	Interval     int               `json:"interval"`
	Timeout      int               `json:"timeout"` // seconds
	MaxAge       int               `json:"max_age"` // seconds
	Capabilities []string          `json:"capabilities"`
	ApiKey       string            `json:"api_key"`
	Cookie       string            `json:"cookie"`
	Settings     *FeedSettingsJSON `json:"settings"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	IndexerID    int64             `json:"indexer_id,omitempty"`
	LastRun      time.Time         `json:"last_run"`
	LastRunData  string            `json:"last_run_data"`
	NextRun      time.Time         `json:"next_run"`
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
