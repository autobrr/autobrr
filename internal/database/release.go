package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

type ReleaseRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewReleaseRepo(log logger.Logger, db *DB) domain.ReleaseRepo {
	return &ReleaseRepo{
		log: log.With().Str("repo", "release").Logger(),
		db:  db,
	}
}

func (repo *ReleaseRepo) Store(ctx context.Context, r *domain.Release) (*domain.Release, error) {
	codecStr := strings.Join(r.Codec, ",")
	hdrStr := strings.Join(r.HDR, ",")

	queryBuilder := repo.db.squirrel.
		Insert("release").
		Columns("filter_status", "rejections", "indexer", "filter", "protocol", "implementation", "timestamp", "group_id", "torrent_id", "info_url", "download_url", "torrent_name", "size", "title", "category", "season", "episode", "year", "resolution", "source", "codec", "container", "hdr", "release_group", "proper", "repack", "website", "type", "origin", "tags", "uploader", "pre_time", "filter_id").
		Values(r.FilterStatus, pq.Array(r.Rejections), r.Indexer, r.FilterName, r.Protocol, r.Implementation, r.Timestamp.Format(time.RFC3339), r.GroupID, r.TorrentID, r.InfoURL, r.TorrentURL, r.TorrentName, r.Size, r.Title, r.Category, r.Season, r.Episode, r.Year, r.Resolution, r.Source, codecStr, r.Container, hdrStr, r.Group, r.Proper, r.Repack, r.Website, r.Type, r.Origin, pq.Array(r.Tags), r.Uploader, r.PreTime, r.FilterID).
		Suffix("RETURNING id").RunWith(repo.db.handler)

	// return values
	var retID int64

	err := queryBuilder.QueryRowContext(ctx).Scan(&retID)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	r.ID = retID

	repo.log.Debug().Msgf("release.store: %+v", r)

	return r, nil
}

func (repo *ReleaseRepo) StoreReleaseActionStatus(ctx context.Context, a *domain.ReleaseActionStatus) error {
	if a.ID != 0 {
		queryBuilder := repo.db.squirrel.
			Update("release_action_status").
			Set("status", a.Status).
			Set("rejections", pq.Array(a.Rejections)).
			Set("timestamp", a.Timestamp).
			Where(sq.Eq{"id": a.ID}).
			Where(sq.Eq{"release_id": a.ReleaseID})

		query, args, err := queryBuilder.ToSql()
		if err != nil {
			return errors.Wrap(err, "error building query")
		}

		_, err = repo.db.handler.ExecContext(ctx, query, args...)
		if err != nil {
			return errors.Wrap(err, "error executing query")
		}

	} else {
		queryBuilder := repo.db.squirrel.
			Insert("release_action_status").
			Columns("status", "action", "type", "client", "filter", "filter_id", "rejections", "timestamp", "release_id").
			Values(a.Status, a.Action, a.Type, a.Client, a.Filter, a.FilterID, pq.Array(a.Rejections), a.Timestamp, a.ReleaseID).
			Suffix("RETURNING id").RunWith(repo.db.handler)

		// return values
		var retID int64

		err := queryBuilder.QueryRowContext(ctx).Scan(&retID)
		if err != nil {
			return errors.Wrap(err, "error executing query")
		}

		a.ID = retID
	}

	repo.log.Trace().Msgf("release.store_release_action_status: %+v", a)

	return nil
}

func (repo *ReleaseRepo) Find(ctx context.Context, params domain.ReleaseQueryParams) ([]*domain.Release, int64, int64, error) {
	tx, err := repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, 0, 0, errors.Wrap(err, "error begin transaction")
	}
	defer tx.Rollback()

	releases, nextCursor, total, err := repo.findReleases(ctx, tx, params)
	if err != nil {
		return nil, nextCursor, total, err
	}

	return releases, nextCursor, total, nil
}

func (repo *ReleaseRepo) findReleases(ctx context.Context, tx *Tx, params domain.ReleaseQueryParams) ([]*domain.Release, int64, int64, error) {
	whereQueryBuilder := sq.And{}
	if params.Cursor > 0 {
		whereQueryBuilder = append(whereQueryBuilder, sq.Lt{"r.id": params.Cursor})
	}

	if params.Search != "" {
		reserved := map[string]string{
			"title":      "r.title",
			"group":      "r.release_group",
			"category":   "r.category",
			"season":     "r.season",
			"episode":    "r.episode",
			"year":       "r.year",
			"resolution": "r.resolution",
			"source":     "r.source",
			"codec":      "r.codec",
			"hdr":        "r.hdr",
			"filter":     "r.filter",
		}

		search := strings.TrimSpace(params.Search)
		for k, v := range reserved {
			r := regexp.MustCompile(fmt.Sprintf(`(?:%s:)(?P<value>'.*?'|".*?"|\S+)`, k))
			if reskey := r.FindAllStringSubmatch(search, -1); len(reskey) != 0 {
				filter := sq.Or{}
				for _, found := range reskey {
					filter = append(filter, ILike(v, strings.ReplaceAll(strings.Trim(strings.Trim(found[1], `"`), `'`), ".", "_")+"%"))
				}

				if len(filter) == 0 {
					continue
				}

				whereQueryBuilder = append(whereQueryBuilder, filter)
				search = strings.TrimSpace(r.ReplaceAllLiteralString(search, ""))
			}
		}

		if len(search) != 0 {
			whereQueryBuilder = append(whereQueryBuilder, sq.Like{"r.torrent_name": search + "%"})
		}
	}

	if params.Filters.Indexers != nil {
		filter := sq.And{}
		for _, v := range params.Filters.Indexers {
			filter = append(filter, sq.Eq{"r.indexer": v})
		}

		if len(filter) > 0 {
			whereQueryBuilder = append(whereQueryBuilder, filter)
		}
	}

	whereQuery, _, err := whereQueryBuilder.ToSql()
	if err != nil {
		return nil, 0, 0, errors.Wrap(err, "error building wherequery")
	}

	subQueryBuilder := repo.db.squirrel.
		Select("r.id").
		Distinct().
		From("release r").
		OrderBy("r.id DESC")

	if params.Limit > 0 {
		subQueryBuilder = subQueryBuilder.Limit(params.Limit)
	} else {
		subQueryBuilder = subQueryBuilder.Limit(20)
	}

	if params.Offset > 0 {
		subQueryBuilder = subQueryBuilder.Offset(params.Offset)
	}

	if len(whereQueryBuilder) != 0 {
		subQueryBuilder = subQueryBuilder.Where(whereQueryBuilder)
	}

	countQuery := repo.db.squirrel.Select("COUNT(*)").From("release r").Where(whereQuery)

	if params.Filters.PushStatus != "" {
		subQueryBuilder = subQueryBuilder.InnerJoin("release_action_status ras ON r.id = ras.release_id").Where(sq.Eq{"ras.status": params.Filters.PushStatus})
		countQuery = countQuery.InnerJoin("release_action_status ras ON r.id = ras.release_id").Where("ras.status = '" + params.Filters.PushStatus + `'`)
	}

	subQuery, subArgs, err := subQueryBuilder.ToSql()
	if err != nil {
		return nil, 0, 0, errors.Wrap(err, "error building subquery")
	}

	queryBuilder := repo.db.squirrel.
		Select("r.id", "r.filter_status", "r.rejections", "r.indexer", "r.filter", "r.protocol", "r.info_url", "r.download_url", "r.title", "r.torrent_name", "r.size", "r.timestamp",
			"ras.id", "ras.status", "ras.action", "ras.type", "ras.client", "ras.filter", "ras.rejections", "ras.timestamp").
		Column(sq.Alias(countQuery, "page_total")).
		From("release r").
		OrderBy("r.id DESC").
		Where("r.id IN ("+subQuery+")", subArgs...).
		LeftJoin("release_action_status ras ON r.id = ras.release_id")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, 0, 0, errors.Wrap(err, "error building query")
	}

	repo.log.Trace().Str("database", "release.find").Msgf("query: '%v', args: '%v'", query, args)

	res := make([]*domain.Release, 0)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, 0, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		return res, 0, 0, errors.Wrap(err, "error rows findreleases")
	}

	var countItems int64 = 0

	for rows.Next() {
		var rls domain.Release
		var ras domain.ReleaseActionStatus

		var rlsindexer, rlsfilter, infoUrl, downloadUrl sql.NullString

		var rasId sql.NullInt64
		var rasStatus, rasAction, rasType, rasClient, rasFilter sql.NullString
		var rasRejections []sql.NullString
		var rasTimestamp sql.NullTime

		if err := rows.Scan(&rls.ID, &rls.FilterStatus, pq.Array(&rls.Rejections), &rlsindexer, &rlsfilter, &rls.Protocol, &infoUrl, &downloadUrl, &rls.Title, &rls.TorrentName, &rls.Size, &rls.Timestamp, &rasId, &rasStatus, &rasAction, &rasType, &rasClient, &rasFilter, pq.Array(&rasRejections), &rasTimestamp, &countItems); err != nil {
			return res, 0, 0, errors.Wrap(err, "error scanning row")
		}

		ras.ID = rasId.Int64
		ras.Status = domain.ReleasePushStatus(rasStatus.String)
		ras.Action = rasAction.String
		ras.Type = domain.ActionType(rasType.String)
		ras.Client = rasClient.String
		ras.Filter = rasFilter.String
		ras.Timestamp = rasTimestamp.Time
		ras.Rejections = []string{}

		for _, rejection := range rasRejections {
			ras.Rejections = append(ras.Rejections, rejection.String)
		}

		idx := 0
		for ; idx < len(res); idx++ {
			if res[idx].ID != rls.ID {
				continue
			}

			res[idx].ActionStatus = append(res[idx].ActionStatus, ras)
			break
		}

		if idx != len(res) {
			continue
		}

		rls.Indexer = rlsindexer.String
		rls.FilterName = rlsfilter.String
		rls.ActionStatus = make([]domain.ReleaseActionStatus, 0)
		rls.InfoURL = infoUrl.String
		rls.TorrentURL = downloadUrl.String

		// only add ActionStatus if it's not empty
		if ras.ID > 0 {
			rls.ActionStatus = append(rls.ActionStatus, ras)
		}

		res = append(res, &rls)
	}

	nextCursor := int64(0)
	if len(res) > 0 {
		lastID := res[len(res)-1].ID
		nextCursor = lastID
	}

	return res, nextCursor, countItems, nil
}

func (repo *ReleaseRepo) FindRecent(ctx context.Context) ([]*domain.Release, error) {
	tx, err := repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, errors.Wrap(err, "error begin transaction")
	}
	defer tx.Rollback()

	releases, _, _, err := repo.findReleases(ctx, tx, domain.ReleaseQueryParams{Limit: 10})
	if err != nil {
		return nil, err
	}

	return releases, nil
}

func (repo *ReleaseRepo) GetIndexerOptions(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT indexer FROM "release" UNION SELECT DISTINCT identifier indexer FROM indexer;`

	repo.log.Trace().Str("database", "release.get_indexers").Msgf("query: '%v'", query)

	res := make([]string, 0)

	rows, err := repo.db.handler.QueryContext(ctx, query)
	if err != nil {
		return res, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		return res, errors.Wrap(err, "rows error")
	}

	for rows.Next() {
		var indexer string

		if err := rows.Scan(&indexer); err != nil {
			return res, errors.Wrap(err, "error scanning row")
		}

		res = append(res, indexer)
	}

	return res, nil
}

func (repo *ReleaseRepo) GetActionStatusByReleaseID(ctx context.Context, releaseID int64) ([]domain.ReleaseActionStatus, error) {
	queryBuilder := repo.db.squirrel.
		Select("id", "status", "action", "type", "client", "filter", "rejections", "timestamp").
		From("release_action_status").
		Where(sq.Eq{"release_id": releaseID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	res := make([]domain.ReleaseActionStatus, 0)

	rows, err := repo.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return res, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		repo.log.Error().Stack().Err(err)
		return res, err
	}

	for rows.Next() {
		var rls domain.ReleaseActionStatus

		var client, filter sql.NullString

		if err := rows.Scan(&rls.ID, &rls.Status, &rls.Action, &rls.Type, &client, &filter, pq.Array(&rls.Rejections), &rls.Timestamp); err != nil {
			return res, errors.Wrap(err, "error scanning row")
		}

		rls.Client = client.String
		rls.Filter = filter.String

		res = append(res, rls)
	}

	return res, nil
}

func (repo *ReleaseRepo) attachActionStatus(ctx context.Context, tx *Tx, releaseID int64) ([]domain.ReleaseActionStatus, error) {
	queryBuilder := repo.db.squirrel.
		Select("id", "status", "action", "type", "client", "filter", "filter_id", "rejections", "timestamp").
		From("release_action_status").
		Where(sq.Eq{"release_id": releaseID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	res := make([]domain.ReleaseActionStatus, 0)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return res, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		return res, errors.Wrap(err, "error rows")
	}

	for rows.Next() {
		var rls domain.ReleaseActionStatus

		var client, filter sql.NullString
		var filterID sql.NullInt64

		if err := rows.Scan(&rls.ID, &rls.Status, &rls.Action, &rls.Type, &client, &filter, &filterID, pq.Array(&rls.Rejections), &rls.Timestamp); err != nil {
			return res, errors.Wrap(err, "error scanning row")
		}

		rls.Client = client.String
		rls.Filter = filter.String
		rls.FilterID = filterID.Int64

		res = append(res, rls)
	}

	return res, nil
}

func (repo *ReleaseRepo) Stats(ctx context.Context) (*domain.ReleaseStats, error) {
	query := `SELECT *
FROM (
	SELECT
	COUNT(*) AS total,
	COUNT(CASE WHEN filter_status = 'FILTER_APPROVED' THEN 0 END) AS filtered_count,
	COUNT(CASE WHEN filter_status = 'FILTER_REJECTED' THEN 0 END) AS filter_rejected_count
	FROM release
) AS zoo
CROSS JOIN (
	SELECT
	COUNT(CASE WHEN status = 'PUSH_APPROVED' THEN 0 END) AS push_approved_count,
	COUNT(CASE WHEN status = 'PUSH_REJECTED' THEN 0 END) AS push_rejected_count
	FROM release_action_status
) AS foo`

	row := repo.db.handler.QueryRowContext(ctx, query)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var rls domain.ReleaseStats

	if err := row.Scan(&rls.TotalCount, &rls.FilteredCount, &rls.FilterRejectedCount, &rls.PushApprovedCount, &rls.PushRejectedCount); err != nil {
		return nil, errors.Wrap(err, "error scanning row")
	}

	return &rls, nil
}

func (repo *ReleaseRepo) Delete(ctx context.Context) error {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `DELETE FROM "release"`)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM release_action_status`)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "error commit transaction delete")
	}

	return nil
}

func (repo *ReleaseRepo) CanDownloadShow(ctx context.Context, title string, season int, episode int) (bool, error) {
	// TODO support non season episode shows
	// if rls.Day > 0 {
	//	// Maybe in the future
	//	// SELECT '' FROM release WHERE Title LIKE %q AND ((Year == %d AND Month == %d AND Day > %d) OR (Year == %d AND Month > %d) OR (Year > %d))"
	//	qs := sql.Query("SELECT torrent_name FROM release WHERE Title LIKE %q AND Year >= %d", rls.Title, rls.Year)
	//
	//	for q := range qs.Rows() {
	//		r := rls.ParseTitle(q)
	//		if r.Year > rls.Year {
	//			return false, fmt.Errorf("stale release year")
	//		}
	//
	//		if r.Month > rls.Month {
	//			return false, fmt.Errorf("stale release month")
	//		}
	//
	//		if r.Month == rls.Month && r.Day > rls.Day {
	//			return false, fmt.Errorf("stale release day")
	//		}
	//	}
	//}

	queryBuilder := repo.db.squirrel.
		Select("COUNT(*)").
		From("release").
		Where(ILike("title", title+"%"))

	if season > 0 && episode > 0 {
		queryBuilder = queryBuilder.Where(sq.Or{
			sq.And{
				sq.Eq{"season": season},
				sq.Gt{"episode": episode},
			},
			sq.Gt{"season": season},
		})
	} else if season > 0 && episode == 0 {
		queryBuilder = queryBuilder.Where(sq.Gt{"season": season})
	} else {
		/* No support for this scenario today. Specifically multi-part specials.
		 * The Database presently does not have Subtitle as a field, but is coming at a future date. */
		return true, nil
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return false, errors.Wrap(err, "error building query")
	}

	row := repo.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return false, err
	}

	var count int

	if err := row.Scan(&count); err != nil {
		return false, err
	}

	if count > 0 {
		return false, nil
	}

	return true, nil
}
