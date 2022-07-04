package database

import (
	"context"
	"database/sql"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog"
)

func NewFeedRepo(log logger.Logger, db *DB) domain.FeedRepo {
	return &FeedRepo{
		log: log.With().Str("repo", "feed").Logger(),
		db:  db,
	}
}

type FeedRepo struct {
	log zerolog.Logger
	db  *DB
}

func (r *FeedRepo) FindByID(ctx context.Context, id int) (*domain.Feed, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"indexer",
			"name",
			"type",
			"enabled",
			"url",
			"interval",
			"api_key",
			"created_at",
			"updated_at",
		).
		From("feed").
		Where("id = ?", id)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var f domain.Feed

	var apiKey sql.NullString

	if err := row.Scan(&f.ID, &f.Indexer, &f.Name, &f.Type, &f.Enabled, &f.URL, &f.Interval, &apiKey, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, errors.Wrap(err, "error scanning row")

	}

	f.ApiKey = apiKey.String

	return &f, nil
}

func (r *FeedRepo) FindByIndexerIdentifier(ctx context.Context, indexer string) (*domain.Feed, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"indexer",
			"name",
			"type",
			"enabled",
			"url",
			"interval",
			"api_key",
			"created_at",
			"updated_at",
		).
		From("feed").
		Where("indexer = ?", indexer)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var f domain.Feed

	var apiKey sql.NullString

	if err := row.Scan(&f.ID, &f.Indexer, &f.Name, &f.Type, &f.Enabled, &f.URL, &f.Interval, &apiKey, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, errors.Wrap(err, "error scanning row")

	}

	f.ApiKey = apiKey.String

	return &f, nil
}

func (r *FeedRepo) Find(ctx context.Context) ([]domain.Feed, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"indexer",
			"name",
			"type",
			"enabled",
			"url",
			"interval",
			"api_key",
			"created_at",
			"updated_at",
		).
		From("feed").
		OrderBy("name ASC")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	feeds := make([]domain.Feed, 0)
	for rows.Next() {
		var f domain.Feed

		var apiKey sql.NullString

		if err := rows.Scan(&f.ID, &f.Indexer, &f.Name, &f.Type, &f.Enabled, &f.URL, &f.Interval, &apiKey, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, errors.Wrap(err, "error scanning row")

		}

		f.ApiKey = apiKey.String

		feeds = append(feeds, f)
	}

	return feeds, nil
}

func (r *FeedRepo) Store(ctx context.Context, feed *domain.Feed) error {
	queryBuilder := r.db.squirrel.
		Insert("feed").
		Columns(
			"name",
			"indexer",
			"type",
			"enabled",
			"url",
			"interval",
			"api_key",
			"indexer_id",
		).
		Values(
			feed.Name,
			feed.Indexer,
			feed.Type,
			feed.Enabled,
			feed.URL,
			feed.Interval,
			feed.ApiKey,
			feed.IndexerID,
		).
		Suffix("RETURNING id").RunWith(r.db.handler)

	var retID int

	if err := queryBuilder.QueryRowContext(ctx).Scan(&retID); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	feed.ID = retID

	return nil
}

func (r *FeedRepo) Update(ctx context.Context, feed *domain.Feed) error {
	queryBuilder := r.db.squirrel.
		Update("feed").
		Set("name", feed.Name).
		Set("indexer", feed.Indexer).
		Set("type", feed.Type).
		Set("enabled", feed.Enabled).
		Set("url", feed.URL).
		Set("interval", feed.Interval).
		Set("api_key", feed.ApiKey).
		Where("id = ?", feed.ID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *FeedRepo) ToggleEnabled(ctx context.Context, id int, enabled bool) error {
	var err error

	queryBuilder := r.db.squirrel.
		Update("feed").
		Set("enabled", enabled).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where("id = ?", id)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}
	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *FeedRepo) Delete(ctx context.Context, id int) error {
	queryBuilder := r.db.squirrel.
		Delete("feed").
		Where("id = ?", id)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Info().Msgf("feed.delete: successfully deleted: %v", id)

	return nil
}
