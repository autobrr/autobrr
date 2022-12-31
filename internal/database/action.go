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
		if action.ClientID != 0 {
			client, err := r.attachDownloadClient(ctx, tx, action.ClientID)
			if err != nil {
				return nil, err
			}
			action.Client = *client
		}
	}

	return actions, nil
}

func (r *ActionRepo) findByFilterID(ctx context.Context, tx *Tx, filterID int) ([]*domain.Action, error) {
	queryBuilder := r.db.squirrel.
		RunWith(tx).
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
			"client_id",
		).
		From("action").
		Where(sq.Eq{"filter_id": filterID})

	rows, err := queryBuilder.Query()
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

		var clientID sql.NullInt32
		// filterID
		var paused, ignoreRules sql.NullBool

		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Enabled, &execCmd, &execArgs, &watchFolder, &category, &tags, &label, &savePath, &paused, &ignoreRules, &a.SkipHashCheck, &contentLayout, &limitDl, &limitUl, &limitRatio, &limitSeedTime, &a.ReAnnounceSkip, &a.ReAnnounceDelete, &a.ReAnnounceInterval, &a.ReAnnounceMaxAttempts, &webhookHost, &webhookType, &webhookMethod, &webhookData, &clientID); err != nil {
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
		RunWith(tx).
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

	row, err := queryBuilder.Query()
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var client domain.DownloadClient
	var settingsJsonStr string

	for row.Next() {
		if err := row.Scan(&client.ID, &client.Name, &client.Type, &client.Enabled, &client.Host, &client.Port, &client.TLS, &client.TLSSkipVerify, &client.Username, &client.Password, &settingsJsonStr); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}
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
		RunWith(r.db.handler).
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
			"client_id",
		).
		From("action")

	rows, err := queryBuilder.Query()
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	actions := make([]domain.Action, 0)
	for rows.Next() {
		var a domain.Action

		var execCmd, execArgs, watchFolder, category, tags, label, savePath, webhookHost, webhookType, webhookMethod, webhookData sql.NullString
		var limitUl, limitDl, limitSeedTime sql.NullInt64
		var limitRatio sql.NullFloat64
		var clientID sql.NullInt32
		var paused, ignoreRules sql.NullBool

		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Enabled, &execCmd, &execArgs, &watchFolder, &category, &tags, &label, &savePath, &paused, &ignoreRules, &limitDl, &limitUl, &limitRatio, &limitSeedTime, &a.ReAnnounceSkip, &a.ReAnnounceDelete, &a.ReAnnounceInterval, &a.ReAnnounceMaxAttempts, &webhookHost, &webhookType, &webhookMethod, &webhookData, &clientID); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		a.Category = category.String
		a.Tags = tags.String
		a.Label = label.String
		a.SavePath = savePath.String
		a.Paused = paused.Bool
		a.IgnoreRules = ignoreRules.Bool

		a.LimitDownloadSpeed = limitDl.Int64
		a.LimitUploadSpeed = limitUl.Int64
		a.LimitRatio = limitRatio.Float64
		a.LimitSeedTime = limitSeedTime.Int64

		a.WebhookHost = webhookHost.String
		a.WebhookType = webhookType.String
		a.WebhookMethod = webhookMethod.String
		a.WebhookData = webhookData.String

		a.ClientID = clientID.Int32

		actions = append(actions, a)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows error")
	}

	return actions, nil
}

func (r *ActionRepo) Delete(actionID int) error {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Delete("action").
		Where(sq.Eq{"id": actionID})

	_, err := queryBuilder.Exec()
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("action.delete: %v", actionID)

	return nil
}

func (r *ActionRepo) DeleteByFilterID(ctx context.Context, filterID int) error {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Delete("action").
		Where(sq.Eq{"filter_id": filterID})

	_, err := queryBuilder.Exec()
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("action.deleteByFilterID: %v", filterID)

	return nil
}

func (r *ActionRepo) Store(ctx context.Context, action domain.Action) (*domain.Action, error) {
	execCmd := toNullString(action.ExecCmd)
	execArgs := toNullString(action.ExecArgs)
	watchFolder := toNullString(action.WatchFolder)
	category := toNullString(action.Category)
	tags := toNullString(action.Tags)
	label := toNullString(action.Label)
	savePath := toNullString(action.SavePath)
	contentLayout := toNullString(string(action.ContentLayout))
	webhookHost := toNullString(action.WebhookHost)
	webhookData := toNullString(action.WebhookData)
	webhookType := toNullString(action.WebhookType)
	webhookMethod := toNullString(action.WebhookMethod)

	limitDL := toNullInt64(action.LimitDownloadSpeed)
	limitUL := toNullInt64(action.LimitUploadSpeed)
	limitRatio := toNullFloat64(action.LimitRatio)
	limitSeedTime := toNullInt64(action.LimitSeedTime)
	clientID := toNullInt32(action.ClientID)
	filterID := toNullInt32(int32(action.FilterID))

	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
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
			"client_id",
			"filter_id",
		).
		Values(
			action.Name,
			action.Type,
			action.Enabled,
			execCmd,
			execArgs,
			watchFolder,
			category,
			tags,
			label,
			savePath,
			action.Paused,
			action.IgnoreRules,
			action.SkipHashCheck,
			contentLayout,
			limitUL,
			limitDL,
			limitRatio,
			limitSeedTime,
			action.ReAnnounceSkip,
			action.ReAnnounceDelete,
			action.ReAnnounceInterval,
			action.ReAnnounceMaxAttempts,
			webhookHost,
			webhookType,
			webhookMethod,
			webhookData,
			clientID,
			filterID,
		).
		Suffix("RETURNING id")

	// return values
	var retID int64

	err := queryBuilder.QueryRow().Scan(&retID)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("action.store: added new %v", retID)
	action.ID = int(retID)

	return &action, nil
}

func (r *ActionRepo) Update(ctx context.Context, action domain.Action) (*domain.Action, error) {
	execCmd := toNullString(action.ExecCmd)
	execArgs := toNullString(action.ExecArgs)
	watchFolder := toNullString(action.WatchFolder)
	category := toNullString(action.Category)
	tags := toNullString(action.Tags)
	label := toNullString(action.Label)
	savePath := toNullString(action.SavePath)
	contentLayout := toNullString(string(action.ContentLayout))
	webhookHost := toNullString(action.WebhookHost)
	webhookType := toNullString(action.WebhookType)
	webhookMethod := toNullString(action.WebhookMethod)
	webhookData := toNullString(action.WebhookData)

	limitDL := toNullInt64(action.LimitDownloadSpeed)
	limitUL := toNullInt64(action.LimitUploadSpeed)
	limitRatio := toNullFloat64(action.LimitRatio)
	limitSeedTime := toNullInt64(action.LimitSeedTime)

	clientID := toNullInt32(action.ClientID)
	filterID := toNullInt32(int32(action.FilterID))

	var err error

	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Update("action").
		Set("name", action.Name).
		Set("type", action.Type).
		Set("enabled", action.Enabled).
		Set("exec_cmd", execCmd).
		Set("exec_args", execArgs).
		Set("watch_folder", watchFolder).
		Set("category", category).
		Set("tags", tags).
		Set("label", label).
		Set("save_path", savePath).
		Set("paused", action.Paused).
		Set("ignore_rules", action.IgnoreRules).
		Set("skip_hash_check", action.SkipHashCheck).
		Set("content_layout", contentLayout).
		Set("limit_upload_speed", limitUL).
		Set("limit_download_speed", limitDL).
		Set("limit_ratio", limitRatio).
		Set("limit_seed_time", limitSeedTime).
		Set("reannounce_skip", action.ReAnnounceSkip).
		Set("reannounce_delete", action.ReAnnounceDelete).
		Set("reannounce_interval", action.ReAnnounceInterval).
		Set("reannounce_max_attempts", action.ReAnnounceMaxAttempts).
		Set("webhook_host", webhookHost).
		Set("webhook_type", webhookType).
		Set("webhook_method", webhookMethod).
		Set("webhook_data", webhookData).
		Set("client_id", clientID).
		Set("filter_id", filterID).
		Where(sq.Eq{"id": action.ID})

	_, err = queryBuilder.Exec()
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("action.update: %v", action.ID)

	return &action, nil
}

func (r *ActionRepo) StoreFilterActions(ctx context.Context, actions []*domain.Action, filterID int64) ([]*domain.Action, error) {
	tx, err := (r.db.handler).Begin()
	if err != nil {
		return nil, errors.Wrap(err, "error begin transaction")
	}

	defer tx.Rollback()

	deleteQueryBuilder := r.db.squirrel.
		RunWith(tx).
		Delete("action").
		Where(sq.Eq{"filter_id": filterID})

	_, err = deleteQueryBuilder.Exec()
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	queryBuilder := r.db.squirrel.
	RunWith(tx).
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
		"client_id",
		"filter_id",
	)

	for _, action := range actions {
		execCmd := toNullString(action.ExecCmd)
		execArgs := toNullString(action.ExecArgs)
		watchFolder := toNullString(action.WatchFolder)
		category := toNullString(action.Category)
		tags := toNullString(action.Tags)
		label := toNullString(action.Label)
		savePath := toNullString(action.SavePath)
		contentLayout := toNullString(string(action.ContentLayout))
		webhookHost := toNullString(action.WebhookHost)
		webhookType := toNullString(action.WebhookType)
		webhookMethod := toNullString(action.WebhookMethod)
		webhookData := toNullString(action.WebhookData)

		limitDL := toNullInt64(action.LimitDownloadSpeed)
		limitUL := toNullInt64(action.LimitUploadSpeed)
		limitRatio := toNullFloat64(action.LimitRatio)
		limitSeedTime := toNullInt64(action.LimitSeedTime)
		clientID := toNullInt32(action.ClientID)


		// scan values
		var retID int

		if err := queryBuilder.Values(
				action.Name,
				action.Type,
				action.Enabled,
				execCmd,
				execArgs,
				watchFolder,
				category,
				tags,
				label,
				savePath,
				action.Paused,
				action.IgnoreRules,
				action.SkipHashCheck,
				contentLayout,
				limitUL,
				limitDL,
				limitRatio,
				limitSeedTime,
				action.ReAnnounceSkip,
				action.ReAnnounceDelete,
				action.ReAnnounceInterval,
				action.ReAnnounceMaxAttempts,
				webhookHost,
				webhookType,
				webhookMethod,
				webhookData,
				clientID,
				filterID,
			).
			Suffix("RETURNING id").
			QueryRow().
			Scan(&retID); err != nil {
				return nil, errors.Wrap(err, "error executing query")
			}

		action.ID = retID

		r.log.Debug().Msgf("action.StoreFilterActions: store '%v' type: '%v' on filter: %v", action.Name, action.Type, filterID)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "error updating filter actions")
	}

	return actions, nil
}

func (r *ActionRepo) ToggleEnabled(actionID int) error {
	queryBuilder := r.db.squirrel.
		RunWith(r.db.handler).
		Update("action").
		Set("enabled", sq.Expr("NOT enabled")).
		Where(sq.Eq{"id": actionID})

	if _, err := queryBuilder.Exec(); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("action.toggleEnabled: %v", actionID)

	return nil
}
