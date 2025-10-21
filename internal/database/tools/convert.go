// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package tools

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/database/migrations"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	_ "modernc.org/sqlite"
)

var defaultTables = []string{
	"users",
	"proxy",
	"indexer",
	"irc_network",
	"irc_channel",
	"release_profile_duplicate",
	"filter",
	"filter_external",
	"filter_indexer",
	"client",
	"action",
	"release",
	"release_action_status",
	"notification",
	"filter_notification",
	"feed",
	"feed_cache",
	"api_key",
	"list",
	"list_filter",
	//"sessions",
	//"schema_migrations",
}

// lists of changes to make to the SQLite and Postgres DBs just before and after migration, respectively
var sqliteFixups = []string{
	// null out references to rows which have been deleted (only necessary for FKs with ON DELETE SET NULL)
	"UPDATE release SET filter_id = NULL WHERE filter_id NOT IN (SELECT id FROM filter)",
	"UPDATE release_action_status SET filter_id = NULL WHERE filter_id NOT IN (SELECT id FROM filter)",
	"UPDATE release_action_status SET action_id = NULL WHERE action_id NOT IN (SELECT id FROM action)",
	"UPDATE indexer SET proxy_id = NULL WHERE proxy_id NOT IN (SELECT id FROM proxy)",
	"UPDATE irc_network SET proxy_id = NULL WHERE proxy_id NOT IN (SELECT id FROM proxy)",
	"UPDATE filter SET release_profile_duplicate_id = NULL WHERE release_profile_duplicate_id NOT IN (SELECT id FROM release_profile_duplicate)",
	"UPDATE action SET client_id = NULL WHERE client_id NOT IN (SELECT id FROM client)",
	"UPDATE feed SET indexer_id = NULL WHERE indexer_id NOT IN (SELECT id FROM indexer)",
	"UPDATE list SET client_id = NULL WHERE client_id NOT IN (SELECT id FROM client)",
}

var postgresFixups = []string{
	// Update PK sequences to start after their most recent values instead of 1
	"SELECT setval('release_id_seq', (SELECT MAX(id) FROM release), true)",
	"SELECT setval('filter_id_seq', (SELECT MAX(id) FROM filter), true)",
	"SELECT setval('client_id_seq', (SELECT MAX(id) FROM client), true)",
	"SELECT setval('feed_id_seq', (SELECT MAX(id) FROM feed), true)",
	"SELECT setval('filter_external_id_seq', (SELECT MAX(id) FROM filter_external), true)",
	"SELECT setval('indexer_id_seq', (SELECT MAX(id) FROM indexer), true)",
	"SELECT setval('irc_channel_id_seq', (SELECT MAX(id) FROM irc_channel), true)",
	"SELECT setval('irc_network_id_seq', (SELECT MAX(id) FROM irc_network), true)",
	"SELECT setval('list_id_seq', (SELECT MAX(id) FROM list), true)",
	"SELECT setval('notification_id_seq', (SELECT MAX(id) FROM notification), true)",
	"SELECT setval('proxy_id_seq', (SELECT MAX(id) FROM proxy), true)",
	"SELECT setval('release_action_status_id_seq', (SELECT MAX(id) FROM release_action_status), true)",
	"SELECT setval('release_profile_duplicate_id_seq', (SELECT MAX(id) FROM release_profile_duplicate), true)",
	"SELECT setval('users_id_seq', (SELECT MAX(id) FROM users), true)",
}

type DBConverter interface {
	Convert(ctx context.Context, opts Opts) error
}

type Opts struct {
	ExcludeTables []string
	DryRun        bool
}

type SqliteToPostgresConverter struct {
	logger                      logger.Logger
	sqliteDBPath, postgresDBURL string
}

func NewConverter(logger logger.Logger, sqliteDBPath, postgresDBURL string) DBConverter {
	return &SqliteToPostgresConverter{
		logger:        logger,
		sqliteDBPath:  sqliteDBPath,
		postgresDBURL: postgresDBURL,
	}
}

func (c *SqliteToPostgresConverter) Convert(ctx context.Context, opts Opts) error {
	startTime := time.Now()

	sqliteDB, err := sql.Open("sqlite", c.sqliteDBPath)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to connect to SQLite database")
		return err
	}
	defer sqliteDB.Close()

	postgresDB, err := sql.Open("postgres", c.postgresDBURL)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to connect to PostgreSQL database")
		return err
	}
	defer postgresDB.Close()

	if err := postgresDB.PingContext(ctx); err != nil {
		c.logger.Error().Err(err).Msg("Failed to ping PostgreSQL database")
		return err
	}

	pgMigrations := migrations.PostgresMigrations(postgresDB, c.logger.With().Logger())
	if err = pgMigrations.Migrate(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to migrate PostgreSQL database")
		return err
	}

	tables := GetTables()

	c.applyFixups(ctx, sqliteDB, sqliteFixups)

	// Store all foreign key violation messages.
	var allFKViolations []string
	for _, table := range tables {
		if slices.Contains(opts.ExcludeTables, table) {
			c.logger.Info().Str("table", table).Msg("Skip ignored table")
			continue
		}
		fkViolations, err := c.migrateTable(ctx, sqliteDB, postgresDB, table, opts.DryRun)
		if err != nil {
			c.logger.Error().Err(err).Str("table", table).Msg("Failed to migrate table")
			continue
		}
		allFKViolations = append(allFKViolations, fkViolations...)
	}

	c.applyFixups(ctx, postgresDB, postgresFixups)

	c.printConversionResult(startTime, allFKViolations)

	return err
}

func (c *SqliteToPostgresConverter) printConversionResult(startTime time.Time, allFKViolations []string) {
	var sb strings.Builder

	sb.WriteString("Convert completed successfully!\n")
	sb.WriteString(fmt.Sprintf("Elapsed time: %s\n", time.Since(startTime)))
	if len(allFKViolations) > 0 {
		sb.WriteString("\nSummary of Foreign Key Violations:\n\n")
		for _, msg := range allFKViolations {
			sb.WriteString(" - " + msg + "\n")
		}
		sb.WriteString("\nThese are due to missing references, likely because the related item in another table no longer exists.\n")
	}
	fmt.Print(sb.String())
}

func GetTables() []string {
	return append([]string(nil), defaultTables...)
}

func (c *SqliteToPostgresConverter) migrateTable(ctx context.Context, sqliteDB, postgresDB *sql.DB, table string, dry bool) ([]string, error) {
	var fkViolationMessages []string

	var rowCount int64
	if err := sqliteDB.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&rowCount); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			c.logger.Error().Err(err).Str("table", table).Msg("Failed to query row count")
			return nil, errors.Wrap(err, "Failed to query row count for table '%s'", table)
		}
	}

	if rowCount == 0 {
		c.logger.Info().Str("table", table).Msg("Table is empty, skipping")
		return nil, nil
	}

	c.logger.Debug().Str("table", table).Int64("rows", rowCount).Msg("Converting table..")

	if dry {
		c.logger.Info().Str("table", table).Msg("Dry run, skipping table")
		return nil, nil
	}

	rows, err := sqliteDB.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %s", table))
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Wrap(err, "Failed to query SQLite table '%s'", table)
		}
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.ColumnTypes()
	if err != nil {
		c.logger.Error().Err(err).Str("table", table).Msg("Failed to get column types")
		return nil, errors.Wrap(err, "Failed to get column types for table '%s'", table)
	}

	colNames, _ := prepareColumns(columns)

	const batchSize = 1000
	c.logger.Debug().Str("table", table).Int("batchSize", batchSize).Int64("rows", rowCount).Int("total_batches", int(rowCount/batchSize)).Msg("total batches")
	var batch [][]interface{}
	var rowsAffected int64

	for rows.Next() {
		values, valuePtrs := prepareValues(columns)

		if err := rows.Scan(valuePtrs...); err != nil {
			c.logger.Error().Err(err).Str("table", table).Msg("Failed to scan row")
			return nil, errors.Wrap(err, "Failed to scan row from SQLite table '%s'", table)
		}

		batch = append(batch, values)

		// When batch is full, insert it
		if len(batch) >= batchSize {
			inserted, violations := c.insertBatch(ctx, postgresDB, table, colNames, columns, batch)
			rowsAffected += inserted
			c.logger.Debug().Str("table", table).Int64("rowsAffected", rowsAffected).Int64("total_rows", rowCount).Msg("rows affected")
			fkViolationMessages = append(fkViolationMessages, violations...)
			batch = batch[:0] // Reset batch
		}
	}

	// Insert any remaining rows in the final batch
	if len(batch) > 0 {
		inserted, violations := c.insertBatch(ctx, postgresDB, table, colNames, columns, batch)
		rowsAffected += inserted
		c.logger.Debug().Str("table", table).Int64("rowsAffected", rowsAffected).Int64("total_rows", rowCount).Msg("rows affected")
		fkViolationMessages = append(fkViolationMessages, violations...)
	}

	c.logger.Info().Msgf("Converted %d rows to table '%s' from SQLite to PostgreSQL", rowsAffected, table)

	return fkViolationMessages, nil
}

func (c *SqliteToPostgresConverter) insertBatch(ctx context.Context, db *sql.DB, table, colNames string, columns []*sql.ColumnType, batch [][]interface{}) (int64, []string) {
	if len(batch) == 0 {
		return 0, nil
	}

	var fkViolations []string

	// Build multi-row INSERT statement
	var placeholders []string
	var allValues []interface{}
	paramIndex := 1

	for _, rowValues := range batch {
		var rowPlaceholders []string
		for range columns {
			rowPlaceholders = append(rowPlaceholders, fmt.Sprintf("$%d", paramIndex))
			paramIndex++
		}
		placeholders = append(placeholders, fmt.Sprintf("(%s)", strings.Join(rowPlaceholders, ", ")))
		allValues = append(allValues, rowValues...)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", table, colNames, strings.Join(placeholders, ", "))

	_, err := db.Exec(query, allValues...)
	if err != nil {
		if isForeignKeyViolation(err) {
			// If batch insert fails due to FK violation, fall back to individual inserts
			// to identify which specific rows are problematic
			return c.insertBatchOneByOne(ctx, db, table, colNames, columns, batch)
		}
		c.logger.Error().Err(err).Str("table", table).Msg("Failed to insert batch into table")
	}

	return int64(len(batch)), fkViolations
}

func (c *SqliteToPostgresConverter) insertBatchOneByOne(ctx context.Context, db *sql.DB, table, colNames string, columns []*sql.ColumnType, batch [][]interface{}) (int64, []string) {
	var fkViolations []string
	var rowsAffected int64

	_, colPlaceholders := prepareColumns(columns)
	insertStmt, err := db.PrepareContext(ctx, fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, colNames, colPlaceholders))
	if err != nil {
		c.logger.Error().Err(err).Str("table", table).Msg("Failed to prepare INSERT statement")
		return 0, nil
	}
	defer insertStmt.Close()

	for _, values := range batch {
		_, err := insertStmt.Exec(values...)
		if err != nil {
			if isForeignKeyViolation(err) {
				message := fmt.Sprintf("Table '%s': %v", table, err)
				fkViolations = append(fkViolations, message)
				continue
			}

			c.logger.Error().Err(err).Str("table", table).Msg("Failed to insert row into table")
		}
		rowsAffected++
	}

	return rowsAffected, fkViolations
}

func prepareColumns(columns []*sql.ColumnType) (colNames, colPlaceholders string) {
	for i, col := range columns {
		colNames += col.Name()
		colPlaceholders += fmt.Sprintf("$%d", i+1)
		if i < len(columns)-1 {
			colNames += ", "
			colPlaceholders += ", "
		}
	}
	return
}

func prepareValues(columns []*sql.ColumnType) ([]interface{}, []interface{}) {
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}
	return values, valuePtrs
}

func isForeignKeyViolation(err error) bool {
	return strings.Contains(err.Error(), "violates foreign key constraint")
}

func (c *SqliteToPostgresConverter) applyFixups(ctx context.Context, sql *sql.DB, stmts []string) {
	for _, stmt := range stmts {
		_, err := sql.ExecContext(ctx, stmt)
		if err != nil {
			log.Printf("Failed to apply fixup %s: %v", stmt, err)
		}
	}
}
