package database

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/domain"
)

type FilterRepo struct {
	db *sql.DB
}

func NewFilterRepo(db *sql.DB) domain.FilterRepo {
	return &FilterRepo{db: db}
}

func (r *FilterRepo) ListFilters() ([]domain.Filter, error) {

	rows, err := r.db.Query("SELECT id, enabled, name, match_releases, except_releases, created_at, updated_at FROM filter")
	if err != nil {
		log.Fatal().Err(err)
	}

	defer rows.Close()

	var filters []domain.Filter
	for rows.Next() {
		var f domain.Filter

		var matchReleases, exceptReleases sql.NullString
		var createdAt, updatedAt string

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, &matchReleases, &exceptReleases, &createdAt, &updatedAt); err != nil {
			log.Error().Stack().Err(err).Msg("filters_list: error scanning data to struct")
		}
		if err != nil {
			return nil, err
		}

		f.MatchReleases = matchReleases.String
		f.ExceptReleases = exceptReleases.String

		ua, _ := time.Parse(time.RFC3339, updatedAt)
		ca, _ := time.Parse(time.RFC3339, createdAt)

		f.UpdatedAt = ua
		f.CreatedAt = ca

		filters = append(filters, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return filters, nil
}

func (r *FilterRepo) FindByID(filterID int) (*domain.Filter, error) {

	row := r.db.QueryRow("SELECT id, enabled, name, min_size, max_size, delay, match_releases, except_releases, use_regex, match_release_groups, except_release_groups, scene, freeleech, freeleech_percent, shows, seasons, episodes, resolutions, codecs, sources, containers, years, match_categories, except_categories, match_uploaders, except_uploaders, tags, except_tags, created_at, updated_at FROM filter WHERE id = ?", filterID)

	var f domain.Filter

	if err := row.Err(); err != nil {
		return nil, err
	}

	var minSize, maxSize, matchReleases, exceptReleases, matchReleaseGroups, exceptReleaseGroups, freeleechPercent, shows, seasons, episodes, years, matchCategories, exceptCategories, matchUploaders, exceptUploaders, tags, exceptTags sql.NullString
	var useRegex, scene, freeleech sql.NullBool
	var delay sql.NullInt32
	var createdAt, updatedAt string

	if err := row.Scan(&f.ID, &f.Enabled, &f.Name, &minSize, &maxSize, &delay, &matchReleases, &exceptReleases, &useRegex, &matchReleaseGroups, &exceptReleaseGroups, &scene, &freeleech, &freeleechPercent, &shows, &seasons, &episodes, pq.Array(&f.Resolutions), pq.Array(&f.Codecs), pq.Array(&f.Sources), pq.Array(&f.Containers), &years, &matchCategories, &exceptCategories, &matchUploaders, &exceptUploaders, &tags, &exceptTags, &createdAt, &updatedAt); err != nil {
		log.Error().Stack().Err(err).Msgf("filter: %v : error scanning data to struct", filterID)
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
	f.MatchCategories = matchCategories.String
	f.ExceptCategories = exceptCategories.String
	f.MatchUploaders = matchUploaders.String
	f.ExceptUploaders = exceptUploaders.String
	f.Tags = tags.String
	f.ExceptTags = exceptTags.String
	f.UseRegex = useRegex.Bool
	f.Scene = scene.Bool
	f.Freeleech = freeleech.Bool

	updatedTime, _ := time.Parse(time.RFC3339, updatedAt)
	createdTime, _ := time.Parse(time.RFC3339, createdAt)

	f.UpdatedAt = updatedTime
	f.CreatedAt = createdTime

	return &f, nil
}

// TODO remove
func (r *FilterRepo) FindFiltersForSite(site string) ([]domain.Filter, error) {

	rows, err := r.db.Query("SELECT id, enabled, name, match_releases, except_releases, created_at, updated_at FROM filter WHERE match_sites LIKE ?", site)
	if err != nil {
		log.Fatal().Err(err)
	}

	defer rows.Close()

	var filters []domain.Filter
	for rows.Next() {
		var f domain.Filter

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, pq.Array(&f.MatchReleases), pq.Array(&f.ExceptReleases), &f.CreatedAt, &f.UpdatedAt); err != nil {
			log.Error().Stack().Err(err).Msg("error scanning data to struct")
		}
		if err != nil {
			return nil, err
		}

		filters = append(filters, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return filters, nil
}

// FindByIndexerIdentifier find active filters only
func (r *FilterRepo) FindByIndexerIdentifier(indexer string) ([]domain.Filter, error) {

	rows, err := r.db.Query(`
		SELECT 
		       f.id,
		       f.enabled,
		       f.name,
		       f.min_size,
		       f.max_size,
		       f.delay,
		       f.match_releases,
		       f.except_releases,
		       f.use_regex,
		       f.match_release_groups,
		       f.except_release_groups,
		       f.scene,
		       f.freeleech,
		       f.freeleech_percent,
		       f.shows,
		       f.seasons,
		       f.episodes,
		       f.resolutions,
		       f.codecs,
		       f.sources,
		       f.containers,
		       f.years,
		       f.match_categories,
		       f.except_categories,
		       f.match_uploaders,
		       f.except_uploaders,
		       f.tags,
		       f.except_tags,
		       f.created_at,
		       f.updated_at
		FROM filter f
				 JOIN filter_indexer fi on f.id = fi.filter_id
				 JOIN indexer i on i.id = fi.indexer_id
		WHERE i.identifier = ?
		AND f.enabled = true`, indexer)
	if err != nil {
		log.Fatal().Err(err)
	}

	defer rows.Close()

	var filters []domain.Filter
	for rows.Next() {
		var f domain.Filter

		var minSize, maxSize, matchReleases, exceptReleases, matchReleaseGroups, exceptReleaseGroups, freeleechPercent, shows, seasons, episodes, years, matchCategories, exceptCategories, matchUploaders, exceptUploaders, tags, exceptTags sql.NullString
		var useRegex, scene, freeleech sql.NullBool
		var delay sql.NullInt32
		var createdAt, updatedAt string

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, &minSize, &maxSize, &delay, &matchReleases, &exceptReleases, &useRegex, &matchReleaseGroups, &exceptReleaseGroups, &scene, &freeleech, &freeleechPercent, &shows, &seasons, &episodes, pq.Array(&f.Resolutions), pq.Array(&f.Codecs), pq.Array(&f.Sources), pq.Array(&f.Containers), &years, &matchCategories, &exceptCategories, &matchUploaders, &exceptUploaders, &tags, &exceptTags, &createdAt, &updatedAt); err != nil {
			log.Error().Stack().Err(err).Msg("error scanning data to struct")
		}
		if err != nil {
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
		f.MatchCategories = matchCategories.String
		f.ExceptCategories = exceptCategories.String
		f.MatchUploaders = matchUploaders.String
		f.ExceptUploaders = exceptUploaders.String
		f.Tags = tags.String
		f.ExceptTags = exceptTags.String
		f.UseRegex = useRegex.Bool
		f.Scene = scene.Bool
		f.Freeleech = freeleech.Bool

		updatedTime, _ := time.Parse(time.RFC3339, updatedAt)
		createdTime, _ := time.Parse(time.RFC3339, createdAt)

		f.UpdatedAt = updatedTime
		f.CreatedAt = createdTime

		filters = append(filters, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return filters, nil
}

func (r *FilterRepo) Store(filter domain.Filter) (*domain.Filter, error) {

	var err error
	if filter.ID != 0 {
		log.Debug().Msg("update existing record")
	} else {
		var res sql.Result

		res, err = r.db.Exec(`INSERT INTO filter (
                    name,
                    enabled,
                    min_size,
                    max_size,
                    delay,
                    match_releases,
                    except_releases,
                    use_regex,
                    match_release_groups,
                    except_release_groups,
                    scene,
                    freeleech,
                    freeleech_percent,
                    shows,
                    seasons,
                    episodes,
                    resolutions,
                    codecs,
                    sources,
                    containers,
                    years,
                    match_categories,
                    except_categories,
                    match_uploaders,
                    except_uploaders,
                    tags,
                    except_tags
                    )
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27) ON CONFLICT DO NOTHING`,
			filter.Name,
			filter.Enabled,
			filter.MinSize,
			filter.MaxSize,
			filter.Delay,
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
			filter.Years,
			filter.MatchCategories,
			filter.ExceptCategories,
			filter.MatchUploaders,
			filter.ExceptUploaders,
			filter.Tags,
			filter.ExceptTags,
		)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error executing query")
			return nil, err
		}

		resId, _ := res.LastInsertId()
		filter.ID = int(resId)
	}

	return &filter, nil
}

func (r *FilterRepo) Update(ctx context.Context, filter domain.Filter) (*domain.Filter, error) {

	//var res sql.Result

	var err error
	_, err = r.db.ExecContext(ctx, `
			UPDATE filter SET 
                    name = ?,
                    enabled = ?,
                    min_size = ?,
                    max_size = ?,
                    delay = ?,
                    match_releases = ?,
                    except_releases = ?,
                    use_regex = ?,
                    match_release_groups = ?,
                    except_release_groups = ?,
                    scene = ?,
                    freeleech = ?,
                    freeleech_percent = ?,
                    shows = ?,
                    seasons = ?,
                    episodes = ?,
                    resolutions = ?,
                    codecs = ?,
                    sources = ?,
                    containers = ?,
                    years = ?,
                    match_categories = ?,
                    except_categories = ?,
                    match_uploaders = ?,
                    except_uploaders = ?,
                    tags = ?,
                    except_tags = ?,
				    updated_at = CURRENT_TIMESTAMP
            WHERE id = ?`,
		filter.Name,
		filter.Enabled,
		filter.MinSize,
		filter.MaxSize,
		filter.Delay,
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
		filter.Years,
		filter.MatchCategories,
		filter.ExceptCategories,
		filter.MatchUploaders,
		filter.ExceptUploaders,
		filter.Tags,
		filter.ExceptTags,
		filter.ID,
	)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error executing query")
		return nil, err
	}

	return &filter, nil
}

func (r *FilterRepo) ToggleEnabled(ctx context.Context, filterID int, enabled bool) error {

	var err error
	_, err = r.db.ExecContext(ctx, `
			UPDATE filter SET 
                    enabled = ?,
				    updated_at = CURRENT_TIMESTAMP
            WHERE id = ?`,
		enabled,
		filterID,
	)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error executing query")
		return err
	}

	return nil
}

func (r *FilterRepo) StoreIndexerConnections(ctx context.Context, filterID int, indexers []domain.Indexer) error {

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	deleteQuery := `DELETE FROM filter_indexer WHERE filter_id = ?`
	_, err = tx.ExecContext(ctx, deleteQuery, filterID)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error deleting indexers for filter: %v", filterID)
		return err
	}

	for _, indexer := range indexers {
		query := `INSERT INTO filter_indexer (filter_id, indexer_id) VALUES ($1, $2)`
		_, err := tx.ExecContext(ctx, query, filterID, indexer.ID)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error executing query")
			return err
		}

		log.Debug().Msgf("filter.indexers: store '%v' on filter: %v", indexer.Name, filterID)
	}

	err = tx.Commit()
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error deleting indexers for filter: %v", filterID)
		return err
	}

	return nil
}

func (r *FilterRepo) StoreIndexerConnection(ctx context.Context, filterID int, indexerID int) error {
	query := `INSERT INTO filter_indexer (filter_id, indexer_id) VALUES ($1, $2)`
	_, err := r.db.ExecContext(ctx, query, filterID, indexerID)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error executing query")
		return err
	}

	return nil
}

func (r *FilterRepo) DeleteIndexerConnections(ctx context.Context, filterID int) error {

	query := `DELETE FROM filter_indexer WHERE filter_id = ?`
	_, err := r.db.ExecContext(ctx, query, filterID)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error executing query")
		return err
	}

	return nil
}

func (r *FilterRepo) Delete(ctx context.Context, filterID int) error {

	_, err := r.db.ExecContext(ctx, `DELETE FROM filter WHERE id = ?`, filterID)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error executing query")
		return err
	}

	log.Info().Msgf("filter.delete: successfully deleted: %v", filterID)

	return nil
}

// Split string to slice. We store comma separated strings and convert to slice
func stringToSlice(str string) []string {
	if str == "" {
		return []string{}
	} else if !strings.Contains(str, ",") {
		return []string{str}
	}

	split := strings.Split(str, ",")

	return split
}
