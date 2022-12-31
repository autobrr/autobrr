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
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, errors.Wrap(err, "error begin transaction")
	}
	defer tx.Rollback()

	filters, err := r.find(ctx, tx, params)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "error commit transaction find releases")
	}

	return filters, nil
}

func (r *FilterRepo) find(ctx context.Context, tx *Tx, params domain.FilterQueryParams) ([]domain.Filter, error) {

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
		Distinct().
		Column(sq.Alias(actionCountQuery, "action_count")).
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

	rows, err := tx.QueryContext(ctx, query, args...)
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

func (r *FilterRepo) ListFilters(ctx context.Context) ([]domain.Filter, error) {
	actionCountQuery := r.db.squirrel.
		RunWith(r.db.handler).
		Select("COUNT(*)").
		From("action a").
		Where(sq.Eq{"a.filter_id": "f.id"})

	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
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

	rows, err := queryBuilder.Query()
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
		RunWith(r.db.handler).
		Select(
			"id",
			"enabled",
			"name",
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
			"match_categories",
			"except_categories",
			"match_uploaders",
			"except_uploaders",
			"tags",
			"except_tags",
			"origins",
			"except_origins",
			"external_script_enabled",
			"external_script_cmd",
			"external_script_args",
			"external_script_expect_status",
			"external_webhook_enabled",
			"external_webhook_host",
			"external_webhook_data",
			"external_webhook_expect_status",
			"created_at",
			"updated_at",
		).
		From("filter").
		Where(sq.Eq{"id": filterID})

	row := queryBuilder.QueryRow()

	var f domain.Filter
	var minSize, maxSize, maxDownloadsUnit, matchReleases, exceptReleases, matchReleaseGroups, exceptReleaseGroups, matchReleaseTags, exceptReleaseTags, freeleechPercent, shows, seasons, episodes, years, artists, albums, matchCategories, exceptCategories, matchUploaders, exceptUploaders, tags, exceptTags, extScriptCmd, extScriptArgs, extWebhookHost, extWebhookData sql.NullString
	var useRegex, scene, freeleech, hasLog, hasCue, perfectFlac, extScriptEnabled, extWebhookEnabled sql.NullBool
	var delay, maxDownloads, logScore, extWebhookStatus, extScriptStatus sql.NullInt32

	if err := row.Scan(&f.ID, &f.Enabled, &f.Name, &minSize, &maxSize, &delay, &f.Priority, &maxDownloads, &maxDownloadsUnit, &matchReleases, &exceptReleases, &useRegex, &matchReleaseGroups, &exceptReleaseGroups, &matchReleaseTags, &exceptReleaseTags, &f.UseRegexReleaseTags, &scene, &freeleech, &freeleechPercent, &f.SmartEpisode, &shows, &seasons, &episodes, pq.Array(&f.Resolutions), pq.Array(&f.Codecs), pq.Array(&f.Sources), pq.Array(&f.Containers), pq.Array(&f.MatchHDR), pq.Array(&f.ExceptHDR), pq.Array(&f.MatchOther), pq.Array(&f.ExceptOther), &years, &artists, &albums, pq.Array(&f.MatchReleaseTypes), pq.Array(&f.Formats), pq.Array(&f.Quality), pq.Array(&f.Media), &logScore, &hasLog, &hasCue, &perfectFlac, &matchCategories, &exceptCategories, &matchUploaders, &exceptUploaders, &tags, &exceptTags, pq.Array(&f.Origins), pq.Array(&f.ExceptOrigins), &extScriptEnabled, &extScriptCmd, &extScriptArgs, &extScriptStatus, &extWebhookEnabled, &extWebhookHost, &extWebhookData, &extWebhookStatus, &f.CreatedAt, &f.UpdatedAt); err != nil {
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
	f.FreeleechPercent = freeleechPercent.String
	f.Shows = shows.String
	f.Seasons = seasons.String
	f.Episodes = episodes.String
	f.Years = years.String
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
	f.UseRegex = useRegex.Bool
	f.Scene = scene.Bool
	f.Freeleech = freeleech.Bool

	f.ExternalScriptEnabled = extScriptEnabled.Bool
	f.ExternalScriptCmd = extScriptCmd.String
	f.ExternalScriptArgs = extScriptArgs.String
	f.ExternalScriptExpectStatus = int(extScriptStatus.Int32)

	f.ExternalWebhookEnabled = extWebhookEnabled.Bool
	f.ExternalWebhookHost = extWebhookHost.String
	f.ExternalWebhookData = extWebhookData.String
	f.ExternalWebhookExpectStatus = int(extWebhookStatus.Int32)

	return &f, nil
}

// FindByIndexerIdentifier find active filters with active indexer only
func (r *FilterRepo) FindByIndexerIdentifier(indexer string) ([]domain.Filter, error) {
	ctx := context.TODO()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error begin transaction")
	}
	defer tx.Rollback()

	filters, err := r.findByIndexerIdentifier(ctx, tx, indexer)
	if err != nil {
		return nil, err
	}

	for i, filter := range filters {
		downloads, err := r.attachDownloadsByFilter(ctx, tx, filter.ID)
		if err != nil {
			continue
		}
		filters[i].Downloads = downloads
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "error finding filter by identifier")
	}

	return filters, nil
}

func (r *FilterRepo) findByIndexerIdentifier(ctx context.Context, tx *Tx, indexer string) ([]domain.Filter, error) {
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
			"f.tags",
			"f.except_tags",
			"f.origins",
			"f.except_origins",
			"f.external_script_enabled",
			"f.external_script_cmd",
			"f.external_script_args",
			"f.external_script_expect_status",
			"f.external_webhook_enabled",
			"f.external_webhook_host",
			"f.external_webhook_data",
			"f.external_webhook_expect_status",
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

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	var filters []domain.Filter
	for rows.Next() {
		var f domain.Filter

		var minSize, maxSize, maxDownloadsUnit, matchReleases, exceptReleases, matchReleaseGroups, exceptReleaseGroups, matchReleaseTags, exceptReleaseTags, freeleechPercent, shows, seasons, episodes, years, artists, albums, matchCategories, exceptCategories, matchUploaders, exceptUploaders, tags, exceptTags, extScriptCmd, extScriptArgs, extWebhookHost, extWebhookData sql.NullString
		var useRegex, scene, freeleech, hasLog, hasCue, perfectFlac, extScriptEnabled, extWebhookEnabled sql.NullBool
		var delay, maxDownloads, logScore, extWebhookStatus, extScriptStatus sql.NullInt32

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, &minSize, &maxSize, &delay, &f.Priority, &maxDownloads, &maxDownloadsUnit, &matchReleases, &exceptReleases, &useRegex, &matchReleaseGroups, &exceptReleaseGroups, &matchReleaseTags, &exceptReleaseTags, &f.UseRegexReleaseTags, &scene, &freeleech, &freeleechPercent, &f.SmartEpisode, &shows, &seasons, &episodes, pq.Array(&f.Resolutions), pq.Array(&f.Codecs), pq.Array(&f.Sources), pq.Array(&f.Containers), pq.Array(&f.MatchHDR), pq.Array(&f.ExceptHDR), pq.Array(&f.MatchOther), pq.Array(&f.ExceptOther), &years, &artists, &albums, pq.Array(&f.MatchReleaseTypes), pq.Array(&f.Formats), pq.Array(&f.Quality), pq.Array(&f.Media), &logScore, &hasLog, &hasCue, &perfectFlac, &matchCategories, &exceptCategories, &matchUploaders, &exceptUploaders, &tags, &exceptTags, pq.Array(&f.Origins), pq.Array(&f.ExceptOrigins), &extScriptEnabled, &extScriptCmd, &extScriptArgs, &extScriptStatus, &extWebhookEnabled, &extWebhookHost, &extWebhookData, &extWebhookStatus, &f.CreatedAt, &f.UpdatedAt); err != nil {
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
		f.FreeleechPercent = freeleechPercent.String
		f.Shows = shows.String
		f.Seasons = seasons.String
		f.Episodes = episodes.String
		f.Years = years.String
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
		f.UseRegex = useRegex.Bool
		f.Scene = scene.Bool
		f.Freeleech = freeleech.Bool

		f.ExternalScriptEnabled = extScriptEnabled.Bool
		f.ExternalScriptCmd = extScriptCmd.String
		f.ExternalScriptArgs = extScriptArgs.String
		f.ExternalScriptExpectStatus = int(extScriptStatus.Int32)

		f.ExternalWebhookEnabled = extWebhookEnabled.Bool
		f.ExternalWebhookHost = extWebhookHost.String
		f.ExternalWebhookData = extWebhookData.String
		f.ExternalWebhookExpectStatus = int(extWebhookStatus.Int32)

		filters = append(filters, f)
	}

	return filters, nil
}

func (r *FilterRepo) Store(ctx context.Context, filter domain.Filter) (*domain.Filter, error) {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
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
			"match_categories",
			"except_categories",
			"match_uploaders",
			"except_uploaders",
			"tags",
			"except_tags",
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
			"external_script_enabled",
			"external_script_cmd",
			"external_script_args",
			"external_script_expect_status",
			"external_webhook_enabled",
			"external_webhook_host",
			"external_webhook_data",
			"external_webhook_expect_status",
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
			filter.MatchCategories,
			filter.ExceptCategories,
			filter.MatchUploaders,
			filter.ExceptUploaders,
			filter.Tags,
			filter.ExceptTags,
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
			filter.ExternalScriptEnabled,
			filter.ExternalScriptCmd,
			filter.ExternalScriptArgs,
			filter.ExternalScriptExpectStatus,
			filter.ExternalWebhookEnabled,
			filter.ExternalWebhookHost,
			filter.ExternalWebhookData,
			filter.ExternalWebhookExpectStatus,
		).
		Suffix("RETURNING id")

	// return values
	var retID int	
	if err := queryBuilder.QueryRow().Scan(&retID); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	filter.ID = retID

	return &filter, nil
}

func (r *FilterRepo) Update(ctx context.Context, filter domain.Filter) (*domain.Filter, error) {
	var err error

	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
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
		Set("match_categories", filter.MatchCategories).
		Set("except_categories", filter.ExceptCategories).
		Set("match_uploaders", filter.MatchUploaders).
		Set("except_uploaders", filter.ExceptUploaders).
		Set("tags", filter.Tags).
		Set("except_tags", filter.ExceptTags).
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
		Set("external_script_enabled", filter.ExternalScriptEnabled).
		Set("external_script_cmd", filter.ExternalScriptCmd).
		Set("external_script_args", filter.ExternalScriptArgs).
		Set("external_script_expect_status", filter.ExternalScriptExpectStatus).
		Set("external_webhook_enabled", filter.ExternalWebhookEnabled).
		Set("external_webhook_host", filter.ExternalWebhookHost).
		Set("external_webhook_data", filter.ExternalWebhookData).
		Set("external_webhook_expect_status", filter.ExternalWebhookExpectStatus).
		Set("updated_at", time.Now().Format(time.RFC3339)).
		Where(sq.Eq{"id": filter.ID})

	if _, err = queryBuilder.Exec(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	return &filter, nil
}

func (r *FilterRepo) UpdatePartial(ctx context.Context, filter domain.FilterUpdate) error {
	var err error

	q := r.db.squirrel.RunWith(r.db.handler).Update("filter")

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
	if filter.Tags != nil {
		q = q.Set("tags", filter.Tags)
	}
	if filter.ExceptTags != nil {
		q = q.Set("except_tags", filter.ExceptTags)
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
	if filter.ExternalScriptEnabled != nil {
		q = q.Set("external_script_enabled", filter.ExternalScriptEnabled)
	}
	if filter.ExternalScriptCmd != nil {
		q = q.Set("external_script_cmd", filter.ExternalScriptCmd)
	}
	if filter.ExternalScriptArgs != nil {
		q = q.Set("external_script_args", filter.ExternalScriptArgs)
	}
	if filter.ExternalScriptExpectStatus != nil {
		q = q.Set("external_script_expect_status", filter.ExternalScriptExpectStatus)
	}
	if filter.ExternalWebhookEnabled != nil {
		q = q.Set("external_webhook_enabled", filter.ExternalWebhookEnabled)
	}
	if filter.ExternalWebhookHost != nil {
		q = q.Set("external_webhook_host", filter.ExternalWebhookHost)
	}
	if filter.ExternalWebhookData != nil {
		q = q.Set("external_webhook_data", filter.ExternalWebhookData)
	}
	if filter.ExternalWebhookExpectStatus != nil {
		q = q.Set("external_webhook_expect_status", filter.ExternalWebhookExpectStatus)
	}

	q = q.Where(sq.Eq{"id": filter.ID})

	result, err := q.Exec()
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
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Update("filter").
		Set("enabled", enabled).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": filterID})

	if _, err := queryBuilder.Exec(); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *FilterRepo) StoreIndexerConnections(ctx context.Context, filterID int, indexers []domain.Indexer) error {
	tx, err := r.db.handler.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	deleteQueryBuilder := r.db.squirrel.
		RunWith(tx).
		Delete("filter_indexer").
		Where(sq.Eq{"filter_id": filterID})

	if _, err := deleteQueryBuilder.Exec(); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	for _, indexer := range indexers {
		queryBuilder := r.db.squirrel.
			RunWith(tx).
			Insert("filter_indexer").Columns("filter_id", "indexer_id").
			Values(filterID, indexer.ID)

		if _, err := queryBuilder.Exec(); err != nil {
			return errors.Wrap(err, "error executing query")
		}

		r.log.Debug().Msgf("filter.StoreIndexerConnections: store '%v' on filter: %v", indexer.Name, filterID)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "error store indexers for filter: %v", filterID)
	}

	return nil
}

func (r *FilterRepo) StoreIndexerConnection(ctx context.Context, filterID int, indexerID int) error {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Insert("filter_indexer").Columns("filter_id", "indexer_id").
		Values(filterID, indexerID)

	if _, err := queryBuilder.Exec(); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *FilterRepo) DeleteIndexerConnections(ctx context.Context, filterID int) error {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Delete("filter_indexer").
		Where(sq.Eq{"filter_id": filterID})

	if _, err := queryBuilder.Exec(); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *FilterRepo) Delete(ctx context.Context, filterID int) error {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Delete("filter").
		Where(sq.Eq{"id": filterID})

	if _, err := queryBuilder.Exec(); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Info().Msgf("filter.delete: successfully deleted: %v", filterID)

	return nil
}

func (r *FilterRepo) attachDownloadsByFilter(ctx context.Context, tx *Tx, filterID int) (*domain.FilterDownloads, error) {
	if r.db.Driver == "sqlite" {
		return r.downloadsByFilterSqlite(ctx, tx, filterID)
	}

	return r.downloadsByFilterPostgres(ctx, tx, filterID)
}

func (r *FilterRepo) downloadsByFilterSqlite(ctx context.Context, tx *Tx, filterID int) (*domain.FilterDownloads, error) {
	query := `SELECT
    IFNULL(SUM(CASE WHEN "release".timestamp >= strftime('%Y-%m-%dT%H:00:00Z', datetime('now','localtime')) THEN 1 ELSE 0 END),0) as "hour_count",
    IFNULL(SUM(CASE WHEN "release".timestamp >= datetime('now', 'localtime', 'start of day') THEN 1 ELSE 0 END),0) as "day_count",
    IFNULL(SUM(CASE WHEN "release".timestamp >= datetime('now', 'localtime', 'weekday 0', '-7 days') THEN 1 ELSE 0 END),0) as "week_count",
    IFNULL(SUM(CASE WHEN "release".timestamp >= datetime('now', 'localtime', 'start of month') THEN 1 ELSE 0 END),0) as "month_count",
    count(*) as "total_count"
FROM "release"
WHERE "release".filter_id = ?;`

	row := tx.QueryRow(query, filterID)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var f domain.FilterDownloads

	if err := row.Scan(&f.HourCount, &f.DayCount, &f.WeekCount, &f.MonthCount, &f.TotalCount); err != nil {
		return nil, errors.Wrap(err, "error scanning stats data sqlite")
	}

	r.log.Trace().Msgf("filter %v downloads: %+v", filterID, &f)

	return &f, nil
}

func (r *FilterRepo) downloadsByFilterPostgres(ctx context.Context, tx *Tx, filterID int) (*domain.FilterDownloads, error) {
	query := `SELECT
    COALESCE(SUM(CASE WHEN "release".timestamp >= date_trunc('hour', CURRENT_TIMESTAMP) THEN 1 ELSE 0 END),0) as "hour_count",
    COALESCE(SUM(CASE WHEN "release".timestamp >= date_trunc('day', CURRENT_DATE) THEN 1 ELSE 0 END),0) as "day_count",
    COALESCE(SUM(CASE WHEN "release".timestamp >= date_trunc('week', CURRENT_DATE) THEN 1 ELSE 0 END),0) as "week_count",
    COALESCE(SUM(CASE WHEN "release".timestamp >= date_trunc('month', CURRENT_DATE) THEN 1 ELSE 0 END),0) as "month_count",
    count(*) as "total_count"
FROM "release"
WHERE "release".filter_id = $1;`

	row := tx.QueryRow(query, filterID)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var f domain.FilterDownloads

	if err := row.Scan(&f.HourCount, &f.DayCount, &f.WeekCount, &f.MonthCount, &f.TotalCount); err != nil {
		return nil, errors.Wrap(err, "error scanning stats data postgres")
	}

	return &f, nil
}
