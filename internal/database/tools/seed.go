// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package tools

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

type Seeder interface {
	Reset() error
	Seed() error
}

type SQLiteSeeder struct {
	dbPath   string
	seedFile string
}

func NewSQLiteSeeder(dbPath, seedFile string) *SQLiteSeeder {
	return &SQLiteSeeder{
		dbPath:   dbPath,
		seedFile: seedFile,
	}
}

func (s *SQLiteSeeder) Reset() error {
	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open %s database: %v", "sqlite", err)
	}
	defer db.Close()

	tables := GetTables()

	for _, table := range tables {
		if err := s.resetTable(db, table); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLiteSeeder) resetTable(db *sql.DB, table string) error {
	if _, err := db.Exec("DELETE FROM ?", table); err != nil {
		return fmt.Errorf("failed to delete rows from table %s: %v", table, err)
	}

	// Update sqlite_sequence, ignore errors for missing sqlite_sequence entry
	if _, err := db.Exec("UPDATE sqlite_sequence SET seq = 0 WHERE name = ?", table); err != nil {
		if !strings.Contains(err.Error(), "no such table") {
			return fmt.Errorf("failed to reset primary key sequence for table %s: %v", table, err)
		}
	}

	return nil
}

func (s *SQLiteSeeder) Seed() error {
	sqlFile, err := os.ReadFile(s.seedFile)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %v", err)
	}

	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open %s database: %v", "sqlite", err)
	}
	defer db.Close()

	sqlCommands := strings.Split(string(sqlFile), ";")
	for _, cmd := range sqlCommands {
		if _, err := db.Exec(cmd); err != nil {
			return fmt.Errorf("failed to execute SQL command: %v", err)
		}
	}

	return nil
}
