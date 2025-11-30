// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog"
)

func NewReleaseCleanupJobRepo(log logger.Logger, db *DB) domain.ReleaseCleanupJobRepo {
	return &ReleaseCleanupJobRepo{
		log: log.With().Str("repo", "release_cleanup_job").Logger(),
		db:  db,
	}
}

type ReleaseCleanupJobRepo struct {
	log zerolog.Logger
	db  *DB
}

func (r *ReleaseCleanupJobRepo) List(ctx context.Context) ([]*domain.ReleaseCleanupJob, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"name",
			"enabled",
			"schedule",
			"older_than",
			"indexers",
			"statuses",
			"last_run",
			"last_run_status",
			"last_run_data",
			"created_at",
			"updated_at",
		).
		From("release_cleanup_job").
		OrderBy("name ASC")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	jobs := make([]*domain.ReleaseCleanupJob, 0)
	for rows.Next() {
		var job domain.ReleaseCleanupJob

		var indexers, statuses, lastRunStatus, lastRunData sql.NullString
		var lastRun sql.NullTime

		if err := rows.Scan(
			&job.ID,
			&job.Name,
			&job.Enabled,
			&job.Schedule,
			&job.OlderThan,
			&indexers,
			&statuses,
			&lastRun,
			&lastRunStatus,
			&lastRunData,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		job.Indexers = indexers.String
		job.Statuses = statuses.String
		job.LastRun = lastRun.Time
		job.LastRunStatus = domain.ReleaseCleanupStatus(lastRunStatus.String)
		job.LastRunData = lastRunData.String

		jobs = append(jobs, &job)
	}

	return jobs, nil
}

func (r *ReleaseCleanupJobRepo) FindByID(ctx context.Context, id int) (*domain.ReleaseCleanupJob, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"name",
			"enabled",
			"schedule",
			"older_than",
			"indexers",
			"statuses",
			"last_run",
			"last_run_status",
			"last_run_data",
			"created_at",
			"updated_at",
		).
		From("release_cleanup_job").
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.Handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var job domain.ReleaseCleanupJob

	var indexers, statuses, lastRunStatus, lastRunData sql.NullString
	var lastRun sql.NullTime

	if err := row.Scan(
		&job.ID,
		&job.Name,
		&job.Enabled,
		&job.Schedule,
		&job.OlderThan,
		&indexers,
		&statuses,
		&lastRun,
		&lastRunStatus,
		&lastRunData,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	job.Indexers = indexers.String
	job.Statuses = statuses.String
	job.LastRun = lastRun.Time
	job.LastRunStatus = domain.ReleaseCleanupStatus(lastRunStatus.String)
	job.LastRunData = lastRunData.String

	return &job, nil
}

func (r *ReleaseCleanupJobRepo) Store(ctx context.Context, job *domain.ReleaseCleanupJob) error {
	var indexers, statuses sql.NullString

	if job.Indexers != "" {
		indexers = sql.NullString{String: job.Indexers, Valid: true}
	}
	if job.Statuses != "" {
		statuses = sql.NullString{String: job.Statuses, Valid: true}
	}

	queryBuilder := r.db.squirrel.
		Insert("release_cleanup_job").
		Columns(
			"name",
			"enabled",
			"schedule",
			"older_than",
			"indexers",
			"statuses",
		).
		Values(
			job.Name,
			job.Enabled,
			job.Schedule,
			job.OlderThan,
			indexers,
			statuses,
		).
		Suffix("RETURNING id").RunWith(r.db.Handler)

	var retID int

	if err := queryBuilder.QueryRowContext(ctx).Scan(&retID); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	job.ID = retID

	return nil
}

func (r *ReleaseCleanupJobRepo) Update(ctx context.Context, job *domain.ReleaseCleanupJob) error {
	var indexers, statuses sql.NullString

	if job.Indexers != "" {
		indexers = sql.NullString{String: job.Indexers, Valid: true}
	}
	if job.Statuses != "" {
		statuses = sql.NullString{String: job.Statuses, Valid: true}
	}

	queryBuilder := r.db.squirrel.
		Update("release_cleanup_job").
		Set("name", job.Name).
		Set("enabled", job.Enabled).
		Set("schedule", job.Schedule).
		Set("older_than", job.OlderThan).
		Set("indexers", indexers).
		Set("statuses", statuses).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": job.ID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return errors.Wrap(err, "error getting rows affected")
	} else if rowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	return nil
}

func (r *ReleaseCleanupJobRepo) UpdateLastRun(ctx context.Context, job *domain.ReleaseCleanupJob) error {
	var lastRunStatus, lastRunData sql.NullString

	if job.LastRunStatus != "" {
		lastRunStatus = sql.NullString{String: string(job.LastRunStatus), Valid: true}
	}
	if job.LastRunData != "" {
		lastRunData = sql.NullString{String: job.LastRunData, Valid: true}
	}

	queryBuilder := r.db.squirrel.
		Update("release_cleanup_job").
		Set("last_run", job.LastRun).
		Set("last_run_status", lastRunStatus).
		Set("last_run_data", lastRunData).
		Where(sq.Eq{"id": job.ID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return errors.Wrap(err, "error getting rows affected")
	} else if rowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	return nil
}

func (r *ReleaseCleanupJobRepo) ToggleEnabled(ctx context.Context, id int, enabled bool) error {
	queryBuilder := r.db.squirrel.
		Update("release_cleanup_job").
		Set("enabled", enabled).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return errors.Wrap(err, "error getting rows affected")
	} else if rowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	return nil
}

func (r *ReleaseCleanupJobRepo) Delete(ctx context.Context, id int) error {
	queryBuilder := r.db.squirrel.
		Delete("release_cleanup_job").
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return errors.Wrap(err, "error getting rows affected")
	} else if rowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	r.log.Debug().Msgf("release_cleanup_job.delete: successfully deleted: %v", id)

	return nil
}
