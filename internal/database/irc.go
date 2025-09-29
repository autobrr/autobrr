// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
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
		Select("id", "enabled", "name", "server", "port", "tls", "pass", "nick", "auth_mechanism", "auth_account", "auth_password", "invite_command", "bouncer_addr", "use_bouncer", "bot_mode", "use_proxy", "proxy_id").
		From("irc_network").
		Where(sq.Eq{"id": id})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}
	r.log.Trace().Str("database", "irc.check_existing_network").Msgf("query: '%s', args: '%v'", query, args)

	var n domain.IrcNetwork

	var pass, nick, inviteCmd, bouncerAddr sql.Null[string]
	var account, password sql.Null[string]
	var tls sql.Null[bool]
	var proxyId sql.Null[int64]

	row := r.db.Handler.QueryRowContext(ctx, query, args...)
	if err := row.Scan(&n.ID, &n.Enabled, &n.Name, &n.Server, &n.Port, &tls, &pass, &nick, &n.Auth.Mechanism, &account, &password, &inviteCmd, &bouncerAddr, &n.UseBouncer, &n.BotMode, &n.UseProxy, &proxyId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}
		return nil, errors.Wrap(err, "error scanning row")
	}

	n.TLS = tls.V
	n.Pass = pass.V
	n.Nick = nick.V
	n.InviteCommand = inviteCmd.V
	n.BouncerAddr = bouncerAddr.V
	n.Auth.Account = account.V
	n.Auth.Password = password.V
	n.ProxyId = proxyId.V

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
		Select("id", "enabled", "name", "server", "port", "tls", "pass", "nick", "auth_mechanism", "auth_account", "auth_password", "invite_command", "bouncer_addr", "use_bouncer", "bot_mode", "use_proxy", "proxy_id").
		From("irc_network").
		Where(sq.Eq{"enabled": true})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	var networks []domain.IrcNetwork
	for rows.Next() {
		var net domain.IrcNetwork

		var pass, nick, inviteCmd, bouncerAddr sql.Null[string]
		var account, password sql.Null[string]
		var tls sql.Null[bool]
		var proxyId sql.Null[int64]

		if err := rows.Scan(&net.ID, &net.Enabled, &net.Name, &net.Server, &net.Port, &tls, &pass, &nick, &net.Auth.Mechanism, &account, &password, &inviteCmd, &bouncerAddr, &net.UseBouncer, &net.BotMode, &net.UseProxy, &proxyId); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		net.TLS = tls.V
		net.Pass = pass.V
		net.Nick = nick.V
		net.InviteCommand = inviteCmd.V
		net.BouncerAddr = bouncerAddr.V
		net.Auth.Account = account.V
		net.Auth.Password = password.V

		net.ProxyId = proxyId.V

		networks = append(networks, net)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error row")
	}

	return networks, nil
}

func (r *IrcRepo) ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "server", "port", "tls", "pass", "nick", "auth_mechanism", "auth_account", "auth_password", "invite_command", "bouncer_addr", "use_bouncer", "bot_mode", "use_proxy", "proxy_id").
		From("irc_network").
		OrderBy("name ASC")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}

	rows, err := r.db.Handler.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	defer rows.Close()

	var networks []domain.IrcNetwork
	for rows.Next() {
		var net domain.IrcNetwork

		var pass, nick, inviteCmd, bouncerAddr sql.Null[string]
		var account, password sql.Null[string]
		var tls sql.Null[bool]
		var proxyId sql.Null[int64]

		if err := rows.Scan(&net.ID, &net.Enabled, &net.Name, &net.Server, &net.Port, &tls, &pass, &nick, &net.Auth.Mechanism, &account, &password, &inviteCmd, &bouncerAddr, &net.UseBouncer, &net.BotMode, &net.UseProxy, &proxyId); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		net.TLS = tls.V
		net.Pass = pass.V
		net.Nick = nick.V
		net.InviteCommand = inviteCmd.V
		net.BouncerAddr = bouncerAddr.V
		net.Auth.Account = account.V
		net.Auth.Password = password.V

		net.ProxyId = proxyId.V

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

	rows, err := r.db.Handler.Query(query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}
	defer rows.Close()

	var channels []domain.IrcChannel
	for rows.Next() {
		var ch domain.IrcChannel
		var pass sql.Null[string]

		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Enabled, &pass); err != nil {
			return nil, errors.Wrap(err, "error scanning row")
		}

		ch.Password = pass.V

		channels = append(channels, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error row")
	}

	return channels, nil
}

func (r *IrcRepo) CheckExistingNetwork(ctx context.Context, network *domain.IrcNetwork) (*domain.IrcNetwork, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "server", "port", "tls", "pass", "nick", "auth_mechanism", "auth_account", "auth_password", "invite_command", "bouncer_addr", "use_bouncer", "bot_mode", "use_proxy", "proxy_id").
		From("irc_network").
		Where(sq.Eq{"server": network.Server}).
		Where(sq.Eq{"port": network.Port}).
		Where(sq.Eq{"nick": network.Nick})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "error building query")
	}
	r.log.Trace().Str("database", "irc.checkExistingNetwork").Msgf("query: '%s', args: '%v'", query, args)

	row := r.db.Handler.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	var net domain.IrcNetwork

	var pass, nick, inviteCmd, bouncerAddr sql.Null[string]
	var account, password sql.Null[string]
	var tls sql.Null[bool]
	var proxyId sql.Null[int64]

	if err = row.Scan(&net.ID, &net.Enabled, &net.Name, &net.Server, &net.Port, &tls, &pass, &nick, &net.Auth.Mechanism, &account, &password, &inviteCmd, &bouncerAddr, &net.UseBouncer, &net.BotMode, &net.UseProxy, &proxyId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no result is not an error in our case
			return nil, nil
		}

		return nil, errors.Wrap(err, "error scanning row")
	}

	net.TLS = tls.V
	net.Pass = pass.V
	net.Nick = nick.V
	net.InviteCommand = inviteCmd.V
	net.BouncerAddr = bouncerAddr.V
	net.Auth.Account = account.V
	net.Auth.Password = password.V

	net.ProxyId = proxyId.V

	return &net, nil
}

func (r *IrcRepo) StoreNetwork(ctx context.Context, network *domain.IrcNetwork) error {
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
			toNullString(network.Name),
			network.Server,
			network.Port,
			network.TLS,
			toNullString(network.Pass),
			toNullString(network.Nick),
			network.Auth.Mechanism,
			toNullString(network.Auth.Account),
			toNullString(network.Auth.Password),
			toNullString(network.InviteCommand),
			toNullString(network.BouncerAddr),
			network.UseBouncer,
			network.BotMode,
		).
		Suffix("RETURNING id").
		RunWith(r.db.Handler)

	if err := queryBuilder.QueryRowContext(ctx).Scan(&network.ID); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return nil
}

func (r *IrcRepo) UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error {
	queryBuilder := r.db.squirrel.
		Update("irc_network").
		Set("enabled", network.Enabled).
		Set("name", toNullString(network.Name)).
		Set("server", network.Server).
		Set("port", network.Port).
		Set("tls", network.TLS).
		Set("pass", toNullString(network.Pass)).
		Set("nick", toNullString(network.Nick)).
		Set("auth_mechanism", network.Auth.Mechanism).
		Set("auth_account", toNullString(network.Auth.Account)).
		Set("auth_password", toNullString(network.Auth.Password)).
		Set("invite_command", toNullString(network.InviteCommand)).
		Set("bouncer_addr", toNullString(network.BouncerAddr)).
		Set("use_bouncer", network.UseBouncer).
		Set("bot_mode", network.BotMode).
		Set("use_proxy", network.UseProxy).
		Set("proxy_id", toNullInt64(network.ProxyId)).
		Set("updated_at", time.Now().Format(time.RFC3339)).
		Where(sq.Eq{"id": network.ID})

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	// update record
	if _, err = r.db.Handler.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return err
}

// TODO create new channel Handler to only add, not delete

func (r *IrcRepo) StoreNetworkChannels(ctx context.Context, networkID int64, channels []domain.IrcChannel) error {
	tx, err := r.db.Handler.BeginTx(ctx, nil)
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
				toNullString(channel.Password),
				networkID,
			).
			Suffix("RETURNING id").
			RunWith(tx)

		// returning
		if err = channelQueryBuilder.QueryRowContext(ctx).Scan(&channel.ID); err != nil {
			return errors.Wrap(err, "error executing query storeNetworkChannels")
		}

		//channelQuery, channelArgs, err := channelQueryBuilder.ToSql()
		//if err != nil {
		//	r.log.Error().Stack().Err(err).Msg("irc.storeNetworkChannels: error building query")
		//	return err
		//}
		//
		//res, err = r.db.Handler.ExecContext(ctx, channelQuery, channelArgs...)
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
	if channel.ID != 0 {
		// update record
		channelQueryBuilder := r.db.squirrel.
			Update("irc_channel").
			Set("enabled", channel.Enabled).
			Set("detached", channel.Detached).
			Set("name", channel.Name).
			Set("password", toNullString(channel.Password)).
			Where(sq.Eq{"id": channel.ID})

		query, args, err := channelQueryBuilder.ToSql()
		if err != nil {
			return errors.Wrap(err, "error building query")
		}

		if _, err := r.db.Handler.ExecContext(ctx, query, args...); err != nil {
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
				toNullString(channel.Password),
				networkID,
			).
			Suffix("RETURNING id").
			RunWith(r.db.Handler)

		// returning
		if err := queryBuilder.QueryRowContext(ctx).Scan(&channel.ID); err != nil {
			return errors.Wrap(err, "error executing query")
		}

		//channelQuery, channelArgs, err := channelQueryBuilder.ToSql()
		//if err != nil {
		//	r.log.Error().Stack().Err(err).Msg("irc.storeChannel: error building query")
		//	return err
		//}
		//
		//res, err := r.db.Handler.Exec(channelQuery, channelArgs...)
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
	// update record
	channelQueryBuilder := r.db.squirrel.
		Update("irc_channel").
		Set("enabled", channel.Enabled).
		Set("detached", channel.Detached).
		Set("name", channel.Name).
		Set("password", toNullString(channel.Password)).
		Where(sq.Eq{"id": channel.ID})

	query, args, err := channelQueryBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "error building query")
	}

	_, err = r.db.Handler.Exec(query, args...)
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

	_, err = r.db.Handler.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}

	return err
}
