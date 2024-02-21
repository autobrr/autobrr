// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
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

type ProxyRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewProxyRepo(log logger.Logger, db *DB) domain.ProxyRepo {
	return &ProxyRepo{
		log: log.With().Str("repo", "proxy").Logger(),
		db:  db,
	}
}

func (r *ProxyRepo) Store(ctx context.Context, p *domain.Proxy) error {
	addr := toNullString(p.Addr)

	user := toNullString(p.User)
	pass := toNullString(p.Pass)

	queryBuilder := r.db.squirrel.
		Insert("proxy").
		Columns(
			"enabled",
			"name",
			"type",
			"addr",
			//"auth_type",
			"auth_user",
			"auth_pass",
			"timeout",
		).
		Values(
			p.Enabled,
			p.Name,
			p.Type,
			addr,
			user,
			pass,
			p.Timeout,
		).
		Suffix("RETURNING id").
		RunWith(r.db.handler)

	var retID int64
	err := queryBuilder.QueryRowContext(ctx).Scan(&retID)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	p.ID = retID

	return nil
}

func (r *ProxyRepo) Update(ctx context.Context, p *domain.Proxy) error {
	user := toNullString(p.User)
	pass := toNullString(p.Pass)

	queryBuilder := r.db.squirrel.
		Update("proxy").
		Set("enabled", p.Enabled).
		Set("name", p.Name).
		Set("addr", p.Addr).
		//Set("auth_type", p.AuthType).
		Set("auth_user", user).
		Set("auth_pass", pass).
		Set("timeout", p.Timeout).
		Set("updated_at", time.Now().Format(time.RFC3339)).
		Where(sq.Eq{"id": p.ID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	// update record
	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return err
}

func (r *ProxyRepo) List(ctx context.Context) ([]domain.Proxy, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"enabled",
			"name",
			"type",
			"addr",
			//"auth_type",
			"auth_user",
			"auth_pass",
			"timeout",
		).
		From("proxy").
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

	var proxies []domain.Proxy
	for rows.Next() {
		var proxy domain.Proxy

		var user, pass sql.NullString

		if err := rows.Scan(&proxy.ID, &proxy.Enabled, &proxy.Name, &proxy.Type, &proxy.Addr, &user, &pass, &proxy.Timeout); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		proxy.User = user.String
		proxy.Pass = pass.String

		proxies = append(proxies, proxy)
	}

	err = rows.Err()
	if err != nil {
		return nil, errors.Wrap(err, "error row")
	}

	return proxies, nil
}

func (r *ProxyRepo) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "error begin transaction")
	}

	defer tx.Rollback()

	queryBuilder := r.db.squirrel.
		Delete("proxy").
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "error commit deleting proxy")
	}

	return nil
}

func (r *ProxyRepo) FindByID(ctx context.Context, id int64) (*domain.Proxy, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"enabled",
			"name",
			"type",
			"addr",
			"auth_user",
			"auth_pass",
			"timeout",
		).
		From("proxy").
		OrderBy("name ASC").
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows := r.db.handler.QueryRowContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var proxy domain.Proxy

	var user, pass sql.NullString

	err = rows.Scan(&proxy.ID, &proxy.Enabled, &proxy.Name, &proxy.Type, &proxy.Addr, &user, &pass, &proxy.Timeout)
	if err != nil {
		return nil, errors.Wrap(err, "error scanning row")
	}

	proxy.User = user.String
	proxy.Pass = pass.String

	return &proxy, nil
}

func (r *ProxyRepo) ToggleEnabled(ctx context.Context, id int64, enabled bool) error {
	queryBuilder := r.db.squirrel.
		Update("proxy").
		Set("enabled", enabled).
		Set("updated_at", time.Now().Format(time.RFC3339)).
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	// update record
	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}
