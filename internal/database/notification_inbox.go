// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

type NotificationInboxRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewNotificationInboxRepo(log zerolog.Logger, db *DB) *NotificationInboxRepo {
	return &NotificationInboxRepo{
		log: log.With().Str("repo", "notification_inbox").Logger(),
		db:  db,
	}
}

func (r *NotificationInboxRepo) Find(ctx context.Context, params domain.InboxQueryParams) (*domain.FindInboxResponse, error) {
	resp := &domain.FindInboxResponse{
		Data: make([]*domain.InboxMessage, 0),
	}

	statsBuilder := r.db.squirrel.
		Select("COUNT(*) AS total", "COUNT(CASE WHEN read_at IS NULL THEN 1 END) AS unread_count").
		From("notification_inbox")

	if len(params.Events) > 0 {
		statsBuilder = statsBuilder.Where(sq.Eq{"event": params.Events})
	}

	statsQuery, statsArgs, err := statsBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	if err := r.db.Handler.QueryRowContext(ctx, statsQuery, statsArgs...).Scan(&resp.AllCount, &resp.UnreadCount); err != nil {
		return nil, errors.Wrap(err, "error scanning row")
	}

	resp.TotalCount = resp.AllCount
	if params.Unread {
		resp.TotalCount = resp.UnreadCount
	}

	queryBuilder := r.db.squirrel.
		Select("id", "event", "title", "message", "release_name", "indexer", "filter_name", "filter_id", "action", "action_client", "rejections", "url", "read_at", "created_at").
		From("notification_inbox").
		OrderBy("id DESC").
		Limit(params.Limit).
		Offset(params.Offset)

	if params.Unread {
		queryBuilder = queryBuilder.Where(sq.Eq{"read_at": nil})
	}
	if len(params.Events) > 0 {
		queryBuilder = queryBuilder.Where(sq.Eq{"event": params.Events})
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}
	defer rows.Close()

	for rows.Next() {
		var msg domain.InboxMessage
		var releaseName, indexer, filterName, action, actionClient, url sql.Null[string]
		var filterID sql.Null[int64]
		var readAt sql.Null[time.Time]

		if err := rows.Scan(&msg.ID, &msg.Event, &msg.Title, &msg.Message, &releaseName, &indexer, &filterName, &filterID, &action, &actionClient, pq.Array(&msg.Rejections), &url, &readAt, &msg.CreatedAt); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		msg.ReleaseName = releaseName.V
		msg.Indexer = indexer.V
		msg.FilterName = filterName.V
		msg.FilterID = int(filterID.V)
		msg.Action = action.V
		msg.ActionClient = actionClient.V
		msg.URL = url.V

		if msg.Rejections == nil {
			msg.Rejections = []string{}
		}

		if readAt.Valid {
			msg.ReadAt = &readAt.V
		}

		resp.Data = append(resp.Data, &msg)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "row error")
	}

	return resp, nil
}

func (r *NotificationInboxRepo) Store(ctx context.Context, msg *domain.InboxMessage) error {
	if msg.Rejections == nil {
		msg.Rejections = []string{}
	}

	queryBuilder := r.db.squirrel.
		Insert("notification_inbox").
		Columns("event", "title", "message", "release_name", "indexer", "filter_name", "filter_id", "action", "action_client", "rejections", "url", "created_at").
		Values(msg.Event, msg.Title, msg.Message, toNullString(msg.ReleaseName), toNullString(msg.Indexer), toNullString(msg.FilterName), toNullInt32(int32(msg.FilterID)), toNullString(msg.Action), toNullString(msg.ActionClient), pq.Array(msg.Rejections), toNullString(msg.URL), msg.CreatedAt).
		Suffix("RETURNING id").RunWith(r.db.Handler)

	if err := queryBuilder.QueryRowContext(ctx).Scan(&msg.ID); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

// MarkRead marks the messages in messageIDs as read, or every unread message when messageIDs is empty.
func (r *NotificationInboxRepo) MarkRead(ctx context.Context, messageIDs []int64, readAt time.Time) error {
	queryBuilder := r.db.squirrel.
		Update("notification_inbox").
		Set("read_at", readAt).
		Where(sq.Eq{"read_at": nil})

	if len(messageIDs) > 0 {
		queryBuilder = queryBuilder.Where(sq.Eq{"id": messageIDs})
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err := r.db.Handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *NotificationInboxRepo) Delete(ctx context.Context, messageIDs []int64) error {
	query, args, err := r.db.squirrel.Delete("notification_inbox").Where(sq.Eq{"id": messageIDs}).ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err := r.db.Handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *NotificationInboxRepo) DeleteAll(ctx context.Context) error {
	query, args, err := r.db.squirrel.Delete("notification_inbox").ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err := r.db.Handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

// Cleanup deletes messages older than params.OlderThan and everything beyond the newest params.MaxMessages.
func (r *NotificationInboxRepo) Cleanup(ctx context.Context, params domain.InboxCleanupParams) (int64, error) {
	// The subquery is built without the driver placeholder format so the outer query numbers every placeholder.
	keepQuery, keepArgs, err := sq.
		Select("id").
		From("notification_inbox").
		OrderBy("id DESC").
		Limit(uint64(params.MaxMessages)).
		ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "error building query")
	}

	queryBuilder := r.db.squirrel.
		Delete("notification_inbox").
		Where(sq.Or{
			sq.Lt{"created_at": params.OlderThan},
			sq.Expr("id NOT IN ("+keepQuery+")", keepArgs...),
		})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "error executing query")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "error getting rows affected")
	}

	return rowsAffected, nil
}
