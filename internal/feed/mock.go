package feed

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
)

type mockFeedRepo struct{}

func (m *mockFeedRepo) UpdateLastRunWithData(_ context.Context, _ int, _ string) error {
	return nil
}

type mockFeedCacheRepo struct{}

func (m *mockFeedCacheRepo) ExistingItems(_ context.Context, _ int, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (m *mockFeedCacheRepo) PutMany(_ context.Context, _ []domain.FeedCacheItem) error {
	return nil
}

type mockReleaseSvc struct{}

func (m *mockReleaseSvc) ProcessMultipleFromIndexer(_ []*domain.Release, _ domain.IndexerMinimal) error {
	return nil
}
