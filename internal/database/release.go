package database

import (
	"context"
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/domain"
)

type ReleaseRepo struct {
	db *sql.DB
}

func NewReleaseRepo(db *sql.DB) domain.ReleaseRepo {
	return &ReleaseRepo{db: db}
}

func (repo *ReleaseRepo) Store(ctx context.Context, r *domain.Release) (*domain.Release, error) {
	query, args, err := sq.
		Insert("release").
		Columns("filter_status", "push_status", "rejections", "indexer", "filter", "protocol", "implementation", "timestamp", "group_id", "torrent_id", "torrent_name", "size", "raw", "title", "category", "season", "episode", "year", "resolution", "source", "codec", "container", "hdr", "audio", "release_group", "region", "language", "edition", "unrated", "hybrid", "proper", "repack", "website", "artists", "type", "format", "bitrate", "log_score", "has_log", "has_cue", "is_scene", "origin", "tags", "freeleech", "freeleech_percent", "uploader", "pre_time").
		Values(r.FilterStatus, r.PushStatus, pq.Array(r.Rejections), r.Indexer, r.FilterName, r.Protocol, r.Implementation, r.Timestamp, r.GroupID, r.TorrentID, r.TorrentName, r.Size, r.Raw, r.Title, r.Category, r.Season, r.Episode, r.Year, r.Resolution, r.Source, r.Codec, r.Container, r.HDR, r.Audio, r.Group, r.Region, r.Language, r.Edition, r.Unrated, r.Hybrid, r.Proper, r.Repack, r.Website, pq.Array(r.Artists), r.Type, r.Format, r.Bitrate, r.LogScore, r.HasLog, r.HasCue, r.IsScene, r.Origin, pq.Array(r.Tags), r.Freeleech, r.FreeleechPercent, r.Uploader, r.PreTime).
		ToSql()

	res, err := repo.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error inserting release")
		return nil, err
	}

	resId, _ := res.LastInsertId()
	r.ID = resId

	log.Trace().Msgf("release.store: %+v", r)

	return r, nil
}

func (repo *ReleaseRepo) UpdatePushStatus(ctx context.Context, id int64, status domain.ReleasePushStatus) error {
	query, args, err := sq.Update("release").Set("push_status", status).Where("id = ?", id).ToSql()

	_, err = repo.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error updating status of release")
		return err
	}

	log.Trace().Msgf("release.update_push_status: id %+v", id)

	return nil
}

func (repo *ReleaseRepo) UpdatePushStatusRejected(ctx context.Context, id int64, rejections string) error {
	r := []string{rejections}

	query, args, err := sq.
		Update("release").
		Set("push_status", domain.ReleasePushStatusRejected).
		Set("rejections", pq.Array(r)).
		Where("id = ?", id).
		ToSql()

	_, err = repo.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error updating status of release")
		return err
	}

	log.Trace().Msgf("release.update_push_status_rejected: id %+v", id)

	return nil
}

func (repo *ReleaseRepo) Find(ctx context.Context, params domain.QueryParams) ([]domain.Release, int64, int64, error) {

	queryBuilder := sq.
		Select("id", "filter_status", "push_status", "rejections", "indexer", "filter", "protocol", "title", "torrent_name", "size", "timestamp", "COUNT() OVER() AS total_count").
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

	rows, err := repo.db.QueryContext(ctx, query, args...)
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
		var timestamp string

		if err := rows.Scan(&rls.ID, &rls.FilterStatus, &rls.PushStatus, pq.Array(&rls.Rejections), &indexer, &filter, &rls.Protocol, &rls.Title, &rls.TorrentName, &rls.Size, &timestamp, &countItems); err != nil {
			log.Error().Stack().Err(err).Msg("release.find: error scanning data to struct")
			return res, 0, 0, err
		}

		rls.Indexer = indexer.String
		rls.FilterName = filter.String

		ca, _ := time.Parse(time.RFC3339, timestamp)
		rls.Timestamp = ca

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

func (repo *ReleaseRepo) Stats(ctx context.Context) (*domain.ReleaseStats, error) {
	query := `SELECT
    COUNT(*) total,
    IFNULL(SUM(CASE WHEN push_status = 'PUSH_APPROVED' THEN 1 ELSE 0 END), 0) push_approved_count,
    IFNULL(SUM(CASE WHEN push_status = 'PUSH_REJECTED' THEN 1 ELSE 0 END), 0) push_rejected_count,
    IFNULL(SUM(CASE WHEN filter_status = 'FILTER_APPROVED' THEN 1 ELSE 0 END), 0) filtered_count,
    IFNULL(SUM(CASE WHEN filter_status = 'FILTER_REJECTED' THEN 1 ELSE 0 END), 0) filter_rejected_count
FROM "release";`

	row := repo.db.QueryRowContext(ctx, query)
	if err := row.Err(); err != nil {
		log.Error().Stack().Err(err).Msg("release.stats: error querying stats")
		return nil, err
	}

	var rls domain.ReleaseStats

	if err := row.Scan(&rls.TotalCount, &rls.PushApprovedCount, &rls.PushRejectedCount, &rls.FilteredCount, &rls.FilterRejectedCount); err != nil {
		log.Error().Stack().Err(err).Msg("release.stats: error scanning stats data to struct")
		return nil, err
	}

	return &rls, nil
}
