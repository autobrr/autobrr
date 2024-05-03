// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog"
)

type IrcRepo struct {
	log zerolog.Logger
	db  *DB
}

func NewIrcRepo(log logger.Logger, db *DB) domain.IrcRepo {
	return &IrcRepo{
		log: log.With().Str("repo", "irc").Logger(),
		db:  db,
	}
}

func (r *IrcRepo) GetNetworkByID(ctx context.Context, id int64) (*domain.IrcNetwork, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "server", "port", "tls", "pass", "nick", "auth_mechanism", "auth_account", "auth_password", "invite_command", "bouncer_addr", "use_bouncer", "bot_mode").
		From("irc_network").
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}
	r.log.Trace().Str("database", "irc.check_existing_network").Msgf("query: '%s', args: '%v'", query, args)

	var n domain.IrcNetwork

	var pass, nick, inviteCmd, bouncerAddr sql.NullString
	var account, password sql.NullString
	var tls sql.NullBool

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Scan(&n.ID, &n.Enabled, &n.Name, &n.Server, &n.Port, &tls, &pass, &nick, &n.Auth.Mechanism, &account, &password, &inviteCmd, &bouncerAddr, &n.UseBouncer, &n.BotMode); err != nil {
		return nil, errors.Wrap(err, "error scanning row")
	}

	n.TLS = tls.Bool
	n.Pass = pass.String
	n.Nick = nick.String
	n.InviteCommand = inviteCmd.String
	n.Auth.Account = account.String
	n.Auth.Password = password.String
	n.BouncerAddr = bouncerAddr.String

	return &n, nil
}

func (r *IrcRepo) DeleteNetwork(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "error begin transaction")
	}

	defer tx.Rollback()

	queryBuilder := r.db.squirrel.
		Delete("irc_channel").
		Where(sq.Eq{"network_id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	netQueryBuilder := r.db.squirrel.
		Delete("irc_network").
		Where(sq.Eq{"id": id})

	netQuery, netArgs, err := netQueryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = tx.ExecContext(ctx, netQuery, netArgs...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "error commit deleting network")
	}

	return nil
}

func (r *IrcRepo) FindActiveNetworks(ctx context.Context) ([]domain.IrcNetwork, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "server", "port", "tls", "pass", "nick", "auth_mechanism", "auth_account", "auth_password", "invite_command", "bouncer_addr", "use_bouncer", "bot_mode").
		From("irc_network").
		Where(sq.Eq{"enabled": true})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	var networks []domain.IrcNetwork
	for rows.Next() {
		var net domain.IrcNetwork

		var pass, nick, inviteCmd, bouncerAddr sql.NullString
		var account, password sql.NullString
		var tls sql.NullBool

		if err := rows.Scan(&net.ID, &net.Enabled, &net.Name, &net.Server, &net.Port, &tls, &pass, &nick, &net.Auth.Mechanism, &account, &password, &inviteCmd, &bouncerAddr, &net.UseBouncer, &net.BotMode); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		net.TLS = tls.Bool
		net.Pass = pass.String
		net.Nick = nick.String
		net.InviteCommand = inviteCmd.String
		net.BouncerAddr = bouncerAddr.String

		net.Auth.Account = account.String
		net.Auth.Password = password.String

		networks = append(networks, net)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error row")
	}

	return networks, nil
}

func (r *IrcRepo) ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "server", "port", "tls", "pass", "nick", "auth_mechanism", "auth_account", "auth_password", "invite_command", "bouncer_addr", "use_bouncer", "bot_mode").
		From("irc_network").
		OrderBy("name ASC")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	var networks []domain.IrcNetwork
	for rows.Next() {
		var net domain.IrcNetwork

		var pass, nick, inviteCmd, bouncerAddr sql.NullString
		var account, password sql.NullString
		var tls sql.NullBool

		if err := rows.Scan(&net.ID, &net.Enabled, &net.Name, &net.Server, &net.Port, &tls, &pass, &nick, &net.Auth.Mechanism, &account, &password, &inviteCmd, &bouncerAddr, &net.UseBouncer, &net.BotMode); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		net.TLS = tls.Bool
		net.Pass = pass.String
		net.Nick = nick.String
		net.InviteCommand = inviteCmd.String
		net.BouncerAddr = bouncerAddr.String

		net.Auth.Account = account.String
		net.Auth.Password = password.String

		networks = append(networks, net)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error row")
	}

	return networks, nil
}

func (r *IrcRepo) ListChannels(networkID int64) ([]domain.IrcChannel, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "name", "enabled", "password").
		From("irc_channel").
		Where(sq.Eq{"network_id": networkID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.handler.Query(query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}
	defer rows.Close()

	var channels []domain.IrcChannel
	for rows.Next() {
		var ch domain.IrcChannel
		var pass sql.NullString

		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Enabled, &pass); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		ch.Password = pass.String

		channels = append(channels, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error row")
	}

	return channels, nil
}

func (r *IrcRepo) CheckExistingNetwork(ctx context.Context, network *domain.IrcNetwork) (*domain.IrcNetwork, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "server", "port", "tls", "pass", "nick", "auth_mechanism", "auth_account", "auth_password", "invite_command", "bouncer_addr", "use_bouncer", "bot_mode").
		From("irc_network").
		Where(sq.Eq{"server": network.Server}).
		Where(sq.Eq{"port": network.Port}).
		Where(sq.Eq{"nick": network.Nick})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}
	r.log.Trace().Str("database", "irc.checkExistingNetwork").Msgf("query: '%s', args: '%v'", query, args)

	row := r.db.handler.QueryRowContext(ctx, query, args...)

	var net domain.IrcNetwork

	var pass, nick, inviteCmd, bouncerAddr sql.NullString
	var account, password sql.NullString
	var tls sql.NullBool

	if err = row.Scan(&net.ID, &net.Enabled, &net.Name, &net.Server, &net.Port, &tls, &pass, &nick, &net.Auth.Mechanism, &account, &password, &inviteCmd, &bouncerAddr, &net.UseBouncer, &net.BotMode); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no result is not an error in our case
			return nil, nil
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	net.TLS = tls.Bool
	net.Pass = pass.String
	net.Nick = nick.String
	net.InviteCommand = inviteCmd.String
	net.BouncerAddr = bouncerAddr.String
	net.Auth.Account = account.String
	net.Auth.Password = password.String

	return &net, nil
}

func (r *IrcRepo) StoreNetwork(ctx context.Context, network *domain.IrcNetwork) error {
	netName := toNullString(network.Name)
	pass := toNullString(network.Pass)
	nick := toNullString(network.Nick)
	inviteCmd := toNullString(network.InviteCommand)
	bouncerAddr := toNullString(network.BouncerAddr)

	account := toNullString(network.Auth.Account)
	password := toNullString(network.Auth.Password)

	var retID int64

	queryBuilder := r.db.squirrel.
		Insert("irc_network").
		Columns(
			"enabled",
			"name",
			"server",
			"port",
			"tls",
			"pass",
			"nick",
			"auth_mechanism",
			"auth_account",
			"auth_password",
			"invite_command",
			"bouncer_addr",
			"use_bouncer",
			"bot_mode",
		).
		Values(
			network.Enabled,
			netName,
			network.Server,
			network.Port,
			network.TLS,
			pass,
			nick,
			network.Auth.Mechanism,
			account,
			password,
			inviteCmd,
			bouncerAddr,
			network.UseBouncer,
			network.BotMode,
		).
		Suffix("RETURNING id").
		RunWith(r.db.handler)

	if err := queryBuilder.QueryRowContext(ctx).Scan(&retID); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	network.ID = retID

	return nil
}

func (r *IrcRepo) UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error {
	netName := toNullString(network.Name)
	pass := toNullString(network.Pass)
	nick := toNullString(network.Nick)
	inviteCmd := toNullString(network.InviteCommand)
	bouncerAddr := toNullString(network.BouncerAddr)

	account := toNullString(network.Auth.Account)
	password := toNullString(network.Auth.Password)

	var err error

	queryBuilder := r.db.squirrel.
		Update("irc_network").
		Set("enabled", network.Enabled).
		Set("name", netName).
		Set("server", network.Server).
		Set("port", network.Port).
		Set("tls", network.TLS).
		Set("pass", pass).
		Set("nick", nick).
		Set("auth_mechanism", network.Auth.Mechanism).
		Set("auth_account", account).
		Set("auth_password", password).
		Set("invite_command", inviteCmd).
		Set("bouncer_addr", bouncerAddr).
		Set("use_bouncer", network.UseBouncer).
		Set("bot_mode", network.BotMode).
		Set("updated_at", time.Now().Format(time.RFC3339)).
		Where(sq.Eq{"id": network.ID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	// update record
	if _, err = r.db.handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return err
}

// TODO create new channel handler to only add, not delete

func (r *IrcRepo) StoreNetworkChannels(ctx context.Context, networkID int64, channels []domain.IrcChannel) error {
	tx, err := r.db.handler.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	queryBuilder := r.db.squirrel.
		Delete("irc_channel").
		Where(sq.Eq{"network_id": networkID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	for _, channel := range channels {
		// values
		pass := toNullString(channel.Password)

		channelQueryBuilder := r.db.squirrel.
			Insert("irc_channel").
			Columns(
				"enabled",
				"detached",
				"name",
				"password",
				"network_id",
			).
			Values(
				channel.Enabled,
				true,
				channel.Name,
				pass,
				networkID,
			).
			Suffix("RETURNING id").
			RunWith(tx)

		// returning
		var retID int64

		if err = channelQueryBuilder.QueryRowContext(ctx).Scan(&retID); err != nil {
			return errors.Wrap(err, "error executing query storeNetworkChannels")
		}

		channel.ID = retID

		//channelQuery, channelArgs, err := channelQueryBuilder.ToSql()
		//if err != nil {
		//	r.log.Error().Stack().Err(err).Msg("irc.storeNetworkChannels: error building query")
		//	return err
		//}
		//
		//res, err = r.db.handler.ExecContext(ctx, channelQuery, channelArgs...)
		//if err != nil {
		//	r.log.Error().Stack().Err(err).Msg("irc.storeNetworkChannels: error executing query")
		//	return err
		//}
		//
		//channel.ID, err = res.LastInsertId()
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "error commit transaction store network")
	}

	return nil
}

func (r *IrcRepo) StoreChannel(ctx context.Context, networkID int64, channel *domain.IrcChannel) error {
	pass := toNullString(channel.Password)

	if channel.ID != 0 {
		// update record
		channelQueryBuilder := r.db.squirrel.
			Update("irc_channel").
			Set("enabled", channel.Enabled).
			Set("detached", channel.Detached).
			Set("name", channel.Name).
			Set("password", pass).
			Where(sq.Eq{"id": channel.ID})

		query, args, err := channelQueryBuilder.ToSql()
		if err != nil {
			return errors.Wrap(err, "error building query")
		}

		if _, err := r.db.handler.ExecContext(ctx, query, args...); err != nil {
			return errors.Wrap(err, "error executing query")
		}
	} else {
		queryBuilder := r.db.squirrel.
			Insert("irc_channel").
			Columns(
				"enabled",
				"detached",
				"name",
				"password",
				"network_id",
			).
			Values(
				channel.Enabled,
				true,
				channel.Name,
				pass,
				networkID,
			).
			Suffix("RETURNING id").
			RunWith(r.db.handler)

		// returning
		var retID int64

		if err := queryBuilder.QueryRowContext(ctx).Scan(&retID); err != nil {
			return errors.Wrap(err, "error executing query")
		}

		channel.ID = retID

		//channelQuery, channelArgs, err := channelQueryBuilder.ToSql()
		//if err != nil {
		//	r.log.Error().Stack().Err(err).Msg("irc.storeChannel: error building query")
		//	return err
		//}
		//
		//res, err := r.db.handler.Exec(channelQuery, channelArgs...)
		//if err != nil {
		//	r.log.Error().Stack().Err(err).Msg("irc.storeChannel: error executing query")
		//	return errors.Wrap(err, "error executing query")
		//	//return err
		//}
		//
		//channel.ID, err = res.LastInsertId()
	}

	return nil
}

func (r *IrcRepo) UpdateChannel(channel *domain.IrcChannel) error {
	pass := toNullString(channel.Password)

	// update record
	channelQueryBuilder := r.db.squirrel.
		Update("irc_channel").
		Set("enabled", channel.Enabled).
		Set("detached", channel.Detached).
		Set("name", channel.Name).
		Set("password", pass).
		Where(sq.Eq{"id": channel.ID})

	query, args, err := channelQueryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return err
}

func (r *IrcRepo) UpdateInviteCommand(networkID int64, invite string) error {

	// update record
	channelQueryBuilder := r.db.squirrel.
		Update("irc_network").
		Set("invite_command", invite).
		Where(sq.Eq{"id": networkID})

	query, args, err := channelQueryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.handler.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return err
}
