// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog"
)

type FeedCacheRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewFeedCacheRepo(log logger.Logger, db *DB) domain.FeedCacheRepo {
	return &FeedCacheRepo{
		log: log.With().Str("module", "database").Str("repo", "feed_cache").Logger(),
		db:  db,
	}
}

func (r *FeedCacheRepo) Get(feedId int, key string) ([]byte, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"value",
			"ttl",
		).
		From("feed_cache").
		Where(sq.Eq{"feed_id": feedId}).
		Where(sq.Eq{"key": key}).
		Where(sq.Gt{"ttl": time.Now()})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.Handler.QueryRow(query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var value []byte
	var ttl time.Time

	if err := row.Scan(&value, &ttl); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	return value, nil
}

func (r *FeedCacheRepo) GetByFeed(ctx context.Context, feedId int) ([]domain.FeedCacheItem, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"feed_id",
			"key",
			"value",
			"ttl",
		).
		From("feed_cache").
		Where(sq.Eq{"feed_id": feedId})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	var data []domain.FeedCacheItem

	for rows.Next() {
		var d domain.FeedCacheItem

		if err := rows.Scan(&d.FeedId, &d.Key, &d.Value, &d.TTL); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		data = append(data, d)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "row error")
	}

	return data, nil
}

func (r *FeedCacheRepo) GetCountByFeed(ctx context.Context, feedId int) (int, error) {
	queryBuilder := r.db.squirrel.
		Select("COUNT(*)").
		From("feed_cache").
		Where(sq.Eq{"feed_id": feedId})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "error building query")
	}

	row := r.db.Handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return 0, errors.Wrap(err, "error executing query")
	}

	var count = 0

	if err := row.Scan(&count); err != nil {
		return 0, errors.Wrap(err, "error scanning row")
	}

	return count, nil
}

func (r *FeedCacheRepo) Exists(feedId int, key string) (bool, error) {
	queryBuilder := r.db.squirrel.
		Select("1").
		Prefix("SELECT EXISTS (").
		From("feed_cache").
		Where(sq.Eq{"feed_id": feedId}).
		Where(sq.Eq{"key": key}).
		Suffix(")")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return false, errors.Wrap(err, "error building query")
	}

	var exists bool
	err = r.db.Handler.QueryRow(query, args...).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, errors.Wrap(err, "error query")
	}

	return exists, nil
}

// ExistingItems checks multiple keys in the cache for a given feed ID
// and returns a map of existing keys to their values
func (r *FeedCacheRepo) ExistingItems(ctx context.Context, feedId int, keys []string) (map[string]bool, error) {
	if len(keys) == 0 {
		return make(map[string]bool), nil
	}

	// Build a query that returns all keys that exist in the cache
	queryBuilder := r.db.squirrel.
		Select("key").
		From("feed_cache").
		Where(sq.Eq{"feed_id": feedId}).
		Where(sq.Eq{"key": keys})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}
	defer rows.Close()

	result := make(map[string]bool)

	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}
		result[key] = true
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "row error")
	}

	return result, nil
}

func (r *FeedCacheRepo) Put(feedId int, key string, val []byte, ttl time.Time) error {
	queryBuilder := r.db.squirrel.
		Insert("feed_cache").
		Columns("feed_id", "key", "value", "ttl").
		Values(feedId, key, val, ttl)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err = r.db.Handler.Exec(query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *FeedCacheRepo) PutMany(ctx context.Context, items []domain.FeedCacheItem) error {
	queryBuilder := r.db.squirrel.
		Insert("feed_cache").
		Columns("feed_id", "key", "value", "ttl")

	for _, item := range items {
		queryBuilder = queryBuilder.Values(item.FeedId, item.Key, item.Value, item.TTL)
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err = r.db.Handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *FeedCacheRepo) Delete(ctx context.Context, feedId int, key string) error {
	queryBuilder := r.db.squirrel.
		Delete("feed_cache").
		Where(sq.Eq{"feed_id": feedId}).
		Where(sq.Eq{"key": key})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return errors.Wrap(err, "error getting rows affected")
	} else if rowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	return nil
}

func (r *FeedCacheRepo) DeleteByFeed(ctx context.Context, feedId int) error {
	queryBuilder := r.db.squirrel.Delete("feed_cache").Where(sq.Eq{"feed_id": feedId})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error exec result")
	}

	r.log.Debug().Msgf("deleted %d rows from feed cache: %d", rows, feedId)

	return nil
}

func (r *FeedCacheRepo) DeleteStale(ctx context.Context) error {
	queryBuilder := r.db.squirrel.Delete("feed_cache")

	if r.db.Driver == "sqlite" {
		queryBuilder = queryBuilder.Where(sq.Expr("ttl < datetime('now', 'localtime', '-30 days')"))
	} else {
		queryBuilder = queryBuilder.Where(sq.Lt{"ttl": time.Now().AddDate(0, 0, -30)})
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error exec result")
	}

	r.log.Debug().Int64("items", rows).Msg("deleted rows from stale feed cache")

	return nil
}
