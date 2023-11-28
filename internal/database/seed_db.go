package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

func OpenDB(dbPath string, dbType string) (*sql.DB, error) {
	db, err := sql.Open(dbType, dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s database: %v", dbType, err)
	}
	return db, nil
}

func ResetDB(dbPath string) error {
	db, err := OpenDB(dbPath, "sqlite")
	if err != nil {
		return err
	}
	defer db.Close()

	tables := GetTables()

	for _, table := range tables {
		if err := resetTable(db, table); err != nil {
			return err
		}
	}

	return nil
}

func resetTable(db *sql.DB, table string) error {
	if _, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
		return fmt.Errorf("failed to delete rows from table %s: %v", table, err)
	}

	// Update sqlite_sequence, ignore errors for missing sqlite_sequence entry
	if _, err := db.Exec(fmt.Sprintf("UPDATE sqlite_sequence SET seq = 0 WHERE name = '%s'", table)); err != nil && !strings.Contains(err.Error(), "no such table") {
		return fmt.Errorf("failed to reset primary key sequence for table %s: %v", table, err)
	}

	return nil
}

func SeedDB(dbPath string, seedDBPath string) error {
	sqlFile, err := os.ReadFile(seedDBPath)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %v", err)
	}

	db, err := OpenDB(dbPath, "sqlite")
	if err != nil {
		return err
	}
	defer db.Close()

	sqlCommands := strings.Split(string(sqlFile), ";")
	for _, cmd := range sqlCommands {
		if err := executeCommand(db, cmd); err != nil {
			return err
		}
	}

	return nil
}

func executeCommand(db *sql.DB, cmd string) error {
	if _, err := db.Exec(cmd); err != nil {
		return fmt.Errorf("failed to execute SQL command: %v", err)
	}
	return nil
}
