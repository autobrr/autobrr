// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

var backupPrefix = "autobrr.db.backup."

func (db *DB) BackupDatabase(shuttingDown bool) error {
	if db.handler == nil {
		return errors.New("backup: invalid database handle")
	}

	switch db.Driver {
	case "sqlite":
		opts := &sql.TxOptions{
			Isolation: sql.LevelSnapshot,
			ReadOnly:  true,
		}

		base := "./"
		if idx := strings.LastIndexByte(db.DSN, byte('/')); idx != -1 {
			base = db.DSN[:idx]
		}

		tx, err := db.handler.BeginTx(context.Background(), opts)
		if err != nil {
			return errors.Wrap(err, "backup: Transaction creation failed")
		}

		defer tx.Commit()
		if err := databaseConsistentSQLite(tx); err != nil {
			return err
		}

		if exists, err := backupDatabaseSQLite(base, db.handler); err != nil {
			if !exists {
				return err
			}

			return nil
		}

		retain := 1
		if !shuttingDown {
			retain++
		}

		return cleanupDatabaseSQLite(base, db, retain)
	}

	return errors.New("backup: not implemented for database type: %s", db.Driver)
}

func databaseConsistentSQLite(tx *sql.Tx) error {
	row := tx.QueryRow("PRAGMA integrity_check;")

	var status string
	if err := row.Scan(&status); err != nil {
		return errors.Wrap(err, "backup integrity unexpected state")
	}

	if status != "ok" {
		return errors.New("backup integrity check failed: %q", status)
	}

	return nil
}

func backupDatabaseSQLite(base string, db *sql.DB) (bool, error) {
	path := filepath.Join(base, fmt.Sprintf("%s%d", backupPrefix, time.Now().Unix()))
	if _, err := os.Stat(path); err == nil {
		return true, errors.New("backup creation failed, already exists %q", path)
	}

	row := db.QueryRow("VACUUM INTO $1", path)
	if err := row.Scan(); err != nil && err != sql.ErrNoRows {
		return false, errors.Wrap(err, "backup vacuum failed")
	}

	return false, nil
}

func cleanupDatabaseSQLite(base string, db *DB, retain int) error {
	files, err := os.ReadDir(base)
	if err != nil {
		return errors.Wrap(err, "backup unable to open base for cleaning %q", base)
	}

	// Scan configuration directory for prefixes matching our backup schema.
	de := make([]int64, 0)
	for _, f := range files {
		if !strings.HasPrefix(f.Name(), backupPrefix) {
			continue
		}

		strNum := strings.TrimPrefix(f.Name(), backupPrefix)
		i, err := strconv.ParseInt(strNum, 10, 64)
		if err != nil {
			db.log.Err(err).Msgf("backup fatal number parsing on %q", f.Name())
			continue
		}

		de = append(de, i)
	}

	// Sort the slices by their respective timestamps.
	sort.SliceStable(de, func(i, j int) bool { return de[i] < de[j] })
	tu := time.Now().Unix()
	for i := len(de) - 1; i > 0 && de[i] > tu; i-- {
		os.Remove(filepath.Join(base, fmt.Sprintf("%s%d", backupPrefix, de[i])))
		de = de[:i]
	}

	// Remove older backups until they hit the specified retain threshold.
	for i := 0; i < len(de)-retain; i++ {
		os.Remove(filepath.Join(base, fmt.Sprintf("%s%d", backupPrefix, de[i])))
	}

	return nil
}
