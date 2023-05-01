// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"time"
)

type FeedCacheRepo interface {
	Get(bucket string, key string) ([]byte, error)
	GetByBucket(ctx context.Context, bucket string) ([]FeedCacheItem, error)
	GetCountByBucket(ctx context.Context, bucket string) (int, error)
	Exists(bucket string, key string) (bool, error)
	Put(bucket string, key string, val []byte, ttl time.Time) error
	Delete(ctx context.Context, bucket string, key string) error
	DeleteBucket(ctx context.Context, bucket string) error
}

type FeedRepo interface {
	FindByID(ctx context.Context, id int) (*Feed, error)
	FindByIndexerIdentifier(ctx context.Context, indexer string) (*Feed, error)
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
	ID           int               `json:"id"`
	Name         string            `json:"name"`
	Indexer      string            `json:"indexer"`
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
	IndexerID    int               `json:"indexer_id,omitempty"`
	Indexerr     FeedIndexer       `json:"-"`
	LastRun      time.Time         `json:"last_run"`
	LastRunData  string            `json:"last_run_data"`
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
	Bucket string    `json:"bucket"`
	Key    string    `json:"key"`
	Value  []byte    `json:"value"`
	TTL    time.Time `json:"ttl"`
}
