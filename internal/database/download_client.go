package database

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type DownloadClientRepo struct {
	log   zerolog.Logger
	db    *DB
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
	defer c.mu.RUnlock()
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

func NewDownloadClientRepo(log logger.Logger, db *DB) domain.DownloadClientRepo {
	return &DownloadClientRepo{
		log:   log.With().Str("repo", "action").Logger(),
		db:    db,
		cache: NewClientCache(),
	}
}

func (r *DownloadClientRepo) List(ctx context.Context) ([]domain.DownloadClient, error) {
	clients := make([]domain.DownloadClient, 0)

	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"name",
			"type",
			"enabled",
			"host",
			"port",
			"tls",
			"tls_skip_verify",
			"username",
			"password",
			"settings",
		).
		From("client")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	for rows.Next() {
		var f domain.DownloadClient
		var settingsJsonStr string

		if err := rows.Scan(&f.ID, &f.Name, &f.Type, &f.Enabled, &f.Host, &f.Port, &f.TLS, &f.TLSSkipVerify, &f.Username, &f.Password, &settingsJsonStr); err != nil {
			return clients, errors.Wrap(err, "error scanning row")
		}

		if settingsJsonStr != "" {
			if err := json.Unmarshal([]byte(settingsJsonStr), &f.Settings); err != nil {
				return clients, errors.Wrap(err, "could not unmarshal download client settings: %v", settingsJsonStr)
			}
		}

		clients = append(clients, f)
	}
	if err := rows.Err(); err != nil {
		return clients, errors.Wrap(err, "rows error")
	}

	return clients, nil
}

func (r *DownloadClientRepo) FindByID(ctx context.Context, id int32) (*domain.DownloadClient, error) {
	// get client from cache
	c := r.cache.Get(int(id))
	if c != nil {
		return c, nil
	}

	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"name",
			"type",
			"enabled",
			"host",
			"port",
			"tls",
			"tls_skip_verify",
			"username",
			"password",
			"settings",
		).
		From("client").
		Where("id = ?", id)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var client domain.DownloadClient
	var settingsJsonStr string

	if err := row.Scan(&client.ID, &client.Name, &client.Type, &client.Enabled, &client.Host, &client.Port, &client.TLS, &client.TLSSkipVerify, &client.Username, &client.Password, &settingsJsonStr); err != nil {
		return nil, errors.Wrap(err, "error scanning row")
	}

	if settingsJsonStr != "" {
		if err := json.Unmarshal([]byte(settingsJsonStr), &client.Settings); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal download client settings: %v", settingsJsonStr)
		}
	}

	return &client, nil
}

func (r *DownloadClientRepo) Store(ctx context.Context, client domain.DownloadClient) (*domain.DownloadClient, error) {
	var err error

	settings := domain.DownloadClientSettings{
		APIKey: client.Settings.APIKey,
		Basic:  client.Settings.Basic,
		Rules:  client.Settings.Rules,
	}

	settingsJson, err := json.Marshal(&settings)
	if err != nil {
		return nil, errors.Wrap(err, "error marshal download client settings %+v", settings)
	}

	queryBuilder := r.db.squirrel.
		Insert("client").
		Columns("name", "type", "enabled", "host", "port", "tls", "tls_skip_verify", "username", "password", "settings").
		Values(client.Name, client.Type, client.Enabled, client.Host, client.Port, client.TLS, client.TLSSkipVerify, client.Username, client.Password, settingsJson).
		Suffix("RETURNING id").RunWith(r.db.handler)

	// return values
	var retID int

	err = queryBuilder.QueryRowContext(ctx).Scan(&retID)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	client.ID = retID

	r.log.Debug().Msgf("download_client.store: %d", client.ID)

	// save to cache
	r.cache.Set(client.ID, &client)

	return &client, nil
}

func (r *DownloadClientRepo) Update(ctx context.Context, client domain.DownloadClient) (*domain.DownloadClient, error) {
	var err error

	settings := domain.DownloadClientSettings{
		APIKey: client.Settings.APIKey,
		Basic:  client.Settings.Basic,
		Rules:  client.Settings.Rules,
	}

	settingsJson, err := json.Marshal(&settings)
	if err != nil {
		return nil, errors.Wrap(err, "error marshal download client settings %+v", settings)
	}

	queryBuilder := r.db.squirrel.
		Update("client").
		Set("name", client.Name).
		Set("type", client.Type).
		Set("enabled", client.Enabled).
		Set("host", client.Host).
		Set("port", client.Port).
		Set("tls", client.TLS).
		Set("tls_skip_verify", client.TLSSkipVerify).
		Set("username", client.Username).
		Set("password", client.Password).
		Set("settings", string(settingsJson)).
		Where("id = ?", client.ID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("download_client.update: %d", client.ID)

	// save to cache
	r.cache.Set(client.ID, &client)

	return &client, nil
}

func (r *DownloadClientRepo) Delete(ctx context.Context, clientID int) error {
	queryBuilder := r.db.squirrel.
		Delete("client").
		Where("id = ?", clientID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	res, err := r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	// remove from cache
	r.cache.Pop(clientID)

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("no rows affected")
	}

	r.log.Info().Msgf("delete download client: %d", clientID)

	return nil
}
