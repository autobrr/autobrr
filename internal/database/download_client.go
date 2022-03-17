package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog/log"
)

type DownloadClientRepo struct {
	db    *SqliteDB
	cache *clientCache
}

type clientCache struct {
	mu      sync.RWMutex
	clients map[int]*domain.DownloadClient
}

func NewClientCache() *clientCache {
	return &clientCache{
		clients: make(map[int]*domain.DownloadClient, 0),
	}
}

func (c *clientCache) Set(id int, client *domain.DownloadClient) {
	c.mu.Lock()
	c.clients[id] = client
	c.mu.Unlock()
}

func (c *clientCache) Get(id int) *domain.DownloadClient {
	c.mu.RLock()
	defer c.mu.Unlock()
	v, ok := c.clients[id]
	if ok {
		return v
	}
	return nil
}

func (c *clientCache) Pop(id int) {
	c.mu.Lock()
	delete(c.clients, id)
	c.mu.Unlock()
}

func NewDownloadClientRepo(db *SqliteDB) domain.DownloadClientRepo {
	return &DownloadClientRepo{
		db:    db,
		cache: NewClientCache(),
	}
}

func (r *DownloadClientRepo) List(ctx context.Context) ([]domain.DownloadClient, error) {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()
	clients := make([]domain.DownloadClient, 0)

	rows, err := r.db.handler.QueryContext(ctx, "SELECT id, name, type, enabled, host, port, tls, tls_skip_verify, username, password, settings FROM client")
	if err != nil {
		log.Error().Stack().Err(err).Msg("could not query download client rows")
		return clients, err
	}

	defer rows.Close()

	for rows.Next() {
		var f domain.DownloadClient
		var settingsJsonStr string

		if err := rows.Scan(&f.ID, &f.Name, &f.Type, &f.Enabled, &f.Host, &f.Port, &f.TLS, &f.TLSSkipVerify, &f.Username, &f.Password, &settingsJsonStr); err != nil {
			log.Error().Stack().Err(err).Msg("could not scan download client to struct")
			return clients, err
		}

		if settingsJsonStr != "" {
			if err := json.Unmarshal([]byte(settingsJsonStr), &f.Settings); err != nil {
				log.Error().Stack().Err(err).Msgf("could not marshal download client settings %v", settingsJsonStr)
				return clients, err
			}
		}

		clients = append(clients, f)
	}
	if err := rows.Err(); err != nil {
		log.Error().Stack().Err(err).Msg("could not query download client rows")
		return clients, err
	}

	return clients, nil
}

func (r *DownloadClientRepo) FindByID(ctx context.Context, id int32) (*domain.DownloadClient, error) {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()

	// get client from cache
	c := r.cache.Get(int(id))
	if c != nil {
		return c, nil
	}

	query := `SELECT id, name, type, enabled, host, port, tls, tls_skip_verify, username, password, settings FROM client WHERE id = ?`

	row := r.db.handler.QueryRowContext(ctx, query, id)
	if err := row.Err(); err != nil {
		log.Error().Stack().Err(err).Msg("could not query download client rows")
		return nil, err
	}

	var client domain.DownloadClient
	var settingsJsonStr string

	if err := row.Scan(&client.ID, &client.Name, &client.Type, &client.Enabled, &client.Host, &client.Port, &client.TLS, &client.TLSSkipVerify, &client.Username, &client.Password, &settingsJsonStr); err != nil {
		log.Error().Stack().Err(err).Msg("could not scan download client to struct")
		return nil, err
	}

	if settingsJsonStr != "" {
		if err := json.Unmarshal([]byte(settingsJsonStr), &client.Settings); err != nil {
			log.Error().Stack().Err(err).Msgf("could not marshal download client settings %v", settingsJsonStr)
			return nil, err
		}
	}

	return &client, nil
}

func (r *DownloadClientRepo) Store(ctx context.Context, client domain.DownloadClient) (*domain.DownloadClient, error) {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()

	var err error

	settings := domain.DownloadClientSettings{
		APIKey: client.Settings.APIKey,
		Basic:  client.Settings.Basic,
		Rules:  client.Settings.Rules,
	}

	settingsJson, err := json.Marshal(&settings)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not marshal download client settings %v", settings)
		return nil, err
	}

	if client.ID != 0 {
		_, err = r.db.handler.ExecContext(ctx, `
			UPDATE 
    			client 
			SET 
			    name = ?, 
			    type = ?, 
			    enabled = ?, 
			    host = ?, 
			    port = ?, 
			    tls = ?, 
			    tls_skip_verify = ?,
			    username = ?, 
			    password = ?, 
			    settings = (?) 
			WHERE
			    id = ?`,
			client.Name,
			client.Type,
			client.Enabled,
			client.Host,
			client.Port,
			client.TLS,
			client.TLSSkipVerify,
			client.Username,
			client.Password,
			string(settingsJson),
			client.ID,
		)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("could not update download client: %v", client)
			return nil, err
		}
	} else {
		var res sql.Result

		res, err = r.db.handler.ExecContext(ctx, `INSERT INTO 
    		client(
    		       name,
    		       type, 
    		       enabled,
    		       host,
    		       port,
    		       tls,
    		       tls_skip_verify,
    		       username,
    		       password,
    		       settings)
			VALUES (?, ?, ?, ?, ?, ? , ?, ?, ?, ?) ON CONFLICT DO NOTHING`,
			client.Name,
			client.Type,
			client.Enabled,
			client.Host,
			client.Port,
			client.TLS,
			client.TLSSkipVerify,
			client.Username,
			client.Password,
			string(settingsJson),
		)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("could not store new download client: %v", client)
			return nil, err
		}

		resId, _ := res.LastInsertId()
		client.ID = int(resId)

		log.Trace().Msgf("download_client: store new record %d", client.ID)
	}

	log.Info().Msgf("store download client: %v", client.Name)
	log.Trace().Msgf("store download client: %+v", client)

	// save to cache
	r.cache.Set(client.ID, &client)

	return &client, nil
}

func (r *DownloadClientRepo) Delete(ctx context.Context, clientID int) error {
	//r.db.lock.RLock()
	//defer r.db.lock.RUnlock()

	res, err := r.db.handler.ExecContext(ctx, `DELETE FROM client WHERE client.id = ?`, clientID)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not delete download client: %d", clientID)
		return err
	}

	// remove from cache
	r.cache.Pop(clientID)

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return err
	}

	log.Info().Msgf("delete download client: %d", clientID)

	return nil
}
