// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package sqlite

import (
	"database/sql"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/migrator"
)

// Migration107 fills the duplicate check keys of grabbed releases.
var Migration107 = &migrator.Migration{Name: "107_release_backfill_duplicate_keys", RunTx: backfillDuplicateKeys}

const batchSize = 1000

// backfillDuplicateKeys computes name_key, title_key and sub_title_key for every release with
// a PUSH_APPROVED action, the only releases a duplicate check can match. The keys are Go
// normalizations with no SQL equivalent.
func backfillDuplicateKeys(tx *sql.Tx) error {
	selectQuery := `SELECT r.id, COALESCE(r.release_name, ''), COALESCE(r.title, ''), COALESCE(r.sub_title, '')
		FROM "release" r
		WHERE r.id > $1
		  AND EXISTS (SELECT 1 FROM release_action_status ras WHERE ras.release_id = r.id AND ras.status = 'PUSH_APPROVED')
		ORDER BY r.id
		LIMIT $2`

	stmt, err := tx.Prepare(`UPDATE "release" SET name_key = $1, title_key = $2, sub_title_key = $3 WHERE id = $4`)
	if err != nil {
		return errors.Wrap(err, "could not prepare release key update")
	}

	defer stmt.Close()

	type releaseKeys struct {
		id          int64
		nameKey     string
		titleKey    string
		subTitleKey string
	}

	var lastID int64

	for {
		rows, err := tx.Query(selectQuery, lastID, batchSize)
		if err != nil {
			return errors.Wrap(err, "could not read releases")
		}

		// lib/pq refuses an exec while rows are open on the same transaction
		batch := make([]releaseKeys, 0, batchSize)

		for rows.Next() {
			var rk releaseKeys
			var name, title, subTitle string

			if err := rows.Scan(&rk.id, &name, &title, &subTitle); err != nil {
				rows.Close()
				return errors.Wrap(err, "could not scan release")
			}

			rk.nameKey = domain.NameKey(name)
			rk.titleKey = domain.TitleKey(title)
			rk.subTitleKey = domain.TitleKey(subTitle)

			batch = append(batch, rk)
		}

		if err := rows.Err(); err != nil {
			rows.Close()
			return errors.Wrap(err, "could not read releases")
		}

		rows.Close()

		for _, rk := range batch {
			if _, err := stmt.Exec(rk.nameKey, rk.titleKey, rk.subTitleKey, rk.id); err != nil {
				return errors.Wrap(err, "could not update keys of release: %d", rk.id)
			}
		}

		if len(batch) < batchSize {
			return nil
		}

		lastID = batch[len(batch)-1].id
	}
}
