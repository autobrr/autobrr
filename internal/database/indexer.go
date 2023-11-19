// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog"
)

type IndexerRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewIndexerRepo(log logger.Logger, db *DB) domain.IndexerRepo {
	return &IndexerRepo{
		log: log.With().Str("module", "database").Str("repo", "indexer").Logger(),
		db:  db,
	}
}

func (r *IndexerRepo) Store(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error) {
	settings, err := json.Marshal(indexer.Settings)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling json data")
	}

	queryBuilder := r.db.squirrel.
		Insert("indexer").Columns("enabled", "name", "identifier", "implementation", "base_url", "settings").
		Values(indexer.Enabled, indexer.Name, indexer.Identifier, indexer.Implementation, indexer.BaseURL, settings).
		Suffix("RETURNING id").RunWith(r.db.handler)

	// return values
	var retID int64

	if err = queryBuilder.QueryRowContext(ctx).Scan(&retID); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	indexer.ID = retID

	return &indexer, nil
}

func (r *IndexerRepo) Update(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error) {
	settings, err := json.Marshal(indexer.Settings)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling json data")
	}

	queryBuilder := r.db.squirrel.
		Update("indexer").
		Set("enabled", indexer.Enabled).
		Set("name", indexer.Name).
		Set("base_url", indexer.BaseURL).
		Set("settings", settings).
		Set("updated_at", time.Now().Format(time.RFC3339)).
		Where(sq.Eq{"id": indexer.ID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	if _, err = r.db.handler.ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	return &indexer, nil
}

func (r *IndexerRepo) List(ctx context.Context) ([]domain.Indexer, error) {
	rows, err := r.db.handler.QueryContext(ctx, "SELECT id, enabled, name, identifier, implementation, base_url, settings FROM indexer ORDER BY name ASC")
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	indexers := make([]domain.Indexer, 0)

	for rows.Next() {
		var f domain.Indexer

		var implementation, baseURL sql.NullString
		var settings string
		var settingsMap map[string]string

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, &f.Identifier, &implementation, &baseURL, &settings); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		f.Implementation = implementation.String
		f.BaseURL = baseURL.String

		if err = json.Unmarshal([]byte(settings), &settingsMap); err != nil {
			return nil, errors.Wrap(err, "error unmarshal settings")
		}

		f.Settings = settingsMap

		indexers = append(indexers, f)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error rows")
	}

	return indexers, nil
}

func (r *IndexerRepo) FindByID(ctx context.Context, id int) (*domain.Indexer, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "identifier", "implementation", "base_url", "settings").
		From("indexer").
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var i domain.Indexer

	var implementation, baseURL, settings sql.NullString

	if err := row.Scan(&i.ID, &i.Enabled, &i.Name, &i.Identifier, &implementation, &baseURL, &settings); err != nil {
		return nil, errors.Wrap(err, "error scanning row")
	}

	i.Implementation = implementation.String
	i.BaseURL = baseURL.String

	var settingsMap map[string]string
	if err = json.Unmarshal([]byte(settings.String), &settingsMap); err != nil {
		return nil, errors.Wrap(err, "error unmarshal settings")
	}

	i.Settings = settingsMap

	return &i, nil

}

func (r *IndexerRepo) FindByFilterID(ctx context.Context, id int) ([]domain.Indexer, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "identifier", "base_url", "settings").
		From("indexer").
		Join("filter_indexer ON indexer.id = filter_indexer.indexer_id").
		Where(sq.Eq{"filter_indexer.filter_id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	indexers := make([]domain.Indexer, 0)
	for rows.Next() {
		var f domain.Indexer

		var settings string
		var settingsMap map[string]string
		var baseURL sql.NullString

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, &f.Identifier, &baseURL, &settings); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		if err = json.Unmarshal([]byte(settings), &settingsMap); err != nil {
			return nil, errors.Wrap(err, "error unmarshal settings")
		}

		f.BaseURL = baseURL.String
		f.Settings = settingsMap

		indexers = append(indexers, f)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error rows")
	}

	return indexers, nil

}

func (r *IndexerRepo) Delete(ctx context.Context, id int) error {
	queryBuilder := r.db.squirrel.
		Delete("indexer").
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error rows affected")
	}

	if rows != 1 {
		return errors.New("error deleting row")
	}

	r.log.Debug().Str("method", "delete").Msgf("successfully deleted indexer with id %v", id)

	return nil
}

func (r *IndexerRepo) ToggleEnabled(ctx context.Context, indexerID int, enabled bool) error {
	var err error

	queryBuilder := r.db.squirrel.
		Update("indexer").
		Set("enabled", enabled).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": indexerID})

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
