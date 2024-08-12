package feed

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
)

type mockFeedRepo struct{}

func (repo mockFeedRepo) UpdateLastRunWithData(ctx context.Context, feedID int, data string) error {
	return nil
}

type mockFeedCacheRepo struct{}

func (repo mockFeedCacheRepo) Exists(feedId int, key string) (bool, error) {
	return false, nil
}

func (repo mockFeedCacheRepo) PutMany(ctx context.Context, items []domain.FeedCacheItem) error {
	return nil
}

type mockReleaseService struct{}

func (s mockReleaseService) ProcessMultiple(releases []*domain.Release) {
	return
}
