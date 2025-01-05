// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog"
)

type DownloadClientRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewDownloadClientRepo(log logger.Logger, db *DB) domain.DownloadClientRepo {
	return &DownloadClientRepo{
		log: log.With().Str("repo", "action").Logger(),
		db:  db,
	}
}

func (r *DownloadClientRepo) List(ctx context.Context) ([]domain.DownloadClient, error) {
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

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			r.log.Error().Err(err).Msg("error closing rows")
		}
	}(rows)

	clients := make([]domain.DownloadClient, 0)

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
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var client domain.DownloadClient
	var settingsJsonStr string

	if err := row.Scan(&client.ID, &client.Name, &client.Type, &client.Enabled, &client.Host, &client.Port, &client.TLS, &client.TLSSkipVerify, &client.Username, &client.Password, &settingsJsonStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	if settingsJsonStr != "" {
		if err := json.Unmarshal([]byte(settingsJsonStr), &client.Settings); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal download client settings: %v", settingsJsonStr)
		}
	}

	return &client, nil
}

func (r *DownloadClientRepo) Store(ctx context.Context, client *domain.DownloadClient) error {
	settings := domain.DownloadClientSettings{
		APIKey:                   client.Settings.APIKey,
		Basic:                    client.Settings.Basic,
		Rules:                    client.Settings.Rules,
		ExternalDownloadClientId: client.Settings.ExternalDownloadClientId,
		ExternalDownloadClient:   client.Settings.ExternalDownloadClient,
		Auth:                     client.Settings.Auth,
	}

	settingsJson, err := json.Marshal(&settings)
	if err != nil {
		return errors.Wrap(err, "error marshal download client settings %+v", settings)
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
		return errors.Wrap(err, "error executing query")
	}

	client.ID = int32(retID)

	r.log.Debug().Msgf("download_client.store: %d", client.ID)

	return nil
}

func (r *DownloadClientRepo) Update(ctx context.Context, client *domain.DownloadClient) error {
	settings := domain.DownloadClientSettings{
		APIKey:                   client.Settings.APIKey,
		Basic:                    client.Settings.Basic,
		Rules:                    client.Settings.Rules,
		ExternalDownloadClientId: client.Settings.ExternalDownloadClientId,
		ExternalDownloadClient:   client.Settings.ExternalDownloadClient,
		Auth:                     client.Settings.Auth,
	}

	settingsJson, err := json.Marshal(&settings)
	if err != nil {
		return errors.Wrap(err, "error marshal download client settings %+v", settings)
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
		Where(sq.Eq{"id": client.ID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error getting rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("no rows updated")
	}

	r.log.Debug().Msgf("download_client.update: %d", client.ID)

	return nil
}

func (r *DownloadClientRepo) Delete(ctx context.Context, clientID int32) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		var txErr error
		if p := recover(); p != nil {
			txErr = tx.Rollback()
			if txErr != nil {
				r.log.Error().Err(txErr).Msg("error rolling back transaction")
			}
			r.log.Error().Msgf("something went terribly wrong panic: %v", p)
		} else if err != nil {
			txErr = tx.Rollback()
			if txErr != nil {
				r.log.Error().Err(txErr).Msg("error rolling back transaction")
			}
		} else {
			// All good, commit
			txErr = tx.Commit()
			if txErr != nil {
				r.log.Error().Err(txErr).Msg("error committing transaction")
			}
		}
	}()

	if err = r.delete(ctx, tx, clientID); err != nil {
		return errors.Wrap(err, "error deleting download client: %d", clientID)
	}

	if err = r.deleteClientFromAction(ctx, tx, clientID); err != nil {
		return errors.Wrap(err, "error deleting download client: %d", clientID)
	}

	r.log.Debug().Msgf("delete download client: %d", clientID)

	return nil
}

func (r *DownloadClientRepo) delete(ctx context.Context, tx *Tx, clientID int32) error {
	queryBuilder := r.db.squirrel.
		Delete("client").
		Where(sq.Eq{"id": clientID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("no rows affected")
	}

	r.log.Debug().Msgf("delete download client: %d", clientID)

	return nil
}

func (r *DownloadClientRepo) deleteClientFromAction(ctx context.Context, tx *Tx, clientID int32) error {
	queryBuilder := r.db.squirrel.
		Update("action").
		Set("enabled", false).
		Set("client_id", 0).
		Where(sq.Eq{"client_id": clientID}).
		Suffix("RETURNING filter_id").RunWith(tx)

	// return values
	var filterID int

	err := queryBuilder.QueryRowContext(ctx).Scan(&filterID)
	if err != nil {
		// this will throw when the client is not connected to any actions
		// it is not an error in this case
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("deleting download client %d from action for filter %d", clientID, filterID)

	return nil
}
