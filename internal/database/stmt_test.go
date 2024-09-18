// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	sq "github.com/Masterminds/squirrel"
	_ "modernc.org/sqlite"
)

func Test_SQLiteStmt(t *testing.T) {
	cfg := &domain.Config{
		LogLevel:     "DISABLED",
		DatabaseType: "sqlite:memory",
	}

	logr := logger.New(cfg)

	db, err := NewDB(cfg, logr)
	if err != nil {
		t.Logf("Error opening sqlite: %#v", err)
		return
	}

	err = db.Open()
	if err != nil {
		t.Logf("Error opening sqlite: %#v", err)
		return
	}
	defer db.handler.Close()

	ctx := context.Background()
	query, err := db.Statement.ToStmt(ctx, "SELECT 1 WHERE 1=$1")
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

func GetMockAction() domain.Action {
	return domain.Action{
		Name:                     "randomAction",
		Type:                     domain.ActionTypeTest,
		Enabled:                  true,
		ExecCmd:                  "/home/user/Downloads/test.sh",
		ExecArgs:                 "WGET_URL",
		WatchFolder:              "/home/user/Downloads",
		Category:                 "HD, 720p",
		Tags:                     "P2P, x264",
		Label:                    "testLabel",
		SavePath:                 "/home/user/Downloads",
		Paused:                   false,
		IgnoreRules:              false,
		SkipHashCheck:            false,
		FirstLastPiecePrio:       false,
		ContentLayout:            domain.ActionContentLayoutOriginal,
		LimitUploadSpeed:         0,
		LimitDownloadSpeed:       0,
		LimitRatio:               0,
		LimitSeedTime:            0,
		ReAnnounceSkip:           false,
		ReAnnounceDelete:         false,
		ReAnnounceInterval:       0,
		ReAnnounceMaxAttempts:    0,
		WebhookHost:              "http://localhost:8080",
		WebhookType:              "test",
		WebhookMethod:            "POST",
		WebhookData:              "testData",
		WebhookHeaders:           []string{"testHeader"},
		ExternalDownloadClientID: 21,
		FilterID:                 1,
		ClientID:                 1,
	}
}

func BenchmarkSQLiteCache(b *testing.B) {
	cfg := &domain.Config{
		LogLevel:     "DISABLED",
		DatabaseType: "sqlite:memory",
	}

	logr := logger.New(cfg)

	db, err := NewDB(cfg, logr)
	if err != nil {
		b.Logf("Error opening sqlite: %#v", err)
		return
	}

	err = db.Open()
	if err != nil {
		b.Logf("Error opening sqlite: %#v", err)
		return
	}
	defer db.handler.Close()

	repo := NewActionRepo(logr, db, nil)

	ctx := context.Background()
	action := GetMockAction()
	if _, err := repo.Store(ctx, action); err != nil {
		b.Logf("fail: %q", err)
		return
	}

	queryBuilder := db.squirrel.
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
		Where(sq.Eq{"a.filter_id": 1})

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		query, args, err := db.Statement.ToSql(ctx, queryBuilder)
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
	cfg := &domain.Config{
		LogLevel:     "DISABLED",
		DatabaseType: "sqlite:memory",
	}

	logr := logger.New(cfg)

	db, err := NewDB(cfg, logr)
	if err != nil {
		b.Logf("Error opening sqlite: %#v", err)
		return
	}

	err = db.Open()
	if err != nil {
		b.Logf("Error opening sqlite: %#v", err)
		return
	}
	defer db.handler.Close()

	repo := NewActionRepo(logr, db, nil)

	ctx := context.Background()
	action := GetMockAction()
	if _, err := repo.Store(ctx, action); err != nil {
		b.Logf("fail: %q", err)
		return
	}

	queryBuilder := db.squirrel.
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

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		query, args, err := queryBuilder.ToSql()
		if err != nil {
			b.Logf("unable to get query: %#v", err)
			break
		}

		rows, _ := db.handler.QueryContext(ctx, query, args...)
		b.StopTimer()
		if rows != nil {
			rows.Close()
		}
	}
}
