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
	db *sql.DB
}

func NewReleaseRepo(db *sql.DB) domain.ReleaseRepo {
	return &ReleaseRepo{db: db}
}

func (r *ReleaseRepo) Store(release domain.Release) (*domain.Release, error) {
	res, err := r.db.Exec(`INSERT INTO "release"(status, rejections, indexer, client, protocol, title, size) VALUES (?, ?, ?, ?, ? ,? ,?) ON CONFLICT DO NOTHING`, release.Status, pq.Array(release.Rejections), release.Indexer, release.Client, release.Protocol, release.Title, release.Size)
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

	queryBuilder := sq.Select("id", "status", "rejections", "indexer", "client", "protocol", "title", "size", "created_at").From("release").OrderBy("created_at DESC")

	if params.Limit > 0 {
		queryBuilder = queryBuilder.Limit(params.Limit)
	} else {
		queryBuilder = queryBuilder.Limit(20)
	}

	if params.Cursor > 0 {
		queryBuilder = queryBuilder.Where(sq.Gt{"id": params.Cursor})
	}

	if params.Filter != nil {
		filter := sq.And{}
		for k, v := range params.Filter {
			filter = append(filter, sq.Eq{k: v})
		}

		queryBuilder = queryBuilder.Where(filter)
	}

	query, args, err := queryBuilder.ToSql()

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

	//releases := make([]domain.Release, 0)
	res := make([]domain.Release, 0)
	//res := []domain.Release{}
	for rows.Next() {
		var rls domain.Release

		if err := rows.Scan(&rls.ID, &rls.Status, pq.Array(&rls.Rejections), &rls.Indexer, &rls.Client, &rls.Protocol, &rls.Title, &rls.Size, &rls.CreatedAt); err != nil {
			log.Error().Stack().Err(err)
			return nil, 0, err
			//return
		}

		//releases = append(releases, rls)
		res = append(res, rls)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
		//return
	}

	nextCursor := int64(0)
	if len(res) > 0 {
		lastID := res[len(res)-1].ID
		nextCursor = lastID
		//nextCursor, _ = strconv.ParseInt(lastID, 10, 64)
	}

	//return releases, nil
	return res, nextCursor, nil
	//return
}
