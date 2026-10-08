// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package tools

import (
	"database/sql"
	"os"
	"strings"

	"github.com/autobrr/autobrr/pkg/errors"

	_ "modernc.org/sqlite"
)

type Seeder interface {
	Reset() error
	Seed() error
	ResetAndSeed() error
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

// Reset deletes all rows from the known tables in a single transaction.
func (s *SQLiteSeeder) Reset() error {
	return s.inTx(func(tx *sql.Tx) error {
		return resetTables(tx)
	})
}

// Seed executes the seed file in a single transaction.
func (s *SQLiteSeeder) Seed() error {
	stmts, err := s.readSeed()
	if err != nil {
		return err
	}

	return s.inTx(func(tx *sql.Tx) error {
		return execSeed(tx, stmts)
	})
}

// ResetAndSeed resets and reseeds in one transaction, so a seed that fails
// to apply leaves the existing rows untouched.
func (s *SQLiteSeeder) ResetAndSeed() error {
	stmts, err := s.readSeed()
	if err != nil {
		return err
	}

	return s.inTx(func(tx *sql.Tx) error {
		if err := resetTables(tx); err != nil {
			return err
		}

		return execSeed(tx, stmts)
	})
}

func (s *SQLiteSeeder) readSeed() ([]string, error) {
	data, err := os.ReadFile(s.seedFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read SQL file")
	}

	var stmts []string
	for stmt := range strings.SplitSeq(string(data), ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		stmts = append(stmts, stmt)
	}

	if len(stmts) == 0 {
		return nil, errors.New("seed file %s contains no SQL statements", s.seedFile)
	}

	return stmts, nil
}

func (s *SQLiteSeeder) inTx(fn func(tx *sql.Tx) error) error {
	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return errors.Wrap(err, "failed to open sqlite database")
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

func resetTables(tx *sql.Tx) error {
	for _, table := range GetTables() {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return errors.Wrap(err, "failed to delete rows from table %s", table)
		}

		// sqlite_sequence only exists once a table with AUTOINCREMENT has been written to
		if _, err := tx.Exec("UPDATE sqlite_sequence SET seq = 0 WHERE name = ?", table); err != nil {
			if !strings.Contains(err.Error(), "no such table") {
				return errors.Wrap(err, "failed to reset primary key sequence for table %s", table)
			}
		}
	}

	return nil
}

func execSeed(tx *sql.Tx, stmts []string) error {
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return errors.Wrap(err, "failed to execute SQL command")
		}
	}

	return nil
}
