package feed

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
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

func (m *mockReleaseSvc) ProcessMultipleFromIndexer(_ context.Context, _ []*domain.Release, _ domain.IndexerMinimal) error {
	return nil
}

// stubFeedCacheRepo is a stateful cache repo for pagination tests: keys written
// via PutMany become visible to later ExistingItems calls.
type stubFeedCacheRepo struct {
	existing map[string]bool
	putErr   error
	putCalls [][]string

	// existingErrOnCall and putErrOnCall fail only the nth call (1-based), 0 never.
	existingErrOnCall int
	putErrOnCall      int
	existingCalls     int
	putAttempts       int
}

func (m *stubFeedCacheRepo) ExistingItems(_ context.Context, _ int, keys []string) (map[string]bool, error) {
	m.existingCalls++
	if m.existingErrOnCall == m.existingCalls {
		return nil, errors.New("existing items failed")
	}

	res := make(map[string]bool, len(keys))
	for _, key := range keys {
		if m.existing[key] {
			res[key] = true
		}
	}

	return res, nil
}

func (m *stubFeedCacheRepo) PutMany(_ context.Context, items []domain.FeedCacheItem) error {
	m.putAttempts++
	if m.putErr != nil {
		return m.putErr
	}
	if m.putErrOnCall == m.putAttempts {
		return errors.New("put many failed")
	}

	keys := make([]string, 0, len(items))
	for _, item := range items {
		m.existing[item.Key] = true
		keys = append(keys, item.Key)
	}

	m.putCalls = append(m.putCalls, keys)

	return nil
}
