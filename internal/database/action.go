// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
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

type ActionRepo struct {
	log        zerolog.Logger
	db         *DB
	clientRepo domain.DownloadClientRepo
}

func NewActionRepo(log logger.Logger, db *DB, clientRepo domain.DownloadClientRepo) domain.ActionRepo {
	return &ActionRepo{
		log:        log.With().Str("repo", "action").Logger(),
		db:         db,
		clientRepo: clientRepo,
	}
}

func (r *ActionRepo) FindByFilterID(ctx context.Context, filterID int) ([]*domain.Action, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	actions, err := r.findByFilterID(ctx, tx, filterID)
	if err != nil {
		return nil, err
	}

	for _, action := range actions {
		if action.ClientID > 0 {
			client, err := r.attachDownloadClient(ctx, tx, action.ClientID)
			if err != nil {
				return nil, err
			}

			if client != nil {
				action.Client = client
			}
		}
	}

	return actions, nil
}

func (r *ActionRepo) findByFilterID(ctx context.Context, tx *Tx, filterID int) ([]*domain.Action, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"name",
			"type",
			"enabled",
			"exec_cmd",
			"exec_args",
			"watch_folder",
			"category",
			"tags",
			"label",
			"save_path",
			"paused",
			"ignore_rules",
			"skip_hash_check",
			"content_layout",
			"limit_download_speed",
			"limit_upload_speed",
			"limit_ratio",
			"limit_seed_time",
			"reannounce_skip",
			"reannounce_delete",
			"reannounce_interval",
			"reannounce_max_attempts",
			"webhook_host",
			"webhook_type",
			"webhook_method",
			"webhook_data",
			"external_client_id",
			"client_id",
		).
		From("action").
		Where(sq.Eq{"filter_id": filterID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	actions := make([]*domain.Action, 0)
	for rows.Next() {
		var a domain.Action

		var execCmd, execArgs, watchFolder, category, tags, label, savePath, contentLayout, webhookHost, webhookType, webhookMethod, webhookData sql.NullString
		var limitUl, limitDl, limitSeedTime sql.NullInt64
		var limitRatio sql.NullFloat64

		var externalClientID, clientID sql.NullInt32
		var paused, ignoreRules sql.NullBool

		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Enabled, &execCmd, &execArgs, &watchFolder, &category, &tags, &label, &savePath, &paused, &ignoreRules, &a.SkipHashCheck, &contentLayout, &limitDl, &limitUl, &limitRatio, &limitSeedTime, &a.ReAnnounceSkip, &a.ReAnnounceDelete, &a.ReAnnounceInterval, &a.ReAnnounceMaxAttempts, &webhookHost, &webhookType, &webhookMethod, &webhookData, &externalClientID, &clientID); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		a.ExecCmd = execCmd.String
		a.ExecArgs = execArgs.String
		a.WatchFolder = watchFolder.String
		a.Category = category.String
		a.Tags = tags.String
		a.Label = label.String
		a.SavePath = savePath.String
		a.Paused = paused.Bool
		a.IgnoreRules = ignoreRules.Bool
		a.ContentLayout = domain.ActionContentLayout(contentLayout.String)

		a.LimitDownloadSpeed = limitDl.Int64
		a.LimitUploadSpeed = limitUl.Int64
		a.LimitRatio = limitRatio.Float64
		a.LimitSeedTime = limitSeedTime.Int64

		a.WebhookHost = webhookHost.String
		a.WebhookType = webhookType.String
		a.WebhookMethod = webhookMethod.String
		a.WebhookData = webhookData.String

		a.ExternalDownloadClientID = externalClientID.Int32
		a.ClientID = clientID.Int32

		actions = append(actions, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "row error")
	}

	return actions, nil
}

func (r *ActionRepo) attachDownloadClient(ctx context.Context, tx *Tx, clientID int32) (*domain.DownloadClient, error) {
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
		Where(sq.Eq{"id": clientID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := tx.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var client domain.DownloadClient
	var settingsJsonStr string

	if err := row.Scan(&client.ID, &client.Name, &client.Type, &client.Enabled, &client.Host, &client.Port, &client.TLS, &client.TLSSkipVerify, &client.Username, &client.Password, &settingsJsonStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Warn().Msgf("no download client with id %d", clientID)
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

func (r *ActionRepo) List(ctx context.Context) ([]domain.Action, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"name",
			"type",
			"enabled",
			"exec_cmd",
			"exec_args",
			"watch_folder",
			"category",
			"tags",
			"label",
			"save_path",
			"paused",
			"ignore_rules",
			"skip_hash_check",
			"content_layout",
			"limit_download_speed",
			"limit_upload_speed",
			"limit_ratio",
			"limit_seed_time",
			"reannounce_skip",
			"reannounce_delete",
			"reannounce_interval",
			"reannounce_max_attempts",
			"webhook_host",
			"webhook_type",
			"webhook_method",
			"webhook_data",
			"external_client_id",
			"client_id",
		).
		From("action")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	actions := make([]domain.Action, 0)
	for rows.Next() {
		var a domain.Action

		var execCmd, execArgs, watchFolder, category, tags, label, savePath, contentLayout, webhookHost, webhookType, webhookMethod, webhookData sql.NullString
		var limitUl, limitDl, limitSeedTime sql.NullInt64
		var limitRatio sql.NullFloat64
		var externalClientID, clientID sql.NullInt32
		var paused, ignoreRules sql.NullBool

		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Enabled, &execCmd, &execArgs, &watchFolder, &category, &tags, &label, &savePath, &paused, &ignoreRules, &a.SkipHashCheck, &contentLayout, &limitDl, &limitUl, &limitRatio, &limitSeedTime, &a.ReAnnounceSkip, &a.ReAnnounceDelete, &a.ReAnnounceInterval, &a.ReAnnounceMaxAttempts, &webhookHost, &webhookType, &webhookMethod, &webhookData, &externalClientID, &clientID); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		a.Category = category.String
		a.Tags = tags.String
		a.Label = label.String
		a.SavePath = savePath.String
		a.Paused = paused.Bool
		a.IgnoreRules = ignoreRules.Bool
		a.ContentLayout = domain.ActionContentLayout(contentLayout.String)

		a.LimitDownloadSpeed = limitDl.Int64
		a.LimitUploadSpeed = limitUl.Int64
		a.LimitRatio = limitRatio.Float64
		a.LimitSeedTime = limitSeedTime.Int64

		a.WebhookHost = webhookHost.String
		a.WebhookType = webhookType.String
		a.WebhookMethod = webhookMethod.String
		a.WebhookData = webhookData.String

		a.ExternalDownloadClientID = externalClientID.Int32
		a.ClientID = clientID.Int32

		actions = append(actions, a)

		if err := rows.Err(); err != nil {
			return nil, errors.Wrap(err, "rows error")
		}
	}

	return actions, nil
}

func (r *ActionRepo) Get(ctx context.Context, req *domain.GetActionRequest) (*domain.Action, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"name",
			"type",
			"enabled",
			"exec_cmd",
			"exec_args",
			"watch_folder",
			"category",
			"tags",
			"label",
			"save_path",
			"paused",
			"ignore_rules",
			"skip_hash_check",
			"content_layout",
			"limit_download_speed",
			"limit_upload_speed",
			"limit_ratio",
			"limit_seed_time",
			"reannounce_skip",
			"reannounce_delete",
			"reannounce_interval",
			"reannounce_max_attempts",
			"webhook_host",
			"webhook_type",
			"webhook_method",
			"webhook_data",
			"external_client_id",
			"client_id",
			"filter_id",
		).
		From("action").
		Where(sq.Eq{"id": req.Id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "rows error")
	}

	var a domain.Action

	var execCmd, execArgs, watchFolder, category, tags, label, savePath, contentLayout, webhookHost, webhookType, webhookMethod, webhookData sql.NullString
	var limitUl, limitDl, limitSeedTime sql.NullInt64
	var limitRatio sql.NullFloat64
	var externalClientID, clientID, filterID sql.NullInt32
	var paused, ignoreRules sql.NullBool

	if err := row.Scan(&a.ID, &a.Name, &a.Type, &a.Enabled, &execCmd, &execArgs, &watchFolder, &category, &tags, &label, &savePath, &paused, &ignoreRules, &a.SkipHashCheck, &contentLayout, &limitDl, &limitUl, &limitRatio, &limitSeedTime, &a.ReAnnounceSkip, &a.ReAnnounceDelete, &a.ReAnnounceInterval, &a.ReAnnounceMaxAttempts, &webhookHost, &webhookType, &webhookMethod, &webhookData, &externalClientID, &clientID, &filterID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	a.ExecCmd = execCmd.String
	a.ExecArgs = execArgs.String
	a.WatchFolder = watchFolder.String
	a.Category = category.String
	a.Tags = tags.String
	a.Label = label.String
	a.SavePath = savePath.String
	a.Paused = paused.Bool
	a.IgnoreRules = ignoreRules.Bool
	a.ContentLayout = domain.ActionContentLayout(contentLayout.String)

	a.LimitDownloadSpeed = limitDl.Int64
	a.LimitUploadSpeed = limitUl.Int64
	a.LimitRatio = limitRatio.Float64
	a.LimitSeedTime = limitSeedTime.Int64

	a.WebhookHost = webhookHost.String
	a.WebhookType = webhookType.String
	a.WebhookMethod = webhookMethod.String
	a.WebhookData = webhookData.String

	a.ExternalDownloadClientID = externalClientID.Int32
	a.ClientID = clientID.Int32
	a.FilterID = int(filterID.Int32)

	return &a, nil
}

func (r *ActionRepo) Delete(ctx context.Context, req *domain.DeleteActionRequest) error {
	queryBuilder := r.db.squirrel.
		Delete("action").
		Where(sq.Eq{"id": req.ActionId})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err = r.db.handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("action.delete: %v", req.ActionId)

	return nil
}

func (r *ActionRepo) DeleteByFilterID(ctx context.Context, filterID int) error {
	queryBuilder := r.db.squirrel.
		Delete("action").
		Where(sq.Eq{"filter_id": filterID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err := r.db.handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("action.deleteByFilterID: %v", filterID)

	return nil
}

func (r *ActionRepo) Store(ctx context.Context, action domain.Action) (*domain.Action, error) {
	queryBuilder := r.db.squirrel.
		Insert("action").
		Columns(
			"name",
			"type",
			"enabled",
			"exec_cmd",
			"exec_args",
			"watch_folder",
			"category",
			"tags",
			"label",
			"save_path",
			"paused",
			"ignore_rules",
			"skip_hash_check",
			"content_layout",
			"limit_upload_speed",
			"limit_download_speed",
			"limit_ratio",
			"limit_seed_time",
			"reannounce_skip",
			"reannounce_delete",
			"reannounce_interval",
			"reannounce_max_attempts",
			"webhook_host",
			"webhook_type",
			"webhook_method",
			"webhook_data",
			"external_client_id",
			"client_id",
			"filter_id",
		).
		Values(
			action.Name,
			action.Type,
			action.Enabled,
			toNullString(action.ExecCmd),
			toNullString(action.ExecArgs),
			toNullString(action.WatchFolder),
			toNullString(action.Category),
			toNullString(action.Tags),
			toNullString(action.Label),
			toNullString(action.SavePath),
			action.Paused,
			action.IgnoreRules,
			action.SkipHashCheck,
			toNullString(string(action.ContentLayout)),
			toNullInt64(action.LimitUploadSpeed),
			toNullInt64(action.LimitDownloadSpeed),
			toNullFloat64(action.LimitRatio),
			toNullInt64(action.LimitSeedTime),
			action.ReAnnounceSkip,
			action.ReAnnounceDelete,
			action.ReAnnounceInterval,
			action.ReAnnounceMaxAttempts,
			toNullString(action.WebhookHost),
			toNullString(action.WebhookType),
			toNullString(action.WebhookMethod),
			toNullString(action.WebhookData),
			toNullInt32(action.ExternalDownloadClientID),
			toNullInt32(action.ClientID),
			toNullInt32(int32(action.FilterID)),
		).
		Suffix("RETURNING id").RunWith(r.db.handler)

	// return values
	var retID int64

	if err := queryBuilder.QueryRowContext(ctx).Scan(&retID); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	action.ID = int(retID)

	r.log.Debug().Msgf("action.store: added new %d", retID)

	return &action, nil
}

func (r *ActionRepo) Update(ctx context.Context, action domain.Action) (*domain.Action, error) {
	queryBuilder := r.db.squirrel.
		Update("action").
		Set("name", action.Name).
		Set("type", action.Type).
		Set("enabled", action.Enabled).
		Set("exec_cmd", toNullString(action.ExecCmd)).
		Set("exec_args", toNullString(action.ExecArgs)).
		Set("watch_folder", toNullString(action.WatchFolder)).
		Set("category", toNullString(action.Category)).
		Set("tags", toNullString(action.Tags)).
		Set("label", toNullString(action.Label)).
		Set("save_path", toNullString(action.SavePath)).
		Set("paused", action.Paused).
		Set("ignore_rules", action.IgnoreRules).
		Set("skip_hash_check", action.SkipHashCheck).
		Set("content_layout", toNullString(string(action.ContentLayout))).
		Set("limit_upload_speed", toNullInt64(action.LimitUploadSpeed)).
		Set("limit_download_speed", toNullInt64(action.LimitDownloadSpeed)).
		Set("limit_ratio", toNullFloat64(action.LimitRatio)).
		Set("limit_seed_time", toNullInt64(action.LimitSeedTime)).
		Set("reannounce_skip", action.ReAnnounceSkip).
		Set("reannounce_delete", action.ReAnnounceDelete).
		Set("reannounce_interval", action.ReAnnounceInterval).
		Set("reannounce_max_attempts", action.ReAnnounceMaxAttempts).
		Set("webhook_host", toNullString(action.WebhookHost)).
		Set("webhook_type", toNullString(action.WebhookType)).
		Set("webhook_method", toNullString(action.WebhookMethod)).
		Set("webhook_data", toNullString(action.WebhookData)).
		Set("external_client_id", toNullInt32(action.ExternalDownloadClientID)).
		Set("client_id", toNullInt32(action.ClientID)).
		Set("filter_id", toNullInt32(int32(action.FilterID))).
		Where(sq.Eq{"id": action.ID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	if _, err := r.db.handler.ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("action.update: %v", action.ID)

	return &action, nil
}

func (r *ActionRepo) StoreFilterActions(ctx context.Context, filterID int64, actions []*domain.Action) ([]*domain.Action, error) {
	tx, err := r.db.handler.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error begin transaction")
	}

	defer tx.Rollback()

	for _, action := range actions {
		action := action

		if action.ID > 0 {
			queryBuilder := r.db.squirrel.
				Update("action").
				Set("name", action.Name).
				Set("type", action.Type).
				Set("enabled", action.Enabled).
				Set("exec_cmd", toNullString(action.ExecCmd)).
				Set("exec_args", toNullString(action.ExecArgs)).
				Set("watch_folder", toNullString(action.WatchFolder)).
				Set("category", toNullString(action.Category)).
				Set("tags", toNullString(action.Tags)).
				Set("label", toNullString(action.Label)).
				Set("save_path", toNullString(action.SavePath)).
				Set("paused", action.Paused).
				Set("ignore_rules", action.IgnoreRules).
				Set("skip_hash_check", action.SkipHashCheck).
				Set("content_layout", toNullString(string(action.ContentLayout))).
				Set("limit_upload_speed", toNullInt64(action.LimitUploadSpeed)).
				Set("limit_download_speed", toNullInt64(action.LimitDownloadSpeed)).
				Set("limit_ratio", toNullFloat64(action.LimitRatio)).
				Set("limit_seed_time", toNullInt64(action.LimitSeedTime)).
				Set("reannounce_skip", action.ReAnnounceSkip).
				Set("reannounce_delete", action.ReAnnounceDelete).
				Set("reannounce_interval", action.ReAnnounceInterval).
				Set("reannounce_max_attempts", action.ReAnnounceMaxAttempts).
				Set("webhook_host", toNullString(action.WebhookHost)).
				Set("webhook_type", toNullString(action.WebhookType)).
				Set("webhook_method", toNullString(action.WebhookMethod)).
				Set("webhook_data", toNullString(action.WebhookData)).
				Set("external_client_id", toNullInt32(action.ExternalDownloadClientID)).
				Set("client_id", toNullInt32(action.ClientID)).
				Set("filter_id", toNullInt64(filterID)).
				Where(sq.Eq{"id": action.ID})

			query, args, err := queryBuilder.ToSql()
			if err != nil {
				return nil, errors.Wrap(err, "error building query")
			}

			if _, err := tx.ExecContext(ctx, query, args...); err != nil {
				return nil, errors.Wrap(err, "error executing query")
			}

			r.log.Trace().Msgf("action.StoreFilterActions: update %d", action.ID)

		} else {
			queryBuilder := r.db.squirrel.
				Insert("action").
				Columns(
					"name",
					"type",
					"enabled",
					"exec_cmd",
					"exec_args",
					"watch_folder",
					"category",
					"tags",
					"label",
					"save_path",
					"paused",
					"ignore_rules",
					"skip_hash_check",
					"content_layout",
					"limit_upload_speed",
					"limit_download_speed",
					"limit_ratio",
					"limit_seed_time",
					"reannounce_skip",
					"reannounce_delete",
					"reannounce_interval",
					"reannounce_max_attempts",
					"webhook_host",
					"webhook_type",
					"webhook_method",
					"webhook_data",
					"external_client_id",
					"client_id",
					"filter_id",
				).
				Values(
					action.Name,
					action.Type,
					action.Enabled,
					toNullString(action.ExecCmd),
					toNullString(action.ExecArgs),
					toNullString(action.WatchFolder),
					toNullString(action.Category),
					toNullString(action.Tags),
					toNullString(action.Label),
					toNullString(action.SavePath),
					action.Paused,
					action.IgnoreRules,
					action.SkipHashCheck,
					toNullString(string(action.ContentLayout)),
					toNullInt64(action.LimitUploadSpeed),
					toNullInt64(action.LimitDownloadSpeed),
					toNullFloat64(action.LimitRatio),
					toNullInt64(action.LimitSeedTime),
					action.ReAnnounceSkip,
					action.ReAnnounceDelete,
					action.ReAnnounceInterval,
					action.ReAnnounceMaxAttempts,
					toNullString(action.WebhookHost),
					toNullString(action.WebhookType),
					toNullString(action.WebhookMethod),
					toNullString(action.WebhookData),
					toNullInt32(action.ExternalDownloadClientID),
					toNullInt32(action.ClientID),
					toNullInt64(filterID),
				).
				Suffix("RETURNING id").RunWith(tx)

			// return values
			var retID int

			if err := queryBuilder.QueryRowContext(ctx).Scan(&retID); err != nil {
				return nil, errors.Wrap(err, "error executing query")
			}

			action.ID = retID

			r.log.Trace().Msgf("action.StoreFilterActions: store %d", action.ID)
		}

		r.log.Debug().Msgf("action.StoreFilterActions: store '%s' type: '%v' on filter: %d", action.Name, action.Type, filterID)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "error updating filter actions")
	}

	return actions, nil
}

func (r *ActionRepo) ToggleEnabled(actionID int) error {
	queryBuilder := r.db.squirrel.
		Update("action").
		Set("enabled", sq.Expr("NOT enabled")).
		Where(sq.Eq{"id": actionID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err := r.db.handler.Exec(query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("action.toggleEnabled: %v", actionID)

	return nil
}
