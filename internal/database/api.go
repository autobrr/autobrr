// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
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
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

func NewAPIRepo(log logger.Logger, db *DB) domain.APIRepo {
	return &APIRepo{
		log: log.With().Str("repo", "api").Logger(),
		db:  db,
	}
}

type APIRepo struct {
	log   zerolog.Logger
	db    *DB
	cache map[string]domain.APIKey
}

func (r *APIRepo) Store(ctx context.Context, key *domain.APIKey) error {
	queryBuilder := r.db.squirrel.
		Insert("api_key").
		Columns(
			"name",
			"key",
			"scopes",
		).
		Values(
			key.Name,
			key.Key,
			pq.Array(key.Scopes),
		).
		Suffix("RETURNING created_at").RunWith(r.db.handler)

	var createdAt time.Time

	if err := queryBuilder.QueryRowContext(ctx).Scan(&createdAt); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	key.CreatedAt = createdAt

	return nil
}

func (r *APIRepo) Delete(ctx context.Context, key string) error {
	queryBuilder := r.db.squirrel.
		Delete("api_key").
		Where(sq.Eq{"key": key})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("successfully deleted: %v", key)

	return nil
}

func (r *APIRepo) GetKeys(ctx context.Context) ([]domain.APIKey, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"name",
			"key",
			"scopes",
			"created_at",
		).
		From("api_key")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	keys := make([]domain.APIKey, 0)
	for rows.Next() {
		var a domain.APIKey

		var name sql.NullString

		if err := rows.Scan(&name, &a.Key, pq.Array(&a.Scopes), &a.CreatedAt); err != nil {
			return nil, errors.Wrap(err, "error scanning row")

		}

		a.Name = name.String

		keys = append(keys, a)
	}

	return keys, nil
}
