// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog"
)

type IndexerRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewIndexerRepo(log zerolog.Logger, db *DB) *IndexerRepo {
	return &IndexerRepo{
		log: log.With().Str("module", "database").Str("repo", "indexer").Logger(),
		db:  db,
	}
}

func (r *IndexerRepo) Store(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error) {
	settings, err := json.Marshal(indexer.Settings)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling json data")
	}

	queryBuilder := r.db.squirrel.
		Insert("indexer").Columns("enabled", "name", "identifier", "identifier_external", "implementation", "base_url", "use_proxy", "proxy_id", "settings").
		Values(indexer.Enabled, indexer.Name, indexer.Identifier, indexer.IdentifierExternal, indexer.Implementation, indexer.BaseURL, indexer.UseProxy, toNullInt64(indexer.ProxyID), settings).
		Suffix("RETURNING id").RunWith(r.db.Handler)

	// return values
	err = queryBuilder.QueryRowContext(ctx).Scan(&indexer.ID)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	return &indexer, nil
}

func (r *IndexerRepo) Update(ctx context.Context, indexer *domain.Indexer) error {
	settings, err := json.Marshal(indexer.Settings)
	if err != nil {
		return errors.Wrap(err, "error marshaling json data")
	}

	queryBuilder := r.db.squirrel.
		Update("indexer").
		Set("enabled", indexer.Enabled).
		Set("name", indexer.Name).
		Set("identifier_external", indexer.IdentifierExternal).
		Set("base_url", indexer.BaseURL).
		Set("use_proxy", indexer.UseProxy).
		Set("proxy_id", toNullInt64(indexer.ProxyID)).
		Set("settings", settings).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": indexer.ID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error rows affected")
	}

	if rowsAffected == 0 {
		return domain.ErrUpdateFailed
	}

	return nil
}

func (r *IndexerRepo) List(ctx context.Context) ([]domain.Indexer, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "identifier", "identifier_external", "implementation", "base_url", "use_proxy", "proxy_id", "settings", "archived", "archived_at").
		From("indexer").
		OrderBy("name ASC")

	query, _, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	indexers := make([]domain.Indexer, 0)

	for rows.Next() {
		var i domain.Indexer

		var identifierExternal, implementation, baseURL sql.Null[string]
		var proxyID sql.Null[int64]
		var archivedAt sql.NullTime
		var settings string
		var settingsMap map[string]string

		if err := rows.Scan(&i.ID, &i.Enabled, &i.Name, &i.Identifier, &identifierExternal, &implementation, &baseURL, &i.UseProxy, &proxyID, &settings, &i.Archived, &archivedAt); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		i.IdentifierExternal = identifierExternal.V
		i.Implementation = domain.IndexerImplementation(implementation.V)
		i.BaseURL = baseURL.V
		i.ProxyID = proxyID.V
		if archivedAt.Valid {
			t := archivedAt.Time
			i.ArchivedAt = &t
		}

		if err = json.Unmarshal([]byte(settings), &settingsMap); err != nil {
			return nil, errors.Wrap(err, "error unmarshal settings")
		}

		i.Settings = settingsMap

		indexers = append(indexers, i)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error rows")
	}

	return indexers, nil
}

func (r *IndexerRepo) FindByID(ctx context.Context, id int) (*domain.Indexer, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "identifier", "identifier_external", "implementation", "base_url", "use_proxy", "proxy_id", "settings", "archived", "archived_at").
		From("indexer").
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.Handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var i domain.Indexer

	var identifierExternal, implementation, baseURL, settings sql.Null[string]
	var proxyID sql.Null[int64]
	var archivedAt sql.NullTime

	if err := row.Scan(&i.ID, &i.Enabled, &i.Name, &i.Identifier, &identifierExternal, &implementation, &baseURL, &i.UseProxy, &proxyID, &settings, &i.Archived, &archivedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	i.IdentifierExternal = identifierExternal.V
	i.Implementation = domain.IndexerImplementation(implementation.V)
	i.BaseURL = baseURL.V
	i.ProxyID = proxyID.V
	if archivedAt.Valid {
		t := archivedAt.Time
		i.ArchivedAt = &t
	}

	var settingsMap map[string]string
	if err = json.Unmarshal([]byte(settings.V), &settingsMap); err != nil {
		return nil, errors.Wrap(err, "error unmarshal settings")
	}

	i.Settings = settingsMap

	return &i, nil
}

func (r *IndexerRepo) GetBy(ctx context.Context, req domain.GetIndexerRequest) (*domain.Indexer, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "identifier", "identifier_external", "implementation", "base_url", "use_proxy", "proxy_id", "settings", "archived", "archived_at").
		From("indexer")

	if req.ID > 0 {
		queryBuilder = queryBuilder.Where(sq.Eq{"id": req.ID})
	} else if req.Name != "" {
		queryBuilder = queryBuilder.Where(sq.Eq{"name": req.Name})
	} else if req.Identifier != "" {
		queryBuilder = queryBuilder.Where(sq.Eq{"identifier": req.Identifier})
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.Handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var i domain.Indexer

	var identifierExternal, implementation, baseURL, settings sql.Null[string]
	var proxyID sql.Null[int64]
	var archivedAt sql.NullTime

	if err := row.Scan(&i.ID, &i.Enabled, &i.Name, &i.Identifier, &identifierExternal, &implementation, &baseURL, &i.UseProxy, &proxyID, &settings, &i.Archived, &archivedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	i.IdentifierExternal = identifierExternal.V
	i.Implementation = domain.IndexerImplementation(implementation.V)
	i.BaseURL = baseURL.V
	i.ProxyID = proxyID.V
	if archivedAt.Valid {
		t := archivedAt.Time
		i.ArchivedAt = &t
	}

	var settingsMap map[string]string
	if err = json.Unmarshal([]byte(settings.V), &settingsMap); err != nil {
		return nil, errors.Wrap(err, "error unmarshal settings")
	}

	i.Settings = settingsMap

	return &i, nil
}

func (r *IndexerRepo) FindByFilterID(ctx context.Context, id int) ([]domain.Indexer, error) {
	queryBuilder := r.db.squirrel.
		Select("indexer.id", "enabled", "name", "identifier", "identifier_external", "base_url", "use_proxy", "proxy_id", "settings", "archived", "archived_at").
		From("indexer").
		Join("filter_indexer ON indexer.id = filter_indexer.indexer_id").
		Where(sq.Eq{"filter_indexer.filter_id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	indexers := make([]domain.Indexer, 0)
	for rows.Next() {
		var i domain.Indexer

		var settings string
		var settingsMap map[string]string
		var identifierExternal, baseURL sql.Null[string]
		var proxyID sql.Null[int64]
		var archivedAt sql.NullTime

		if err := rows.Scan(&i.ID, &i.Enabled, &i.Name, &i.Identifier, &identifierExternal, &baseURL, &i.UseProxy, &proxyID, &settings, &i.Archived, &archivedAt); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		if err = json.Unmarshal([]byte(settings), &settingsMap); err != nil {
			return nil, errors.Wrap(err, "error unmarshal settings")
		}

		i.IdentifierExternal = identifierExternal.V
		i.BaseURL = baseURL.V
		i.ProxyID = proxyID.V
		if archivedAt.Valid {
			t := archivedAt.Time
			i.ArchivedAt = &t
		}
		i.Settings = settingsMap

		indexers = append(indexers, i)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error rows")
	}

	return indexers, nil

}

func (r *IndexerRepo) Delete(ctx context.Context, id int) error {
	queryBuilder := r.db.squirrel.
		Delete("indexer").
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error rows affected")
	}

	if rowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	r.log.Debug().Str("method", "delete").Int("indexer_id", id).Msg("successfully deleted indexer")

	return nil
}

func (r *IndexerRepo) ToggleEnabled(ctx context.Context, indexerID int, enabled bool) error {
	queryBuilder := r.db.squirrel.
		Update("indexer").
		Set("enabled", enabled).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": indexerID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error rows affected")
	}

	if rowsAffected == 0 {
		return domain.ErrUpdateFailed
	}

	return nil
}

// ArchiveByIdentifier marks an orphaned indexer row as archived (a tombstone). Idempotent:
// only touches rows that are not already archived, and never overwrites an existing
// archived_at stamp.
func (r *IndexerRepo) ArchiveByIdentifier(ctx context.Context, identifier string) error {
	queryBuilder := r.db.squirrel.
		Update("indexer").
		Set("archived", true).
		Set("archived_at", sq.Expr("COALESCE(archived_at, CURRENT_TIMESTAMP)")).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"identifier": identifier}).
		Where(sq.Eq{"archived": false})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err := r.db.Handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

// UnarchiveByIdentifier clears the archived flag, e.g. when a removed indexer's definition
// comes back (a re-added custom definition). Idempotent.
func (r *IndexerRepo) UnarchiveByIdentifier(ctx context.Context, identifier string) error {
	queryBuilder := r.db.squirrel.
		Update("indexer").
		Set("archived", false).
		Set("archived_at", nil).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"identifier": identifier}).
		Where(sq.Eq{"archived": true})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err := r.db.Handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

// UpsertDeprecation writes (or refreshes) the metadata for a deprecated indexer, keyed by
// identifier so it survives even when the indexer row was hard-deleted.
func (r *IndexerRepo) UpsertDeprecation(ctx context.Context, d domain.IndexerDeprecation) error {
	queryBuilder := r.db.squirrel.
		Insert("indexer_deprecation").
		Columns("identifier", "name", "reason", "issue_url", "alias_of", "deprecated_at").
		Values(d.Identifier, d.Name, d.Reason, d.IssueURL, toNullString(d.AliasOf), d.DeprecatedAt.Format(time.RFC3339)).
		Suffix("ON CONFLICT (identifier) DO UPDATE SET name = EXCLUDED.name, reason = EXCLUDED.reason, issue_url = EXCLUDED.issue_url, alias_of = EXCLUDED.alias_of, deprecated_at = EXCLUDED.deprecated_at")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err := r.db.Handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

// ListDeprecations returns all known indexer deprecations, with a count of how many filters
// still reference each one (its cleanup blast radius).
func (r *IndexerRepo) ListDeprecations(ctx context.Context) ([]domain.IndexerDeprecation, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"d.identifier",
			"d.name",
			"d.reason",
			"d.issue_url",
			"d.alias_of",
			"d.deprecated_at",
			"(SELECT COUNT(*) FROM filter_indexer fi JOIN indexer i ON fi.indexer_id = i.id WHERE i.identifier = d.identifier) AS filter_count",
		).
		From("indexer_deprecation d").
		OrderBy("d.name ASC")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	deprecations := make([]domain.IndexerDeprecation, 0)
	for rows.Next() {
		var d domain.IndexerDeprecation
		var name, reason, issueURL, aliasOf sql.Null[string]
		var deprecatedAt sql.NullTime

		if err := rows.Scan(&d.Identifier, &name, &reason, &issueURL, &aliasOf, &deprecatedAt, &d.FilterCount); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		d.Name = name.V
		d.Reason = reason.V
		d.IssueURL = issueURL.V
		d.AliasOf = aliasOf.V
		if deprecatedAt.Valid {
			d.DeprecatedAt = deprecatedAt.Time
		}

		deprecations = append(deprecations, d)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error rows")
	}

	return deprecations, nil
}
