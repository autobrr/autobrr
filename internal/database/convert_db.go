package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

var tables = []string{
	"users", "indexer", "irc_network", "irc_channel", "client", "filter",
	"filter_external", "action", "notification", "filter_indexer", "release",
	"release_action_status", "feed", "api_key",
}

func Migrate(sqliteDBPath, postgresDBURL string) error {
	startTime := time.Now()

	sqliteDB, err := sql.Open("sqlite3", sqliteDBPath)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite database: %v", err)
	}
	defer sqliteDB.Close()

	postgresDB, err := sql.Open("postgres", postgresDBURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL database: %v", err)
	}
	defer postgresDB.Close()

	tables := GetTables()

	// Store all foreign key violation messages.
	var allFKViolations []string
	for _, table := range tables {
		fkViolations := MigrateTable(sqliteDB, postgresDB, table)
		allFKViolations = append(allFKViolations, fkViolations...)
	}
	fmt.Println("Migration completed successfully!")
	fmt.Printf("Elapsed time: %s\n", time.Since(startTime))
	if len(allFKViolations) > 0 {
		fmt.Println("\nSummary of Foreign Key Violations:")
		fmt.Println()
		for _, msg := range allFKViolations {
			fmt.Println(msg)
		}
		fmt.Println()
		fmt.Println("These are due to missing references, likely because the related item in another table no longer exists.")
	}
	return err
}

func GetTables() []string {
	return append([]string(nil), tables...)
}

func MigrateTable(sqliteDB, postgresDB *sql.DB, table string) []string {
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
	log.Printf("Migrated %d rows to table '%s' from SQLite to PostgreSQL\n", rowsAffected, table)
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
