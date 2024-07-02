// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

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

func (repo *ReleaseRepo) Store(ctx context.Context, r *domain.Release) error {
	codecStr := strings.Join(r.Codec, ",")
	hdrStr := strings.Join(r.HDR, ",")
	audioStr := r.AudioString()

	queryBuilder := repo.db.squirrel.
		Insert("release").
		Columns("filter_status", "rejections", "indexer", "filter", "protocol", "implementation", "timestamp", "group_id", "torrent_id", "info_url", "download_url", "torrent_name", "size", "title", "category", "season", "episode", "year", "month", "day", "resolution", "source", "codec", "container", "hdr", "audio", "audio_channels", "release_group", "proper", "repack", "website", "type", "origin", "tags", "uploader", "pre_time", "filter_id").
		Values(r.FilterStatus, pq.Array(r.Rejections), r.Indexer.Identifier, r.FilterName, r.Protocol, r.Implementation, r.Timestamp.Format(time.RFC3339), r.GroupID, r.TorrentID, r.InfoURL, r.DownloadURL, r.TorrentName, r.Size, r.Title, r.Category, r.Season, r.Episode, r.Year, r.Month, r.Day, r.Resolution, r.Source, codecStr, r.Container, hdrStr, audioStr, r.AudioChannels, r.Group, r.Proper, r.Repack, r.Website, r.Type, r.Origin, pq.Array(r.Tags), r.Uploader, r.PreTime, r.FilterID).
		Suffix("RETURNING id").RunWith(repo.db.handler)

	// return values
	var retID int64

	if err := queryBuilder.QueryRowContext(ctx).Scan(&retID); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.ID = retID

	repo.log.Debug().Msgf("release.store: %+v", r)

	return nil
}

func (repo *ReleaseRepo) StoreReleaseActionStatus(ctx context.Context, status *domain.ReleaseActionStatus) error {
	if status.ID != 0 {
		queryBuilder := repo.db.squirrel.
			Update("release_action_status").
			Set("status", status.Status).
			Set("rejections", pq.Array(status.Rejections)).
			Set("timestamp", status.Timestamp.Format(time.RFC3339)).
			Where(sq.Eq{"id": status.ID}).
			Where(sq.Eq{"release_id": status.ReleaseID})

		query, args, err := queryBuilder.ToSql()
		if err != nil {
			return errors.Wrap(err, "error building query")
		}

		if _, err = repo.db.handler.ExecContext(ctx, query, args...); err != nil {
			return errors.Wrap(err, "error executing query")
		}

	} else {
		queryBuilder := repo.db.squirrel.
			Insert("release_action_status").
			Columns("status", "action", "action_id", "type", "client", "filter", "filter_id", "rejections", "timestamp", "release_id").
			Values(status.Status, status.Action, status.ActionID, status.Type, status.Client, status.Filter, status.FilterID, pq.Array(status.Rejections), status.Timestamp.Format(time.RFC3339), status.ReleaseID).
			Suffix("RETURNING id").RunWith(repo.db.handler)

		// return values
		var retID int64

		if err := queryBuilder.QueryRowContext(ctx).Scan(&retID); err != nil {
			return errors.Wrap(err, "error executing query")
		}

		status.ID = retID
	}

	repo.log.Trace().Msgf("release.store_release_action_status: %+v", status)

	return nil
}

func (repo *ReleaseRepo) StoreDuplicateProfile(ctx context.Context, profile *domain.DuplicateReleaseProfile) error {
	if profile.ID == 0 {
		queryBuilder := repo.db.squirrel.
			Insert("release_profile_duplicate").
			Columns("name", "protocol", "release_name", "title", "season", "episode", "year", "month", "day", "resolution", "source", "codec", "container", "hdr", "audio", "release_group", "website", "proper", "repack").
			Values(profile.Name, profile.Protocol, profile.ReleaseName, profile.Title, profile.Season, profile.Episode, profile.Year, profile.Month, profile.Day, profile.Resolution, profile.Source, profile.Codec, profile.Container, profile.HDR, profile.Audio, profile.Group, profile.Website, profile.Proper, profile.Repack).
			Suffix("RETURNING id").
			RunWith(repo.db.handler)

		// return values
		var retID int64

		err := queryBuilder.QueryRowContext(ctx).Scan(&retID)
		if err != nil {
			return errors.Wrap(err, "error executing query")
		}

		profile.ID = retID
	} else {
		queryBuilder := repo.db.squirrel.
			Update("release_profile_duplicate").
			Set("name", profile.Name).
			Set("protocol", profile.Protocol).
			Set("release_name", profile.ReleaseName).
			Set("title", profile.Title).
			Set("season", profile.Season).
			Set("episode", profile.Episode).
			Set("year", profile.Year).
			Set("month", profile.Month).
			Set("day", profile.Day).
			Set("resolution", profile.Resolution).
			Set("source", profile.Source).
			Set("codec", profile.Codec).
			Set("container", profile.Container).
			Set("hdr", profile.HDR).
			Set("audio", profile.Audio).
			Set("release_group", profile.Group).
			Set("website", profile.Website).
			Set("proper", profile.Proper).
			Set("repack", profile.Repack).
			Where(sq.Eq{"id": profile.ID}).
			RunWith(repo.db.handler)

		_, err := queryBuilder.ExecContext(ctx)
		if err != nil {
			return errors.Wrap(err, "error executing query")
		}
	}

	repo.log.Debug().Msgf("release.StoreDuplicateProfile: %+v", profile)

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
			r := regexp.MustCompile(fmt.Sprintf(`(?i)(?:%s:)(?P<value>'.*?'|".*?"|\S+)`, k))
			if reskey := r.FindAllStringSubmatch(search, -1); len(reskey) != 0 {
				filter := sq.Or{}
				for _, found := range reskey {
					filter = append(filter, repo.db.ILike(v, strings.ReplaceAll(strings.Trim(strings.Trim(found[1], `"`), `'`), ".", "_")+"%"))
				}

				if len(filter) == 0 {
					continue
				}

				whereQueryBuilder = append(whereQueryBuilder, filter)
				search = strings.TrimSpace(r.ReplaceAllLiteralString(search, ""))
			}
		}

		if len(search) != 0 {
			if len(whereQueryBuilder) > 1 {
				whereQueryBuilder = append(whereQueryBuilder, repo.db.ILike("r.torrent_name", "%"+search+"%"))
			} else {
				whereQueryBuilder = append(whereQueryBuilder, repo.db.ILike("r.torrent_name", search+"%"))
			}
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

		// using sq.Eq for countQuery breaks search with Postgres.
		countQuery = countQuery.InnerJoin("release_action_status ras ON r.id = ras.release_id").Where("ras.status = '" + params.Filters.PushStatus + `'`)
	}

	subQuery, subArgs, err := subQueryBuilder.ToSql()
	if err != nil {
		return nil, 0, 0, errors.Wrap(err, "error building subquery")
	}

	queryBuilder := repo.db.squirrel.
		Select("r.id", "r.filter_status", "r.rejections", "r.indexer", "r.filter", "r.protocol", "r.info_url", "r.download_url", "r.title", "r.torrent_name", "r.size", "r.category", "r.season", "r.episode", "r.year", "r.resolution", "r.source", "r.codec", "r.container", "r.release_group", "r.timestamp",
			"ras.id", "ras.status", "ras.action", "ras.action_id", "ras.type", "ras.client", "ras.filter", "ras.filter_id", "ras.release_id", "ras.rejections", "ras.timestamp").
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

		var rlsindexer, rlsfilter, infoUrl, downloadUrl, codec sql.Null[string]

		var rasId, rasFilterId, rasReleaseId, rasActionId sql.Null[int64]
		var rasStatus, rasAction, rasType, rasClient, rasFilter sql.Null[string]
		var rasRejections []sql.Null[string]
		var rasTimestamp sql.NullTime

		if err := rows.Scan(&rls.ID, &rls.FilterStatus, pq.Array(&rls.Rejections), &rlsindexer, &rlsfilter, &rls.Protocol, &infoUrl, &downloadUrl, &rls.Title, &rls.TorrentName, &rls.Size, &rls.Category, &rls.Season, &rls.Episode, &rls.Year, &rls.Resolution, &rls.Source, &codec, &rls.Container, &rls.Group, &rls.Timestamp, &rasId, &rasStatus, &rasAction, &rasActionId, &rasType, &rasClient, &rasFilter, &rasFilterId, &rasReleaseId, pq.Array(&rasRejections), &rasTimestamp, &countItems); err != nil {
			return res, 0, 0, errors.Wrap(err, "error scanning row")
		}

		//for _, codec := range codecs {
		//	rls.Codec = append(rls.Codec, codec.V)
		//
		//}

		ras.ID = rasId.V
		ras.Status = domain.ReleasePushStatus(rasStatus.V)
		ras.Action = rasAction.V
		ras.ActionID = rasActionId.V
		ras.Type = domain.ActionType(rasType.V)
		ras.Client = rasClient.V
		ras.Filter = rasFilter.V
		ras.FilterID = rasFilterId.V
		ras.Timestamp = rasTimestamp.Time
		ras.ReleaseID = rasReleaseId.V
		ras.Rejections = []string{}

		for _, rejection := range rasRejections {
			ras.Rejections = append(ras.Rejections, rejection.V)
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

		rls.Indexer.Identifier = rlsindexer.V
		rls.FilterName = rlsfilter.V
		rls.ActionStatus = make([]domain.ReleaseActionStatus, 0)
		rls.InfoURL = infoUrl.V
		rls.DownloadURL = downloadUrl.V
		rls.Codec = strings.Split(codec.V, ",")

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

func (repo *ReleaseRepo) FindDuplicateReleaseProfiles(ctx context.Context) ([]*domain.DuplicateReleaseProfile, error) {
	queryBuilder := repo.db.squirrel.
		Select(
			"id",
			"name",
			"protocol",
			"release_name",
			"title",
			"year",
			"month",
			"day",
			"source",
			"resolution",
			"codec",
			"container",
			"hdr",
			"audio",
			"release_group",
			"season",
			"episode",
			"website",
			"proper",
			"repack",
		).
		From("release_profile_duplicate")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := repo.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error rows FindDuplicateReleaseProfiles")
	}

	res := make([]*domain.DuplicateReleaseProfile, 0)

	for rows.Next() {
		var p domain.DuplicateReleaseProfile

		err := rows.Scan(&p.ID, &p.Name, &p.Protocol, &p.ReleaseName, &p.Title, &p.Year, &p.Month, &p.Day, &p.Source, &p.Resolution, &p.Codec, &p.Container, &p.HDR, &p.Audio, &p.Group, &p.Season, &p.Episode, &p.Website, &p.Proper, &p.Repack)
		if err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		res = append(res, &p)
	}

	return res, nil
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
		Select("id", "status", "action", "action_id", "type", "client", "filter", "release_id", "rejections", "timestamp").
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

		var client, filter sql.Null[string]
		var actionId sql.Null[int64]

		if err := rows.Scan(&rls.ID, &rls.Status, &rls.Action, &actionId, &rls.Type, &client, &filter, &rls.ReleaseID, pq.Array(&rls.Rejections), &rls.Timestamp); err != nil {
			return res, errors.Wrap(err, "error scanning row")
		}

		rls.ActionID = actionId.V
		rls.Client = client.V
		rls.Filter = filter.V

		res = append(res, rls)
	}

	return res, nil
}

func (repo *ReleaseRepo) Get(ctx context.Context, req *domain.GetReleaseRequest) (*domain.Release, error) {
	queryBuilder := repo.db.squirrel.
		Select("r.id", "r.filter_status", "r.rejections", "r.indexer", "r.filter", "r.filter_id", "r.protocol", "r.implementation", "r.info_url", "r.download_url", "r.title", "r.torrent_name", "r.category", "r.size", "r.group_id", "r.torrent_id", "r.uploader", "r.timestamp").
		From("release r").
		OrderBy("r.id DESC").
		Where(sq.Eq{"r.id": req.Id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	repo.log.Trace().Str("database", "release.find").Msgf("query: '%s', args: '%v'", query, args)

	row := repo.db.handler.QueryRowContext(ctx, query, args...)
	if row.Err() != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error rows find release")
	}

	var rls domain.Release

	var indexerName, filterName, infoUrl, downloadUrl, groupId, torrentId, category, uploader sql.Null[string]
	var filterId sql.Null[int64]

	if err := row.Scan(&rls.ID, &rls.FilterStatus, pq.Array(&rls.Rejections), &indexerName, &filterName, &filterId, &rls.Protocol, &rls.Implementation, &infoUrl, &downloadUrl, &rls.Title, &rls.TorrentName, &category, &rls.Size, &groupId, &torrentId, &uploader, &rls.Timestamp); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "error scanning row")
	}

	rls.Indexer.Identifier = indexerName.V
	rls.FilterName = filterName.V
	rls.FilterID = filterId.V
	rls.ActionStatus = make([]domain.ReleaseActionStatus, 0)
	rls.InfoURL = infoUrl.V
	rls.DownloadURL = downloadUrl.V
	rls.Category = category.V
	rls.GroupID = groupId.V
	rls.TorrentID = torrentId.V
	rls.Uploader = uploader.V

	return &rls, nil
}

func (repo *ReleaseRepo) GetActionStatus(ctx context.Context, req *domain.GetReleaseActionStatusRequest) (*domain.ReleaseActionStatus, error) {
	queryBuilder := repo.db.squirrel.
		Select("id", "status", "action", "action_id", "type", "client", "filter", "filter_id", "release_id", "rejections", "timestamp").
		From("release_action_status").
		Where(sq.Eq{"id": req.Id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := repo.db.handler.QueryRowContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	if err := row.Err(); err != nil {
		repo.log.Error().Stack().Err(err)
		return nil, err
	}

	var rls domain.ReleaseActionStatus

	var client, filter sql.Null[string]
	var actionId, filterId sql.Null[int64]

	if err := row.Scan(&rls.ID, &rls.Status, &rls.Action, &actionId, &rls.Type, &client, &filter, &filterId, &rls.ReleaseID, pq.Array(&rls.Rejections), &rls.Timestamp); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	rls.ActionID = actionId.V
	rls.Client = client.V
	rls.Filter = filter.V
	rls.FilterID = filterId.V

	return &rls, nil
}

func (repo *ReleaseRepo) attachActionStatus(ctx context.Context, tx *Tx, releaseID int64) ([]domain.ReleaseActionStatus, error) {
	queryBuilder := repo.db.squirrel.
		Select("id", "status", "action", "action_id", "type", "client", "filter", "filter_id", "release_id", "rejections", "timestamp").
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

		var client, filter sql.Null[string]
		var actionId, filterID sql.Null[int64]

		if err := rows.Scan(&rls.ID, &rls.Status, &rls.Action, &actionId, &rls.Type, &client, &filter, &filterID, &rls.ReleaseID, pq.Array(&rls.Rejections), &rls.Timestamp); err != nil {
			return res, errors.Wrap(err, "error scanning row")
		}

		rls.ActionID = actionId.V
		rls.Client = client.V
		rls.Filter = filter.V
		rls.FilterID = filterID.V

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
	COUNT(CASE WHEN status = 'PUSH_REJECTED' THEN 0 END) AS push_rejected_count,
	COUNT(CASE WHEN status = 'PUSH_ERROR' THEN 0 END) AS push_error_count
	FROM release_action_status
) AS foo`

	row := repo.db.handler.QueryRowContext(ctx, query)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var rls domain.ReleaseStats

	if err := row.Scan(&rls.TotalCount, &rls.FilteredCount, &rls.FilterRejectedCount, &rls.PushApprovedCount, &rls.PushRejectedCount, &rls.PushErrorCount); err != nil {
		return nil, errors.Wrap(err, "error scanning row")
	}

	return &rls, nil
}

func (repo *ReleaseRepo) Delete(ctx context.Context, req *domain.DeleteReleaseRequest) error {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "could not start transaction")
	}

	defer func() {
		var txErr error
		if p := recover(); p != nil {
			txErr = tx.Rollback()
			if txErr != nil {
				repo.log.Error().Err(txErr).Msg("error rolling back transaction")
			}
			repo.log.Error().Msgf("something went terribly wrong panic: %v", p)
		} else if err != nil {
			txErr = tx.Rollback()
			if txErr != nil {
				repo.log.Error().Err(txErr).Msg("error rolling back transaction")
			}
		} else {
			// All good, commit
			txErr = tx.Commit()
			if txErr != nil {
				repo.log.Error().Err(txErr).Msg("error committing transaction")
			}
		}
	}()

	qb := repo.db.squirrel.Delete("release")

	if req.OlderThan > 0 {
		if repo.db.Driver == "sqlite" {
			qb = qb.Where(fmt.Sprintf("timestamp < strftime('%%Y-%%m-%%dT%%H:00:00', datetime('now','-%d hours'))", req.OlderThan))
		} else {
			// postgres compatible
			thresholdTime := time.Now().Add(time.Duration(-req.OlderThan) * time.Hour)
			qb = qb.Where(sq.Lt{
				//"timestamp": fmt.Sprintf("(now() - interval '%d hours')", req.OlderThan),
				"timestamp": thresholdTime,
			})
		}
	}

	if len(req.Indexers) > 0 {
		qb = qb.Where(sq.Eq{"indexer": req.Indexers})
	}

	if len(req.ReleaseStatuses) > 0 {
		subQuery := sq.Select("release_id").From("release_action_status").Where(sq.Eq{"status": req.ReleaseStatuses})
		subQueryText, subQueryArgs, err := subQuery.ToSql()
		if err != nil {
			return errors.Wrap(err, "error building subquery")
		}
		qb = qb.Where("id IN ("+subQueryText+")", subQueryArgs...)
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building SQL query")
	}

	repo.log.Trace().Str("query", query).Interface("args", args).Msg("Executing combined delete query")

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		repo.log.Error().Err(err).Str("query", query).Interface("args", args).Msg("Error executing combined delete query")
		return errors.Wrap(err, "error executing delete query")
	}

	deletedRows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error fetching rows affected")
	}

	repo.log.Debug().Msgf("deleted %d rows from release table", deletedRows)

	// clean up orphaned rows
	orphanedResult, err := tx.ExecContext(ctx, `DELETE FROM release_action_status WHERE release_id NOT IN (SELECT id FROM "release")`)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	deletedRowsOrphaned, err := orphanedResult.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error fetching rows affected")
	}

	repo.log.Debug().Msgf("deleted %d orphaned rows from release table", deletedRowsOrphaned)

	return nil
}

func (repo *ReleaseRepo) DeleteReleaseProfileDuplicate(ctx context.Context, id int64) error {
	qb := repo.db.squirrel.Delete("release_profile_duplicate").Where(sq.Eq{"id": id})

	query, args, err := qb.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building SQL query")
	}

	_, err = repo.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing delete query")
	}

	//deletedRows, err := result.RowsAffected()
	//if err != nil {
	//	return errors.Wrap(err, "error fetching rows affected")
	//}
	//
	//repo.log.Debug().Msgf("deleted %d rows from release table", deletedRows)

	repo.log.Debug().Msgf("deleted duplicate release profile: %d", id)

	return nil
}

func (repo *ReleaseRepo) CheckSmartEpisodeCanDownload(ctx context.Context, p *domain.SmartEpisodeParams) (bool, error) {
	queryBuilder := repo.db.squirrel.
		Select("COUNT(*)").
		From("release r").
		LeftJoin("release_action_status ras ON r.id = ras.release_id").
		Where(sq.And{
			repo.db.ILike("r.title", p.Title+"%"),
			sq.Eq{"ras.status": "PUSH_APPROVED"},
		})

	if p.Proper {
		queryBuilder = queryBuilder.Where(sq.Eq{"r.proper": p.Proper})
	}
	if p.Repack {
		queryBuilder = queryBuilder.Where(sq.And{
			sq.Eq{"r.repack": p.Repack},
			repo.db.ILike("r.release_group", p.Group),
		})
	}

	if p.Season > 0 && p.Episode > 0 {
		queryBuilder = queryBuilder.Where(sq.Or{
			sq.And{
				sq.Eq{"r.season": p.Season},
				sq.Gt{"r.episode": p.Episode},
			},
			sq.Gt{"r.season": p.Season},
		})
	} else if p.Season > 0 && p.Episode == 0 {
		queryBuilder = queryBuilder.Where(sq.Gt{"r.season": p.Season})
	} else if p.Year > 0 && p.Month > 0 && p.Day > 0 {
		queryBuilder = queryBuilder.Where(sq.Or{
			sq.And{
				sq.Eq{"r.year": p.Year},
				sq.Eq{"r.month": p.Month},
				sq.Gt{"r.day": p.Day},
			},
			sq.And{
				sq.Eq{"r.year": p.Year},
				sq.Gt{"r.month": p.Month},
			},
			sq.Gt{"r.year": p.Year},
		})
	} else {
		/* No support for this scenario today. Specifically multi-part specials.
		 * The Database presently does not have Subtitle as a field, but is coming at a future date. */
		return true, nil
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return false, errors.Wrap(err, "error building query")
	}

	repo.log.Trace().Str("method", "CheckSmartEpisodeCanDownload").Str("query", query).Interface("args", args).Msgf("executing query")

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

func (repo *ReleaseRepo) UpdateBaseURL(ctx context.Context, indexer string, oldBaseURL, newBaseURL string) error {
	tx, err := repo.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		var txErr error
		if p := recover(); p != nil {
			txErr = tx.Rollback()
			if txErr != nil {
				repo.log.Error().Err(txErr).Msg("error rolling back transaction")
			}
			repo.log.Error().Msgf("something went terribly wrong panic: %v", p)
		} else if err != nil {
			txErr = tx.Rollback()
			if txErr != nil {
				repo.log.Error().Err(txErr).Msg("error rolling back transaction")
			}
		} else {
			// All good, commit
			txErr = tx.Commit()
			if txErr != nil {
				repo.log.Error().Err(txErr).Msg("error committing transaction")
			}
		}
	}()

	queryBuilder := repo.db.squirrel.
		RunWith(tx).
		Update("release").
		Set("download_url", sq.Expr("REPLACE(download_url, ?, ?)", oldBaseURL, newBaseURL)).
		Set("info_url", sq.Expr("REPLACE(info_url, ?, ?)", oldBaseURL, newBaseURL)).
		Where(sq.Eq{"indexer": indexer})

	result, err := queryBuilder.ExecContext(ctx)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error getting rows affected")
	}

	repo.log.Trace().Msgf("release updated (%d) base urls from %q to %q", rowsAffected, oldBaseURL, newBaseURL)

	return nil
}

func (repo *ReleaseRepo) CheckIsDuplicateRelease(ctx context.Context, profile *domain.DuplicateReleaseProfile, release *domain.ReleaseNormalized) (bool, error) {
	queryBuilder := repo.db.squirrel.
		Select("r.id, r.torrent_name, r.title, ras.action, ras.status").
		From("release r").
		LeftJoin("release_action_status ras ON r.id = ras.release_id").
		Where("ras.status = 'PUSH_APPROVED'")

	if profile.Title {
		queryBuilder = queryBuilder.Where(sq.Eq{"LOWER(r.title)": release.Title})
	}

	if profile.ReleaseName {
		queryBuilder = queryBuilder.Where(sq.Eq{"LOWER(r.torrent_name)": release.TorrentName})
	}

	if profile.Year {
		queryBuilder = queryBuilder.Where(sq.Eq{"r.year": release.Year})
	}

	if profile.Month {
		queryBuilder = queryBuilder.Where(sq.Eq{"r.month": release.Month})
	}

	if profile.Day {
		queryBuilder = queryBuilder.Where(sq.Eq{"r.day": release.Day})
	}

	if profile.Source {
		queryBuilder = queryBuilder.Where(sq.Eq{"LOWER(r.source)": release.Source})
	}

	if profile.Container {
		queryBuilder = queryBuilder.Where(sq.Eq{"LOWER(r.container)": release.Container})
	}

	if profile.Codec {
		var and sq.And
		for _, codec := range release.Codec {
			and = append(and, repo.db.ILike("r.codec", "%"+codec+"%"))
		}
		queryBuilder = queryBuilder.Where(and)
	}

	if profile.Resolution {
		queryBuilder = queryBuilder.Where(sq.Eq{"LOWER(r.resolution)": release.Resolution})
	}

	if profile.HDR {
		var and sq.And
		for _, hdr := range release.HDR {
			and = append(and, repo.db.ILike("r.hdr", "%"+hdr+"%"))
		}
		queryBuilder = queryBuilder.Where(and)
	}

	if profile.Audio {
		queryBuilder = queryBuilder.Where(sq.Eq{"r.audio": release.Audio})
	}

	if profile.Group {
		queryBuilder = queryBuilder.Where(sq.Eq{"LOWER(r.release_group)": release.Group})
	}

	if profile.Season {
		queryBuilder = queryBuilder.Where(sq.Eq{"r.season": release.Season})
	}

	if profile.Episode {
		queryBuilder = queryBuilder.Where(sq.Eq{"r.episode": release.Episode})
	}

	if profile.Website {
		queryBuilder = queryBuilder.Where(sq.Eq{"r.website": release.Website})
	}

	if profile.Proper {
		queryBuilder = queryBuilder.Where(sq.Eq{"r.proper": release.Proper})
	}

	if profile.Repack {
		//queryBuilder = queryBuilder.Where(sq.Eq{"r.repack": release.Repack})
		queryBuilder = queryBuilder.Where(sq.And{
			sq.Eq{"r.repack": release.Repack},
			sq.Eq{"LOWER(r.release_group)": release.Group},
		})
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return false, errors.Wrap(err, "error building query")
	}

	repo.log.Trace().Str("database", "release.FindDuplicateReleases").Msgf("query: %q, args: %q", query, args)

	rows, err := repo.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return false, err
	}

	if err := rows.Err(); err != nil {
		return false, errors.Wrap(err, "error rows CheckIsDuplicateRelease")
	}

	type result struct {
		id      int
		release string
		title   string
		action  string
		status  string
	}

	var res []result

	for rows.Next() {
		r := result{}
		if err := rows.Scan(&r.id, &r.release, &r.title, &r.action, &r.status); err != nil {
			return false, errors.Wrap(err, "error scan CheckIsDuplicateRelease")
		}
		res = append(res, r)
	}

	repo.log.Trace().Str("database", "release.FindDuplicateReleases").Msgf("found duplicate releases: %+v", res)

	if len(res) == 0 {
		return false, nil
	}

	return true, nil
}
