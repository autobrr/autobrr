package domain

import (
	"context"
	"time"
)

type FeedCacheRepo interface {
	Get(bucket string, key string) ([]byte, error)
	Exists(bucket string, key string) (bool, error)
	Put(bucket string, key string, val []byte, ttl time.Time) error
	Delete(bucket string, key string) error
}

type FeedRepo interface {
	FindByID(ctx context.Context, id int) (*Feed, error)
	FindByIndexerIdentifier(ctx context.Context, indexer string) (*Feed, error)
	Find(ctx context.Context) ([]Feed, error)
	Store(ctx context.Context, feed *Feed) error
	Update(ctx context.Context, feed *Feed) error
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
	Capabilities []string          `json:"capabilities"`
	ApiKey       string            `json:"api_key"`
	Settings     map[string]string `json:"settings"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	IndexerID    int               `json:"-"`
	Indexerr     FeedIndexer       `json:"-"`
}

type FeedIndexer struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
}

type FeedType string

const (
	FeedTypeTorznab FeedType = "TORZNAB"
)
