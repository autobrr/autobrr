// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package tools

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var tables = []string{
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
	"feed",
	"feed_cache",
	"api_key",
	"list",
	"list_filter",
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

type Converter interface {
	Convert() error
}

type SqliteToPostgresConverter struct {
	sqliteDBPath, postgresDBURL string
}

func NewConverter(sqliteDBPath, postgresDBURL string) Converter {
	return &SqliteToPostgresConverter{
		sqliteDBPath:  sqliteDBPath,
		postgresDBURL: postgresDBURL,
	}
}

func (c *SqliteToPostgresConverter) Convert() error {
	startTime := time.Now()

	sqliteDB, err := sql.Open("sqlite", c.sqliteDBPath)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite database: %v", err)
	}
	defer sqliteDB.Close()

	postgresDB, err := sql.Open("postgres", c.postgresDBURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL database: %v", err)
	}
	defer postgresDB.Close()

	tables := GetTables()

	applyFixups(sqliteDB, sqliteFixups)

	// Store all foreign key violation messages.
	var allFKViolations []string
	for _, table := range tables {
		fkViolations := c.migrateTable(sqliteDB, postgresDB, table)
		allFKViolations = append(allFKViolations, fkViolations...)
	}

	applyFixups(postgresDB, postgresFixups)

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
	return append([]string(nil), tables...)
}

func (c *SqliteToPostgresConverter) migrateTable(sqliteDB, postgresDB *sql.DB, table string) []string {
	var fkViolationMessages []string

	rows, err := sqliteDB.Query(fmt.Sprintf("SELECT * FROM %s", table))
	if err != nil {
		log.Fatalf("Failed to query SQLite table '%s': %v", table, err)
	}
	defer rows.Close()

	columns, err := rows.ColumnTypes()
	if err != nil {
		log.Fatalf("Failed to get column types for table '%s': %v", table, err)
	}

	// Prepare the INSERT statement for PostgreSQL.
	colNames, colPlaceholders := prepareColumns(columns)
	insertStmt, err := postgresDB.Prepare(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, colNames, colPlaceholders))
	if err != nil {
		log.Fatalf("Failed to prepare INSERT statement for table '%s': %v", table, err)
	}
	defer insertStmt.Close()

	var rowsAffected int64

	for rows.Next() {
		values, valuePtrs := prepareValues(columns)

		if err := rows.Scan(valuePtrs...); err != nil {
			log.Fatalf("Failed to scan row from SQLite table '%s': %v", table, err)
		}

		_, err := insertStmt.Exec(values...)
		if err != nil {
			if isForeignKeyViolation(err) {
				// Record foreign key violation message.
				message := fmt.Sprintf("Table '%s': %v", table, err)
				fkViolationMessages = append(fkViolationMessages, message)
				continue
			}
		} else {
			rowsAffected++
		}
	}
	log.Printf("Converted %d rows to table '%s' from SQLite to PostgreSQL\n", rowsAffected, table)
	return fkViolationMessages
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

func applyFixups(sql *sql.DB, stmts []string) {
	for _, stmt := range stmts {
		_, err := sql.Exec(stmt)
		if err != nil {
			log.Printf("Failed to apply fixup %s: %v", stmt, err)
		}
	}
}
