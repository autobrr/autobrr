// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/migrator"
)

// Migration104 rewrites every SQLite timestamp to UTC so stored timestamps sort as text.
var Migration104 = &migrator.Migration{Name: "104_timestamps_to_utc", RunTx: migrateTimestampsToUTC}

// timeLayout is the layout the SQLite driver binds time.Time values with, set through the
// _time_format DSN parameter. Written in UTC it sorts as text.
const timeLayout = "2006-01-02 15:04:05.999999999-07:00"

// defaultTimeLayout is what DEFAULT CURRENT_TIMESTAMP writes: UTC without an offset.
const defaultTimeLayout = "2006-01-02 15:04:05"

// Older releases wrote timestamps in whatever layout the driver or the caller used:
// RFC3339 with the host offset, the driver's default time.Time.String() output, and a
// space separated layout with an offset from an earlier driver.
var legacyTimeLayouts = []string{
	time.RFC3339Nano,
	timeLayout,
	"2006-01-02 15:04:05.999999999 -0700 MST",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02",
}

var timestampColumns = []struct {
	table   string
	columns []string
}{
	{table: "users", columns: []string{"created_at", "updated_at"}},
	{table: "proxy", columns: []string{"created_at", "updated_at"}},
	{table: "notification", columns: []string{"created_at", "updated_at"}},
	{table: "indexer", columns: []string{"created_at", "updated_at", "archived_at"}},
	{table: "indexer_deprecation", columns: []string{"deprecated_at"}},
	{table: "irc_network", columns: []string{"connected_since", "created_at", "updated_at"}},
	{table: "filter", columns: []string{"created_at", "updated_at"}},
	{table: "filter_notification", columns: []string{"created_at"}},
	{table: "release", columns: []string{"timestamp"}},
	{table: "release_action_status", columns: []string{"timestamp"}},
	{table: "feed", columns: []string{"last_run", "created_at", "updated_at"}},
	{table: "feed_cache", columns: []string{"ttl"}},
	{table: "api_key", columns: []string{"created_at"}},
	{table: "list", columns: []string{"last_refresh_time", "created_at", "updated_at"}},
	{table: "release_cleanup_job", columns: []string{"last_run", "created_at", "updated_at"}},
	{table: "notification_inbox", columns: []string{"read_at", "created_at"}},
	{table: "matrix_network", columns: []string{"token_expires_at", "created_at", "updated_at"}},
	{table: "matrix_room", columns: []string{"created_at", "updated_at"}},
}

// parseTimestamp parses a stored SQLite timestamp and reports whether it has to be
// rewritten to sort correctly as UTC text.
func parseTimestamp(raw string) (time.Time, bool, bool) {
	if t, err := time.Parse(defaultTimeLayout, raw); err == nil {
		return t, false, true
	}

	// time.Time.String() appends the monotonic clock reading when it has one
	value, _, _ := strings.Cut(raw, " m=")

	for _, layout := range legacyTimeLayouts {
		t, err := time.Parse(layout, value)
		if err != nil {
			continue
		}

		return t, t.UTC().Format(timeLayout) != raw, true
	}

	return time.Time{}, false, false
}

// migrateTimestampsToUTC rewrites every stored timestamp to UTC in timeLayout.
// Values it cannot parse are left as they are.
func migrateTimestampsToUTC(tx *sql.Tx) error {
	type rewrite struct {
		rowID int64
		value string
	}

	for _, tc := range timestampColumns {
		for _, column := range tc.columns {
			rows, err := tx.Query(fmt.Sprintf(`SELECT rowid, CAST(%s AS TEXT) FROM "%s" WHERE %s IS NOT NULL`, column, tc.table, column))
			if err != nil {
				return errors.Wrap(err, "could not read %s.%s", tc.table, column)
			}

			var rewrites []rewrite

			for rows.Next() {
				var rowID int64
				var raw string

				if err := rows.Scan(&rowID, &raw); err != nil {
					rows.Close()
					return errors.Wrap(err, "could not scan %s.%s", tc.table, column)
				}

				if t, changed, ok := parseTimestamp(raw); ok && changed {
					rewrites = append(rewrites, rewrite{rowID: rowID, value: t.UTC().Format(timeLayout)})
				}
			}

			if err := rows.Err(); err != nil {
				rows.Close()
				return errors.Wrap(err, "could not read %s.%s", tc.table, column)
			}

			rows.Close()

			if len(rewrites) == 0 {
				continue
			}

			stmt, err := tx.Prepare(fmt.Sprintf(`UPDATE "%s" SET %s = ? WHERE rowid = ?`, tc.table, column))
			if err != nil {
				return errors.Wrap(err, "could not prepare update of %s.%s", tc.table, column)
			}

			for _, rw := range rewrites {
				if _, err := stmt.Exec(rw.value, rw.rowID); err != nil {
					stmt.Close()
					return errors.Wrap(err, "could not update %s.%s", tc.table, column)
				}
			}

			stmt.Close()
		}
	}

	return nil
}
