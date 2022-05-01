package database

import (
	"database/sql"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/domain"
)

type FeedCacheRepo struct {
	db *DB
}

func NewFeedCacheRepo(db *DB) domain.FeedCacheRepo {
	return &FeedCacheRepo{
		db: db,
	}
}

func (r *FeedCacheRepo) Get(bucket string, key string) ([]byte, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"value",
			"ttl",
		).
		From("feed_cache").
		Where("bucket = ?", bucket).
		Where("key = ?", key).
		Where("ttl > ?", time.Now())

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("feedCache.Get: error building query")
		return nil, err
	}

	row := r.db.handler.QueryRow(query, args...)
	if err := row.Err(); err != nil {
		log.Error().Stack().Err(err).Msg("feedCache.Get: query error")
		return nil, err
	}

	var value []byte
	var ttl time.Duration

	if err := row.Scan(&value, &ttl); err != nil && err != sql.ErrNoRows {
		log.Error().Stack().Err(err).Msg("feedCache.Get: error scanning row")
		return nil, err
	}

	return value, nil
}

func (r *FeedCacheRepo) Exists(bucket string, key string) (bool, error) {
	queryBuilder := r.db.squirrel.
		Select("1").
		Prefix("SELECT EXISTS (").
		From("feed_cache").
		Where("bucket = ?", bucket).
		Where("key = ?", key).
		Suffix(")")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("feedCache.Exists: error building query")
		return false, err
	}

	var exists bool
	err = r.db.handler.QueryRow(query, args...).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		log.Error().Stack().Err(err).Msg("feedCache.Exists: query error")
	}

	return exists, nil
}

func (r *FeedCacheRepo) Put(bucket string, key string, val []byte, ttl time.Time) error {
	queryBuilder := r.db.squirrel.
		Insert("feed_cache").
		Columns("bucket", "key", "value", "ttl").
		Values(bucket, key, val, ttl)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("feedCache.Put: error building query")
		return err
	}

	if _, err = r.db.handler.Exec(query, args...); err != nil {
		log.Error().Stack().Err(err).Msg("feedCache.Put: error executing query")
		return err
	}

	return nil
}

func (r *FeedCacheRepo) Delete(bucket string, key string) error {
	//TODO implement me
	panic("implement me")
}
