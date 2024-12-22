package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
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

func (r *ListRepo) List(ctx context.Context) ([]*domain.List, error) {
	qb := r.db.squirrel.Select(
		"id",
		"name",
		"enabled",
		"type",
		"client_id",
		"url",
		"headers",
		"api_key",
		"match_release",
		"tags_included",
		"tags_excluded",
		"include_unmonitored",
		"include_alternate_titles",
		"last_refresh_time",
		"last_refresh_status",
		"last_refresh_data",
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

	lists := make([]*domain.List, 0)
	for rows.Next() {
		var list domain.List

		var url, apiKey, lastRefreshStatus, lastRefreshData sql.Null[string]
		var lastRefreshTime sql.Null[time.Time]

		err = rows.Scan(&list.ID, &list.Name, &list.Enabled, &list.Type, &list.ClientID, &url, pq.Array(&list.Headers), &list.APIKey, &list.MatchRelease, pq.Array(&list.TagsInclude), pq.Array(&list.TagsExclude), &list.IncludeUnmonitored, &list.IncludeAlternateTitles, &lastRefreshTime, &lastRefreshStatus, &lastRefreshData, &list.CreatedAt, &list.UpdatedAt)
		if err != nil {
			return nil, err
		}

		list.URL = url.V
		list.APIKey = apiKey.V
		list.LastRefreshTime = lastRefreshTime.V
		list.LastRefreshData = lastRefreshData.V
		list.LastRefreshStatus = domain.ListRefreshStatus(lastRefreshStatus.V)
		list.Filters = make([]domain.ListFilter, 0)

		lists = append(lists, &list)
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
		"match_release",
		"tags_included",
		"tags_excluded",
		"include_unmonitored",
		"include_alternate_titles",
		"last_refresh_time",
		"last_refresh_status",
		"last_refresh_data",
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

	var url, apiKey sql.Null[string]

	err = row.Scan(&list.ID, &list.Name, &list.Enabled, &list.Type, &list.ClientID, &url, pq.Array(&list.Headers), &list.APIKey, &list.MatchRelease, pq.Array(&list.TagsInclude), pq.Array(&list.TagsExclude), &list.IncludeUnmonitored, &list.IncludeAlternateTitles, &list.LastRefreshTime, &list.LastRefreshStatus, &list.LastRefreshData, &list.CreatedAt, &list.UpdatedAt)
	if err != nil {
		return nil, err
	}

	list.URL = url.V
	list.APIKey = apiKey.V

	return &list, nil
}

func (r *ListRepo) Store(ctx context.Context, list *domain.List) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "error begin transaction")
	}

	defer tx.Rollback()

	qb := r.db.squirrel.Insert("list").
		Columns(
			"name",
			"enabled",
			"type",
			"client_id",
			"url",
			"headers",
			"api_key",
			"match_release",
			"tags_included",
			"tags_excluded",
			"include_unmonitored",
			"include_alternate_titles",
		).
		Values(
			list.Name,
			list.Enabled,
			list.Type,
			list.ClientID,
			list.URL,
			pq.Array(list.Headers),
			list.APIKey,
			list.MatchRelease,
			pq.Array(list.TagsInclude),
			pq.Array(list.TagsExclude),
			list.IncludeUnmonitored,
			list.IncludeAlternateTitles,
		).Suffix("RETURNING id").RunWith(tx)

	//query, args, err := qb.ToSql()
	//if err != nil {
	//	return err
	//}

	if err := qb.QueryRowContext(ctx).Scan(&list.ID); err != nil {
		return err
	}

	if err := r.StoreListFilterConnection(ctx, tx, list); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "error storing list and filters")
	}

	return nil
}

func (r *ListRepo) Update(ctx context.Context, list *domain.List) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "error begin transaction")
	}

	defer tx.Rollback()

	qb := r.db.squirrel.Update("list").
		Set("name", list.Name).
		Set("enabled", list.Enabled).
		Set("type", list.Type).
		Set("client_id", list.ClientID).
		Set("url", list.URL).
		Set("headers", pq.Array(list.Headers)).
		Set("api_key", list.APIKey).
		Set("match_release", list.MatchRelease).
		Set("tags_included", pq.Array(list.TagsInclude)).
		Set("tags_excluded", pq.Array(list.TagsExclude)).
		Set("include_unmonitored", list.IncludeUnmonitored).
		Set("include_alternate_titles", list.IncludeAlternateTitles).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": list.ID})

	query, args, err := qb.ToSql()
	if err != nil {
		return err
	}

	results, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	if err := r.StoreListFilterConnection(ctx, tx, list); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "error updating filter actions")
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

func (r *ListRepo) UpdateLastRefresh(ctx context.Context, list *domain.List) error {
	qb := r.db.squirrel.Update("list").
		Set("last_refresh_time", list.LastRefreshTime).
		Set("last_refresh_status", list.LastRefreshStatus).
		Set("last_refresh_data", list.LastRefreshData).
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

func (r *ListRepo) StoreListFilterConnection(ctx context.Context, tx *Tx, list *domain.List) error {
	qb := r.db.squirrel.Delete("list_filter").Where(sq.Eq{"list_id": list.ID})

	query, args, err := qb.ToSql()
	if err != nil {
		return err
	}

	results, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return err
	}

	r.log.Trace().Int64("rows_affected", rowsAffected).Msg("deleted list filters")

	//if rowsAffected == 0 {
	//	return domain.ErrUpdateFailed
	//}

	for _, filter := range list.Filters {
		qb := r.db.squirrel.Insert("list_filter").
			Columns(
				"list_id",
				"filter_id",
			).
			Values(
				list.ID,
				filter.ID,
			)

		query, args, err := qb.ToSql()
		if err != nil {
			return err
		}

		results, err := tx.ExecContext(ctx, query, args...)
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
	}

	return nil
}

func (r *ListRepo) GetListFilters(ctx context.Context, listID int64) ([]domain.ListFilter, error) {
	qb := r.db.squirrel.Select(
		"f.id",
		"f.name",
	).
		From("list_filter lf").
		Join(
			"filter f ON f.id = lf.filter_id",
		).
		OrderBy(
			"f.name ASC",
		).
		Where(sq.Eq{"lf.list_id": listID})

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	filters := make([]domain.ListFilter, 0)
	for rows.Next() {
		var filter domain.ListFilter
		err = rows.Scan(&filter.ID, &filter.Name)
		if err != nil {
			return nil, err
		}

		filters = append(filters, filter)
	}

	return filters, nil
}
