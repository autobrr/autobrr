package database

import (
	"context"
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/domain"
)

type ReleaseRepo struct {
	db *SqliteDB
}

func NewReleaseRepo(db *SqliteDB) domain.ReleaseRepo {
	return &ReleaseRepo{db: db}
}

func (repo *ReleaseRepo) Store(ctx context.Context, r *domain.Release) (*domain.Release, error) {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()

	query, args, err := sq.
		Insert("release").
		Columns("filter_status", "rejections", "indexer", "filter", "protocol", "implementation", "timestamp", "group_id", "torrent_id", "torrent_name", "size", "raw", "title", "category", "season", "episode", "year", "resolution", "source", "codec", "container", "hdr", "audio", "release_group", "region", "language", "edition", "unrated", "hybrid", "proper", "repack", "website", "artists", "type", "format", "quality", "log_score", "has_log", "has_cue", "is_scene", "origin", "tags", "freeleech", "freeleech_percent", "uploader", "pre_time").
		Values(r.FilterStatus, pq.Array(r.Rejections), r.Indexer, r.FilterName, r.Protocol, r.Implementation, r.Timestamp, r.GroupID, r.TorrentID, r.TorrentName, r.Size, r.Raw, r.Title, r.Category, r.Season, r.Episode, r.Year, r.Resolution, r.Source, r.Codec, r.Container, r.HDR, r.Audio, r.Group, r.Region, r.Language, r.Edition, r.Unrated, r.Hybrid, r.Proper, r.Repack, r.Website, pq.Array(r.Artists), r.Type, r.Format, r.Quality, r.LogScore, r.HasLog, r.HasCue, r.IsScene, r.Origin, pq.Array(r.Tags), r.Freeleech, r.FreeleechPercent, r.Uploader, r.PreTime).
		ToSql()

	res, err := repo.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error inserting release")
		return nil, err
	}

	resId, _ := res.LastInsertId()
	r.ID = resId

	log.Trace().Msgf("release.store: %+v", r)

	return r, nil
}

func (repo *ReleaseRepo) StoreReleaseActionStatus(ctx context.Context, a *domain.ReleaseActionStatus) error {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()

	if a.ID != 0 {
		query, args, err := sq.
			Update("release_action_status").
			Set("status", a.Status).
			Set("rejections", pq.Array(a.Rejections)).
			Set("timestamp", a.Timestamp).
			Where("id = ?", a.ID).
			Where("release_id = ?", a.ReleaseID).
			ToSql()

		_, err = repo.db.handler.ExecContext(ctx, query, args...)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error updating status of release")
			return err
		}

	} else {
		query, args, err := sq.
			Insert("release_action_status").
			Columns("status", "action", "type", "rejections", "timestamp", "release_id").
			Values(a.Status, a.Action, a.Type, pq.Array(a.Rejections), a.Timestamp, a.ReleaseID).
			ToSql()

		res, err := repo.db.handler.ExecContext(ctx, query, args...)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error inserting status of release")
			return err
		}

		resId, _ := res.LastInsertId()
		a.ID = resId
	}

	log.Trace().Msgf("release.store_release_action_status: %+v", a)

	return nil
}

func (repo *ReleaseRepo) Find(ctx context.Context, params domain.QueryParams) ([]domain.Release, int64, int64, error) {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()

	queryBuilder := sq.
		Select("id", "filter_status", "rejections", "indexer", "filter", "protocol", "title", "torrent_name", "size", "timestamp", "COUNT() OVER() AS total_count").
		From("release").
		OrderBy("timestamp DESC")

	if params.Limit > 0 {
		queryBuilder = queryBuilder.Limit(params.Limit)
	} else {
		queryBuilder = queryBuilder.Limit(20)
	}

	if params.Offset > 0 {
		queryBuilder = queryBuilder.Offset(params.Offset)
	}

	if params.Cursor > 0 {
		queryBuilder = queryBuilder.Where(sq.Lt{"id": params.Cursor})
	}

	if params.Filter != nil {
		filter := sq.And{}
		for k, v := range params.Filter {
			filter = append(filter, sq.Eq{k: v})
		}

		queryBuilder = queryBuilder.Where(filter)
	}

	query, args, err := queryBuilder.ToSql()
	log.Trace().Str("database", "release.find").Msgf("query: '%v', args: '%v'", query, args)

	res := make([]domain.Release, 0)

	rows, err := repo.db.handler.QueryContext(ctx, query, args...)
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

		// get action status
		actionStatus, err := repo.GetActionStatusByReleaseID(ctx, rls.ID)
		if err != nil {
			log.Error().Stack().Err(err).Msg("release.find: error getting action status")
			return res, 0, 0, err
		}

		rls.ActionStatus = actionStatus

		res = append(res, rls)
	}

	nextCursor := int64(0)
	if len(res) > 0 {
		lastID := res[len(res)-1].ID
		nextCursor = lastID
		//nextCursor, _ = strconv.ParseInt(lastID, 10, 64)
	}

	return res, nextCursor, countItems, nil
}

func (repo *ReleaseRepo) GetActionStatusByReleaseID(ctx context.Context, releaseID int64) ([]domain.ReleaseActionStatus, error) {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()

	queryBuilder := sq.
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

func (repo *ReleaseRepo) Stats(ctx context.Context) (*domain.ReleaseStats, error) {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()

	query := `SELECT COUNT(*)                                                                      total,
       IFNULL(SUM(CASE WHEN filter_status = 'FILTER_APPROVED' THEN 1 ELSE 0 END), 0) filtered_count,
       IFNULL(SUM(CASE WHEN filter_status = 'FILTER_REJECTED' THEN 1 ELSE 0 END), 0) filter_rejected_count,
       (SELECT IFNULL(SUM(CASE WHEN status = 'PUSH_APPROVED' THEN 1 ELSE 0 END), 0)
        FROM "release_action_status") AS                                             push_approved_count,
       (SELECT IFNULL(SUM(CASE WHEN status = 'PUSH_REJECTED' THEN 1 ELSE 0 END), 0)
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
