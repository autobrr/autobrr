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

func (r *FeedCacheRepo) Get(bucket string, key string) ([]byte, error) {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Select(
			"value",
			"ttl",
		).
		From("feed_cache").
		Where(sq.Eq{"bucket": bucket}).
		Where(sq.Eq{"key": key}).
		Where(sq.Gt{"ttl": time.Now()})

	rows, err := queryBuilder.Query()
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	var value []byte
	var ttl time.Duration

	if err := rows.Scan(&value, &ttl); err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "error scanning row")
	}

	return value, nil
}

func (r *FeedCacheRepo) GetByBucket(ctx context.Context, bucket string) ([]domain.FeedCacheItem, error) {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Select(
			"bucket",
			"key",
			"value",
			"ttl",
		).
		From("feed_cache").
		Where(sq.Eq{"bucket": bucket})

	rows, err := queryBuilder.Query()
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	var data []domain.FeedCacheItem

	for rows.Next() {
		var d domain.FeedCacheItem

		if err := rows.Scan(&d.Bucket, &d.Key, &d.Value, &d.TTL); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		data = append(data, d)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "row error")
	}

	return data, nil
}

func (r *FeedCacheRepo) GetCountByBucket(ctx context.Context, bucket string) (int, error) {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Select("COUNT(*)").
		From("feed_cache").
		Where(sq.Eq{"bucket": bucket})

	row := queryBuilder.QueryRow()

	var count = 0
	if err := row.Scan(&count); err != nil {
		return 0, errors.Wrap(err, "error scanning row")
	}

	return count, nil
}

func (r *FeedCacheRepo) Exists(bucket string, key string) (bool, error) {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Select("1").
		Prefix("SELECT EXISTS (").
		From("feed_cache").
		Where(sq.Eq{"bucket": bucket}).
		Where(sq.Eq{"key": key}).
		Suffix(")")

	var exists bool
	err := queryBuilder.QueryRow().Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, errors.Wrap(err, "error query")
	}

	return exists, nil
}

func (r *FeedCacheRepo) Put(bucket string, key string, val []byte, ttl time.Time) error {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Insert("feed_cache").
		Columns("bucket", "key", "value", "ttl").
		Values(bucket, key, val, ttl)

	if _, err := queryBuilder.Exec(); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *FeedCacheRepo) Delete(ctx context.Context, bucket string, key string) error {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Delete("feed_cache").
		Where(sq.Eq{"bucket": bucket}).
		Where(sq.Eq{"key": key})

	if _, err := queryBuilder.Exec(); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *FeedCacheRepo) DeleteBucket(ctx context.Context, bucket string) error {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Delete("feed_cache").
		Where(sq.Eq{"bucket": bucket})

	result, err := queryBuilder.Exec()
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error exec result")
	}

	if rows == 0 {
		return errors.Wrap(err, "error no rows affected")
	}

	return nil
}
