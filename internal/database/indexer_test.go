package database

import "github.com/autobrr/autobrr/internal/domain"

func getMockIndexer() domain.Indexer {
	return domain.Indexer{
		ID:             0,
		Name:           "indexer1",
		Identifier:     "indexer1",
		Enabled:        true,
		Implementation: "meh",
		BaseURL:        "ok",
		Settings:       nil,
	}
}
