// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jellydator/ttlcache/v3"
)

type StatementCache struct {
	db    *sql.DB
	cache *ttlcache.Cache[string, *sql.Stmt]
}

func NewStatementCache(db *sql.DB) *StatementCache {
	s := &StatementCache{
		db:    db,
		cache: ttlcache.New[string, *sql.Stmt](ttlcache.WithTTL[string, *sql.Stmt](5 * time.Minute)),
	}

	//s.cache.OnEviction(func(ctx context.Context, reason ttlcache.EvictionReason, item *ttlcache.Item[string, *sql.Stmt]) {
	//	if stmt := item.Value(); stmt != nil {
	//		stmt.Close()
	//	}
	//})

	// cache eviction needs to be started
	go s.cache.Start()

	return s
}

func (s *StatementCache) ToSql(ctx context.Context, queryBuilder any) (*sql.Stmt, []interface{}, error) {
	var query string
	var args []interface{}
	var err error

	switch queryBuilder.(type) {
	case sq.CaseBuilder:
		query, args, err = queryBuilder.(sq.CaseBuilder).ToSql()
	case sq.DeleteBuilder:
		query, args, err = queryBuilder.(sq.DeleteBuilder).ToSql()
	case sq.InsertBuilder:
		query, args, err = queryBuilder.(sq.InsertBuilder).ToSql()
	case sq.SelectBuilder:
		query, args, err = queryBuilder.(sq.SelectBuilder).ToSql()
	case sq.UpdateBuilder:
		query, args, err = queryBuilder.(sq.UpdateBuilder).ToSql()
	default:
		return nil, nil, fmt.Errorf("unimplemented type for ToSql: %T", queryBuilder)
	}

	item := s.cache.Get(query)
	if item != nil {
		return item.Value(), args, err
	}

	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, nil, err
	}

	s.cache.Set(query, stmt, ttlcache.DefaultTTL)

	return stmt, args, err
}

func (s *StatementCache) ToStmt(ctx context.Context, query string) (*sql.Stmt, error) {
	item := s.cache.Get(query)
	if item != nil {
		return item.Value(), nil
	}

	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}

	s.cache.Set(query, stmt, ttlcache.DefaultTTL)

	return stmt, nil
}
