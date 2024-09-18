// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package stmtcache

import (
	"context"
	"database/sql"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jellydator/ttlcache/v3"
)

var cache = ttlcache.New[string, *sql.Stmt](
	ttlcache.WithTTL[string, *sql.Stmt](5 * time.Minute),
)

func init() {
	cache.OnEviction(func(ctx context.Context, reason ttlcache.EvictionReason, item *ttlcache.Item[string, *sql.Stmt]) {
		if stmt := item.Value(); stmt != nil {
			stmt.Close()
		}
	})
}

func ToSql[T sq.CaseBuilder | sq.DeleteBuilder | sq.InsertBuilder | sq.SelectBuilder | sq.StatementBuilderType | sq.UpdateBuilder](ctx context.Context, db *sql.DB, queryBuilder T) (*sql.Stmt, []interface{}, error) {
	var abstract interface{}
	abstract = &queryBuilder // so fucking stupid this is a thing. was supposed to be fixed in 1.19.

	var query string
	var args []interface{}
	var err error

	switch abstract.(type) {
	case *sq.CaseBuilder:
		query, args, err = abstract.(*sq.CaseBuilder).ToSql()
	case *sq.DeleteBuilder:
		query, args, err = abstract.(*sq.DeleteBuilder).ToSql()
	case *sq.InsertBuilder:
		query, args, err = abstract.(*sq.InsertBuilder).ToSql()
	case *sq.SelectBuilder:
		query, args, err = abstract.(*sq.SelectBuilder).ToSql()
	case *sq.UpdateBuilder:
		query, args, err = abstract.(*sq.UpdateBuilder).ToSql()
	default:
		return nil, nil, errors.New("unimplemented type for ToSql")
	}

	item := cache.Get(query)
	if item == nil {
		stmt, err := db.PrepareContext(ctx, query)
		if err != nil {
			return nil, nil, err
		}

		item = cache.Set(query, stmt, ttlcache.DefaultTTL)
	}

	return item.Value(), args, err
}

func ToStmt(ctx context.Context, db *sql.DB, query string) (*sql.Stmt, error) {
	item := cache.Get(query)
	if item == nil {
		stmt, err := db.PrepareContext(ctx, query)
		if err != nil {
			return nil, err
		}

		item = cache.Set(query, stmt, ttlcache.DefaultTTL)
	}

	return item.Value(), nil
}
