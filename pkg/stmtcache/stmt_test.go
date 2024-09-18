// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package stmtcache

import (
	"context"
	"database/sql"
	"testing"

	sq "github.com/Masterminds/squirrel"

	_ "modernc.org/sqlite"
)

func Test_SQLiteStmt(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:")
	if err != nil {
		t.Logf("Error opening sqlite: %#v", err)
		return
	}

	defer db.Close()

	ctx := context.Background()
	query, err := ToStmt(ctx, db, "SELECT 1 WHERE 1=$1")
	if err != nil {
		t.Logf("fail: %q", err)
		t.Fail()
	}

	rows, err := query.QueryContext(ctx, 1)
	if err != nil {
		t.Logf("fail: %q", err)
		t.Fail()
	}

	defer rows.Close()
	for rows.Next() {
		t.Logf("Row!")
	}
}

func BenchmarkSQLiteCache(b *testing.B) {
	db, err := sql.Open("sqlite", "file::memory:")
	if err != nil {
		b.Logf("Error opening sqlite: %#v", err)
		return
	}

	defer db.Close()
	queryBuilder := sq.
		Select(
			"a.id",
			"a.name",
			"a.type",
			"a.enabled",
			"a.exec_cmd",
			"a.exec_args",
			"a.watch_folder",
			"a.category",
			"a.tags",
			"a.label",
			"a.save_path",
			"a.paused",
			"a.ignore_rules",
			"a.first_last_piece_prio",
			"a.skip_hash_check",
			"a.content_layout",
			"a.priority",
			"a.limit_download_speed",
			"a.limit_upload_speed",
			"a.limit_ratio",
			"a.limit_seed_time",
			"a.reannounce_skip",
			"a.reannounce_delete",
			"a.reannounce_interval",
			"a.reannounce_max_attempts",
			"a.webhook_host",
			"a.webhook_type",
			"a.webhook_method",
			"a.webhook_data",
			"a.external_client_id",
			"a.external_client",
			"a.client_id",
		).
		From("action a").
		Where(sq.Eq{"a.filter_id": 24})

	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		query, args, err := ToSql(ctx, db, queryBuilder)
		if err != nil {
			b.Logf("unable to get stmt: %#v", err)
			break
		}

		rows, _ := query.QueryContext(ctx, args...)
		b.StopTimer()
		if rows != nil {
			rows.Close()
		}
	}
}

func BenchmarkSQLite(b *testing.B) {
	db, err := sql.Open("sqlite", "file::memory:")
	if err != nil {
		b.Logf("Error opening sqlite: %#v", err)
		return
	}

	defer db.Close()
	queryBuilder := sq.
		Select(
			"a.id",
			"a.name",
			"a.type",
			"a.enabled",
			"a.exec_cmd",
			"a.exec_args",
			"a.watch_folder",
			"a.category",
			"a.tags",
			"a.label",
			"a.save_path",
			"a.paused",
			"a.ignore_rules",
			"a.first_last_piece_prio",
			"a.skip_hash_check",
			"a.content_layout",
			"a.priority",
			"a.limit_download_speed",
			"a.limit_upload_speed",
			"a.limit_ratio",
			"a.limit_seed_time",
			"a.reannounce_skip",
			"a.reannounce_delete",
			"a.reannounce_interval",
			"a.reannounce_max_attempts",
			"a.webhook_host",
			"a.webhook_type",
			"a.webhook_method",
			"a.webhook_data",
			"a.external_client_id",
			"a.external_client",
			"a.client_id",
		).
		From("action a").
		Where(sq.Eq{"a.filter_id": 24})

	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		query, args, err := queryBuilder.ToSql()
		if err != nil {
			b.Logf("unable to get query: %#v", err)
			break
		}

		rows, _ := db.QueryContext(ctx, query, args...)
		b.StopTimer()
		if rows != nil {
			rows.Close()
		}
	}
}
