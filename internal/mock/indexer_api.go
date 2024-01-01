// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package mock

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
)

type IndexerApiClient interface {
	GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error)
	TestAPI(ctx context.Context) (bool, error)
}

type IndexerClient struct {
	url    string
	APIKey string
}

type OptFunc func(client *IndexerClient)

func WithUrl(url string) OptFunc {
	return func(c *IndexerClient) {
		c.url = url
	}
}

func NewMockClient(apiKey string, opts ...OptFunc) IndexerApiClient {
	c := &IndexerClient{
		url:    "",
		APIKey: apiKey,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *IndexerClient) GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error) {
	if torrentID == "" {
		return nil, errors.New("mock client: must have torrentID")
	}

	r := &domain.TorrentBasic{
		Id:       torrentID,
		InfoHash: "",
		Size:     "10GB",
	}

	return r, nil

}

// TestAPI try api access against torrents page
func (c *IndexerClient) TestAPI(ctx context.Context) (bool, error) {
	return true, nil
}
