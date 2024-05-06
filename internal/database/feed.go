// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"encoding/json"

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

func (r *FeedRepo) FindByID(ctx context.Context, id int64) (*domain.Feed, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"f.id",
			"i.id",
			"i.identifier",
			"i.identifier_external",
			"i.name",
			"f.name",
			"f.type",
			"f.enabled",
			"f.url",
			"f.interval",
			"f.timeout",
			"f.max_age",
			"f.api_key",
			"f.cookie",
			"f.settings",
			"f.created_at",
			"f.updated_at",
		).
		From("feed f").
		Join("indexer i ON f.indexer_id = i.id").
		Where(sq.Eq{"f.id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var f domain.Feed

	var apiKey, cookie, settings sql.Null[string]

	if err := row.Scan(&f.ID, &f.Indexer.ID, &f.Indexer.Identifier, &f.Indexer.IdentifierExternal, &f.Indexer.Name, &f.Name, &f.Type, &f.Enabled, &f.URL, &f.Interval, &f.Timeout, &f.MaxAge, &apiKey, &cookie, &settings, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, errors.Wrap(err, "error scanning row")
	}

	f.ApiKey = apiKey.V
	f.Cookie = cookie.V

	if settings.Valid {
		var settingsJson domain.FeedSettingsJSON
		if err = json.Unmarshal([]byte(settings.V), &settingsJson); err != nil {
			return nil, errors.Wrap(err, "error unmarshal settings")
		}

		f.Settings = &settingsJson
	}

	return &f, nil
}

func (r *FeedRepo) FindByIndexerIdentifier(ctx context.Context, indexer string) (*domain.Feed, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"f.id",
			"i.id",
			"i.identifier",
			"i.identifier_external",
			"i.name",
			"f.name",
			"f.type",
			"f.enabled",
			"f.url",
			"f.interval",
			"f.timeout",
			"f.max_age",
			"f.api_key",
			"f.cookie",
			"f.settings",
			"f.created_at",
			"f.updated_at",
		).
		From("feed f").
		Join("indexer i ON f.indexer_id = i.id").
		Where(sq.Eq{"i.name": indexer})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var f domain.Feed

	var apiKey, cookie, settings sql.Null[string]

	if err := row.Scan(&f.ID, &f.Indexer.ID, &f.Indexer.Identifier, &f.Indexer.IdentifierExternal, &f.Indexer.Name, &f.Name, &f.Type, &f.Enabled, &f.URL, &f.Interval, &f.Timeout, &f.MaxAge, &apiKey, &cookie, &settings, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, errors.Wrap(err, "error scanning row")
	}

	f.ApiKey = apiKey.V
	f.Cookie = cookie.V

	var settingsJson domain.FeedSettingsJSON
	if err = json.Unmarshal([]byte(settings.V), &settingsJson); err != nil {
		return nil, errors.Wrap(err, "error unmarshal settings")
	}

	f.Settings = &settingsJson

	return &f, nil
}

func (r *FeedRepo) Find(ctx context.Context) ([]domain.Feed, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"f.id",
			"i.id",
			"i.identifier",
			"i.identifier_external",
			"i.name",
			"f.name",
			"f.type",
			"f.enabled",
			"f.url",
			"f.interval",
			"f.timeout",
			"f.max_age",
			"f.api_key",
			"f.cookie",
			"f.last_run",
			"f.last_run_data",
			"f.settings",
			"f.created_at",
			"f.updated_at",
		).
		From("feed f").
		Join("indexer i ON f.indexer_id = i.id").
		OrderBy("f.name ASC")

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

		var apiKey, cookie, lastRunData, settings sql.Null[string]
		var lastRun sql.NullTime

		if err := rows.Scan(&f.ID, &f.Indexer.ID, &f.Indexer.Identifier, &f.Indexer.IdentifierExternal, &f.Indexer.Name, &f.Name, &f.Type, &f.Enabled, &f.URL, &f.Interval, &f.Timeout, &f.MaxAge, &apiKey, &cookie, &lastRun, &lastRunData, &settings, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		f.LastRun = lastRun.Time
		f.LastRunData = lastRunData.V
		f.ApiKey = apiKey.V
		f.Cookie = cookie.V

		f.Settings = &domain.FeedSettingsJSON{
			DownloadType: domain.FeedDownloadTypeTorrent,
		}

		if settings.Valid {
			var settingsJson domain.FeedSettingsJSON
			if err = json.Unmarshal([]byte(settings.V), &settingsJson); err != nil {
				return nil, errors.Wrap(err, "error unmarshal settings")
			}

			f.Settings = &settingsJson
		}

		feeds = append(feeds, f)
	}

	return feeds, nil
}

func (r *FeedRepo) GetLastRunDataByID(ctx context.Context, id int64) (string, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"last_run_data",
		).
		From("feed").
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return "", errors.Wrap(err, "error building query")
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return "", errors.Wrap(err, "error executing query")
	}

	var data sql.Null[string]

	if err := row.Scan(&data); err != nil {
		return "", errors.Wrap(err, "error scanning row")
	}

	return data.V, nil
}

func (r *FeedRepo) Store(ctx context.Context, feed *domain.Feed) error {
	settings, err := json.Marshal(feed.Settings)
	if err != nil {
		return errors.Wrap(err, "error marshaling feed settings json data")
	}

	queryBuilder := r.db.squirrel.
		Insert("feed").
		Columns(
			"name",
			"type",
			"enabled",
			"url",
			"interval",
			"timeout",
			"api_key",
			"indexer_id",
			"settings",
		).
		Values(
			feed.Name,
			feed.Type,
			feed.Enabled,
			feed.URL,
			feed.Interval,
			feed.Timeout,
			feed.ApiKey,
			feed.IndexerID,
			settings,
		).
		Suffix("RETURNING id").RunWith(r.db.handler)

	var retID int64

	if err := queryBuilder.QueryRowContext(ctx).Scan(&retID); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	feed.ID = retID

	return nil
}

func (r *FeedRepo) Update(ctx context.Context, feed *domain.Feed) error {
	settings, err := json.Marshal(feed.Settings)
	if err != nil {
		return errors.Wrap(err, "error marshaling feed settings json data")
	}

	queryBuilder := r.db.squirrel.
		Update("feed").
		Set("name", feed.Name).
		Set("type", feed.Type).
		Set("enabled", feed.Enabled).
		Set("url", feed.URL).
		Set("interval", feed.Interval).
		Set("timeout", feed.Timeout).
		Set("max_age", feed.MaxAge).
		Set("api_key", feed.ApiKey).
		Set("cookie", feed.Cookie).
		Set("settings", settings).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": feed.ID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.handler.ExecContext(ctx, query, args...)
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

func (r *FeedRepo) UpdateLastRun(ctx context.Context, feedID int64) error {
	queryBuilder := r.db.squirrel.
		Update("feed").
		Set("last_run", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": feedID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.handler.ExecContext(ctx, query, args...)
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

func (r *FeedRepo) UpdateLastRunWithData(ctx context.Context, feedID int64, data string) error {
	queryBuilder := r.db.squirrel.
		Update("feed").
		Set("last_run", sq.Expr("CURRENT_TIMESTAMP")).
		Set("last_run_data", data).
		Where(sq.Eq{"id": feedID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.handler.ExecContext(ctx, query, args...)
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

func (r *FeedRepo) ToggleEnabled(ctx context.Context, id int64, enabled bool) error {
	var err error

	queryBuilder := r.db.squirrel.
		Update("feed").
		Set("enabled", enabled).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}
	result, err := r.db.handler.ExecContext(ctx, query, args...)
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

func (r *FeedRepo) Delete(ctx context.Context, id int64) error {
	queryBuilder := r.db.squirrel.
		Delete("feed").
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return errors.Wrap(err, "error getting rows affected")
	} else if rowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	r.log.Debug().Msgf("feed.delete: successfully deleted: %v", id)

	return nil
}
