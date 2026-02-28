// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package filter

import (
	"context"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/rs/zerolog"
)

// RateLimiter provides in-memory token bucket rate limiting for filters.
// It maintains separate buckets per filter ID to enforce MaxDownloads limits
// without database round-trips on every release.
type RateLimiter struct {
	log     zerolog.Logger
	repo    domain.FilterRepo
	buckets sync.Map // map[int]*filterBucket
}

// filterBucket represents a token bucket for a single filter.
type filterBucket struct {
	mu            sync.Mutex
	filterID      int
	maxDownloads  int
	unit          domain.FilterMaxDownloadsUnit
	tokens        int                 // available tokens
	usedTokens    map[string]struct{} // track which releases acquired tokens (by release ID/hash)
	lastReset     time.Time
	resetInterval time.Duration
}

// TokenReceipt represents a token that was acquired and can be released.
//type TokenReceipt struct {
//	filterID  int
//	releaseID string
//}

// NewRateLimiter creates a new rate limiter service.
func NewRateLimiter(log logger.Logger, repo domain.FilterRepo) *RateLimiter {
	return &RateLimiter{
		log:  log.With().Str("module", "filter-ratelimit").Logger(),
		repo: repo,
	}
}

// InitializeFromDB loads current download counts from the database and initializes
// token buckets for all filters with rate limits enabled.
func (rl *RateLimiter) InitializeFromDB(ctx context.Context, filters []*domain.Filter) error {
	rl.log.Debug().Msg("initializing rate limiter from database")

	for _, filter := range filters {
		if !filter.IsMaxDownloadsLimitEnabled() {
			continue
		}

		// Get current download counts from DB
		if err := rl.repo.GetFilterDownloadCount(ctx, filter); err != nil {
			rl.log.Error().Err(err).Int("filter_id", filter.ID).Msg("failed to get download count for filter")
			continue
		}

		// Calculate remaining tokens based on unit
		var currentCount int
		switch filter.MaxDownloadsUnit {
		case domain.FilterMaxDownloadsHour:
			currentCount = filter.Downloads.HourCount
		case domain.FilterMaxDownloadsDay:
			currentCount = filter.Downloads.DayCount
		case domain.FilterMaxDownloadsWeek:
			currentCount = filter.Downloads.WeekCount
		case domain.FilterMaxDownloadsMonth:
			currentCount = filter.Downloads.MonthCount
		case domain.FilterMaxDownloadsEver:
			currentCount = filter.Downloads.TotalCount
		}

		remainingTokens := filter.MaxDownloads - currentCount
		if remainingTokens < 0 {
			remainingTokens = 0
		}

		bucket := &filterBucket{
			filterID:      filter.ID,
			maxDownloads:  filter.MaxDownloads,
			unit:          filter.MaxDownloadsUnit,
			tokens:        remainingTokens,
			usedTokens:    make(map[string]struct{}),
			lastReset:     time.Now(),
			resetInterval: getResetInterval(filter.MaxDownloadsUnit),
		}

		rl.buckets.Store(filter.ID, bucket)

		rl.log.Debug().Int("filter_id", filter.ID).Str("filter_name", filter.Name).Int("max_downloads", filter.MaxDownloads).Str("unit", string(filter.MaxDownloadsUnit)).Int("current_count", currentCount).Int("remaining_tokens", remainingTokens).Msg("initialized filter bucket")
	}

	return nil
}

// TryAcquire attempts to acquire a token for the given filter and release.
// Returns a TokenReceipt if successful, or nil if the limit has been reached.
func (rl *RateLimiter) TryAcquire(filter *domain.Filter, releaseID string) *domain.FilterRateLimitTokenReceipt {
	if !filter.IsMaxDownloadsLimitEnabled() {
		// No rate limit configured, always allow
		return &domain.FilterRateLimitTokenReceipt{FilterID: filter.ID, ReleaseID: releaseID}
	}

	// Get or create bucket for this filter
	bucketInterface, _ := rl.buckets.LoadOrStore(filter.ID, &filterBucket{
		filterID:      filter.ID,
		maxDownloads:  filter.MaxDownloads,
		unit:          filter.MaxDownloadsUnit,
		tokens:        filter.MaxDownloads,
		usedTokens:    make(map[string]struct{}),
		lastReset:     time.Now(),
		resetInterval: getResetInterval(filter.MaxDownloadsUnit),
	})

	bucket := bucketInterface.(*filterBucket)
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Check if we need to reset the bucket based on time window
	now := time.Now()
	if rl.shouldReset(bucket, now) {
		rl.resetBucket(bucket, filter, now)
	}

	// Check if we have tokens available
	if bucket.tokens <= 0 {
		rl.log.Debug().
			Int("filter_id", filter.ID).
			Str("release_id", releaseID).
			Int("max_downloads", bucket.maxDownloads).
			Str("unit", string(bucket.unit)).
			Msg("rate limit reached, rejecting release")
		return nil
	}

	// Acquire token
	bucket.tokens--
	bucket.usedTokens[releaseID] = struct{}{}

	rl.log.Trace().
		Int("filter_id", filter.ID).
		Str("release_id", releaseID).
		Int("remaining_tokens", bucket.tokens).
		Msg("acquired token for release")

	return &domain.FilterRateLimitTokenReceipt{
		FilterID:  filter.ID,
		ReleaseID: releaseID,
	}
}

// Release returns a token to the bucket if the action failed or was rejected.
// This ensures accurate counting even when releases don't complete successfully.
func (rl *RateLimiter) Release(receipt *domain.FilterRateLimitTokenReceipt) {
	if receipt == nil {
		return
	}

	bucketInterface, ok := rl.buckets.Load(receipt.FilterID)
	if !ok {
		return
	}

	bucket := bucketInterface.(*filterBucket)
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Only release if this release actually acquired a token
	if _, acquired := bucket.usedTokens[receipt.ReleaseID]; !acquired {
		return
	}

	// Return token and remove from used set
	bucket.tokens++
	delete(bucket.usedTokens, receipt.ReleaseID)

	rl.log.Trace().
		Int("filter_id", receipt.FilterID).
		Str("release_id", receipt.ReleaseID).
		Int("remaining_tokens", bucket.tokens).
		Msg("released token back to bucket")
}

// UpdateBucket updates an existing bucket's configuration when filter settings change.
func (rl *RateLimiter) UpdateBucket(filter *domain.Filter) {
	if !filter.IsMaxDownloadsLimitEnabled() {
		// Remove bucket if rate limiting was disabled
		rl.buckets.Delete(filter.ID)
		return
	}

	bucketInterface, ok := rl.buckets.Load(filter.ID)
	if !ok {
		// Bucket doesn't exist yet, will be created on first use
		return
	}

	bucket := bucketInterface.(*filterBucket)
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Update configuration
	oldMax := bucket.maxDownloads
	bucket.maxDownloads = filter.MaxDownloads
	bucket.unit = filter.MaxDownloadsUnit
	bucket.resetInterval = getResetInterval(filter.MaxDownloadsUnit)

	// Adjust tokens if max increased/decreased
	tokenDiff := filter.MaxDownloads - oldMax
	bucket.tokens += tokenDiff
	if bucket.tokens < 0 {
		bucket.tokens = 0
	}
	if bucket.tokens > filter.MaxDownloads {
		bucket.tokens = filter.MaxDownloads
	}

	rl.log.Debug().Int("filter_id", filter.ID).Int("old_max", oldMax).Int("new_max", filter.MaxDownloads).Int("tokens", bucket.tokens).Msg("updated filter bucket configuration")
}

// shouldReset determines if the bucket should be reset based on the time window.
func (rl *RateLimiter) shouldReset(bucket *filterBucket, now time.Time) bool {
	switch bucket.unit {
	case domain.FilterMaxDownloadsHour:
		// Reset if we're in a new hour
		lastHour := bucket.lastReset.Truncate(time.Hour)
		currentHour := now.Truncate(time.Hour)
		return currentHour.After(lastHour)

	case domain.FilterMaxDownloadsDay:
		// Reset if we're in a new day
		lastDay := bucket.lastReset.Truncate(24 * time.Hour)
		currentDay := now.Truncate(24 * time.Hour)
		return currentDay.After(lastDay)

	case domain.FilterMaxDownloadsWeek:
		// Reset if we're in a new week (Sunday to Saturday)
		lastWeek := startOfWeek(bucket.lastReset)
		currentWeek := startOfWeek(now)
		return currentWeek.After(lastWeek)

	case domain.FilterMaxDownloadsMonth:
		// Reset if we're in a new month
		lastMonth := startOfMonth(bucket.lastReset)
		currentMonth := startOfMonth(now)
		return currentMonth.After(lastMonth)

	case domain.FilterMaxDownloadsEver:
		// Never reset
		return false

	default:
		return false
	}
}

// resetBucket resets the token count and clears used tokens.
func (rl *RateLimiter) resetBucket(bucket *filterBucket, filter *domain.Filter, now time.Time) {
	bucket.tokens = bucket.maxDownloads
	bucket.usedTokens = make(map[string]struct{})
	bucket.lastReset = now

	rl.log.Debug().Int("filter_id", bucket.filterID).Str("unit", string(bucket.unit)).Msg("reset filter bucket for new time window")
}

// getResetInterval returns the duration for the given unit.
func getResetInterval(unit domain.FilterMaxDownloadsUnit) time.Duration {
	switch unit {
	case domain.FilterMaxDownloadsHour:
		return time.Hour
	case domain.FilterMaxDownloadsDay:
		return 24 * time.Hour
	case domain.FilterMaxDownloadsWeek:
		return 7 * 24 * time.Hour
	case domain.FilterMaxDownloadsMonth:
		return 30 * 24 * time.Hour // Approximate
	case domain.FilterMaxDownloadsEver:
		return 0 // Never resets
	default:
		return 0
	}
}

// startOfWeek returns the start of the week (Monday) for the given time.
func startOfWeek(t time.Time) time.Time {
	weekday := t.Weekday()
	// Go's Weekday: Sunday = 0, Monday = 1, ..., Saturday = 6
	// We want Monday as the start, so:
	// - If it's Sunday (0), go back 6 days to previous Monday
	// - Otherwise, go back (weekday - 1) days to Monday
	var daysToSubtract int
	if weekday == time.Sunday {
		daysToSubtract = 6
	} else {
		daysToSubtract = int(weekday) - 1
	}
	return t.AddDate(0, 0, -daysToSubtract).Truncate(24 * time.Hour)
}

// startOfMonth returns the start of the month for the given time.
func startOfMonth(t time.Time) time.Time {
	year, month, _ := t.Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
}
