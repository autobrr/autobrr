package database

import (
	"context"
	"encoding/json"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/rs/zerolog/log"
)

type IndexerRepo struct {
	db *SqliteDB
}

func NewIndexerRepo(db *SqliteDB) domain.IndexerRepo {
	return &IndexerRepo{
		db: db,
	}
}

func (r *IndexerRepo) Store(indexer domain.Indexer) (*domain.Indexer, error) {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()

	settings, err := json.Marshal(indexer.Settings)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error marshaling json data")
		return nil, err
	}

	res, err := r.db.handler.Exec(`INSERT INTO indexer (enabled, name, identifier, settings) VALUES (?, ?, ?, ?)`, indexer.Enabled, indexer.Name, indexer.Identifier, settings)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error executing query")
		return nil, err
	}

	id, _ := res.LastInsertId()
	indexer.ID = id

	return &indexer, nil
}

func (r *IndexerRepo) Update(indexer domain.Indexer) (*domain.Indexer, error) {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()

	sett, err := json.Marshal(indexer.Settings)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error marshaling json data")
		return nil, err
	}

	_, err = r.db.handler.Exec(`UPDATE indexer SET enabled = ?, name = ?, settings = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, indexer.Enabled, indexer.Name, sett, indexer.ID)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error executing query")
		return nil, err
	}

	return &indexer, nil
}

func (r *IndexerRepo) List() ([]domain.Indexer, error) {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()

	rows, err := r.db.handler.Query("SELECT id, enabled, name, identifier, settings FROM indexer ORDER BY name ASC")
	if err != nil {
		log.Fatal().Err(err)
	}

	defer rows.Close()

	var indexers []domain.Indexer
	for rows.Next() {
		var f domain.Indexer

		var settings string
		var settingsMap map[string]string

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, &f.Identifier, &settings); err != nil {
			log.Error().Stack().Err(err).Msg("indexer.list: error scanning data to struct")
		}
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(settings), &settingsMap)
		if err != nil {
			log.Error().Stack().Err(err).Msg("indexer.list: error unmarshal settings")
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

func (r *IndexerRepo) FindByFilterID(id int) ([]domain.Indexer, error) {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()

	rows, err := r.db.handler.Query(`
		SELECT i.id, i.enabled, i.name, i.identifier
		FROM indexer i
			JOIN filter_indexer fi on i.id = fi.indexer_id
		WHERE fi.filter_id = ?`, id)
	if err != nil {
		log.Fatal().Err(err)
	}

	defer rows.Close()

	var indexers []domain.Indexer
	for rows.Next() {
		var f domain.Indexer

		//var settings string
		//var settingsMap map[string]string

		if err := rows.Scan(&f.ID, &f.Enabled, &f.Name, &f.Identifier); err != nil {
			log.Error().Stack().Err(err).Msg("indexer.list: error scanning data to struct")
		}
		if err != nil {
			return nil, err
		}

		//err = json.Unmarshal([]byte(settings), &settingsMap)
		//if err != nil {
		//	log.Error().Stack().Err(err).Msg("indexer.list: error unmarshal settings")
		//	return nil, err
		//}
		//
		//f.Settings = settingsMap

		indexers = append(indexers, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return indexers, nil

}

func (r *IndexerRepo) Delete(ctx context.Context, id int) error {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()

	query := `DELETE FROM indexer WHERE id = ?`

	_, err := r.db.handler.ExecContext(ctx, query, id)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("indexer.delete: error executing query: '%v'", query)
		return err
	}

	log.Debug().Msgf("indexer.delete: id %v", id)

	return nil
}
