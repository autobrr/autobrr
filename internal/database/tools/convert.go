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
	"action",
	"api_key",
	"client",
	"feed",
	"filter",
	"filter_external",
	"filter_indexer",
	"indexer",
	"irc_channel",
	"irc_network",
	"notification",
	"release",
	"release_action_status",
	"users",
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

	// Store all foreign key violation messages.
	var allFKViolations []string
	for _, table := range tables {
		fkViolations := c.migrateTable(sqliteDB, postgresDB, table)
		allFKViolations = append(allFKViolations, fkViolations...)
	}

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

	rows, err := sqliteDB.Query("SELECT * FROM ?", table)
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
