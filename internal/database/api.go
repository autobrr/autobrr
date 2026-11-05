// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

func NewAPIRepo(log zerolog.Logger, db *DB) *APIRepo {
	return &APIRepo{
		log: log.With().Str("repo", "api").Logger(),
		db:  db,
	}
}

type APIRepo struct {
	log zerolog.Logger
	db  *DB
}

func (r *APIRepo) Store(ctx context.Context, key *domain.APIKey) error {
	queryBuilder := r.db.squirrel.
		Insert("api_key").
		Columns(
			"name",
			"key_hash",
			"scopes",
		).
		Values(
			key.Name,
			key.KeyHash,
			pq.Array(key.Scopes),
		).
		Suffix("RETURNING id, created_at").RunWith(r.db.Handler)

	if err := queryBuilder.QueryRowContext(ctx).Scan(&key.ID, &key.CreatedAt); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *APIRepo) Update(ctx context.Context, key *domain.APIKey) error {
	queryBuilder := r.db.squirrel.
		Update("api_key").
		Set("name", key.Name).
		Set("scopes", pq.Array(key.Scopes)).
		Where(sq.Eq{"id": key.ID}).
		Suffix("RETURNING key_hash, created_at").RunWith(r.db.Handler)

	if err := queryBuilder.QueryRowContext(ctx).Scan(&key.KeyHash, &key.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrRecordNotFound
		}

		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *APIRepo) Delete(ctx context.Context, id int) error {
	queryBuilder := r.db.squirrel.Delete("api_key").Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Int("api_key_id", id).Msg("successfully deleted")

	return nil
}

func (r *APIRepo) List(ctx context.Context) ([]domain.APIKey, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "name", "key_hash", "scopes", "created_at").
		From("api_key").
		OrderBy("id")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	keys := make([]domain.APIKey, 0)
	for rows.Next() {
		var a domain.APIKey

		var name sql.NullString

		if err := rows.Scan(&a.ID, &name, &a.KeyHash, pq.Array(&a.Scopes), &a.CreatedAt); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		a.Name = name.String

		keys = append(keys, a)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error rows")
	}

	return keys, nil
}

func (r *APIRepo) FindByID(ctx context.Context, id int) (*domain.APIKey, error) {
	return r.findOne(ctx, sq.Eq{"id": id})
}

func (r *APIRepo) FindByKeyHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	return r.findOne(ctx, sq.Eq{"key_hash": keyHash})
}

func (r *APIRepo) findOne(ctx context.Context, where sq.Eq) (*domain.APIKey, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "name", "key_hash", "scopes", "created_at").
		From("api_key").
		Where(where)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.Handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var apiKey domain.APIKey

	var name sql.NullString

	if err := row.Scan(&apiKey.ID, &name, &apiKey.KeyHash, pq.Array(&apiKey.Scopes), &apiKey.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	apiKey.Name = name.String

	return &apiKey, nil
}
