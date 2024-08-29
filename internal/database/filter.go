// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

type FilterRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewFilterRepo(log logger.Logger, db *DB) domain.FilterRepo {
	return &FilterRepo{
		log: log.With().Str("repo", "filter").Logger(),
		db:  db,
	}
}

func (r *FilterRepo) Find(ctx context.Context, params domain.FilterQueryParams) ([]domain.Filter, error) {
	filters, err := r.find(ctx, params)
	if err != nil {
		return nil, err
	}

	return filters, nil
}

func (r *FilterRepo) find(ctx context.Context, params domain.FilterQueryParams) ([]domain.Filter, error) {
	actionCountQuery := r.db.squirrel.
		Select("COUNT(*)").
		From("action a").
		Where("a.filter_id = f.id")

	actionEnabledCountQuery := r.db.squirrel.
		Select("COUNT(*)").
		From("action a").
		Where("a.filter_id = f.id").
		Where("a.enabled = '1'")

	queryBuilder := r.db.squirrel.
		Select(
			"f.id",
			"f.enabled",
			"f.name",
			"f.priority",
			"f.created_at",
			"f.updated_at",
		).
		Distinct().
		Column(sq.Alias(actionCountQuery, "action_count")).
		Column(sq.Alias(actionEnabledCountQuery, "actions_enabled_count")).
		LeftJoin("filter_indexer fi ON f.id = fi.filter_id").
		LeftJoin("indexer i ON i.id = fi.indexer_id").
		From("filter f")

	if params.Search != "" {
		queryBuilder = queryBuilder.Where(sq.Like{"f.name": params.Search + "%"})
	}

	if len(params.Sort) > 0 {
		for field, order := range params.Sort {
			queryBuilder = queryBuilder.OrderBy(fmt.Sprintf("f.%v %v", field, strings.ToUpper(order)))
		}
	} else {
		queryBuilder = queryBuilder.OrderBy("f.name ASC")
	}

	if params.Filters.Indexers != nil {
		filter := sq.And{}
		for _, v := range params.Filters.Indexers {
			filter = append(filter, sq.Eq{"i.identifier": v})
		}
		queryBuilder = queryBuilder.Where(filter)
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}
	defer rows.Close()

	var filters []domain.Filter
	for rows.Next() {
		var f domain.Filter

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, &f.Priority, &f.CreatedAt, &f.UpdatedAt, &f.ActionsCount, &f.ActionsEnabledCount); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		filters = append(filters, f)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "row error")
	}

	return filters, nil
}

func (r *FilterRepo) ListFilters(ctx context.Context) ([]domain.Filter, error) {
	actionCountQuery := r.db.squirrel.
		Select("COUNT(*)").
		From("action a").
		Where("a.filter_id = f.id")

	queryBuilder := r.db.squirrel.
		Select(
			"f.id",
			"f.enabled",
			"f.name",
			"f.priority",
			"f.created_at",
			"f.updated_at",
		).
		Column(sq.Alias(actionCountQuery, "action_count")).
		From("filter f").
		OrderBy("f.name ASC")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	var filters []domain.Filter
	for rows.Next() {
		var f domain.Filter

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, &f.Priority, &f.CreatedAt, &f.UpdatedAt, &f.ActionsCount); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		filters = append(filters, f)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "row error")
	}

	return filters, nil
}

func (r *FilterRepo) FindByID(ctx context.Context, filterID int) (*domain.Filter, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"f.id",
			"f.enabled",
			"f.name",
			"f.min_size",
			"f.max_size",
			"f.delay",
			"f.priority",
			"f.max_downloads",
			"f.max_downloads_unit",
			"f.match_releases",
			"f.except_releases",
			"f.use_regex",
			"f.match_release_groups",
			"f.except_release_groups",
			"f.match_release_tags",
			"f.except_release_tags",
			"f.use_regex_release_tags",
			"f.match_description",
			"f.except_description",
			"f.use_regex_description",
			"f.scene",
			"f.freeleech",
			"f.freeleech_percent",
			"f.smart_episode",
			"f.shows",
			"f.seasons",
			"f.episodes",
			"f.resolutions",
			"f.codecs",
			"f.sources",
			"f.containers",
			"f.match_hdr",
			"f.except_hdr",
			"f.match_other",
			"f.except_other",
			"f.years",
			"f.months",
			"f.days",
			"f.artists",
			"f.albums",
			"f.release_types_match",
			"f.formats",
			"f.quality",
			"f.media",
			"f.log_score",
			"f.has_log",
			"f.has_cue",
			"f.perfect_flac",
			"f.match_categories",
			"f.except_categories",
			"f.match_uploaders",
			"f.except_uploaders",
			"f.match_language",
			"f.except_language",
			"f.tags",
			"f.except_tags",
			"f.tags_match_logic",
			"f.except_tags_match_logic",
			"f.origins",
			"f.except_origins",
			"f.min_seeders",
			"f.max_seeders",
			"f.min_leechers",
			"f.max_leechers",
			"f.created_at",
			"f.updated_at",
		).
		From("filter f").
		Where(sq.Eq{"f.id": filterID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error row")
	}

	var f domain.Filter

	// filter
	var minSize, maxSize, maxDownloadsUnit, matchReleases, exceptReleases, matchReleaseGroups, exceptReleaseGroups, matchReleaseTags, exceptReleaseTags, matchDescription, exceptDescription, freeleechPercent, shows, seasons, episodes, years, months, days, artists, albums, matchCategories, exceptCategories, matchUploaders, exceptUploaders, tags, exceptTags, tagsMatchLogic, exceptTagsMatchLogic sql.NullString
	var useRegex, scene, freeleech, hasLog, hasCue, perfectFlac sql.NullBool
	var delay, maxDownloads, logScore sql.NullInt32

	err = row.Scan(
		&f.ID,
		&f.Enabled,
		&f.Name,
		&minSize,
		&maxSize,
		&delay,
		&f.Priority,
		&maxDownloads,
		&maxDownloadsUnit,
		&matchReleases,
		&exceptReleases,
		&useRegex,
		&matchReleaseGroups,
		&exceptReleaseGroups,
		&matchReleaseTags,
		&exceptReleaseTags,
		&f.UseRegexReleaseTags,
		&matchDescription,
		&exceptDescription,
		&f.UseRegexDescription,
		&scene,
		&freeleech,
		&freeleechPercent,
		&f.SmartEpisode,
		&shows,
		&seasons,
		&episodes,
		pq.Array(&f.Resolutions),
		pq.Array(&f.Codecs),
		pq.Array(&f.Sources),
		pq.Array(&f.Containers),
		pq.Array(&f.MatchHDR),
		pq.Array(&f.ExceptHDR),
		pq.Array(&f.MatchOther),
		pq.Array(&f.ExceptOther),
		&years,
		&months,
		&days,
		&artists,
		&albums,
		pq.Array(&f.MatchReleaseTypes),
		pq.Array(&f.Formats),
		pq.Array(&f.Quality),
		pq.Array(&f.Media),
		&logScore,
		&hasLog,
		&hasCue,
		&perfectFlac,
		&matchCategories,
		&exceptCategories,
		&matchUploaders,
		&exceptUploaders,
		pq.Array(&f.MatchLanguage),
		pq.Array(&f.ExceptLanguage),
		&tags,
		&exceptTags,
		&tagsMatchLogic,
		&exceptTagsMatchLogic,
		pq.Array(&f.Origins),
		pq.Array(&f.ExceptOrigins),
		&f.MinSeeders,
		&f.MaxSeeders,
		&f.MinLeechers,
		&f.MaxLeechers,
		&f.CreatedAt,
		&f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	f.MinSize = minSize.String
	f.MaxSize = maxSize.String
	f.Delay = int(delay.Int32)
	f.MaxDownloads = int(maxDownloads.Int32)
	f.MaxDownloadsUnit = domain.FilterMaxDownloadsUnit(maxDownloadsUnit.String)
	f.MatchReleases = matchReleases.String
	f.ExceptReleases = exceptReleases.String
	f.MatchReleaseGroups = matchReleaseGroups.String
	f.ExceptReleaseGroups = exceptReleaseGroups.String
	f.MatchReleaseTags = matchReleaseTags.String
	f.ExceptReleaseTags = exceptReleaseTags.String
	f.MatchDescription = matchDescription.String
	f.ExceptDescription = exceptDescription.String
	f.FreeleechPercent = freeleechPercent.String
	f.Shows = shows.String
	f.Seasons = seasons.String
	f.Episodes = episodes.String
	f.Years = years.String
	f.Months = months.String
	f.Days = days.String
	f.Artists = artists.String
	f.Albums = albums.String
	f.LogScore = int(logScore.Int32)
	f.Log = hasLog.Bool
	f.Cue = hasCue.Bool
	f.PerfectFlac = perfectFlac.Bool
	f.MatchCategories = matchCategories.String
	f.ExceptCategories = exceptCategories.String
	f.MatchUploaders = matchUploaders.String
	f.ExceptUploaders = exceptUploaders.String
	f.Tags = tags.String
	f.ExceptTags = exceptTags.String
	f.TagsMatchLogic = tagsMatchLogic.String
	f.ExceptTagsMatchLogic = exceptTagsMatchLogic.String
	f.UseRegex = useRegex.Bool
	f.Scene = scene.Bool
	f.Freeleech = freeleech.Bool

	return &f, nil
}

// FindByIndexerIdentifier find active filters with active indexer only
func (r *FilterRepo) FindByIndexerIdentifier(ctx context.Context, indexer string) ([]*domain.Filter, error) {
	return r.findByIndexerIdentifier(ctx, indexer)
}

func (r *FilterRepo) findByIndexerIdentifier(ctx context.Context, indexer string) ([]*domain.Filter, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"f.id",
			"f.enabled",
			"f.name",
			"f.min_size",
			"f.max_size",
			"f.delay",
			"f.priority",
			"f.max_downloads",
			"f.max_downloads_unit",
			"f.match_releases",
			"f.except_releases",
			"f.use_regex",
			"f.match_release_groups",
			"f.except_release_groups",
			"f.match_release_tags",
			"f.except_release_tags",
			"f.use_regex_release_tags",
			"f.match_description",
			"f.except_description",
			"f.use_regex_description",
			"f.scene",
			"f.freeleech",
			"f.freeleech_percent",
			"f.smart_episode",
			"f.shows",
			"f.seasons",
			"f.episodes",
			"f.resolutions",
			"f.codecs",
			"f.sources",
			"f.containers",
			"f.match_hdr",
			"f.except_hdr",
			"f.match_other",
			"f.except_other",
			"f.years",
			"f.months",
			"f.days",
			"f.artists",
			"f.albums",
			"f.release_types_match",
			"f.formats",
			"f.quality",
			"f.media",
			"f.log_score",
			"f.has_log",
			"f.has_cue",
			"f.perfect_flac",
			"f.match_categories",
			"f.except_categories",
			"f.match_uploaders",
			"f.except_uploaders",
			"f.match_language",
			"f.except_language",
			"f.tags",
			"f.except_tags",
			"f.tags_match_logic",
			"f.except_tags_match_logic",
			"f.origins",
			"f.except_origins",
			"f.min_seeders",
			"f.max_seeders",
			"f.min_leechers",
			"f.max_leechers",
			"f.created_at",
			"f.updated_at",
		).
		From("filter f").
		Join("filter_indexer fi ON f.id = fi.filter_id").
		Join("indexer i ON i.id = fi.indexer_id").
		Where(sq.Eq{"i.identifier": indexer}).
		Where(sq.Eq{"i.enabled": true}).
		Where(sq.Eq{"f.enabled": true}).
		OrderBy("f.priority DESC")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	var filters []*domain.Filter

	for rows.Next() {
		var f domain.Filter

		var minSize, maxSize, maxDownloadsUnit, matchReleases, exceptReleases, matchReleaseGroups, exceptReleaseGroups, matchReleaseTags, exceptReleaseTags, matchDescription, exceptDescription, freeleechPercent, shows, seasons, episodes, years, months, days, artists, albums, matchCategories, exceptCategories, matchUploaders, exceptUploaders, tags, exceptTags, tagsMatchLogic, exceptTagsMatchLogic sql.NullString
		var useRegex, scene, freeleech, hasLog, hasCue, perfectFlac sql.NullBool
		var delay, maxDownloads, logScore sql.NullInt32

		err := rows.Scan(
			&f.ID,
			&f.Enabled,
			&f.Name,
			&minSize,
			&maxSize,
			&delay,
			&f.Priority,
			&maxDownloads,
			&maxDownloadsUnit,
			&matchReleases,
			&exceptReleases,
			&useRegex,
			&matchReleaseGroups,
			&exceptReleaseGroups,
			&matchReleaseTags,
			&exceptReleaseTags,
			&f.UseRegexReleaseTags,
			&matchDescription,
			&exceptDescription,
			&f.UseRegexDescription,
			&scene,
			&freeleech,
			&freeleechPercent,
			&f.SmartEpisode,
			&shows,
			&seasons,
			&episodes,
			pq.Array(&f.Resolutions),
			pq.Array(&f.Codecs),
			pq.Array(&f.Sources),
			pq.Array(&f.Containers),
			pq.Array(&f.MatchHDR),
			pq.Array(&f.ExceptHDR),
			pq.Array(&f.MatchOther),
			pq.Array(&f.ExceptOther),
			&years,
			&months,
			&days,
			&artists,
			&albums,
			pq.Array(&f.MatchReleaseTypes),
			pq.Array(&f.Formats),
			pq.Array(&f.Quality),
			pq.Array(&f.Media),
			&logScore,
			&hasLog,
			&hasCue,
			&perfectFlac,
			&matchCategories,
			&exceptCategories,
			&matchUploaders,
			&exceptUploaders,
			pq.Array(&f.MatchLanguage),
			pq.Array(&f.ExceptLanguage),
			&tags,
			&exceptTags,
			&tagsMatchLogic,
			&exceptTagsMatchLogic,
			pq.Array(&f.Origins),
			pq.Array(&f.ExceptOrigins),
			&f.MinSeeders,
			&f.MaxSeeders,
			&f.MinLeechers,
			&f.MaxLeechers,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		f.MinSize = minSize.String
		f.MaxSize = maxSize.String
		f.Delay = int(delay.Int32)
		f.MaxDownloads = int(maxDownloads.Int32)
		f.MaxDownloadsUnit = domain.FilterMaxDownloadsUnit(maxDownloadsUnit.String)
		f.MatchReleases = matchReleases.String
		f.ExceptReleases = exceptReleases.String
		f.MatchReleaseGroups = matchReleaseGroups.String
		f.ExceptReleaseGroups = exceptReleaseGroups.String
		f.MatchReleaseTags = matchReleaseTags.String
		f.ExceptReleaseTags = exceptReleaseTags.String
		f.MatchDescription = matchDescription.String
		f.ExceptDescription = exceptDescription.String
		f.FreeleechPercent = freeleechPercent.String
		f.Shows = shows.String
		f.Seasons = seasons.String
		f.Episodes = episodes.String
		f.Years = years.String
		f.Months = months.String
		f.Days = days.String
		f.Artists = artists.String
		f.Albums = albums.String
		f.LogScore = int(logScore.Int32)
		f.Log = hasLog.Bool
		f.Cue = hasCue.Bool
		f.PerfectFlac = perfectFlac.Bool
		f.MatchCategories = matchCategories.String
		f.ExceptCategories = exceptCategories.String
		f.MatchUploaders = matchUploaders.String
		f.ExceptUploaders = exceptUploaders.String
		f.Tags = tags.String
		f.ExceptTags = exceptTags.String
		f.TagsMatchLogic = tagsMatchLogic.String
		f.ExceptTagsMatchLogic = exceptTagsMatchLogic.String
		f.UseRegex = useRegex.Bool
		f.Scene = scene.Bool
		f.Freeleech = freeleech.Bool

		f.Rejections = []string{}

		filters = append(filters, &f)
	}

	return filters, nil
}

func (r *FilterRepo) FindExternalFiltersByID(ctx context.Context, filterId int) ([]domain.FilterExternal, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"fe.id",
			"fe.name",
			"fe.idx",
			"fe.type",
			"fe.enabled",
			"fe.exec_cmd",
			"fe.exec_args",
			"fe.exec_expect_status",
			"fe.webhook_host",
			"fe.webhook_method",
			"fe.webhook_data",
			"fe.webhook_headers",
			"fe.webhook_expect_status",
			"fe.webhook_retry_status",
			"fe.webhook_retry_attempts",
			"fe.webhook_retry_delay_seconds",
		).
		From("filter_external fe").
		Where(sq.Eq{"fe.filter_id": filterId})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}
		return nil, errors.Wrap(err, "error executing query")
	}

	var externalFilters []domain.FilterExternal

	for rows.Next() {
		var external domain.FilterExternal

		// filter external
		var extExecCmd, extExecArgs, extWebhookHost, extWebhookMethod, extWebhookHeaders, extWebhookData, extWebhookRetryStatus sql.NullString
		var extWebhookStatus, extWebhookRetryAttempts, extWebhookDelaySeconds, extExecStatus sql.NullInt32

		if err := rows.Scan(
			&external.ID,
			&external.Name,
			&external.Index,
			&external.Type,
			&external.Enabled,
			&extExecCmd,
			&extExecArgs,
			&extExecStatus,
			&extWebhookHost,
			&extWebhookMethod,
			&extWebhookData,
			&extWebhookHeaders,
			&extWebhookStatus,
			&extWebhookRetryStatus,
			&extWebhookRetryAttempts,
			&extWebhookDelaySeconds,
		); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		external.ExecCmd = extExecCmd.String
		external.ExecArgs = extExecArgs.String
		external.ExecExpectStatus = int(extExecStatus.Int32)

		external.WebhookHost = extWebhookHost.String
		external.WebhookMethod = extWebhookMethod.String
		external.WebhookData = extWebhookData.String
		external.WebhookHeaders = extWebhookHeaders.String
		external.WebhookExpectStatus = int(extWebhookStatus.Int32)
		external.WebhookRetryStatus = extWebhookRetryStatus.String
		external.WebhookRetryAttempts = int(extWebhookRetryAttempts.Int32)
		external.WebhookRetryDelaySeconds = int(extWebhookDelaySeconds.Int32)

		externalFilters = append(externalFilters, external)
	}

	return externalFilters, nil
}

func (r *FilterRepo) Store(ctx context.Context, filter *domain.Filter) error {
	queryBuilder := r.db.squirrel.
		Insert("filter").
		Columns(
			"name",
			"enabled",
			"min_size",
			"max_size",
			"delay",
			"priority",
			"max_downloads",
			"max_downloads_unit",
			"match_releases",
			"except_releases",
			"use_regex",
			"match_release_groups",
			"except_release_groups",
			"match_release_tags",
			"except_release_tags",
			"use_regex_release_tags",
			"match_description",
			"except_description",
			"use_regex_description",
			"scene",
			"freeleech",
			"freeleech_percent",
			"smart_episode",
			"shows",
			"seasons",
			"episodes",
			"resolutions",
			"codecs",
			"sources",
			"containers",
			"match_hdr",
			"except_hdr",
			"match_other",
			"except_other",
			"years",
			"months",
			"days",
			"match_categories",
			"except_categories",
			"match_uploaders",
			"except_uploaders",
			"match_language",
			"except_language",
			"tags",
			"except_tags",
			"tags_match_logic",
			"except_tags_match_logic",
			"artists",
			"albums",
			"release_types_match",
			"formats",
			"quality",
			"media",
			"log_score",
			"has_log",
			"has_cue",
			"perfect_flac",
			"origins",
			"except_origins",
			"min_seeders",
			"max_seeders",
			"min_leechers",
			"max_leechers",
		).
		Values(
			filter.Name,
			filter.Enabled,
			filter.MinSize,
			filter.MaxSize,
			filter.Delay,
			filter.Priority,
			filter.MaxDownloads,
			filter.MaxDownloadsUnit,
			filter.MatchReleases,
			filter.ExceptReleases,
			filter.UseRegex,
			filter.MatchReleaseGroups,
			filter.ExceptReleaseGroups,
			filter.MatchReleaseTags,
			filter.ExceptReleaseTags,
			filter.UseRegexReleaseTags,
			filter.MatchDescription,
			filter.ExceptDescription,
			filter.UseRegexDescription,
			filter.Scene,
			filter.Freeleech,
			filter.FreeleechPercent,
			filter.SmartEpisode,
			filter.Shows,
			filter.Seasons,
			filter.Episodes,
			pq.Array(filter.Resolutions),
			pq.Array(filter.Codecs),
			pq.Array(filter.Sources),
			pq.Array(filter.Containers),
			pq.Array(filter.MatchHDR),
			pq.Array(filter.ExceptHDR),
			pq.Array(filter.MatchOther),
			pq.Array(filter.ExceptOther),
			filter.Years,
			filter.Months,
			filter.Days,
			filter.MatchCategories,
			filter.ExceptCategories,
			filter.MatchUploaders,
			filter.ExceptUploaders,
			pq.Array(filter.MatchLanguage),
			pq.Array(filter.ExceptLanguage),
			filter.Tags,
			filter.ExceptTags,
			filter.TagsMatchLogic,
			filter.ExceptTagsMatchLogic,
			filter.Artists,
			filter.Albums,
			pq.Array(filter.MatchReleaseTypes),
			pq.Array(filter.Formats),
			pq.Array(filter.Quality),
			pq.Array(filter.Media),
			filter.LogScore,
			filter.Log,
			filter.Cue,
			filter.PerfectFlac,
			pq.Array(filter.Origins),
			pq.Array(filter.ExceptOrigins),
			filter.MinSeeders,
			filter.MaxSeeders,
			filter.MinLeechers,
			filter.MaxLeechers,
		).
		Suffix("RETURNING id").RunWith(r.db.handler)

	// return values
	var retID int

	if err := queryBuilder.QueryRowContext(ctx).Scan(&retID); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	filter.ID = retID

	return nil
}

func (r *FilterRepo) Update(ctx context.Context, filter *domain.Filter) error {
	var err error

	queryBuilder := r.db.squirrel.
		Update("filter").
		Set("name", filter.Name).
		Set("enabled", filter.Enabled).
		Set("min_size", filter.MinSize).
		Set("max_size", filter.MaxSize).
		Set("delay", filter.Delay).
		Set("priority", filter.Priority).
		Set("max_downloads", filter.MaxDownloads).
		Set("max_downloads_unit", filter.MaxDownloadsUnit).
		Set("use_regex", filter.UseRegex).
		Set("match_releases", filter.MatchReleases).
		Set("except_releases", filter.ExceptReleases).
		Set("match_release_groups", filter.MatchReleaseGroups).
		Set("except_release_groups", filter.ExceptReleaseGroups).
		Set("match_release_tags", filter.MatchReleaseTags).
		Set("except_release_tags", filter.ExceptReleaseTags).
		Set("use_regex_release_tags", filter.UseRegexReleaseTags).
		Set("match_description", filter.MatchDescription).
		Set("except_description", filter.ExceptDescription).
		Set("use_regex_description", filter.UseRegexDescription).
		Set("scene", filter.Scene).
		Set("freeleech", filter.Freeleech).
		Set("freeleech_percent", filter.FreeleechPercent).
		Set("smart_episode", filter.SmartEpisode).
		Set("shows", filter.Shows).
		Set("seasons", filter.Seasons).
		Set("episodes", filter.Episodes).
		Set("resolutions", pq.Array(filter.Resolutions)).
		Set("codecs", pq.Array(filter.Codecs)).
		Set("sources", pq.Array(filter.Sources)).
		Set("containers", pq.Array(filter.Containers)).
		Set("match_hdr", pq.Array(filter.MatchHDR)).
		Set("except_hdr", pq.Array(filter.ExceptHDR)).
		Set("match_other", pq.Array(filter.MatchOther)).
		Set("except_other", pq.Array(filter.ExceptOther)).
		Set("years", filter.Years).
		Set("months", filter.Months).
		Set("days", filter.Days).
		Set("match_categories", filter.MatchCategories).
		Set("except_categories", filter.ExceptCategories).
		Set("match_uploaders", filter.MatchUploaders).
		Set("except_uploaders", filter.ExceptUploaders).
		Set("match_language", pq.Array(filter.MatchLanguage)).
		Set("except_language", pq.Array(filter.ExceptLanguage)).
		Set("tags", filter.Tags).
		Set("except_tags", filter.ExceptTags).
		Set("tags_match_logic", filter.TagsMatchLogic).
		Set("except_tags_match_logic", filter.ExceptTagsMatchLogic).
		Set("artists", filter.Artists).
		Set("albums", filter.Albums).
		Set("release_types_match", pq.Array(filter.MatchReleaseTypes)).
		Set("formats", pq.Array(filter.Formats)).
		Set("quality", pq.Array(filter.Quality)).
		Set("media", pq.Array(filter.Media)).
		Set("log_score", filter.LogScore).
		Set("has_log", filter.Log).
		Set("has_cue", filter.Cue).
		Set("perfect_flac", filter.PerfectFlac).
		Set("origins", pq.Array(filter.Origins)).
		Set("except_origins", pq.Array(filter.ExceptOrigins)).
		Set("min_seeders", filter.MinSeeders).
		Set("max_seeders", filter.MaxSeeders).
		Set("min_leechers", filter.MinLeechers).
		Set("max_leechers", filter.MaxLeechers).
		Set("updated_at", time.Now().Format(time.RFC3339)).
		Where(sq.Eq{"id": filter.ID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.handler.ExecContext(ctx, query, args...)
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

func (r *FilterRepo) UpdatePartial(ctx context.Context, filter domain.FilterUpdate) error {
	var err error

	q := r.db.squirrel.Update("filter")

	if filter.Name != nil {
		q = q.Set("name", filter.Name)
	}
	if filter.Enabled != nil {
		q = q.Set("enabled", filter.Enabled)
	}
	if filter.MinSize != nil {
		q = q.Set("min_size", filter.MinSize)
	}
	if filter.MaxSize != nil {
		q = q.Set("max_size", filter.MaxSize)
	}
	if filter.Delay != nil {
		q = q.Set("delay", filter.Delay)
	}
	if filter.Priority != nil {
		q = q.Set("priority", filter.Priority)
	}
	if filter.MaxDownloads != nil {
		q = q.Set("max_downloads", filter.MaxDownloads)
	}
	if filter.MaxDownloadsUnit != nil {
		q = q.Set("max_downloads_unit", filter.MaxDownloadsUnit)
	}
	if filter.UseRegex != nil {
		q = q.Set("use_regex", filter.UseRegex)
	}
	if filter.MatchReleases != nil {
		q = q.Set("match_releases", filter.MatchReleases)
	}
	if filter.ExceptReleases != nil {
		q = q.Set("except_releases", filter.ExceptReleases)
	}
	if filter.MatchReleaseGroups != nil {
		q = q.Set("match_release_groups", filter.MatchReleaseGroups)
	}
	if filter.ExceptReleaseGroups != nil {
		q = q.Set("except_release_groups", filter.ExceptReleaseGroups)
	}
	if filter.MatchReleaseTags != nil {
		q = q.Set("match_release_tags", filter.MatchReleaseTags)
	}
	if filter.ExceptReleaseTags != nil {
		q = q.Set("except_release_tags", filter.ExceptReleaseTags)
	}
	if filter.UseRegexReleaseTags != nil {
		q = q.Set("use_regex_release_tags", filter.UseRegexReleaseTags)
	}
	if filter.MatchDescription != nil {
		q = q.Set("match_description", filter.MatchDescription)
	}
	if filter.ExceptDescription != nil {
		q = q.Set("except_description", filter.ExceptDescription)
	}
	if filter.UseRegexDescription != nil {
		q = q.Set("use_regex_description", filter.UseRegexDescription)
	}
	if filter.Scene != nil {
		q = q.Set("scene", filter.Scene)
	}
	if filter.Freeleech != nil {
		q = q.Set("freeleech", filter.Freeleech)
	}
	if filter.FreeleechPercent != nil {
		q = q.Set("freeleech_percent", filter.FreeleechPercent)
	}
	if filter.SmartEpisode != nil {
		q = q.Set("smart_episode", filter.SmartEpisode)
	}
	if filter.Shows != nil {
		q = q.Set("shows", filter.Shows)
	}
	if filter.Seasons != nil {
		q = q.Set("seasons", filter.Seasons)
	}
	if filter.Episodes != nil {
		q = q.Set("episodes", filter.Episodes)
	}
	if filter.Resolutions != nil {
		q = q.Set("resolutions", pq.Array(filter.Resolutions))
	}
	if filter.Codecs != nil {
		q = q.Set("codecs", pq.Array(filter.Codecs))
	}
	if filter.Sources != nil {
		q = q.Set("sources", pq.Array(filter.Sources))
	}
	if filter.Containers != nil {
		q = q.Set("containers", pq.Array(filter.Containers))
	}
	if filter.MatchHDR != nil {
		q = q.Set("match_hdr", pq.Array(filter.MatchHDR))
	}
	if filter.ExceptHDR != nil {
		q = q.Set("except_hdr", pq.Array(filter.ExceptHDR))
	}
	if filter.MatchOther != nil {
		q = q.Set("match_other", pq.Array(filter.MatchOther))
	}
	if filter.ExceptOther != nil {
		q = q.Set("except_other", pq.Array(filter.ExceptOther))
	}
	if filter.Years != nil {
		q = q.Set("years", filter.Years)
	}
	if filter.Months != nil {
		q = q.Set("months", filter.Months)
	}
	if filter.Days != nil {
		q = q.Set("days", filter.Days)
	}
	if filter.MatchCategories != nil {
		q = q.Set("match_categories", filter.MatchCategories)
	}
	if filter.ExceptCategories != nil {
		q = q.Set("except_categories", filter.ExceptCategories)
	}
	if filter.MatchUploaders != nil {
		q = q.Set("match_uploaders", filter.MatchUploaders)
	}
	if filter.ExceptUploaders != nil {
		q = q.Set("except_uploaders", filter.ExceptUploaders)
	}
	if filter.MatchLanguage != nil {
		q = q.Set("match_language", pq.Array(filter.MatchLanguage))
	}
	if filter.ExceptLanguage != nil {
		q = q.Set("except_language", pq.Array(filter.ExceptLanguage))
	}
	if filter.Tags != nil {
		q = q.Set("tags", filter.Tags)
	}
	if filter.ExceptTags != nil {
		q = q.Set("except_tags", filter.ExceptTags)
	}
	if filter.TagsMatchLogic != nil {
		q = q.Set("tags_match_logic", filter.TagsMatchLogic)
	}
	if filter.ExceptTagsMatchLogic != nil {
		q = q.Set("except_tags_match_logic", filter.ExceptTagsMatchLogic)
	}
	if filter.Artists != nil {
		q = q.Set("artists", filter.Artists)
	}
	if filter.Albums != nil {
		q = q.Set("albums", filter.Albums)
	}
	if filter.MatchReleaseTypes != nil {
		q = q.Set("release_types_match", pq.Array(filter.MatchReleaseTypes))
	}
	if filter.Formats != nil {
		q = q.Set("formats", pq.Array(filter.Formats))
	}
	if filter.Quality != nil {
		q = q.Set("quality", pq.Array(filter.Quality))
	}
	if filter.Media != nil {
		q = q.Set("media", pq.Array(filter.Media))
	}
	if filter.LogScore != nil {
		q = q.Set("log_score", filter.LogScore)
	}
	if filter.Log != nil {
		q = q.Set("has_log", filter.Log)
	}
	if filter.Cue != nil {
		q = q.Set("has_cue", filter.Cue)
	}
	if filter.PerfectFlac != nil {
		q = q.Set("perfect_flac", filter.PerfectFlac)
	}
	if filter.Origins != nil {
		q = q.Set("origins", pq.Array(filter.Origins))
	}
	if filter.ExceptOrigins != nil {
		q = q.Set("except_origins", pq.Array(filter.ExceptOrigins))
	}
	if filter.MinSeeders != nil {
		q = q.Set("min_seeders", filter.MinSeeders)
	}
	if filter.MaxSeeders != nil {
		q = q.Set("max_seeders", filter.MaxSeeders)
	}
	if filter.MinLeechers != nil {
		q = q.Set("min_leechers", filter.MinLeechers)
	}
	if filter.MaxLeechers != nil {
		q = q.Set("max_leechers", filter.MaxLeechers)
	}

	q = q.Where(sq.Eq{"id": filter.ID})

	query, args, err := q.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if count == 0 {
		return errors.New("no rows affected")
	}

	return nil
}

func (r *FilterRepo) ToggleEnabled(ctx context.Context, filterID int, enabled bool) error {
	var err error

	queryBuilder := r.db.squirrel.
		Update("filter").
		Set("enabled", enabled).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": filterID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.handler.ExecContext(ctx, query, args...)
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

func (r *FilterRepo) StoreIndexerConnections(ctx context.Context, filterID int, indexers []domain.Indexer) error {
	tx, err := r.db.handler.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	deleteQueryBuilder := r.db.squirrel.
		Delete("filter_indexer").
		Where(sq.Eq{"filter_id": filterID})

	deleteQuery, deleteArgs, err := deleteQueryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = tx.ExecContext(ctx, deleteQuery, deleteArgs...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if len(indexers) == 0 {
		if err := tx.Commit(); err != nil {
			return errors.Wrap(err, "error store indexers for filter: %d", filterID)
		}

		return nil
	}

	queryBuilder := r.db.squirrel.
		Insert("filter_indexer").
		Columns("filter_id", "indexer_id")

	for _, indexer := range indexers {
		queryBuilder = queryBuilder.Values(filterID, indexer.ID)
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "error store indexers for filter: %d", filterID)
	}

	r.log.Debug().Msgf("filter.StoreIndexerConnections: indexers on filter: %d", filterID)

	return nil
}

func (r *FilterRepo) StoreIndexerConnection(ctx context.Context, filterID int, indexerID int) error {
	queryBuilder := r.db.squirrel.
		Insert("filter_indexer").Columns("filter_id", "indexer_id").
		Values(filterID, indexerID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *FilterRepo) DeleteIndexerConnections(ctx context.Context, filterID int) error {
	queryBuilder := r.db.squirrel.
		Delete("filter_indexer").
		Where(sq.Eq{"filter_id": filterID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *FilterRepo) DeleteFilterExternal(ctx context.Context, filterID int) error {
	queryBuilder := r.db.squirrel.
		Delete("filter_external").
		Where(sq.Eq{"filter_id": filterID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *FilterRepo) Delete(ctx context.Context, filterID int) error {
	queryBuilder := r.db.squirrel.
		Delete("filter").
		Where(sq.Eq{"id": filterID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return errors.Wrap(err, "error getting rows affected")
	} else if rowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	r.log.Debug().Msgf("filter.delete: successfully deleted: %v", filterID)

	return nil
}

// GetDownloadsByFilterId looks up how many `PENDING` or `PUSH_APPROVED`
// releases there have been for the given filter in the current time window
// starting at the start of the unit (since the beginning of the most recent
// hour/day/week).
//
// See also
// https://github.com/autobrr/autobrr/pull/1285#pullrequestreview-1795913581
func (r *FilterRepo) GetDownloadsByFilterId(ctx context.Context, filterID int) (*domain.FilterDownloads, error) {
	if r.db.Driver == "sqlite" {
		return r.downloadsByFilterSqlite(ctx, filterID)
	}

	return r.downloadsByFilterPostgres(ctx, filterID)
}

func (r *FilterRepo) downloadsByFilterSqlite(ctx context.Context, filterID int) (*domain.FilterDownloads, error) {
	query := `SELECT
	COUNT(CASE WHEN CAST(strftime('%s', datetime(release_action_status.timestamp, 'localtime')) AS INTEGER) >= CAST(strftime('%s', strftime('%Y-%m-%dT%H:00:00', datetime('now','localtime'))) AS INTEGER) THEN 1 END) as "hour_count",
	COUNT(CASE WHEN CAST(strftime('%s', datetime(release_action_status.timestamp, 'localtime')) AS INTEGER) >= CAST(strftime('%s', datetime('now', 'localtime', 'start of day')) AS INTEGER) THEN 1 END) as "day_count",
	COUNT(CASE WHEN CAST(strftime('%s', datetime(release_action_status.timestamp, 'localtime')) AS INTEGER) >= CAST(strftime('%s', datetime('now', 'localtime', 'weekday 0', '-7 days', 'start of day')) AS INTEGER) THEN 1 END) as "week_count",
	COUNT(CASE WHEN CAST(strftime('%s', datetime(release_action_status.timestamp, 'localtime')) AS INTEGER) >= CAST(strftime('%s', datetime('now', 'localtime', 'start of month')) AS INTEGER) THEN 1 END) as "month_count",
	COUNT(*) as "total_count"
FROM release_action_status
WHERE (release_action_status.status = 'PUSH_APPROVED' OR release_action_status.status = 'PENDING') AND release_action_status.filter_id = ?;`

	row := r.db.handler.QueryRowContext(ctx, query, filterID)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var f domain.FilterDownloads

	if err := row.Scan(&f.HourCount, &f.DayCount, &f.WeekCount, &f.MonthCount, &f.TotalCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning stats data sqlite")
	}

	r.log.Trace().Msgf("filter %v downloads: %+v", filterID, &f)

	return &f, nil
}

func (r *FilterRepo) downloadsByFilterPostgres(ctx context.Context, filterID int) (*domain.FilterDownloads, error) {
	query := `SELECT
    COALESCE(SUM(CASE WHEN release_action_status.timestamp >= date_trunc('hour', CURRENT_TIMESTAMP) THEN 1 ELSE 0 END),0) as "hour_count",
    COALESCE(SUM(CASE WHEN release_action_status.timestamp >= date_trunc('day', CURRENT_DATE) THEN 1 ELSE 0 END),0) as "day_count",
    COALESCE(SUM(CASE WHEN release_action_status.timestamp >= date_trunc('week', CURRENT_DATE) THEN 1 ELSE 0 END),0) as "week_count",
    COALESCE(SUM(CASE WHEN release_action_status.timestamp >= date_trunc('month', CURRENT_DATE) THEN 1 ELSE 0 END),0) as "month_count",
    count(*) as "total_count"
FROM release_action_status
WHERE (release_action_status.status = 'PUSH_APPROVED' OR release_action_status.status = 'PENDING') AND release_action_status.filter_id = $1;`

	row := r.db.handler.QueryRowContext(ctx, query, filterID)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var f domain.FilterDownloads

	if err := row.Scan(&f.HourCount, &f.DayCount, &f.WeekCount, &f.MonthCount, &f.TotalCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning stats data postgres")
	}

	return &f, nil
}

func (r *FilterRepo) StoreFilterExternal(ctx context.Context, filterID int, externalFilters []domain.FilterExternal) error {
	tx, err := r.db.handler.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	deleteQueryBuilder := r.db.squirrel.
		Delete("filter_external").
		Where(sq.Eq{"filter_id": filterID})

	deleteQuery, deleteArgs, err := deleteQueryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = tx.ExecContext(ctx, deleteQuery, deleteArgs...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if len(externalFilters) == 0 {
		if err := tx.Commit(); err != nil {
			return errors.Wrap(err, "error delete external filters for filter: %d", filterID)
		}

		return nil
	}

	qb := r.db.squirrel.
		Insert("filter_external").
		Columns(
			"name",
			"idx",
			"type",
			"enabled",
			"exec_cmd",
			"exec_args",
			"exec_expect_status",
			"webhook_host",
			"webhook_method",
			"webhook_data",
			"webhook_headers",
			"webhook_expect_status",
			"webhook_retry_status",
			"webhook_retry_attempts",
			"webhook_retry_delay_seconds",
			"filter_id",
		)

	for _, external := range externalFilters {
		qb = qb.Values(
			external.Name,
			external.Index,
			external.Type,
			external.Enabled,
			toNullString(external.ExecCmd),
			toNullString(external.ExecArgs),
			toNullInt32(int32(external.ExecExpectStatus)),
			toNullString(external.WebhookHost),
			toNullString(external.WebhookMethod),
			toNullString(external.WebhookData),
			toNullString(external.WebhookHeaders),
			toNullInt32(int32(external.WebhookExpectStatus)),
			toNullString(external.WebhookRetryStatus),
			toNullInt32(int32(external.WebhookRetryAttempts)),
			toNullInt32(int32(external.WebhookRetryDelaySeconds)),
			filterID,
		)
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "error store external filters for filter: %d", filterID)
	}

	r.log.Debug().Msgf("filter.StoreFilterExternal: store external filters on filter: %d", filterID)

	return nil
}
