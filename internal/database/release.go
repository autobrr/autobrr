package database

import (
	"context"
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type ReleaseRepo struct {
	db *DB
}

func NewReleaseRepo(db *DB) domain.ReleaseRepo {
	return &ReleaseRepo{db: db}
}

func (repo *ReleaseRepo) Store(ctx context.Context, r *domain.Release) (*domain.Release, error) {
	queryBuilder := repo.db.squirrel.
		Insert("release").
		Columns("filter_status", "rejections", "indexer", "filter", "protocol", "implementation", "timestamp", "group_id", "torrent_id", "torrent_name", "size", "raw", "title", "category", "season", "episode", "year", "resolution", "source", "codec", "container", "hdr", "audio", "release_group", "region", "language", "edition", "unrated", "hybrid", "proper", "repack", "website", "artists", "type", "format", "quality", "log_score", "has_log", "has_cue", "is_scene", "origin", "tags", "freeleech", "freeleech_percent", "uploader", "pre_time").
		Values(r.FilterStatus, pq.Array(r.Rejections), r.Indexer, r.FilterName, r.Protocol, r.Implementation, r.Timestamp, r.GroupID, r.TorrentID, r.TorrentName, r.Size, r.Raw, r.Title, r.Category, r.Season, r.Episode, r.Year, r.Resolution, r.Source, r.Codec, r.Container, r.HDR, r.Audio, r.Group, r.Region, r.Language, r.Edition, r.Unrated, r.Hybrid, r.Proper, r.Repack, r.Website, pq.Array(r.Artists), r.Type, r.Format, r.Quality, r.LogScore, r.HasLog, r.HasCue, r.IsScene, r.Origin, pq.Array(r.Tags), r.Freeleech, r.FreeleechPercent, r.Uploader, r.PreTime).
		Suffix("RETURNING id").RunWith(repo.db.handler)

	// return values
	var retID int64

	err := queryBuilder.QueryRowContext(ctx).Scan(&retID)
	if err != nil {
		log.Error().Stack().Err(err).Msg("release.store: error executing query")
		return nil, err
	}

	r.ID = retID

	log.Debug().Msgf("release.store: %+v", r)

	return r, nil
}

func (repo *ReleaseRepo) StoreReleaseActionStatus(ctx context.Context, a *domain.ReleaseActionStatus) error {
	if a.ID != 0 {
		queryBuilder := repo.db.squirrel.
			Update("release_action_status").
			Set("status", a.Status).
			Set("rejections", pq.Array(a.Rejections)).
			Set("timestamp", a.Timestamp).
			Where("id = ?", a.ID).
			Where("release_id = ?", a.ReleaseID)

		query, args, err := queryBuilder.ToSql()
		if err != nil {
			log.Error().Stack().Err(err).Msg("release.store: error building query")
			return err
		}

		_, err = repo.db.handler.ExecContext(ctx, query, args...)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error updating status of release")
			return err
		}

	} else {
		queryBuilder := repo.db.squirrel.
			Insert("release_action_status").
			Columns("status", "action", "type", "rejections", "timestamp", "release_id").
			Values(a.Status, a.Action, a.Type, pq.Array(a.Rejections), a.Timestamp, a.ReleaseID).
			Suffix("RETURNING id").RunWith(repo.db.handler)

		// return values
		var retID int64

		err := queryBuilder.QueryRowContext(ctx).Scan(&retID)
		if err != nil {
			log.Error().Stack().Err(err).Msg("release.storeReleaseActionStatus: error executing query")
			return err
		}

		a.ID = retID
	}

	log.Trace().Msgf("release.store_release_action_status: %+v", a)

	return nil
}

func (repo *ReleaseRepo) Find(ctx context.Context, params domain.ReleaseQueryParams) ([]*domain.Release, int64, int64, error) {
	tx, err := repo.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, 0, 0, err
	}
	defer tx.Rollback()

	releases, nextCursor, total, err := repo.findReleases(ctx, tx, params)
	if err != nil {
		return nil, nextCursor, total, err
	}

	for _, release := range releases {
		statuses, err := repo.attachActionStatus(ctx, tx, release.ID)
		if err != nil {
			return releases, nextCursor, total, err
		}
		release.ActionStatus = statuses
	}

	return releases, nextCursor, total, nil
}

func (repo *ReleaseRepo) findReleases(ctx context.Context, tx *Tx, params domain.ReleaseQueryParams) ([]*domain.Release, int64, int64, error) {
	queryBuilder := repo.db.squirrel.
		Select("r.id", "r.filter_status", "r.rejections", "r.indexer", "r.filter", "r.protocol", "r.title", "r.torrent_name", "r.size", "r.timestamp", "COUNT(*) OVER() AS total_count").
		From("release r").
		OrderBy("r.timestamp DESC")

	if params.Limit > 0 {
		queryBuilder = queryBuilder.Limit(params.Limit)
	} else {
		queryBuilder = queryBuilder.Limit(20)
	}

	if params.Offset > 0 {
		queryBuilder = queryBuilder.Offset(params.Offset)
	}

	if params.Cursor > 0 {
		queryBuilder = queryBuilder.Where(sq.Lt{"r.id": params.Cursor})
	}

	if params.Filters.Indexers != nil {
		filter := sq.And{}
		for _, v := range params.Filters.Indexers {
			filter = append(filter, sq.Eq{"r.indexer": v})
		}

		queryBuilder = queryBuilder.Where(filter)
	}

	if params.Filters.PushStatus != "" {
		queryBuilder = queryBuilder.InnerJoin("release_action_status ras ON r.id = ras.release_id").Where(sq.Eq{"ras.status": params.Filters.PushStatus})
	}

	query, args, err := queryBuilder.ToSql()
	log.Trace().Str("database", "release.find").Msgf("query: '%v', args: '%v'", query, args)

	res := make([]*domain.Release, 0)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error fetching releases")
		return res, 0, 0, nil
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		log.Error().Stack().Err(err)
		return res, 0, 0, err
	}

	var countItems int64 = 0

	for rows.Next() {
		var rls domain.Release

		var indexer, filter sql.NullString

		if err := rows.Scan(&rls.ID, &rls.FilterStatus, pq.Array(&rls.Rejections), &indexer, &filter, &rls.Protocol, &rls.Title, &rls.TorrentName, &rls.Size, &rls.Timestamp, &countItems); err != nil {
			log.Error().Stack().Err(err).Msg("release.find: error scanning data to struct")
			return res, 0, 0, err
		}

		rls.Indexer = indexer.String
		rls.FilterName = filter.String

		res = append(res, &rls)
	}

	nextCursor := int64(0)
	if len(res) > 0 {
		lastID := res[len(res)-1].ID
		nextCursor = lastID
	}

	return res, nextCursor, countItems, nil
}

func (repo *ReleaseRepo) GetIndexerOptions(ctx context.Context) ([]string, error) {

	query := `SELECT DISTINCT indexer FROM "release" UNION SELECT DISTINCT identifier indexer FROM indexer;`

	log.Trace().Str("database", "release.get_indexers").Msgf("query: '%v'", query)

	res := make([]string, 0)

	rows, err := repo.db.handler.QueryContext(ctx, query)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error fetching indexer list")
		return res, err
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		log.Error().Stack().Err(err)
		return res, err
	}

	for rows.Next() {
		var indexer string

		if err := rows.Scan(&indexer); err != nil {
			log.Error().Stack().Err(err).Msg("release.find: error scanning data to struct")
			return res, err
		}

		res = append(res, indexer)
	}

	return res, nil
}

func (repo *ReleaseRepo) GetActionStatusByReleaseID(ctx context.Context, releaseID int64) ([]domain.ReleaseActionStatus, error) {

	queryBuilder := repo.db.squirrel.
		Select("id", "status", "action", "type", "rejections", "timestamp").
		From("release_action_status").
		Where("release_id = ?", releaseID)

	query, args, err := queryBuilder.ToSql()

	res := make([]domain.ReleaseActionStatus, 0)

	rows, err := repo.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error fetching releases")
		return res, nil
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		log.Error().Stack().Err(err)
		return res, err
	}

	for rows.Next() {
		var rls domain.ReleaseActionStatus

		if err := rows.Scan(&rls.ID, &rls.Status, &rls.Action, &rls.Type, pq.Array(&rls.Rejections), &rls.Timestamp); err != nil {
			log.Error().Stack().Err(err).Msg("release.find: error scanning data to struct")
			return res, err
		}

		res = append(res, rls)
	}

	return res, nil
}

func (repo *ReleaseRepo) attachActionStatus(ctx context.Context, tx *Tx, releaseID int64) ([]domain.ReleaseActionStatus, error) {

	queryBuilder := repo.db.squirrel.
		Select("id", "status", "action", "type", "rejections", "timestamp").
		From("release_action_status").
		Where("release_id = ?", releaseID)

	query, args, err := queryBuilder.ToSql()

	res := make([]domain.ReleaseActionStatus, 0)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error fetching releases")
		return res, nil
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		log.Error().Stack().Err(err)
		return res, err
	}

	for rows.Next() {
		var rls domain.ReleaseActionStatus

		if err := rows.Scan(&rls.ID, &rls.Status, &rls.Action, &rls.Type, pq.Array(&rls.Rejections), &rls.Timestamp); err != nil {
			log.Error().Stack().Err(err).Msg("release.find: error scanning data to struct")
			return res, err
		}

		res = append(res, rls)
	}

	return res, nil
}

func (repo *ReleaseRepo) Stats(ctx context.Context) (*domain.ReleaseStats, error) {

	query := `SELECT COUNT(*)                                                                      total,
       COALESCE(SUM(CASE WHEN filter_status = 'FILTER_APPROVED' THEN 1 ELSE 0 END), 0) AS filtered_count,
       COALESCE(SUM(CASE WHEN filter_status = 'FILTER_REJECTED' THEN 1 ELSE 0 END), 0) AS filter_rejected_count,
       (SELECT COALESCE(SUM(CASE WHEN status = 'PUSH_APPROVED' THEN 1 ELSE 0 END), 0)
        FROM "release_action_status") AS                                             push_approved_count,
       (SELECT COALESCE(SUM(CASE WHEN status = 'PUSH_REJECTED' THEN 1 ELSE 0 END), 0)
        FROM "release_action_status") AS                                             push_rejected_count
FROM "release";`

	row := repo.db.handler.QueryRowContext(ctx, query)
	if err := row.Err(); err != nil {
		log.Error().Stack().Err(err).Msg("release.stats: error querying stats")
		return nil, err
	}

	var rls domain.ReleaseStats

	if err := row.Scan(&rls.TotalCount, &rls.FilteredCount, &rls.FilterRejectedCount, &rls.PushApprovedCount, &rls.PushRejectedCount); err != nil {
		log.Error().Stack().Err(err).Msg("release.stats: error scanning stats data to struct")
		return nil, err
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
		log.Error().Stack().Err(err).Msg("error deleting all releases")
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM release_action_status`)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error deleting all release_action_status")
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error deleting all releases")
		return err

	}

	return nil
}
