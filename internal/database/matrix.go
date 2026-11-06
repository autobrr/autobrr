// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog"
)

type MatrixRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewMatrixRepo(log zerolog.Logger, db *DB) *MatrixRepo {
	return &MatrixRepo{
		log: log.With().Str("repo", "matrix").Logger(),
		db:  db,
	}
}

func (r *MatrixRepo) networkQuery() sq.SelectBuilder {
	return r.db.squirrel.
		Select(
			"id",
			"enabled",
			"name",
			"homeserver",
			"auth_method",
			"user_id",
			"device_id",
			"access_token",
			"refresh_token",
			"token_expires_at",
			"oauth_client_id",
			"oauth_token_endpoint",
		).
		From("matrix_network")
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMatrixNetwork(row rowScanner) (*domain.MatrixNetwork, error) {
	var network domain.MatrixNetwork

	var (
		authMethod, userID, deviceID, accessToken, refreshToken sql.Null[string]
		clientID, tokenEndpoint                                 sql.Null[string]
		tokenExpiresAt                                          sql.Null[time.Time]
	)

	if err := row.Scan(&network.ID, &network.Enabled, &network.Name, &network.Homeserver, &authMethod, &userID, &deviceID, &accessToken, &refreshToken, &tokenExpiresAt, &clientID, &tokenEndpoint); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	network.AuthMethod = domain.MatrixAuthMethod(authMethod.V)
	network.UserID = userID.V
	network.DeviceID = deviceID.V
	network.AccessToken = accessToken.V
	network.RefreshToken = refreshToken.V
	network.OAuthClientID = clientID.V
	network.OAuthTokenEndpoint = tokenEndpoint.V

	if tokenExpiresAt.Valid {
		network.TokenExpiresAt = &tokenExpiresAt.V
	}

	return &network, nil
}

func (r *MatrixRepo) findNetworks(ctx context.Context, queryBuilder sq.SelectBuilder) ([]domain.MatrixNetwork, error) {
	query, args, err := queryBuilder.OrderBy("name ASC").ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}
	defer rows.Close()

	networks := make([]domain.MatrixNetwork, 0)
	for rows.Next() {
		network, err := scanMatrixNetwork(rows)
		if err != nil {
			return nil, err
		}

		networks = append(networks, *network)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "row error")
	}

	rooms, err := r.listRooms(ctx, nil)
	if err != nil {
		return nil, err
	}

	for i := range networks {
		networks[i].Rooms = rooms[networks[i].ID]
		if networks[i].Rooms == nil {
			networks[i].Rooms = make([]domain.MatrixRoom, 0)
		}
	}

	return networks, nil
}

// ListNetworks returns every network with its rooms.
func (r *MatrixRepo) ListNetworks(ctx context.Context) ([]domain.MatrixNetwork, error) {
	return r.findNetworks(ctx, r.networkQuery())
}

// FindActiveNetworks returns the enabled networks with their rooms.
func (r *MatrixRepo) FindActiveNetworks(ctx context.Context) ([]domain.MatrixNetwork, error) {
	return r.findNetworks(ctx, r.networkQuery().Where(sq.Eq{"enabled": true}))
}

// FindNetworksByHomeserver returns the networks on a normalized homeserver URL.
func (r *MatrixRepo) FindNetworksByHomeserver(ctx context.Context, homeserver string) ([]domain.MatrixNetwork, error) {
	return r.findNetworks(ctx, r.networkQuery().Where(sq.Eq{"homeserver": homeserver}))
}

// GetNetworkByID returns a network with its rooms.
func (r *MatrixRepo) GetNetworkByID(ctx context.Context, id int64) (*domain.MatrixNetwork, error) {
	query, args, err := r.networkQuery().Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	row := r.db.Handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error row")
	}

	network, err := scanMatrixNetwork(row)
	if err != nil {
		return nil, err
	}

	rooms, err := r.listRooms(ctx, &id)
	if err != nil {
		return nil, err
	}

	network.Rooms = rooms[id]
	if network.Rooms == nil {
		network.Rooms = make([]domain.MatrixRoom, 0)
	}

	return network, nil
}

func (r *MatrixRepo) listRooms(ctx context.Context, networkID *int64) (map[int64][]domain.MatrixRoom, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "network_id", "enabled", "name", "room_id").
		From("matrix_room").
		OrderBy("name ASC")

	if networkID != nil {
		queryBuilder = queryBuilder.Where(sq.Eq{"network_id": *networkID})
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

	rooms := make(map[int64][]domain.MatrixRoom)
	for rows.Next() {
		var (
			room   domain.MatrixRoom
			netID  int64
			roomID sql.Null[string]
		)

		if err := rows.Scan(&room.ID, &netID, &room.Enabled, &room.Name, &roomID); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		room.RoomID = roomID.V

		rooms[netID] = append(rooms[netID], room)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "row error")
	}

	return rooms, nil
}

// StoreNetwork inserts a network and its rooms.
func (r *MatrixRepo) StoreNetwork(ctx context.Context, network *domain.MatrixNetwork) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "error begin transaction")
	}

	defer tx.Rollback()

	queryBuilder := r.db.squirrel.
		Insert("matrix_network").
		Columns("enabled", "name", "homeserver", "auth_method", "access_token").
		Values(network.Enabled, network.Name, network.Homeserver, toNullString(network.AuthMethod.String()), toNullString(network.AccessToken)).
		Suffix("RETURNING id").
		RunWith(tx)

	if err := queryBuilder.QueryRowContext(ctx).Scan(&network.ID); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if err := r.syncRooms(ctx, tx, network.ID, network.Rooms); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "error storing matrix network")
	}

	return nil
}

// UpdateNetwork updates the network settings and syncs its rooms. Credentials are
// written by UpdateTokens, except that a new homeserver drops them in the same
// transaction: they belong to an account on the old one.
func (r *MatrixRepo) UpdateNetwork(ctx context.Context, network *domain.MatrixNetwork) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "error begin transaction")
	}

	defer tx.Rollback()

	homeserverQuery, homeserverArgs, err := r.db.squirrel.Select("homeserver").From("matrix_network").Where(sq.Eq{"id": network.ID}).ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	var currentHomeserver string
	if err := tx.QueryRowContext(ctx, homeserverQuery, homeserverArgs...).Scan(&currentHomeserver); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrRecordNotFound
		}

		return errors.Wrap(err, "error scanning row")
	}

	queryBuilder := r.db.squirrel.
		Update("matrix_network").
		Set("enabled", network.Enabled).
		Set("name", network.Name).
		Set("homeserver", network.Homeserver).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": network.ID})

	if currentHomeserver != network.Homeserver {
		for _, column := range []string{"auth_method", "user_id", "device_id", "access_token", "refresh_token", "token_expires_at", "oauth_client_id", "oauth_token_endpoint"} {
			queryBuilder = queryBuilder.Set(column, nil)
		}
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return errors.Wrap(err, "error getting rows affected")
	} else if rowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	if err := r.syncRooms(ctx, tx, network.ID, network.Rooms); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "error updating matrix network")
	}

	return nil
}

// syncRooms makes the stored rooms match rooms by name: missing rooms are inserted,
// existing ones keep their resolved room id, and the rest are deleted.
func (r *MatrixRepo) syncRooms(ctx context.Context, tx *Tx, networkID int64, rooms []domain.MatrixRoom) error {
	names := make([]string, 0, len(rooms))
	for _, room := range rooms {
		names = append(names, room.Name)
	}

	deleteQueryBuilder := r.db.squirrel.
		Delete("matrix_room").
		Where(sq.Eq{"network_id": networkID})

	if len(names) > 0 {
		deleteQueryBuilder = deleteQueryBuilder.Where(sq.NotEq{"name": names})
	}

	deleteQuery, deleteArgs, err := deleteQueryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err := tx.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	existingQuery, existingArgs, err := r.db.squirrel.Select("name").From("matrix_room").Where(sq.Eq{"network_id": networkID}).ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	rows, err := tx.QueryContext(ctx, existingQuery, existingArgs...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	var existing []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return errors.Wrap(err, "error scanning row")
		}

		existing = append(existing, name)
	}

	rows.Close()

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "row error")
	}

	for _, room := range rooms {
		if slices.Contains(existing, room.Name) {
			updateQuery, updateArgs, err := r.db.squirrel.
				Update("matrix_room").
				Set("enabled", room.Enabled).
				Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
				Where(sq.Eq{"network_id": networkID, "name": room.Name}).
				ToSql()
			if err != nil {
				return errors.Wrap(err, "error building query")
			}

			if _, err := tx.ExecContext(ctx, updateQuery, updateArgs...); err != nil {
				return errors.Wrap(err, "error executing query")
			}

			continue
		}

		insertQuery, insertArgs, err := r.db.squirrel.
			Insert("matrix_room").
			Columns("network_id", "enabled", "name", "room_id").
			Values(networkID, room.Enabled, room.Name, toNullString(room.RoomID)).
			ToSql()
		if err != nil {
			return errors.Wrap(err, "error building query")
		}

		if _, err := tx.ExecContext(ctx, insertQuery, insertArgs...); err != nil {
			return errors.Wrap(err, "error executing query")
		}
	}

	return nil
}

// UpdateTokens writes a login or token refresh in one statement, so a rotated
// single-use refresh token is never stored without its access token.
func (r *MatrixRepo) UpdateTokens(ctx context.Context, networkID int64, tokens domain.MatrixTokens) error {
	var expiresAt sql.Null[time.Time]
	if tokens.TokenExpiresAt != nil {
		expiresAt = sql.Null[time.Time]{V: *tokens.TokenExpiresAt, Valid: true}
	}

	queryBuilder := r.db.squirrel.
		Update("matrix_network").
		Set("auth_method", toNullString(tokens.AuthMethod.String())).
		Set("user_id", toNullString(tokens.UserID)).
		Set("device_id", toNullString(tokens.DeviceID)).
		Set("access_token", toNullString(tokens.AccessToken)).
		Set("refresh_token", toNullString(tokens.RefreshToken)).
		Set("token_expires_at", expiresAt).
		Set("oauth_client_id", toNullString(tokens.OAuthClientID)).
		Set("oauth_token_endpoint", toNullString(tokens.OAuthTokenEndpoint)).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": networkID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return errors.Wrap(err, "error getting rows affected")
	} else if rowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	return nil
}

// UpdateRoomID stores the room id a room alias resolved to.
func (r *MatrixRepo) UpdateRoomID(ctx context.Context, id int64, roomID string) error {
	query, args, err := r.db.squirrel.
		Update("matrix_room").
		Set("room_id", toNullString(roomID)).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err := r.db.Handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

// ToggleNetworkEnabled enables or disables a network.
func (r *MatrixRepo) ToggleNetworkEnabled(ctx context.Context, id int64, enabled bool) error {
	query, args, err := r.db.squirrel.
		Update("matrix_network").
		Set("enabled", enabled).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := r.db.Handler.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return errors.Wrap(err, "error getting rows affected")
	} else if rowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	return nil
}

// DeleteNetwork deletes a network and its rooms. Rooms are deleted explicitly
// because SQLite does not enforce the foreign key cascade.
func (r *MatrixRepo) DeleteNetwork(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "error begin transaction")
	}

	defer tx.Rollback()

	roomQuery, roomArgs, err := r.db.squirrel.Delete("matrix_room").Where(sq.Eq{"network_id": id}).ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err := tx.ExecContext(ctx, roomQuery, roomArgs...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	query, args, err := r.db.squirrel.Delete("matrix_network").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return errors.Wrap(err, "error getting rows affected")
	} else if rowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "error deleting matrix network")
	}

	return nil
}
