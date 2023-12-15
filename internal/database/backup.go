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

		if err := databaseConsistent(tx); err != nil {
			return err
		}
		defer tx.Commit()

		if exists, err := backupDatabase(base, db.handler); err != nil {
			if exists {
				return err
			}

			return nil
		}

		retain := 1
		if !shuttingDown {
			retain++
		}

		return cleanupDatabase(base, db, retain)
	}

	return errors.New("backup: not implemented for database type: %s", db.Driver)
}

func databaseConsistent(tx *sql.Tx) error {
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

func backupDatabase(base string, db *sql.DB) (bool, error) {
	path := filepath.Join(base, fmt.Sprintf("autobrr.db.backup.%d", time.Now().Unix()))
	if _, err := os.Stat(path); err == nil {
		return true, errors.New("backup creation failed, already exists %q", path)
	}

	row := db.QueryRow("VACUUM INTO $1", path)
	if err := row.Scan(); err != nil && err != sql.ErrNoRows {
		return false, errors.Wrap(err, "backup vacuum failed")
	}

	return false, nil
}

func cleanupDatabase(base string, db *DB, retain int) error {
	files, err := os.ReadDir(base)
	if err != nil {
		return errors.Wrap(err, "backup unable to open base for cleaning %q", base)
	}

	de := make([]int64, 0)
	for _, f := range files {
		if !strings.HasPrefix(f.Name(), "autobrr.db.backup.") {
			continue
		}

		strNum := strings.TrimPrefix(f.Name(), "autobrr.db.backup.")
		i, err := strconv.ParseInt(strNum, 10, 64)
		if err != nil {
			db.log.Err(err).Msgf("backup fatal number parsing on %q", f.Name())
			continue
		}

		de = append(de, i)
	}

	sort.SliceStable(de, func(i, j int) bool { return de[i] < de[j] })
	tu := time.Now().Unix()
	for i := len(de) - 1; i > 0 && de[i] > tu; i-- {
		os.Remove(filepath.Join(base, "autobrr.db.backup."+fmt.Sprintf("%d", de[i])))
		de = de[:i]
	}

	for i := 0; i < len(de)-retain; i++ {
		os.Remove(filepath.Join(base, "autobrr.db.backup."+fmt.Sprintf("%d", de[i])))
	}

	return nil
}
