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

type ListRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewListRepo(log logger.Logger, db *DB) domain.ListRepo {
	return &ListRepo{
		log: log.With().Str("repo", "list").Logger(),
		db:  db,
	}
}

func (r *ListRepo) List(ctx context.Context) ([]domain.List, error) {
	qb := r.db.squirrel.Select(
		"id",
		"name",
		"enabled",
		"type",
		"client_id",
		"url",
		"headers",
		"api_key",
		"filters",
		"match_release",
		"tags_include",
		"tags_exclude",
		"include_monitored",
		"last_refresh_time",
		"last_refresh_status",
		"last_refresh_error",
		"created_at",
		"updated_at",
	).
		From("list").
		OrderBy("name ASC")

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	lists := make([]domain.List, 0)
	for rows.Next() {
		var list domain.List

		// TODO handle null values
		err = rows.Scan(&list.ID, &list.Name, &list.Enabled, &list.Type, &list.ClientID, &list.URL, &list.Headers, &list.APIKey, &list.Filters, &list.MatchRelease, &list.TagsInclude, &list.TagsExclude, &list.IncludeUnmonitored, &list.LastRefreshTime, &list.LastRefreshStatus, &list.LastRefreshError, &list.CreatedAt, &list.UpdatedAt)
		if err != nil {
			return nil, err
		}

		lists = append(lists, list)
	}

	return lists, nil
}

func (r *ListRepo) FindByID(ctx context.Context, listID int64) (*domain.List, error) {
	qb := r.db.squirrel.Select(
		"id",
		"name",
		"enabled",
		"type",
		"client_id",
		"url",
		"headers",
		"api_key",
		"filters",
		"match_release",
		"tags_include",
		"tags_exclude",
		"include_monitored",
		"last_refresh_time",
		"last_refresh_status",
		"last_refresh_error",
		"created_at",
		"updated_at",
	).
		From("list").
		Where(sq.Eq{"id": listID})

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, err
	}

	var list domain.List

	err = row.Scan(&list.ID, &list.Name, &list.Enabled)
	if err != nil {
		return nil, err
	}

	return &list, nil
}

func (r *ListRepo) Store(ctx context.Context, list *domain.List) error {
	qb := r.db.squirrel.Insert("list").
		Columns(
			"name",
			"enabled",
			"type",
			"client_id",
			"url",
			"headers",
			"api_key",
			"filters",
			"match_release",
			"tags_include",
			"tags_exclude",
		).
		Values(
			list.Name,
			list.Enabled,
			list.Type,
			list.ClientID,
			list.URL,
			list.Headers,
			list.APIKey,
			list.Filters,
			list.MatchRelease,
			list.TagsInclude,
			list.TagsExclude,
		).Suffix("RETURNING id").RunWith(r.db.handler)

	//query, args, err := qb.ToSql()
	//if err != nil {
	//	return err
	//}

	if err := qb.QueryRowContext(ctx).Scan(&list.ID); err != nil {
		return err
	}

	//results, err := r.db.handler.ExecContext(ctx, query, args...)
	//if err != nil {
	//	return err
	//}

	return nil
}

func (r *ListRepo) Update(ctx context.Context, list *domain.List) error {
	qb := r.db.squirrel.Update("list").
		Set("name", list.Name).
		Set("enabled", list.Enabled).
		Set("type", list.Type).
		Set("client_id", list.ClientID).
		Set("url", list.URL).
		Set("headers", list.Headers).
		Set("api_key", list.APIKey).
		Set("filters", list.Filters).
		Set("match_release", list.MatchRelease).
		Set("tags_include", list.TagsInclude).
		Set("tags_exclude", list.TagsExclude).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": list.ID})

	query, args, err := qb.ToSql()
	if err != nil {
		return err
	}

	results, err := r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrUpdateFailed
	}

	return nil
}

func (r *ListRepo) UpdateLastRefresh(ctx context.Context, list domain.List) error {
	qb := r.db.squirrel.Update("list").
		Set("last_refresh_time", list.LastRefreshTime).
		Set("last_refresh_status", list.LastRefreshStatus).
		Set("last_refresh_error", list.LastRefreshError).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": list.ID})

	query, args, err := qb.ToSql()
	if err != nil {
		return err
	}

	results, err := r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrUpdateFailed
	}

	return nil
}

func (r *ListRepo) Delete(ctx context.Context, listID int64) error {
	qb := r.db.squirrel.Delete("list").From("list").Where(sq.Eq{"id": listID})
	query, args, err := qb.ToSql()
	if err != nil {
		return err
	}

	results, err := r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrDeleteFailed
	}

	return nil
}

func (r *ListRepo) ToggleEnabled(ctx context.Context, listID int64, enabled bool) error {
	qb := r.db.squirrel.Update("list").
		Set("enabled", enabled).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": listID})

	query, args, err := qb.ToSql()
	if err != nil {
		return err
	}

	results, err := r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrUpdateFailed
	}

	return nil
}
