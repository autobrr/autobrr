// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

type ReleaseRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewReleaseRepo(log zerolog.Logger, db *DB) *ReleaseRepo {
	return &ReleaseRepo{
		log: log.With().Str("repo", "release").Logger(),
		db:  db,
	}
}

func (repo *ReleaseRepo) Store(ctx context.Context, r *domain.Release) error {
	var (
		codecStr    = strings.Join(r.Codec, ",")
		hdrStr      = strings.Join(r.HDR, ",")
		audioStr    = strings.Join(r.Audio, ",")
		editionStr  = strings.Join(r.Edition, ",")
		cutStr      = strings.Join(r.Cut, ",")
		languageStr = strings.Join(r.Language, ",")
	)

	var downloadVolumeFactor sql.Null[float64]
	if factor, ok := r.DownloadVolumeFactor(); ok {
		downloadVolumeFactor = sql.Null[float64]{V: factor, Valid: true}
	}

	var publishTime sql.Null[time.Time]
	if !r.PublishDate.IsZero() {
		publishTime = sql.Null[time.Time]{V: r.PublishDate, Valid: true}
	}

	var metadata sql.Null[string]
	if data, err := json.Marshal(r.Metadata()); err != nil {
		return errors.Wrap(err, "could not marshal release metadata")
	} else if string(data) != "{}" {
		metadata = sql.Null[string]{V: string(data), Valid: true}
	}

	categories := r.Categories
	if categories == nil {
		categories = []string{}
	}

	indexerFlags := r.IndexerFlags
	if indexerFlags == nil {
		indexerFlags = []string{}
	}

	queryBuilder := repo.db.squirrel.
		Insert("release").
		Columns("indexer", "indexer_id", "filter", "protocol", "implementation", "timestamp", "announce_type", "group_id", "external_id", "info_url", "download_url", "release_name", "normalized_hash", "name_key", "size", "title", "title_key", "sub_title", "sub_title_key", "category", "categories", "season", "episode", "year", "month", "day", "resolution", "source", "codec", "container", "dynamic_range", "audio", "audio_channels", "release_group", "proper", "repack", "region", "language", "cut", "edition", "hybrid", "media_processing", "website", "type", "origin", "tags", "uploader", "pre_time", "other", "indexer_flags", "download_volume_factor", "publish_time", "id_imdb", "id_tmdb", "id_tvdb", "id_igdb", "info_hash", "trace_id", "metadata", "filter_id").
		Values(r.Indexer.Identifier, toNullInt64(int64(r.Indexer.ID)), r.FilterName, r.Protocol, r.Implementation, r.Timestamp, r.AnnounceType, r.GroupID, r.TorrentID, r.InfoURL, r.DownloadURL, r.TorrentName, r.NormalizedHash, domain.NameKey(r.TorrentName), r.Size, r.Title, domain.TitleKey(r.Title), r.SubTitle, domain.TitleKey(r.SubTitle), r.Category, pq.Array(categories), r.Season, r.Episode, r.Year, r.Month, r.Day, r.Resolution, r.Source, codecStr, r.Container, hdrStr, audioStr, r.AudioChannels, r.Group, r.Proper, r.Repack, r.Region, languageStr, cutStr, editionStr, r.Hybrid, r.MediaProcessing, r.Website, r.Type.String(), r.Origin, pq.Array(r.Tags), r.Uploader, r.PreTime, pq.Array(r.Other), pq.Array(indexerFlags), downloadVolumeFactor, publishTime, toNullString(r.MetaIMDB), toNullInt64(int64(r.MetaTMDB)), toNullInt64(int64(r.MetaTVDB)), toNullInt64(int64(r.MetaIGDB)), toNullString(r.TorrentHash), toNullString(r.TraceID), metadata, toNullInt64(int64(r.FilterID))).
		Suffix("RETURNING id").RunWith(repo.db.Handler)

	q, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	repo.log.Debug().Str("query", q).Interface("args", args).Msg("store release")

	if err := queryBuilder.QueryRowContext(ctx).Scan(&r.ID); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	repo.log.Debug().Interface("release", r).Msg("release created")

	return nil
}

func (repo *ReleaseRepo) Update(ctx context.Context, r *domain.Release) error {
	queryBuilder := repo.db.squirrel.
		Update("release").
		Set("size", r.Size).
		Set("info_hash", toNullString(r.TorrentHash)).
		Where(sq.Eq{"id": r.ID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err = repo.db.Handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	repo.log.Debug().Int64("release_id", r.ID).Str("release", r.TorrentName).Msg("release updated")

	return nil
}

func (repo *ReleaseRepo) StoreReleaseActionStatus(ctx context.Context, status *domain.ReleaseActionStatus) error {
	if status.ID != 0 {
		queryBuilder := repo.db.squirrel.
			Update("release_action_status").
			Set("status", status.Status).
			Set("rejections", pq.Array(status.Rejections)).
			Set("timestamp", status.Timestamp).
			Where(sq.Eq{"id": status.ID}).
			Where(sq.Eq{"release_id": status.ReleaseID})

		query, args, err := queryBuilder.ToSql()
		if err != nil {
			return errors.Wrap(err, "error building query")
		}

		if _, err = repo.db.Handler.ExecContext(ctx, query, args...); err != nil {
			return errors.Wrap(err, "error executing query")
		}

	} else {
		queryBuilder := repo.db.squirrel.
			Insert("release_action_status").
			Columns("status", "action", "action_id", "type", "client", "filter", "filter_id", "rejections", "timestamp", "release_id").
			Values(status.Status, status.Action, status.ActionID, status.Type, status.Client, status.Filter, status.FilterID, pq.Array(status.Rejections), status.Timestamp, status.ReleaseID).
			Suffix("RETURNING id").RunWith(repo.db.Handler)

		if err := queryBuilder.QueryRowContext(ctx).Scan(&status.ID); err != nil {
			return errors.Wrap(err, "error executing query")
		}
	}

	repo.log.Trace().Interface("status", status).Msg("release action status created")

	return nil
}

func (repo *ReleaseRepo) StoreDuplicateProfile(ctx context.Context, profile *domain.DuplicateReleaseProfile) error {
	if profile.ID == 0 {
		queryBuilder := repo.db.squirrel.
			Insert("release_profile_duplicate").
			Columns("name", "protocol", "release_name", "hash", "title", "sub_title", "season", "episode", "year", "month", "day", "resolution", "source", "codec", "container", "dynamic_range", "audio", "release_group", "website", "proper", "repack", "hybrid", "edition", "language", "media_processing").
			Values(profile.Name, profile.Protocol, profile.ReleaseName, profile.Hash, profile.Title, profile.SubTitle, profile.Season, profile.Episode, profile.Year, profile.Month, profile.Day, profile.Resolution, profile.Source, profile.Codec, profile.Container, profile.DynamicRange, profile.Audio, profile.Group, profile.Website, profile.Proper, profile.Repack, profile.Hybrid, profile.Edition, profile.Language, profile.MediaProcessing).
			Suffix("RETURNING id").
			RunWith(repo.db.Handler)

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
			Set("hash", profile.Hash).
			Set("title", profile.Title).
			Set("sub_title", profile.SubTitle).
			Set("season", profile.Season).
			Set("episode", profile.Episode).
			Set("year", profile.Year).
			Set("month", profile.Month).
			Set("day", profile.Day).
			Set("resolution", profile.Resolution).
			Set("source", profile.Source).
			Set("codec", profile.Codec).
			Set("container", profile.Container).
			Set("dynamic_range", profile.DynamicRange).
			Set("audio", profile.Audio).
			Set("release_group", profile.Group).
			Set("website", profile.Website).
			Set("proper", profile.Proper).
			Set("repack", profile.Repack).
			Set("hybrid", profile.Hybrid).
			Set("edition", profile.Edition).
			Set("language", profile.Language).
			Set("media_processing", profile.MediaProcessing).
			Where(sq.Eq{"id": profile.ID}).
			RunWith(repo.db.Handler)

		_, err := queryBuilder.ExecContext(ctx)
		if err != nil {
			return errors.Wrap(err, "error executing query")
		}
	}

	repo.log.Debug().Interface("profile", profile).Msg("duplicate profile created")

	return nil
}

func (repo *ReleaseRepo) Find(ctx context.Context, params domain.ReleaseQueryParams) (*domain.FindReleasesResponse, error) {
	tx, err := repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, errors.Wrap(err, "error begin transaction")
	}
	defer tx.Rollback()

	resp, err := repo.findReleases(ctx, tx, params)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

var reservedSearch = map[string]*regexp.Regexp{
	"r.title":         regexp.MustCompile(`(?i)(?:` + `title` + `:)(?P<value>'.*?'|".*?"|\S+)`),
	"r.release_group": regexp.MustCompile(`(?i)(?:` + `group` + `:)(?P<value>'.*?'|".*?"|\S+)`),
	"r.category":      regexp.MustCompile(`(?i)(?:` + `category` + `:)(?P<value>'.*?'|".*?"|\S+)`),
	"r.season":        regexp.MustCompile(`(?i)(?:` + `season` + `:)(?P<value>'.*?'|".*?"|\S+)`),
	"r.episode":       regexp.MustCompile(`(?i)(?:` + `episode` + `:)(?P<value>'.*?'|".*?"|\S+)`),
	"r.year":          regexp.MustCompile(`(?i)(?:` + `year` + `:)(?P<value>'.*?'|".*?"|\S+)`),
	"r.resolution":    regexp.MustCompile(`(?i)(?:` + `resolution` + `:)(?P<value>'.*?'|".*?"|\S+)`),
	"r.source":        regexp.MustCompile(`(?i)(?:` + `source` + `:)(?P<value>'.*?'|".*?"|\S+)`),
	"r.codec":         regexp.MustCompile(`(?i)(?:` + `codec` + `:)(?P<value>'.*?'|".*?"|\S+)`),
	"r.dynamic_range": regexp.MustCompile(`(?i)(?:` + `hdr` + `:)(?P<value>'.*?'|".*?"|\S+)`),
	"r.type":          regexp.MustCompile(`(?i)(?:` + `type` + `:)(?P<value>'.*?'|".*?"|\S+)`),
	"r.filter":        regexp.MustCompile(`(?i)(?:` + `filter` + `:)(?P<value>'.*?'|".*?"|\S+)`),
}

func (repo *ReleaseRepo) findReleases(ctx context.Context, tx *Tx, params domain.ReleaseQueryParams) (*domain.FindReleasesResponse, error) {
	whereQueryBuilder := sq.And{}
	if params.Cursor > 0 {
		whereQueryBuilder = append(whereQueryBuilder, sq.Lt{"r.id": params.Cursor})
	}

	if params.Search != "" {
		search := strings.TrimSpace(params.Search)
		for dbField, regex := range reservedSearch {
			if reskey := regex.FindAllStringSubmatch(search, -1); len(reskey) != 0 {
				filter := sq.Or{}
				for _, found := range reskey {
					filter = append(filter, repo.db.ILike(dbField, strings.ReplaceAll(strings.Trim(strings.Trim(found[1], `"`), `'`), ".", "_")+"%"))
				}

				if len(filter) == 0 {
					continue
				}

				whereQueryBuilder = append(whereQueryBuilder, filter)
				search = strings.TrimSpace(regex.ReplaceAllLiteralString(search, ""))
			}
		}

		if len(search) != 0 {
			if len(whereQueryBuilder) > 1 {
				whereQueryBuilder = append(whereQueryBuilder, repo.db.ILike("r.release_name", "%"+search+"%"))
			} else {
				whereQueryBuilder = append(whereQueryBuilder, repo.db.ILike("r.release_name", search+"%"))
			}
		}
	}

	if params.Filters.Indexers != nil {
		filter := sq.Or{}
		for _, v := range params.Filters.Indexers {
			filter = append(filter, sq.Eq{"r.indexer": v})
		}

		if len(filter) > 0 {
			whereQueryBuilder = append(whereQueryBuilder, filter)
		}
	}

	whereQuery, _, err := whereQueryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building where query")
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
		return nil, errors.Wrap(err, "error building subquery")
	}

	queryBuilder := repo.db.squirrel.
		Select(
			"r.id",
			"r.indexer",
			"i.id",
			"COALESCE(i.name, d.name, r.indexer) AS indexer_name",
			"i.identifier_external",
			"r.filter",
			"r.protocol",
			"r.announce_type",
			"r.info_url",
			"r.download_url",
			"r.title",
			"r.sub_title",
			"r.release_name",
			"r.normalized_hash",
			"r.size",
			"r.category",
			"r.season",
			"r.episode",
			"r.year",
			"r.resolution",
			"r.source",
			"r.codec",
			"r.container",
			"r.dynamic_range",
			"r.audio",
			"r.audio_channels",
			"r.release_group",
			"r.region",
			"r.language",
			"r.edition",
			"r.cut",
			"r.hybrid",
			"r.proper",
			"r.repack",
			"r.website",
			"r.media_processing",
			"r.type",
			"r.timestamp",
			"ras.id", "ras.status", "ras.action", "ras.action_id", "ras.type", "ras.client", "ras.filter", "ras.filter_id", "ras.release_id", "ras.rejections", "ras.timestamp",
		).
		Column(sq.Alias(countQuery, "page_total")).
		From("release r").
		OrderBy("r.id DESC").
		Where("r.id IN ("+subQuery+")", subArgs...).
		LeftJoin("release_action_status ras ON r.id = ras.release_id").
		LeftJoin("indexer i ON r.indexer = i.identifier").
		LeftJoin("indexer_deprecation d ON r.indexer = d.identifier")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	repo.log.Trace().Str("database", "release.find").Str("query", query).Interface("args", args).Msg("find releases")

	resp := &domain.FindReleasesResponse{
		Data:       make([]*domain.Release, 0),
		TotalCount: 0,
		NextCursor: 0,
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return resp, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		return resp, errors.Wrap(err, "error rows findreleases")
	}

	for rows.Next() {
		var rls domain.Release
		var ras domain.ReleaseActionStatus

		var rlsIndexer, rlsIndexerName, rlsIndexerExternalName, rlsFilter, rlsAnnounceType, infoUrl, downloadUrl, subTitle, normalizedHash, codec, hdr, rlsType, audioStr, audioChannels, region, languageStr, editionStr, cutStr, website, mediaProcessing sql.NullString
		var hybrid sql.NullBool

		var rlsIndexerID sql.NullInt64
		var rasId, rasFilterId, rasReleaseId, rasActionId sql.NullInt64
		var rasStatus, rasAction, rasType, rasClient, rasFilter sql.NullString
		var rasRejections []sql.NullString
		var rasTimestamp sql.NullTime

		if err := rows.Scan(
			&rls.ID,
			&rlsIndexer,
			&rlsIndexerID,
			&rlsIndexerName,
			&rlsIndexerExternalName,
			&rlsFilter,
			&rls.Protocol,
			&rlsAnnounceType,
			&infoUrl,
			&downloadUrl,
			&rls.Title,
			&subTitle,
			&rls.TorrentName,
			&normalizedHash,
			&rls.Size,
			&rls.Category,
			&rls.Season,
			&rls.Episode,
			&rls.Year,
			&rls.Resolution,
			&rls.Source,
			&codec,
			&rls.Container,
			&hdr,
			&audioStr,
			&audioChannels,
			&rls.Group,
			&region,
			&languageStr,
			&editionStr,
			&cutStr,
			&hybrid,
			&rls.Proper,
			&rls.Repack,
			&website,
			&mediaProcessing,
			&rlsType,
			&rls.Timestamp,
			&rasId, &rasStatus, &rasAction, &rasActionId, &rasType, &rasClient, &rasFilter, &rasFilterId, &rasReleaseId, pq.Array(&rasRejections), &rasTimestamp, &resp.TotalCount,
		); err != nil {
			return resp, errors.Wrap(err, "error scanning row")
		}

		//for _, codec := range codecs {
		//	rls.Codec = append(rls.Codec, codec.String)
		//
		//}

		ras.ID = rasId.Int64
		ras.Status = domain.ReleasePushStatus(rasStatus.String)
		ras.Action = rasAction.String
		ras.ActionID = rasActionId.Int64
		ras.Type = domain.ActionType(rasType.String)
		ras.Client = rasClient.String
		ras.Filter = rasFilter.String
		ras.FilterID = rasFilterId.Int64
		ras.Timestamp = rasTimestamp.Time
		ras.ReleaseID = rasReleaseId.Int64
		ras.Rejections = []string{}

		for _, rejection := range rasRejections {
			ras.Rejections = append(ras.Rejections, rejection.String)
		}

		idx := 0
		for ; idx < len(resp.Data); idx++ {
			if resp.Data[idx].ID != rls.ID {
				continue
			}

			resp.Data[idx].ActionStatus = append(resp.Data[idx].ActionStatus, ras)
			break
		}

		if idx != len(resp.Data) {
			continue
		}

		rls.Indexer.Identifier = rlsIndexer.String
		rls.Indexer.ID = int(rlsIndexerID.Int64)
		rls.Indexer.Name = rlsIndexerName.String
		rls.Indexer.IdentifierExternal = rlsIndexerExternalName.String

		rls.FilterName = rlsFilter.String
		rls.AnnounceType = domain.AnnounceType(rlsAnnounceType.String)
		rls.ActionStatus = make([]domain.ReleaseActionStatus, 0)
		rls.InfoURL = infoUrl.String
		rls.DownloadURL = downloadUrl.String
		rls.SubTitle = subTitle.String
		rls.NormalizedHash = normalizedHash.String
		rls.Codec = strings.Split(codec.String, ",")
		rls.HDR = strings.Split(hdr.String, ",")
		rls.Audio = strings.Split(audioStr.String, ",")
		rls.AudioChannels = audioChannels.String
		rls.Language = strings.Split(languageStr.String, ",")
		rls.Region = region.String
		rls.Edition = strings.Split(editionStr.String, ",")
		rls.Cut = strings.Split(cutStr.String, ",")
		rls.Hybrid = hybrid.Bool
		rls.Website = website.String
		rls.MediaProcessing = mediaProcessing.String
		//rls.Type = rlsType.String
		if rlsType.Valid {
			rls.ParseType(rlsType.String)
		}

		// only add ActionStatus if it's not empty
		if ras.ID > 0 {
			rls.ActionStatus = append(rls.ActionStatus, ras)
		}

		resp.Data = append(resp.Data, &rls)
	}

	if len(resp.Data) > 0 {
		lastID := resp.Data[len(resp.Data)-1].ID
		resp.NextCursor = lastID
	}

	return resp, nil
}

func (repo *ReleaseRepo) FindDuplicateReleaseProfiles(ctx context.Context) ([]*domain.DuplicateReleaseProfile, error) {
	queryBuilder := repo.db.squirrel.
		Select(
			"id",
			"name",
			"protocol",
			"release_name",
			"hash",
			"title",
			"sub_title",
			"year",
			"month",
			"day",
			"source",
			"resolution",
			"codec",
			"container",
			"dynamic_range",
			"audio",
			"release_group",
			"season",
			"episode",
			"website",
			"proper",
			"repack",
			"hybrid",
			"edition",
			"language",
			"media_processing",
		).
		From("release_profile_duplicate")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := repo.db.Handler.QueryContext(ctx, query, args...)
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

		err := rows.Scan(&p.ID, &p.Name, &p.Protocol, &p.ReleaseName, &p.Hash, &p.Title, &p.SubTitle, &p.Year, &p.Month, &p.Day, &p.Source, &p.Resolution, &p.Codec, &p.Container, &p.DynamicRange, &p.Audio, &p.Group, &p.Season, &p.Episode, &p.Website, &p.Proper, &p.Repack, &p.Hybrid, &p.Edition, &p.Language, &p.MediaProcessing)
		if err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		res = append(res, &p)
	}

	return res, nil
}

func (repo *ReleaseRepo) GetIndexerOptions(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT indexer FROM "release" UNION SELECT DISTINCT identifier indexer FROM indexer;`

	res := make([]string, 0)

	rows, err := repo.db.Handler.QueryContext(ctx, query)
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

	rows, err := repo.db.Handler.QueryContext(ctx, query, args...)
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
		var actionId sql.NullInt64

		if err := rows.Scan(&rls.ID, &rls.Status, &rls.Action, &actionId, &rls.Type, &client, &filter, &rls.ReleaseID, pq.Array(&rls.Rejections), &rls.Timestamp); err != nil {
			return res, errors.Wrap(err, "error scanning row")
		}

		rls.ActionID = actionId.Int64
		rls.Client = client.String
		rls.Filter = filter.String

		res = append(res, rls)
	}

	return res, nil
}

func (repo *ReleaseRepo) Get(ctx context.Context, req *domain.GetReleaseRequest) (*domain.Release, error) {
	queryBuilder := repo.db.squirrel.
		Select(
			"r.id",
			"r.indexer",
			"r.filter",
			"r.filter_id",
			"r.protocol",
			"r.implementation",
			"r.announce_type",
			"r.info_url",
			"r.download_url",
			"r.title",
			"r.sub_title",
			"r.release_name",
			"r.normalized_hash",
			"r.category",
			"r.size",
			"r.group_id",
			"r.external_id",
			"r.season",
			"r.episode",
			"r.year",
			"r.month",
			"r.day",
			"r.resolution",
			"r.source",
			"r.codec",
			"r.container",
			"r.dynamic_range",
			"r.audio",
			"r.audio_channels",
			"r.release_group",
			"r.proper",
			"r.repack",
			"r.region",
			"r.language",
			"r.cut",
			"r.edition",
			"r.hybrid",
			"r.media_processing",
			"r.website",
			"r.type",
			"r.origin",
			"r.tags",
			"r.uploader",
			"r.pre_time",
			"r.other",
			"r.timestamp",
		).
		From("release r").
		OrderBy("r.id DESC").
		Where(sq.Eq{"r.id": req.Id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	repo.log.Trace().Str("database", "release.find").Str("query", query).Interface("args", args).Msg("get release")

	row := repo.db.Handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var rls domain.Release

	var indexerName, filterName, announceType, infoUrl, downloadUrl, subTitle, normalizedHash, groupId, torrentId, category, resolution, source, codec, container, hdr, audio, audioChannels, releaseGroup, region, language, cut, edition, mediaProcessing, website, releaseType, origin, uploader, preTime sql.NullString
	var filterId sql.NullInt64
	var season, episode, year, month, day sql.NullInt64
	var proper, repack, hybrid sql.NullBool

	if err := row.Scan(
		&rls.ID,
		&indexerName,
		&filterName,
		&filterId,
		&rls.Protocol,
		&rls.Implementation,
		&announceType,
		&infoUrl,
		&downloadUrl,
		&rls.Title,
		&subTitle,
		&rls.TorrentName,
		&normalizedHash,
		&category,
		&rls.Size,
		&groupId,
		&torrentId,
		&season,
		&episode,
		&year,
		&month,
		&day,
		&resolution,
		&source,
		&codec,
		&container,
		&hdr,
		&audio,
		&audioChannels,
		&releaseGroup,
		&proper,
		&repack,
		&region,
		&language,
		&cut,
		&edition,
		&hybrid,
		&mediaProcessing,
		&website,
		&releaseType,
		&origin,
		pq.Array(&rls.Tags),
		&uploader,
		&preTime,
		pq.Array(&rls.Other),
		&rls.Timestamp,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	rls.Indexer.Identifier = indexerName.String
	rls.FilterName = filterName.String
	rls.FilterID = int(filterId.Int64)
	rls.ActionStatus = make([]domain.ReleaseActionStatus, 0)
	rls.AnnounceType = domain.AnnounceType(announceType.String)
	rls.InfoURL = infoUrl.String
	rls.DownloadURL = downloadUrl.String
	rls.SubTitle = subTitle.String
	rls.NormalizedHash = normalizedHash.String
	rls.Category = category.String
	rls.GroupID = groupId.String
	rls.TorrentID = torrentId.String
	rls.Season = int(season.Int64)
	rls.Episode = int(episode.Int64)
	rls.Year = int(year.Int64)
	rls.Month = int(month.Int64)
	rls.Day = int(day.Int64)
	rls.Resolution = resolution.String
	rls.Source = source.String
	if codec.String != "" {
		rls.Codec = strings.Split(codec.String, ",")
	}
	rls.Container = container.String
	if hdr.String != "" {
		rls.HDR = strings.Split(hdr.String, ",")
	}
	if audio.String != "" {
		rls.Audio = strings.Split(audio.String, ",")
	}
	rls.AudioChannels = audioChannels.String
	rls.Group = releaseGroup.String
	rls.Proper = proper.Bool
	rls.Repack = repack.Bool
	rls.Region = region.String
	if language.String != "" {
		rls.Language = strings.Split(language.String, ",")
	}
	if cut.String != "" {
		rls.Cut = strings.Split(cut.String, ",")
	}
	if edition.String != "" {
		rls.Edition = strings.Split(edition.String, ",")
	}
	rls.Hybrid = hybrid.Bool
	rls.MediaProcessing = mediaProcessing.String
	rls.Website = website.String
	if releaseType.Valid {
		rls.ParseType(releaseType.String)
	}
	rls.Origin = origin.String
	rls.Uploader = uploader.String
	rls.PreTime = preTime.String

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

	row := repo.db.Handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var rls domain.ReleaseActionStatus

	var client, filter sql.NullString
	var actionId, filterId sql.NullInt64

	if err := row.Scan(&rls.ID, &rls.Status, &rls.Action, &actionId, &rls.Type, &client, &filter, &filterId, &rls.ReleaseID, pq.Array(&rls.Rejections), &rls.Timestamp); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	rls.ActionID = actionId.Int64
	rls.Client = client.String
	rls.Filter = filter.String
	rls.FilterID = filterId.Int64

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

		var client, filter sql.NullString
		var actionId, filterID sql.NullInt64

		if err := rows.Scan(&rls.ID, &rls.Status, &rls.Action, &actionId, &rls.Type, &client, &filter, &filterID, &rls.ReleaseID, pq.Array(&rls.Rejections), &rls.Timestamp); err != nil {
			return res, errors.Wrap(err, "error scanning row")
		}

		rls.ActionID = actionId.Int64
		rls.Client = client.String
		rls.Filter = filter.String
		rls.FilterID = filterID.Int64

		res = append(res, rls)
	}

	return res, nil
}

func (repo *ReleaseRepo) Stats(ctx context.Context) (*domain.ReleaseStats, error) {
	var rls domain.ReleaseStats

	// only releases a filter approved are stored
	if err := repo.db.Handler.QueryRowContext(ctx, `SELECT COUNT(*) FROM release`).Scan(&rls.TotalCount); err != nil {
		return nil, errors.Wrap(err, "error scanning row")
	}

	rls.FilteredCount = rls.TotalCount

	pushRows, err := repo.db.Handler.QueryContext(ctx, `SELECT status, COUNT(*) FROM release_action_status GROUP BY status`)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer pushRows.Close()

	for pushRows.Next() {
		var status sql.NullString
		var count int64

		if err := pushRows.Scan(&status, &count); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		switch domain.ReleasePushStatus(status.String) {
		case domain.ReleasePushStatusApproved:
			rls.PushApprovedCount = count
		case domain.ReleasePushStatusRejected:
			rls.PushRejectedCount = count
		case domain.ReleasePushStatusErr:
			rls.PushErrorCount = count
		}
	}

	if err := pushRows.Err(); err != nil {
		return nil, errors.Wrap(err, "error rows")
	}

	return &rls, nil
}

// statsWindow bounds the dashboard widget queries to the last N days,
// or nothing for all-time (days <= 0).
type statsWindow struct {
	days   int
	now    time.Time
	cutoff time.Time
}

func (repo *ReleaseRepo) statsWindow(days int) statsWindow {
	w := statsWindow{days: days, now: time.Now().UTC()}
	if days > 0 {
		w.cutoff = w.now.AddDate(0, 0, -(days - 1)).Truncate(24 * time.Hour)
	}
	return w
}

func (w statsWindow) apply(qb sq.SelectBuilder, column string) sq.SelectBuilder {
	if w.days > 0 {
		return qb.Where(sq.GtOrEq{column: w.cutoff})
	}
	return qb
}

// dates returns the continuous run of day keys to emit: the fixed window,
// or earliest-seen through today for all-time.
func (w statsWindow) dates(minKey string) []string {
	start := w.cutoff
	if w.days <= 0 {
		if minKey == "" {
			return nil
		}
		parsed, err := time.Parse("2006-01-02", minKey)
		if err != nil {
			return nil
		}
		start = parsed
	}

	var out []string
	for d := start; !d.After(w.now); d = d.AddDate(0, 0, 1) {
		out = append(out, d.Format("2006-01-02"))
	}
	return out
}

func minDateKey[T any](m map[string]*T) string {
	minKey := ""
	for key := range m {
		if minKey == "" || key < minKey {
			minKey = key
		}
	}
	return minKey
}

func (repo *ReleaseRepo) statsDayExpr() string {
	if repo.db.Driver == "sqlite" {
		// substr instead of strftime: stored timestamps are UTC text starting with
		// YYYY-MM-DD HH, and skipping per-row date parsing roughly halves the scan cost
		return "substr(timestamp, 1, 10)"
	}
	return "to_char(timestamp, 'YYYY-MM-DD')"
}

func (repo *ReleaseRepo) statsHourExpr() string {
	if repo.db.Driver == "sqlite" {
		return "CAST(substr(timestamp, 12, 2) AS INTEGER)"
	}
	return "EXTRACT(HOUR FROM timestamp)::int"
}

func (repo *ReleaseRepo) statsQueryRows(ctx context.Context, qb sq.SelectBuilder, scan func(rows *sql.Rows) error) error {
	query, args, err := qb.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	rows, err := repo.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	for rows.Next() {
		if err := scan(rows); err != nil {
			return errors.Wrap(err, "error scanning row")
		}
	}

	return errors.Wrap(rows.Err(), "error rows")
}

func (repo *ReleaseRepo) StatsActivity(ctx context.Context, days int) (*domain.ReleaseActivityStats, error) {
	window := repo.statsWindow(days)

	daily := map[string]*domain.ReleaseActivityDaily{}
	bucket := func(day string) *domain.ReleaseActivityDaily {
		if b, ok := daily[day]; ok {
			return b
		}
		b := &domain.ReleaseActivityDaily{Date: day}
		daily[day] = b
		return b
	}

	err := repo.statsQueryRows(ctx,
		window.apply(repo.db.squirrel.Select(repo.statsDayExpr()+" AS day", "COUNT(*)").From("release").GroupBy("1"), "timestamp"),
		func(rows *sql.Rows) error {
			var day sql.NullString
			var count int64
			if err := rows.Scan(&day, &count); err != nil {
				return err
			}
			if day.Valid {
				bucket(day.String).MatchedCount = count
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	err = repo.statsQueryRows(ctx,
		window.apply(repo.db.squirrel.Select(repo.statsDayExpr()+" AS day", "status", "COUNT(*)").From("release_action_status").GroupBy("1", "2"), "timestamp"),
		func(rows *sql.Rows) error {
			var day, status sql.NullString
			var count int64
			if err := rows.Scan(&day, &status, &count); err != nil {
				return err
			}
			if !day.Valid {
				return nil
			}
			switch domain.ReleasePushStatus(status.String) {
			case domain.ReleasePushStatusApproved:
				bucket(day.String).PushApprovedCount = count
			case domain.ReleasePushStatusRejected:
				bucket(day.String).PushRejectedCount = count
			case domain.ReleasePushStatusErr:
				bucket(day.String).PushErrorCount = count
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	stats := &domain.ReleaseActivityStats{Days: days, Daily: []domain.ReleaseActivityDaily{}}
	for _, day := range window.dates(minDateKey(daily)) {
		if b, ok := daily[day]; ok {
			stats.Daily = append(stats.Daily, *b)
		} else {
			stats.Daily = append(stats.Daily, domain.ReleaseActivityDaily{Date: day})
		}
	}

	return stats, nil
}

func (repo *ReleaseRepo) StatsVolume(ctx context.Context, days int) (*domain.ReleaseVolumeStats, error) {
	window := repo.statsWindow(days)

	daily := map[string]*domain.ReleaseVolumeDaily{}

	err := repo.statsQueryRows(ctx,
		window.apply(repo.db.squirrel.Select(repo.statsDayExpr()+" AS day", "SUM(size)").From("release").
			Where(sq.Expr("id IN (SELECT DISTINCT release_id FROM release_action_status WHERE status = ?)", string(domain.ReleasePushStatusApproved))).
			GroupBy("1"), "timestamp"),
		func(rows *sql.Rows) error {
			var day sql.NullString
			var size sql.NullInt64
			if err := rows.Scan(&day, &size); err != nil {
				return err
			}
			if day.Valid {
				daily[day.String] = &domain.ReleaseVolumeDaily{Date: day.String, DownloadedBytes: size.Int64}
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	stats := &domain.ReleaseVolumeStats{Days: days, Daily: []domain.ReleaseVolumeDaily{}}
	for _, day := range window.dates(minDateKey(daily)) {
		if b, ok := daily[day]; ok {
			stats.Daily = append(stats.Daily, *b)
		} else {
			stats.Daily = append(stats.Daily, domain.ReleaseVolumeDaily{Date: day})
		}
	}

	return stats, nil
}

func (repo *ReleaseRepo) StatsHeatmap(ctx context.Context, days int) (*domain.ReleaseHeatmapStats, error) {
	window := repo.statsWindow(days)

	stats := &domain.ReleaseHeatmapStats{Days: days, Heatmap: make([]int64, 7*24)}

	err := repo.statsQueryRows(ctx,
		window.apply(repo.db.squirrel.Select(repo.statsDayExpr()+" AS day", repo.statsHourExpr()+" AS hour", "COUNT(*)").From("release").GroupBy("1", "2"), "timestamp"),
		func(rows *sql.Rows) error {
			var day sql.NullString
			var hour sql.NullInt64
			var count int64
			if err := rows.Scan(&day, &hour, &count); err != nil {
				return err
			}
			if !day.Valid || !hour.Valid || hour.Int64 < 0 || hour.Int64 > 23 {
				return nil
			}
			date, err := time.Parse("2006-01-02", day.String)
			if err != nil {
				return nil
			}
			stats.Heatmap[int64(date.Weekday())*24+hour.Int64] += count
			return nil
		})
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (repo *ReleaseRepo) StatsTopIndexers(ctx context.Context, days int) (*domain.ReleaseTopIndexersStats, error) {
	window := repo.statsWindow(days)

	indexers := map[string]*domain.ReleaseIndexerStats{}
	bucket := func(name string) *domain.ReleaseIndexerStats {
		if b, ok := indexers[name]; ok {
			return b
		}
		b := &domain.ReleaseIndexerStats{Indexer: name}
		indexers[name] = b
		return b
	}

	err := repo.statsQueryRows(ctx,
		window.apply(repo.db.squirrel.Select("indexer", "COUNT(*)").From("release").
			Where("indexer IS NOT NULL AND indexer != ''").GroupBy("1"), "timestamp"),
		func(rows *sql.Rows) error {
			var name string
			var count int64
			if err := rows.Scan(&name, &count); err != nil {
				return err
			}
			bucket(name).MatchedCount = count
			return nil
		})
	if err != nil {
		return nil, err
	}

	err = repo.statsQueryRows(ctx,
		window.apply(repo.db.squirrel.Select("r.indexer", "COUNT(*)").From("release_action_status ras").
			InnerJoin("release r ON r.id = ras.release_id").
			Where(sq.Eq{"ras.status": string(domain.ReleasePushStatusApproved)}).
			Where("r.indexer IS NOT NULL AND r.indexer != ''").
			GroupBy("1"), "ras.timestamp"),
		func(rows *sql.Rows) error {
			var name string
			var count int64
			if err := rows.Scan(&name, &count); err != nil {
				return err
			}
			bucket(name).PushApprovedCount = count
			return nil
		})
	if err != nil {
		return nil, err
	}

	stats := &domain.ReleaseTopIndexersStats{Days: days, Top: []domain.ReleaseIndexerStats{}}
	for _, b := range indexers {
		stats.Top = append(stats.Top, *b)
	}
	slices.SortFunc(stats.Top, func(a, b domain.ReleaseIndexerStats) int {
		if c := cmp.Compare(b.PushApprovedCount, a.PushApprovedCount); c != 0 {
			return c
		}
		if c := cmp.Compare(b.MatchedCount, a.MatchedCount); c != 0 {
			return c
		}
		return strings.Compare(a.Indexer, b.Indexer)
	})
	if len(stats.Top) > 10 {
		stats.Top = stats.Top[:10]
	}

	return stats, nil
}

func (repo *ReleaseRepo) StatsTopFilters(ctx context.Context, days int) (*domain.ReleaseTopFiltersStats, error) {
	window := repo.statsWindow(days)

	filters := map[string]*domain.ReleaseFilterStats{}
	bucket := func(name string) *domain.ReleaseFilterStats {
		if b, ok := filters[name]; ok {
			return b
		}
		b := &domain.ReleaseFilterStats{Filter: name}
		filters[name] = b
		return b
	}

	err := repo.statsQueryRows(ctx,
		window.apply(repo.db.squirrel.Select("filter", "COUNT(*)").From("release").
			Where("filter IS NOT NULL AND filter != ''").GroupBy("1"), "timestamp"),
		func(rows *sql.Rows) error {
			var name string
			var count int64
			if err := rows.Scan(&name, &count); err != nil {
				return err
			}
			bucket(name).MatchedCount = count
			return nil
		})
	if err != nil {
		return nil, err
	}

	err = repo.statsQueryRows(ctx,
		window.apply(repo.db.squirrel.Select("filter", "COUNT(*)").From("release_action_status").
			Where(sq.Eq{"status": string(domain.ReleasePushStatusApproved)}).
			Where("filter IS NOT NULL AND filter != ''").
			GroupBy("1"), "timestamp"),
		func(rows *sql.Rows) error {
			var name string
			var count int64
			if err := rows.Scan(&name, &count); err != nil {
				return err
			}
			bucket(name).PushApprovedCount = count
			return nil
		})
	if err != nil {
		return nil, err
	}

	stats := &domain.ReleaseTopFiltersStats{Days: days, Top: []domain.ReleaseFilterStats{}}
	for _, b := range filters {
		stats.Top = append(stats.Top, *b)
	}
	slices.SortFunc(stats.Top, func(a, b domain.ReleaseFilterStats) int {
		if c := cmp.Compare(b.MatchedCount, a.MatchedCount); c != 0 {
			return c
		}
		if c := cmp.Compare(b.PushApprovedCount, a.PushApprovedCount); c != 0 {
			return c
		}
		return strings.Compare(a.Filter, b.Filter)
	})
	if len(stats.Top) > 10 {
		stats.Top = stats.Top[:10]
	}

	return stats, nil
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
			repo.log.Error().Interface("panic", p).Msg("something went terribly wrong")
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
		qb = qb.Where(sq.Lt{"timestamp": time.Now().Add(time.Duration(-req.OlderThan) * time.Hour)})
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

		// If PUSH_APPROVED is not in the delete list, exclude releases that have
		// any approved action - a release pushed to at least one client must be kept.
		approvedInList := slices.Contains(req.ReleaseStatuses, string(domain.ReleasePushStatusApproved))
		if !approvedInList {
			excludeSubQuery := sq.Select("release_id").From("release_action_status").Where(sq.Eq{"status": string(domain.ReleasePushStatusApproved)})
			excludeText, excludeArgs, err := excludeSubQuery.ToSql()
			if err != nil {
				return errors.Wrap(err, "error building approved-exclusion subquery")
			}
			qb = qb.Where("id NOT IN ("+excludeText+")", excludeArgs...)
		}
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

	repo.log.Debug().Int64("rows_affected", deletedRows).Msg("deleted rows from release table")

	// clean up orphaned rows
	orphanedResult, err := tx.ExecContext(ctx, `DELETE FROM release_action_status WHERE release_id NOT IN (SELECT id FROM "release")`)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	deletedRowsOrphaned, err := orphanedResult.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error fetching rows affected")
	}

	repo.log.Debug().Int64("rows_affected", deletedRowsOrphaned).Msg("deleted orphaned rows from release table")

	return nil
}

func (repo *ReleaseRepo) DeleteReleaseProfileDuplicate(ctx context.Context, id int64) error {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "error begin transaction")
	}

	defer tx.Rollback()

	// SQLite runs without foreign key enforcement, so ON DELETE SET NULL never fires there
	updateQueryBuilder := repo.db.squirrel.
		Update("filter").
		Set("release_profile_duplicate_id", nil).
		Where(sq.Eq{"release_profile_duplicate_id": id})

	updateQuery, updateArgs, err := updateQueryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err := tx.ExecContext(ctx, updateQuery, updateArgs...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	deleteQueryBuilder := repo.db.squirrel.Delete("release_profile_duplicate").Where(sq.Eq{"id": id})

	deleteQuery, deleteArgs, err := deleteQueryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := tx.ExecContext(ctx, deleteQuery, deleteArgs...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return errors.Wrap(err, "error getting rows affected")
	} else if rowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "error deleting duplicate release profile")
	}

	repo.log.Debug().Int64("id", id).Msg("deleted duplicate release profile")

	return nil
}

func (repo *ReleaseRepo) CheckSmartEpisodeCanDownload(ctx context.Context, p *domain.SmartEpisodeParams) (bool, error) {
	titleKey := domain.TitleKey(p.Title)
	if titleKey == "" {
		return true, nil
	}

	queryBuilder := repo.db.squirrel.
		Select("COUNT(*)").
		From("release r").
		LeftJoin("release_action_status ras ON r.id = ras.release_id").
		Where(sq.And{
			sq.Eq{"r.title_key": titleKey},
			sq.Eq{"ras.status": []string{"PUSH_APPROVED", "PENDING"}},
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

	repo.log.Trace().Str("method", "CheckSmartEpisodeCanDownload").Str("query", query).Interface("args", args).Msg("executing query")

	row := repo.db.Handler.QueryRowContext(ctx, query, args...)
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
			repo.log.Error().Interface("panic", p).Msg("something went terribly wrong")
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

	repo.log.Trace().Int64("rows_affected", rowsAffected).Str("old_url", oldBaseURL).Str("new_url", newBaseURL).Msg("release updated base urls")

	return nil
}

// FindDuplicate returns the grabbed or in-flight release that release duplicates under the
// profile, or nil. A grabbed match wins over an in-flight one. Stored keys narrow the candidates
// through an index and profile.Matches decides.
func (repo *ReleaseRepo) FindDuplicate(ctx context.Context, profile *domain.DuplicateReleaseProfile, release *domain.Release) (*domain.DuplicateMatch, error) {
	if !profile.ReleaseName && !profile.Title {
		repo.log.Warn().Int64("profile_id", profile.ID).Str("profile", profile.Name).Msg("duplicate profile compares neither release name nor title, skipping check")
		return nil, nil
	}

	queryBuilder := repo.db.squirrel.
		Select(
			"r.id",
			"COALESCE(r.protocol, '')",
			"COALESCE(r.release_name, '')",
			"COALESCE(r.title, '')",
			"COALESCE(r.sub_title, '')",
			"COALESCE(r.year, 0)",
			"COALESCE(r.month, 0)",
			"COALESCE(r.day, 0)",
			"COALESCE(r.season, 0)",
			"COALESCE(r.episode, 0)",
			"COALESCE(r.region, '')",
			"COALESCE(r.source, '')",
			"COALESCE(r.resolution, '')",
			"COALESCE(r.container, '')",
			"COALESCE(r.website, '')",
			"COALESCE(r.release_group, '')",
			"COALESCE(r.codec, '')",
			"COALESCE(r.dynamic_range, '')",
			"COALESCE(r.language, '')",
			"COALESCE(r.audio, '')",
			"COALESCE(r.audio_channels, '')",
			"COALESCE(r.cut, '')",
			"COALESCE(r.edition, '')",
			"COALESCE(r.hybrid, FALSE)",
			"COALESCE(r.media_processing, '')",
			"COALESCE(r.proper, FALSE)",
			"COALESCE(r.repack, FALSE)",
			"r.other",
			"EXISTS (SELECT 1 FROM release_action_status ras WHERE ras.release_id = r.id AND ras.status = 'PUSH_APPROVED')",
		).
		From("release r").
		Where("EXISTS (SELECT 1 FROM release_action_status ras WHERE ras.release_id = r.id AND ras.status IN ('PUSH_APPROVED', 'PENDING'))").
		OrderBy("r.id DESC")

	if release.ID != 0 {
		queryBuilder = queryBuilder.Where(sq.NotEq{"r.id": release.ID})
	}

	if profile.UsesNameKey(release) {
		nameKey := domain.NameKey(release.TorrentName)
		if nameKey == "" {
			return nil, nil
		}

		queryBuilder = queryBuilder.Where(sq.Eq{"r.name_key": nameKey})
	} else {
		// releases stored before v1.56.0 have no parsed columns to compare
		queryBuilder = queryBuilder.
			Where(sq.Eq{"r.title_key": domain.TitleKey(release.Title)}).
			Where(sq.NotEq{"r.media_processing": nil})

		if profile.Year {
			queryBuilder = queryBuilder.Where(sq.Eq{"r.year": release.Year})
		}

		if profile.Month {
			queryBuilder = queryBuilder.Where(sq.Eq{"r.month": release.Month})
		}

		if profile.Day {
			queryBuilder = queryBuilder.Where(sq.Eq{"r.day": release.Day})
		}

		if profile.Season {
			queryBuilder = queryBuilder.Where(sq.Eq{"r.season": release.Season})
		}

		if profile.Episode {
			queryBuilder = queryBuilder.Where(sq.Eq{"r.episode": release.Episode})
		}
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := repo.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	var inFlight *domain.DuplicateMatch

	for rows.Next() {
		var candidate domain.Release
		var codec, dynamicRange, language, audio, cut, edition string
		var approved bool

		if err := rows.Scan(
			&candidate.ID,
			&candidate.Protocol,
			&candidate.TorrentName,
			&candidate.Title,
			&candidate.SubTitle,
			&candidate.Year,
			&candidate.Month,
			&candidate.Day,
			&candidate.Season,
			&candidate.Episode,
			&candidate.Region,
			&candidate.Source,
			&candidate.Resolution,
			&candidate.Container,
			&candidate.Website,
			&candidate.Group,
			&codec,
			&dynamicRange,
			&language,
			&audio,
			&candidate.AudioChannels,
			&cut,
			&edition,
			&candidate.Hybrid,
			&candidate.MediaProcessing,
			&candidate.Proper,
			&candidate.Repack,
			pq.Array(&candidate.Other),
			&approved,
		); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		candidate.Codec = strings.Split(codec, ",")
		candidate.HDR = strings.Split(dynamicRange, ",")
		candidate.Language = strings.Split(language, ",")
		candidate.Audio = strings.Split(audio, ",")
		candidate.Cut = strings.Split(cut, ",")
		candidate.Edition = strings.Split(edition, ",")

		if !profile.Matches(release, &candidate) {
			continue
		}

		repo.log.Debug().Int64("profile_id", profile.ID).Str("profile", profile.Name).Int64("duplicate_of", candidate.ID).Str("duplicate_name", candidate.TorrentName).Bool("in_flight", !approved).Msg("found duplicate release")

		if approved {
			return &domain.DuplicateMatch{ReleaseID: candidate.ID, ReleaseName: candidate.TorrentName}, nil
		}

		if inFlight == nil {
			inFlight = &domain.DuplicateMatch{ReleaseID: candidate.ID, ReleaseName: candidate.TorrentName, InFlight: true}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "row error")
	}

	return inFlight, nil
}

// FailPendingActionStatuses marks every PENDING action status as PUSH_ERROR. It runs at startup,
// when no push can still be in flight, so a push interrupted by a restart stops counting as a
// grab and no longer blocks its duplicates.
func (repo *ReleaseRepo) FailPendingActionStatuses(ctx context.Context) (int64, error) {
	queryBuilder := repo.db.squirrel.
		Update("release_action_status").
		Set("status", domain.ReleasePushStatusErr).
		Set("rejections", pq.Array([]string{"interrupted by restart"})).
		Where(sq.Eq{"status": domain.ReleasePushStatusPending})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "error building query")
	}

	result, err := repo.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "error executing query")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "error getting rows affected")
	}

	return rowsAffected, nil
}

func (r *ReleaseRepo) ListCleanupJobs(ctx context.Context) ([]*domain.ReleaseCleanupJob, error) {
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

func (r *ReleaseRepo) FindCleanupJobByID(ctx context.Context, id int) (*domain.ReleaseCleanupJob, error) {
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

func (r *ReleaseRepo) StoreCleanupJob(ctx context.Context, job *domain.ReleaseCleanupJob) error {
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

func (r *ReleaseRepo) UpdateCleanupJob(ctx context.Context, job *domain.ReleaseCleanupJob) error {
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

func (r *ReleaseRepo) UpdateCleanupJobLastRun(ctx context.Context, job *domain.ReleaseCleanupJob) error {
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

func (r *ReleaseRepo) CleanupJobToggleEnabled(ctx context.Context, id int, enabled bool) error {
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

func (r *ReleaseRepo) DeleteCleanupJob(ctx context.Context, id int) error {
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

	r.log.Debug().Int("id", id).Msg("successfully deleted release cleanup job")

	return nil
}
