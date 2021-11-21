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

func (r *ReleaseRepo) Store(release domain.Release) (*domain.Release, error) {
	res, err := r.db.Exec(`INSERT INTO "release"(status, rejections, indexer, protocol, title, size) VALUES (?, ?, ?, ? ,? ,?) ON CONFLICT DO NOTHING`, release.Status, pq.Array(release.Rejections), release.Indexer, release.Protocol, release.Title, release.Size)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error storing release")
		return nil, err
	}

	log.Debug().Msgf("store release: %+v", release)

	resId, _ := res.LastInsertId()
	release.ID = resId

	return &release, nil
}

func (r *ReleaseRepo) Find(ctx context.Context, params domain.QueryParams) ([]domain.Release, int64, error) {

	queryBuilder := sq.Select("id", "status", "rejections", "indexer", "client", "filter", "protocol", "title", "size", "created_at").From("release").OrderBy("created_at DESC")

	if params.Limit > 0 {
		queryBuilder = queryBuilder.Limit(params.Limit)
	} else {
		queryBuilder = queryBuilder.Limit(20)
	}

	if params.Cursor > 0 {
		//queryBuilder = queryBuilder.Where(sq.Gt{"id": params.Cursor})
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
	log.Trace().Msgf("release.find: query: '%v', args: '%v'", query, args)

	//rows, err := r.db.QueryContext(ctx, `
	//	SELECT
	//	       id, status, rejections, indexer, client, protocol, title, size, created_at
	//	FROM "release"
	//	ORDER BY
	//		 created_at DESC
	//	LIMIT ?`, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error fetching releases")
		//return
		return nil, 0, nil
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		log.Error().Stack().Err(err)
		return nil, 0, err
	}

	res := make([]domain.Release, 0)

	for rows.Next() {
		var rls domain.Release

		var indexer, client, filter sql.NullString
		var createdAt string

		if err := rows.Scan(&rls.ID, &rls.Status, pq.Array(&rls.Rejections), &indexer, &client, &filter, &rls.Protocol, &rls.Title, &rls.Size, &createdAt); err != nil {
			log.Error().Stack().Err(err).Msg("release.find: error scanning data to struct")
			return nil, 0, err
		}

		rls.Indexer = indexer.String
		//rls.Client = client.String
		//rls.Filter = filter.String

		ca, _ := time.Parse(time.RFC3339, createdAt)
		rls.Timestamp = ca

		res = append(res, rls)
	}

	nextCursor := int64(0)
	if len(res) > 0 {
		lastID := res[len(res)-1].ID
		nextCursor = lastID
		//nextCursor, _ = strconv.ParseInt(lastID, 10, 64)
	}

	return res, nextCursor, nil
}

func (r *ReleaseRepo) Stats(ctx context.Context) (*domain.ReleaseStats, error) {
	query := `SELECT
    COUNT(*) total,
    sum(case when status = 'PUSH_APPROVED' then 1 else 0 end) push_approved_count,
    sum(case when status = 'PUSH_REJECTED' then 1 else 0 end) push_rejected_count,
    sum(case when status = 'FILTERED' then 1 else 0 end) filtered_count,
    sum(case when status = 'FILTER_REJECTED' then 1 else 0 end) filter_rejected_count
FROM "release";`

	row := r.db.QueryRowContext(ctx, query)
	if err := row.Err(); err != nil {
		log.Error().Stack().Err(err).Msg("release: %v : error query row")
		return nil, err
	}

	var rls domain.ReleaseStats

	if err := row.Scan(&rls.TotalCount, &rls.PushApprovedCount, &rls.PushRejectedCount, &rls.FilteredCount, &rls.FilterRejectedCount); err != nil {
		log.Error().Stack().Err(err).Msg("release: %v : error scanning data to struct")
		return nil, err
	}

	return &rls, nil
}
