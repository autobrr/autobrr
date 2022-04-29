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

type FilterRepo struct {
	db *DB
}

func NewFilterRepo(db *DB) domain.FilterRepo {
	return &FilterRepo{db: db}
}

func (r *FilterRepo) ListFilters(ctx context.Context) ([]domain.Filter, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"enabled",
			"name",
			"match_releases",
			"except_releases",
			"created_at",
			"updated_at",
		).
		From("filter").
		OrderBy("name ASC")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.list: error building query")
		return nil, err
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.list: error executing query")
		return nil, err
	}

	defer rows.Close()

	var filters []domain.Filter
	for rows.Next() {
		var f domain.Filter

		var matchReleases, exceptReleases sql.NullString

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, &matchReleases, &exceptReleases, &f.CreatedAt, &f.UpdatedAt); err != nil {
			log.Error().Stack().Err(err).Msg("filter.list: error scanning row")
			return nil, err
		}

		f.MatchReleases = matchReleases.String
		f.ExceptReleases = exceptReleases.String

		filters = append(filters, f)
	}
	if err := rows.Err(); err != nil {
		log.Error().Stack().Err(err).Msg("filter.list: row error")
		return nil, err
	}

	return filters, nil
}

func (r *FilterRepo) FindByID(ctx context.Context, filterID int) (*domain.Filter, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"enabled",
			"name",
			"min_size",
			"max_size",
			"delay",
			"priority",
			"match_releases",
			"except_releases",
			"use_regex",
			"match_release_groups",
			"except_release_groups",
			"scene",
			"freeleech",
			"freeleech_percent",
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
			"created_at",
			"updated_at",
		).
		From("filter").
		Where("id = ?", filterID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.findByID: error building query")
		return nil, err
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		log.Error().Stack().Err(err).Msg("filter.findByID: error query row")
		return nil, err
	}

	var f domain.Filter
	var minSize, maxSize, matchReleases, exceptReleases, matchReleaseGroups, exceptReleaseGroups, freeleechPercent, shows, seasons, episodes, years, artists, albums, matchCategories, exceptCategories, matchUploaders, exceptUploaders, tags, exceptTags, origins sql.NullString
	var useRegex, scene, freeleech, hasLog, hasCue, perfectFlac sql.NullBool
	var delay, logScore sql.NullInt32

	if err := row.Scan(&f.ID, &f.Enabled, &f.Name, &minSize, &maxSize, &delay, &f.Priority, &matchReleases, &exceptReleases, &useRegex, &matchReleaseGroups, &exceptReleaseGroups, &scene, &freeleech, &freeleechPercent, &shows, &seasons, &episodes, pq.Array(&f.Resolutions), pq.Array(&f.Codecs), pq.Array(&f.Sources), pq.Array(&f.Containers), pq.Array(&f.MatchHDR), pq.Array(&f.ExceptHDR), pq.Array(&f.MatchOther), pq.Array(&f.ExceptOther), &years, &artists, &albums, pq.Array(&f.MatchReleaseTypes), pq.Array(&f.Formats), pq.Array(&f.Quality), pq.Array(&f.Media), &logScore, &hasLog, &hasCue, &perfectFlac, &matchCategories, &exceptCategories, &matchUploaders, &exceptUploaders, &tags, &exceptTags, &origins, &f.CreatedAt, &f.UpdatedAt); err != nil {
		log.Error().Stack().Err(err).Msgf("filter.findByID: %v : error scanning row", filterID)
		return nil, err
	}

	f.MinSize = minSize.String
	f.MaxSize = maxSize.String
	f.Delay = int(delay.Int32)
	f.MatchReleases = matchReleases.String
	f.ExceptReleases = exceptReleases.String
	f.MatchReleaseGroups = matchReleaseGroups.String
	f.ExceptReleaseGroups = exceptReleaseGroups.String
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
	f.Origins = origins.String

	return &f, nil
}

// FindByIndexerIdentifier find active filters with active indexer only
func (r *FilterRepo) FindByIndexerIdentifier(indexer string) ([]domain.Filter, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"f.id",
			"f.enabled",
			"f.name",
			"f.min_size",
			"f.max_size",
			"f.delay",
			"f.priority",
			"f.match_releases",
			"f.except_releases",
			"f.use_regex",
			"f.match_release_groups",
			"f.except_release_groups",
			"f.scene",
			"f.freeleech",
			"f.freeleech_percent",
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
			"f.created_at",
			"f.updated_at",
		).
		From("filter f").
		Join("filter_indexer fi ON f.id = fi.filter_id").
		Join("indexer i ON i.id = fi.indexer_id").
		Where("i.identifier = ?", indexer).
		Where("i.enabled = ?", true).
		Where("f.enabled = ?", true).
		OrderBy("f.priority DESC")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.findByIndexerIdentifier: error building query")
		return nil, err
	}

	rows, err := r.db.handler.Query(query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.findByIndexerIdentifier: error executing query")
		return nil, err
	}

	defer rows.Close()

	var filters []domain.Filter
	for rows.Next() {
		var f domain.Filter

		var minSize, maxSize, matchReleases, exceptReleases, matchReleaseGroups, exceptReleaseGroups, freeleechPercent, shows, seasons, episodes, years, artists, albums, matchCategories, exceptCategories, matchUploaders, exceptUploaders, tags, exceptTags, origins sql.NullString
		var useRegex, scene, freeleech, hasLog, hasCue, perfectFlac sql.NullBool
		var delay, logScore sql.NullInt32

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, &minSize, &maxSize, &delay, &f.Priority, &matchReleases, &exceptReleases, &useRegex, &matchReleaseGroups, &exceptReleaseGroups, &scene, &freeleech, &freeleechPercent, &shows, &seasons, &episodes, pq.Array(&f.Resolutions), pq.Array(&f.Codecs), pq.Array(&f.Sources), pq.Array(&f.Containers), pq.Array(&f.MatchHDR), pq.Array(&f.ExceptHDR), pq.Array(&f.MatchOther), pq.Array(&f.ExceptOther), &years, &artists, &albums, pq.Array(&f.MatchReleaseTypes), pq.Array(&f.Formats), pq.Array(&f.Quality), pq.Array(&f.Media), &logScore, &hasLog, &hasCue, &perfectFlac, &matchCategories, &exceptCategories, &matchUploaders, &exceptUploaders, &tags, &exceptTags, &origins, &f.CreatedAt, &f.UpdatedAt); err != nil {
			log.Error().Stack().Err(err).Msg("filter.findByIndexerIdentifier: error scanning row")
			return nil, err
		}

		f.MinSize = minSize.String
		f.MaxSize = maxSize.String
		f.Delay = int(delay.Int32)
		f.MatchReleases = matchReleases.String
		f.ExceptReleases = exceptReleases.String
		f.MatchReleaseGroups = matchReleaseGroups.String
		f.ExceptReleaseGroups = exceptReleaseGroups.String
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
		f.Origins = origins.String

		filters = append(filters, f)
	}

	return filters, nil
}

func (r *FilterRepo) Store(ctx context.Context, filter domain.Filter) (*domain.Filter, error) {
	queryBuilder := r.db.squirrel.
		Insert("filter").
		Columns(
			"name",
			"enabled",
			"min_size",
			"max_size",
			"delay",
			"priority",
			"match_releases",
			"except_releases",
			"use_regex",
			"match_release_groups",
			"except_release_groups",
			"scene",
			"freeleech",
			"freeleech_percent",
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
		).
		Values(
			filter.Name,
			filter.Enabled,
			filter.MinSize,
			filter.MaxSize,
			filter.Delay,
			filter.Priority,
			filter.MatchReleases,
			filter.ExceptReleases,
			filter.UseRegex,
			filter.MatchReleaseGroups,
			filter.ExceptReleaseGroups,
			filter.Scene,
			filter.Freeleech,
			filter.FreeleechPercent,
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
			filter.Origins,
		).
		Suffix("RETURNING id").RunWith(r.db.handler)

	// return values
	var retID int

	err := queryBuilder.QueryRowContext(ctx).Scan(&retID)
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.store: error executing query")
		return nil, err
	}

	filter.ID = retID

	return &filter, nil
}

func (r *FilterRepo) Update(ctx context.Context, filter domain.Filter) (*domain.Filter, error) {
	var err error

	queryBuilder := r.db.squirrel.
		Update("filter").
		Set("name", filter.Name).
		Set("enabled", filter.Enabled).
		Set("min_size", filter.MinSize).
		Set("max_size", filter.MaxSize).
		Set("delay", filter.Delay).
		Set("priority", filter.Priority).
		Set("use_regex", filter.UseRegex).
		Set("match_releases", filter.MatchReleases).
		Set("except_releases", filter.ExceptReleases).
		Set("match_release_groups", filter.MatchReleaseGroups).
		Set("except_release_groups", filter.ExceptReleaseGroups).
		Set("scene", filter.Scene).
		Set("freeleech", filter.Freeleech).
		Set("freeleech_percent", filter.FreeleechPercent).
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
		Set("origins", filter.Origins).
		Set("updated_at", time.Now().Format(time.RFC3339)).
		Where("id = ?", filter.ID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.update: error building query")
		return nil, err
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.update: error executing query")
		return nil, err
	}

	return &filter, nil
}

func (r *FilterRepo) ToggleEnabled(ctx context.Context, filterID int, enabled bool) error {
	var err error

	queryBuilder := r.db.squirrel.
		Update("filter").
		Set("enabled", enabled).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where("id = ?", filterID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.toggleEnabled: error building query")
		return err
	}
	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.toggleEnabled: error executing query")
		return err
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
		Where("filter_id = ?", filterID)

	deleteQuery, deleteArgs, err := deleteQueryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.StoreIndexerConnections: error building query")
		return err
	}
	_, err = tx.ExecContext(ctx, deleteQuery, deleteArgs...)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("filter.StoreIndexerConnections: error deleting indexers for filter: %v", filterID)
		return err
	}

	for _, indexer := range indexers {
		queryBuilder := r.db.squirrel.
			Insert("filter_indexer").Columns("filter_id", "indexer_id").
			Values(filterID, indexer.ID)

		query, args, err := queryBuilder.ToSql()
		if err != nil {
			log.Error().Stack().Err(err).Msg("filter.StoreIndexerConnections: error building query")
			return err
		}
		_, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			log.Error().Stack().Err(err).Msg("filter.StoreIndexerConnections: error executing query")
			return err
		}

		log.Debug().Msgf("filter.StoreIndexerConnections: store '%v' on filter: %v", indexer.Name, filterID)
	}

	err = tx.Commit()
	if err != nil {
		log.Error().Stack().Err(err).Msgf("filter.StoreIndexerConnections: error storing indexers for filter: %v", filterID)
		return err
	}

	return nil
}

func (r *FilterRepo) StoreIndexerConnection(ctx context.Context, filterID int, indexerID int) error {
	queryBuilder := r.db.squirrel.
		Insert("filter_indexer").Columns("filter_id", "indexer_id").
		Values(filterID, indexerID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.storeIndexerConnection: error building query")
		return err
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.storeIndexerConnection: error executing query")
		return err
	}

	return nil
}

func (r *FilterRepo) DeleteIndexerConnections(ctx context.Context, filterID int) error {
	queryBuilder := r.db.squirrel.
		Delete("filter_indexer").
		Where("filter_id = ?", filterID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.deleteIndexerConnections: error building query")
		return err
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.deleteIndexerConnections: error executing query")
		return err
	}

	return nil
}

func (r *FilterRepo) Delete(ctx context.Context, filterID int) error {
	queryBuilder := r.db.squirrel.
		Delete("filter").
		Where("id = ?", filterID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.delete: error building query")
		return err
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("filter.delete: error executing query")
		return err
	}

	log.Info().Msgf("filter.delete: successfully deleted: %v", filterID)

	return nil
}

// Split string to slice. We store comma separated strings and convert to slice
//func stringToSlice(str string) []string {
//	if str == "" {
//		return []string{}
//	} else if !strings.Contains(str, ",") {
//		return []string{str}
//	}
//
//	split := strings.Split(str, ",")
//
//	return split
//}
