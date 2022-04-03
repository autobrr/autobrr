package database

import (
	"context"
	"database/sql"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type NotificationRepo struct {
	db *DB
}

func NewNotificationRepo(db *DB) domain.NotificationRepo {
	return &NotificationRepo{
		db: db,
	}
}

func (r *NotificationRepo) Find(ctx context.Context, params domain.NotificationQueryParams) ([]domain.Notification, int, error) {

	queryBuilder := r.db.squirrel.
		Select("id", "name", "type", "enabled", "events", "created_at", "updated_at", "COUNT(*) OVER() AS total_count").
		From("notification").
		OrderBy("name")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("notification.find: error building query")
		return nil, 0, err
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("notification.find: error executing query")
		return nil, 0, err
	}

	defer rows.Close()

	notifications := make([]domain.Notification, 0)
	totalCount := 0
	for rows.Next() {
		var n domain.Notification

		//var token, apiKey, webhook, title, icon, host, username, password, channel, targets, devices sql.NullString
		//if err := rows.Scan(&n.ID, &n.Name, &n.Type, &n.Enabled, pq.Array(&n.Events), &token, &apiKey, &webhook, &title, &icon, &host, &username, &password, &channel, &targets, &devices, &n.CreatedAt, &n.UpdatedAt); err != nil {
		//var token, apiKey, webhook, title, icon, host, username, password, channel, targets, devices sql.NullString
		if err := rows.Scan(&n.ID, &n.Name, &n.Type, &n.Enabled, pq.Array(&n.Events), &n.CreatedAt, &n.UpdatedAt, &totalCount); err != nil {
			log.Error().Stack().Err(err).Msg("notification.find: error scanning row")
			return nil, 0, err
		}

		//n.Token = token.String
		//n.APIKey = apiKey.String
		//n.Webhook = webhook.String
		//n.Title = title.String
		//n.Icon = icon.String
		//n.Host = host.String
		//n.Username = username.String
		//n.Password = password.String
		//n.Channel = channel.String
		//n.Targets = targets.String
		//n.Devices = devices.String

		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return notifications, totalCount, nil
}

func (r *NotificationRepo) List(ctx context.Context) ([]domain.Notification, error) {

	//queryBuilder := r.db.squirrel.
	//	Select("r.id", "r.filter_status", "r.rejections", "r.indexer", "r.filter", "r.protocol", "r.title", "r.torrent_name", "r.size", "r.timestamp", "COUNT(*) OVER() AS total_count").
	//	From("release r").
	//	OrderBy("r.timestamp DESC")
	//
	//query, args, err := queryBuilder.ToSql()

	rows, err := r.db.handler.QueryContext(ctx, "SELECT id, name, type, enabled, events, token, api_key, webhook, title, icon, host, username, password, channel, targets, devices, created_at, updated_at FROM notification ORDER BY name ASC")
	if err != nil {
		log.Error().Stack().Err(err).Msg("filters_list: error query data")
		return nil, err
	}

	defer rows.Close()

	var notifications []domain.Notification
	for rows.Next() {
		var n domain.Notification
		//var eventsSlice []string

		var token, apiKey, webhook, title, icon, host, username, password, channel, targets, devices sql.NullString
		if err := rows.Scan(&n.ID, &n.Name, &n.Type, &n.Enabled, pq.Array(&n.Events), &token, &apiKey, &webhook, &title, &icon, &host, &username, &password, &channel, &targets, &devices, &n.CreatedAt, &n.UpdatedAt); err != nil {
			log.Error().Stack().Err(err).Msg("notification_list: error scanning data to struct")
			return nil, err
		}

		//n.Events = ([]domain.NotificationEvent)(eventsSlice)
		n.Token = token.String
		n.APIKey = apiKey.String
		n.Webhook = webhook.String
		n.Title = title.String
		n.Icon = icon.String
		n.Host = host.String
		n.Username = username.String
		n.Password = password.String
		n.Channel = channel.String
		n.Targets = targets.String
		n.Devices = devices.String

		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notifications, nil
}

func (r *NotificationRepo) FindByID(ctx context.Context, id int) (*domain.Notification, error) {

	queryBuilder := r.db.squirrel.
		Select(
			"id",
			"name",
			"type",
			"enabled",
			"events",
			"token",
			"created_at",
			"updated_at",
		).
		From("notification").
		Where("id = ?", id)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("notification.findByID: error building query")
		return nil, err
	}

	//row := r.db.handler.QueryRowContext(ctx, "SELECT id, name, type, enabled, events, token, api_key, webhook, title, icon, host, username, password, channel, targets, devices, created_at, updated_at FROM notification WHERE id = ?", id)
	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, err
	}

	var n domain.Notification

	var token, apiKey, webhook, title, icon, host, username, password, channel, targets, devices sql.NullString
	if err := row.Scan(&n.ID, &n.Name, &n.Type, &n.Enabled, pq.Array(&n.Events), &token, &apiKey, &webhook, &title, &icon, &host, &username, &password, &channel, &targets, &devices, &n.CreatedAt, &n.UpdatedAt); err != nil {
		log.Error().Stack().Err(err).Msg("notification.findByID: error scanning row")
		return nil, err
	}

	n.Token = token.String
	n.APIKey = apiKey.String
	n.Webhook = webhook.String
	n.Title = title.String
	n.Icon = icon.String
	n.Host = host.String
	n.Username = username.String
	n.Password = password.String
	n.Channel = channel.String
	n.Targets = targets.String
	n.Devices = devices.String

	return &n, nil
}

func (r *NotificationRepo) Store(ctx context.Context, notification domain.Notification) (*domain.Notification, error) {
	webhook := toNullString(notification.Webhook)

	queryBuilder := r.db.squirrel.
		Insert("notification").
		Columns(
			"name",
			"type",
			"enabled",
			"events",
			"webhook",
		).
		Values(
			notification.Name,
			notification.Type,
			notification.Enabled,
			pq.Array(notification.Events),
			webhook,
		).
		Suffix("RETURNING id").RunWith(r.db.handler)

	// return values
	var retID int64

	err := queryBuilder.QueryRowContext(ctx).Scan(&retID)
	if err != nil {
		log.Error().Stack().Err(err).Msg("notification.store: error executing query")
		return nil, err
	}

	log.Debug().Msgf("notification.store: added new %v", retID)
	notification.ID = int(retID)

	return &notification, nil
}

func (r *NotificationRepo) Update(ctx context.Context, notification domain.Notification) (*domain.Notification, error) {
	webhook := toNullString(notification.Webhook)

	queryBuilder := r.db.squirrel.
		Update("notification").
		Set("name", notification.Name).
		Set("type", notification.Type).
		Set("enabled", notification.Enabled).
		Set("events", pq.Array(notification.Events)).
		Set("webhook", webhook).
		Where("id = ?", notification.ID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("action.update: error building query")
		return nil, err
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("notification.update: error executing query")
		return nil, err
	}

	log.Debug().Msgf("notification.update: %v", notification.Name)

	return &notification, nil
}

func (r *NotificationRepo) Delete(ctx context.Context, notificationID int) error {
	queryBuilder := r.db.squirrel.
		Delete("notification").
		Where("id = ?", notificationID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Error().Stack().Err(err).Msg("notification.delete: error building query")
		return err
	}

	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error().Stack().Err(err).Msg("notification.delete: error executing query")
		return err
	}

	log.Info().Msgf("notification.delete: successfully deleted: %v", notificationID)

	return nil
}
