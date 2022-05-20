package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
)

type IndexerRepo struct {
	log logger.Logger
	db  *DB
}

func NewIndexerRepo(log logger.Logger, db *DB) domain.IndexerRepo {
	return &IndexerRepo{
		log: log,
		db:  db,
	}
}

func (r *IndexerRepo) Store(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error) {
	settings, err := json.Marshal(indexer.Settings)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("error marshaling json data")
		return nil, err
	}

	queryBuilder := r.db.squirrel.
		Insert("indexer").Columns("enabled", "name", "identifier", "implementation", "settings").
		Values(indexer.Enabled, indexer.Name, indexer.Identifier, indexer.Implementation, settings).
		Suffix("RETURNING id").RunWith(r.db.handler)

	// return values
	var retID int64

	err = queryBuilder.QueryRowContext(ctx).Scan(&retID)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("indexer.store: error executing query")
		return nil, err
	}

	indexer.ID = retID

	return &indexer, nil
}

func (r *IndexerRepo) Update(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error) {
	settings, err := json.Marshal(indexer.Settings)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("error marshaling json data")
		return nil, err
	}

	queryBuilder := r.db.squirrel.
		Update("indexer").
		Set("enabled", indexer.Enabled).
		Set("name", indexer.Name).
		Set("settings", settings).
		Set("updated_at", time.Now().Format(time.RFC3339)).
		Where("id = ?", indexer.ID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("indexer.update: error building query")
		return nil, err
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("indexer.update: error executing query")
		return nil, err
	}

	return &indexer, nil
}

func (r *IndexerRepo) List(ctx context.Context) ([]domain.Indexer, error) {
	rows, err := r.db.handler.QueryContext(ctx, "SELECT id, enabled, name, identifier, implementation, settings FROM indexer ORDER BY name ASC")
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("indexer.list: error query indexer")
		return nil, err
	}

	defer rows.Close()

	var indexers []domain.Indexer
	for rows.Next() {
		var f domain.Indexer

		var implementation sql.NullString
		var settings string
		var settingsMap map[string]string

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, &f.Identifier, &implementation, &settings); err != nil {
			r.log.Error().Stack().Err(err).Msg("indexer.list: error scanning data to struct")
			return nil, err
		}

		f.Implementation = implementation.String

		err = json.Unmarshal([]byte(settings), &settingsMap)
		if err != nil {
			r.log.Error().Stack().Err(err).Msg("indexer.list: error unmarshal settings")
			return nil, err
		}

		f.Settings = settingsMap

		indexers = append(indexers, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return indexers, nil
}

func (r *IndexerRepo) FindByFilterID(ctx context.Context, id int) ([]domain.Indexer, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "identifier", "settings").
		From("indexer").
		Join("filter_indexer ON indexer.id = filter_indexer.indexer_id").
		Where("filter_indexer.filter_id = ?", id)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.check_existing_network: error fetching data")
		return nil, err
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("indexer.find_by_filter_id: error query indexer")
		return nil, err
	}

	defer rows.Close()

	indexers := make([]domain.Indexer, 0)
	for rows.Next() {
		var f domain.Indexer

		var settings string
		var settingsMap map[string]string

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, &f.Identifier, &settings); err != nil {
			r.log.Error().Stack().Err(err).Msg("indexer.find_by_filter_id: error scanning data to struct")
			return nil, err
		}

		err = json.Unmarshal([]byte(settings), &settingsMap)
		if err != nil {
			r.log.Error().Stack().Err(err).Msg("indexer.find_by_filter_id: error unmarshal settings")
			return nil, err
		}

		f.Settings = settingsMap

		indexers = append(indexers, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return indexers, nil

}

func (r *IndexerRepo) Delete(ctx context.Context, id int) error {
	queryBuilder := r.db.squirrel.
		Delete("indexer").
		Where("id = ?", id)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("indexer.delete: error building query")
		return err
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		r.log.Error().Stack().Err(err).Msgf("indexer.delete: error executing query: '%v'", query)
		return err
	}

	r.log.Debug().Msgf("indexer.delete: id %v", id)

	return nil
}
