// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

type NotificationRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewNotificationRepo(log logger.Logger, db *DB) domain.NotificationRepo {
	return &NotificationRepo{
		log: log.With().Str("repo", "notification").Logger(),
		db:  db,
	}
}

func (r *NotificationRepo) Find(ctx context.Context, params domain.NotificationQueryParams) ([]domain.Notification, int, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "name", "type", "enabled", "events", "webhook", "token", "api_key", "channel", "priority", "topic", "host", "username", "password", "created_at", "updated_at", "COUNT(*) OVER() AS total_count").
		From("notification").
		OrderBy("name")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, 0, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	notifications := make([]domain.Notification, 0)
	totalCount := 0
	for rows.Next() {
		n := domain.NewNotification()

		var webhook, token, apiKey, channel, host, topic, username, password sql.Null[string]

		if err := rows.Scan(&n.ID, &n.Name, &n.Type, &n.Enabled, pq.Array(&n.Events), &webhook, &token, &apiKey, &channel, &n.Priority, &topic, &host, &username, &password, &n.CreatedAt, &n.UpdatedAt, &totalCount); err != nil {
			return nil, 0, errors.Wrap(err, "error scanning row")
		}

		n.APIKey = apiKey.V
		n.Webhook = webhook.V
		n.Token = token.V
		n.Channel = channel.V
		n.Topic = topic.V
		n.Host = host.V
		n.Username = username.V
		n.Password = password.V

		notifications = append(notifications, *n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, errors.Wrap(err, "error rows find")
	}

	return notifications, totalCount, nil
}

func (r *NotificationRepo) List(ctx context.Context) ([]domain.Notification, error) {
	rows, err := r.db.Handler.QueryContext(ctx, "SELECT id, name, type, enabled, events, token, api_key,  webhook, title, icon, host, username, password, channel, targets, devices, priority, topic, created_at, updated_at FROM notification ORDER BY name ASC")
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	var notifications []domain.Notification
	for rows.Next() {
		n := domain.NewNotification()
		//var eventsSlice []string

		var token, apiKey, webhook, title, icon, host, username, password, channel, targets, devices, topic sql.Null[string]
		if err := rows.Scan(&n.ID, &n.Name, &n.Type, &n.Enabled, pq.Array(&n.Events), &token, &apiKey, &webhook, &title, &icon, &host, &username, &password, &channel, &targets, &devices, &n.Priority, &topic, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		//n.Events = ([]domain.NotificationEvent)(eventsSlice)
		n.Token = token.V
		n.APIKey = apiKey.V
		n.Webhook = webhook.V
		n.Title = title.V
		n.Icon = icon.V
		n.Host = host.V
		n.Username = username.V
		n.Password = password.V
		n.Channel = channel.V
		n.Targets = targets.V
		n.Devices = devices.V
		n.Topic = topic.V

		notifications = append(notifications, *n)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error rows list")
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
			"api_key",
			"webhook",
			"title",
			"icon",
			"host",
			"username",
			"password",
			"channel",
			"targets",
			"devices",
			"priority",
			"topic",
			"created_at",
			"updated_at",
		).
		From("notification").
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.Handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	n := domain.NewNotification()

	var token, apiKey, webhook, title, icon, host, username, password, channel, targets, devices, topic sql.Null[string]
	if err := row.Scan(&n.ID, &n.Name, &n.Type, &n.Enabled, pq.Array(&n.Events), &token, &apiKey, &webhook, &title, &icon, &host, &username, &password, &channel, &targets, &devices, &n.Priority, &topic, &n.CreatedAt, &n.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	n.Token = token.V
	n.APIKey = apiKey.V
	n.Webhook = webhook.V
	n.Title = title.V
	n.Icon = icon.V
	n.Host = host.V
	n.Username = username.V
	n.Password = password.V
	n.Channel = channel.V
	n.Targets = targets.V
	n.Devices = devices.V
	n.Topic = topic.V

	return n, nil
}

func (r *NotificationRepo) Store(ctx context.Context, notification *domain.Notification) error {
	queryBuilder := r.db.squirrel.
		Insert("notification").
		Columns(
			"name",
			"type",
			"enabled",
			"events",
			"webhook",
			"token",
			"api_key",
			"channel",
			"priority",
			"topic",
			"host",
			"username",
			"password",
		).
		Values(
			notification.Name,
			notification.Type,
			notification.Enabled,
			pq.Array(notification.Events),
			toNullString(notification.Webhook),
			toNullString(notification.Token),
			toNullString(notification.APIKey),
			toNullString(notification.Channel),
			notification.Priority,
			toNullString(notification.Topic),
			toNullString(notification.Host),
			toNullString(notification.Username),
			toNullString(notification.Password),
		).
		Suffix("RETURNING id").RunWith(r.db.Handler)

	if err := queryBuilder.QueryRowContext(ctx).Scan(&notification.ID); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("notification.store: added new %v", notification.ID)

	return nil
}

func (r *NotificationRepo) Update(ctx context.Context, notification *domain.Notification) error {
	queryBuilder := r.db.squirrel.
		Update("notification").
		Set("name", notification.Name).
		Set("type", notification.Type).
		Set("enabled", notification.Enabled).
		Set("events", pq.Array(notification.Events)).
		Set("webhook", toNullString(notification.Webhook)).
		Set("token", toNullString(notification.Token)).
		Set("api_key", toNullString(notification.APIKey)).
		Set("channel", toNullString(notification.Channel)).
		Set("priority", notification.Priority).
		Set("topic", toNullString(notification.Topic)).
		Set("host", toNullString(notification.Host)).
		Set("username", toNullString(notification.Username)).
		Set("password", toNullString(notification.Password)).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": notification.ID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err = r.db.Handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("notification.update: %v", notification.Name)

	return nil
}

func (r *NotificationRepo) Delete(ctx context.Context, notificationID int) error {
	queryBuilder := r.db.squirrel.
		Delete("notification").
		Where(sq.Eq{"id": notificationID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err = r.db.Handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	r.log.Debug().Msgf("notification.delete: successfully deleted: %v", notificationID)

	return nil
}

func (r *NotificationRepo) GetNotificationFilters(ctx context.Context, notificationID int) ([]domain.FilterNotification, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"fn.filter_id",
			"fn.notification_id",
			"fn.events",
		).
		From("filter_notification fn").
		Where(sq.Eq{"fn.notification_id": notificationID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}
	defer rows.Close()

	var notifications []domain.FilterNotification
	for rows.Next() {
		var fn domain.FilterNotification
		var events pq.StringArray

		if err := rows.Scan(&fn.FilterID, &fn.NotificationID, &events); err != nil {
			return nil, errors.Wrap(err, "error scanning filter notification")
		}

		fn.Events = events
		notifications = append(notifications, fn)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating over filter notifications")
	}

	return notifications, nil
}

func (r *NotificationRepo) GetFilterNotifications(ctx context.Context, filterID int) ([]domain.FilterNotification, error) {
	queryBuilder := r.db.squirrel.
		Select(
			"fn.filter_id",
			"fn.notification_id",
			"fn.events",
		).
		From("filter_notification fn").
		Where(sq.Eq{"fn.filter_id": filterID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}
	defer rows.Close()

	var notifications []domain.FilterNotification
	for rows.Next() {
		var fn domain.FilterNotification
		var events pq.StringArray

		if err := rows.Scan(&fn.FilterID, &fn.NotificationID, &events); err != nil {
			return nil, errors.Wrap(err, "error scanning filter notification")
		}

		fn.Events = events
		notifications = append(notifications, fn)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating over filter notifications")
	}

	return notifications, nil
}

func (r *NotificationRepo) StoreFilterNotifications(ctx context.Context, filterID int, notifications []domain.FilterNotification) error {
	tx, err := r.db.Handler.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if err := r.deleteFilterNotifications(ctx, tx, filterID); err != nil {
		return errors.Wrap(err, "failed to delete existing filter notifications")
	}

	if len(notifications) == 0 {
		if err := tx.Commit(); err != nil {
			return errors.Wrap(err, "failed to commit transaction after deleting filter notifications for filter %d", filterID)
		}
		r.log.Debug().Msgf("filter.StoreFilterNotifications: deleted all notifications for filter: %d", filterID)
		return nil
	}

	if err := r.insertFilterNotifications(ctx, tx, filterID, notifications); err != nil {
		return errors.Wrap(err, "failed to insert filter notifications")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction for storing filter notifications for filter %d", filterID)
	}

	r.log.Debug().Msgf("filter.StoreFilterNotifications: stored %d notifications for filter %d", len(notifications), filterID)
	return nil
}

// deleteFilterNotifications handles the deletion of existing filter notifications within a transaction
func (r *NotificationRepo) deleteFilterNotifications(ctx context.Context, tx *sql.Tx, filterID int) error {
	deleteQuery, deleteArgs, err := r.db.squirrel.Delete("filter_notification").Where(sq.Eq{"filter_id": filterID}).ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build delete query")
	}

	if _, err := tx.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
		return errors.Wrap(err, "failed to execute delete query")
	}

	return nil
}

// insertFilterNotifications handles the insertion of new filter notifications within a transaction
func (r *NotificationRepo) insertFilterNotifications(ctx context.Context, tx *sql.Tx, filterID int, notifications []domain.FilterNotification) error {
	insertBuilder := r.db.squirrel.Insert("filter_notification").Columns("filter_id", "notification_id", "events")

	for _, notification := range notifications {
		insertBuilder = insertBuilder.Values(
			filterID,
			notification.NotificationID,
			pq.Array(notification.Events),
		)
	}

	query, args, err := insertBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build insert query")
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "failed to execute insert query")
	}

	return nil
}

func (r *NotificationRepo) DeleteFilterNotifications(ctx context.Context, filterID int) error {
	queryBuilder := r.db.squirrel.
		Delete("filter_notification").
		Where(sq.Eq{"filter_id": filterID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error getting rows affected")
	}

	r.log.Debug().Msgf("filter.DeleteFilterNotifications: deleted %d notifications for filter: %d", rowsAffected, filterID)

	return nil
}
