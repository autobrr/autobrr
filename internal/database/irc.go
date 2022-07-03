package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/pkg/errors"
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
		Select("id", "enabled", "name", "server", "port", "tls", "pass", "invite_command", "nickserv_account", "nickserv_password").
		From("irc_network").
		Where("id = ?", id)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.getNetworkByID: error building query")
		return nil, err
	}
	r.log.Trace().Str("database", "irc.check_existing_network").Msgf("query: '%v', args: '%v'", query, args)

	var n domain.IrcNetwork

	var pass, inviteCmd sql.NullString
	var nsAccount, nsPassword sql.NullString
	var tls sql.NullBool

	row := r.db.handler.QueryRowContext(ctx, query, args...)
	if err := row.Scan(&n.ID, &n.Enabled, &n.Name, &n.Server, &n.Port, &tls, &pass, &inviteCmd, &nsAccount, &nsPassword); err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.getNetworkByID: error executing query")
		return nil, err
	}

	n.TLS = tls.Bool
	n.Pass = pass.String
	n.InviteCommand = inviteCmd.String
	n.NickServ.Account = nsAccount.String
	n.NickServ.Password = nsPassword.String

	return &n, nil
}

func (r *IrcRepo) DeleteNetwork(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	queryBuilder := r.db.squirrel.
		Delete("irc_channel").
		Where("network_id = ?", id)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.deleteNetwork: error building query")
		return err
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.deleteNetwork: error executing query")
		return err
	}

	netQueryBuilder := r.db.squirrel.
		Delete("irc_network").
		Where("id = ?", id)

	netQuery, netArgs, err := netQueryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.deleteNetwork: error building query")
		return err
	}

	_, err = tx.ExecContext(ctx, netQuery, netArgs...)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.deleteNetwork: error executing query")
		return err
	}

	err = tx.Commit()
	if err != nil {
		r.log.Error().Stack().Err(err).Msgf("irc.deleteNetwork: error deleting network %v", id)
		return err

	}

	return nil
}

func (r *IrcRepo) FindActiveNetworks(ctx context.Context) ([]domain.IrcNetwork, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "server", "port", "tls", "pass", "invite_command", "nickserv_account", "nickserv_password").
		From("irc_network").
		Where("enabled = ?", true)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.findActiveNetworks: error building query")
		return nil, err
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.findActiveNetworks: error executing query")
		return nil, err
	}

	defer rows.Close()

	var networks []domain.IrcNetwork
	for rows.Next() {
		var net domain.IrcNetwork

		var pass, inviteCmd sql.NullString
		var nsAccount, nsPassword sql.NullString
		var tls sql.NullBool

		if err := rows.Scan(&net.ID, &net.Enabled, &net.Name, &net.Server, &net.Port, &tls, &pass, &inviteCmd, &nsAccount, &nsPassword); err != nil {
			r.log.Error().Stack().Err(err).Msg("irc.findActiveNetworks: error scanning row")
			return nil, err
		}

		net.TLS = tls.Bool
		net.Pass = pass.String
		net.InviteCommand = inviteCmd.String

		net.NickServ.Account = nsAccount.String
		net.NickServ.Password = nsPassword.String

		networks = append(networks, net)
	}
	if err := rows.Err(); err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.findActiveNetworks: row error")
		return nil, err
	}

	return networks, nil
}

func (r *IrcRepo) ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "server", "port", "tls", "pass", "invite_command", "nickserv_account", "nickserv_password").
		From("irc_network").
		OrderBy("name ASC")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.listNetworks: error building query")
		return nil, err
	}

	rows, err := r.db.handler.QueryContext(ctx, query, args...)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.listNetworks: error executing query")
		return nil, err
	}

	defer rows.Close()

	var networks []domain.IrcNetwork
	for rows.Next() {
		var net domain.IrcNetwork

		var pass, inviteCmd sql.NullString
		var nsAccount, nsPassword sql.NullString
		var tls sql.NullBool

		if err := rows.Scan(&net.ID, &net.Enabled, &net.Name, &net.Server, &net.Port, &tls, &pass, &inviteCmd, &nsAccount, &nsPassword); err != nil {
			r.log.Error().Stack().Err(err).Msg("irc.listNetworks: error scanning row")
			return nil, err
		}

		net.TLS = tls.Bool
		net.Pass = pass.String
		net.InviteCommand = inviteCmd.String

		net.NickServ.Account = nsAccount.String
		net.NickServ.Password = nsPassword.String

		networks = append(networks, net)
	}
	if err := rows.Err(); err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.listNetworks: row error")
		return nil, err
	}

	return networks, nil
}

func (r *IrcRepo) ListChannels(networkID int64) ([]domain.IrcChannel, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "name", "enabled", "password").
		From("irc_channel").
		Where("network_id = ?", networkID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.listChannels: error building query")
		return nil, err
	}

	rows, err := r.db.handler.Query(query, args...)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.listChannels: error executing query")
		return nil, err
	}
	defer rows.Close()

	var channels []domain.IrcChannel
	for rows.Next() {
		var ch domain.IrcChannel
		var pass sql.NullString

		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Enabled, &pass); err != nil {
			r.log.Error().Stack().Err(err).Msg("irc.listChannels: error scanning row")
			return nil, err
		}

		ch.Password = pass.String

		channels = append(channels, ch)
	}
	if err := rows.Err(); err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.listChannels: error row")
		return nil, err
	}

	return channels, nil
}

func (r *IrcRepo) CheckExistingNetwork(ctx context.Context, network *domain.IrcNetwork) (*domain.IrcNetwork, error) {
	queryBuilder := r.db.squirrel.
		Select("id", "enabled", "name", "server", "port", "tls", "pass", "invite_command", "nickserv_account", "nickserv_password").
		From("irc_network").
		Where("server = ?", network.Server).
		Where("nickserv_account = ?", network.NickServ.Account)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.checkExistingNetwork: error building query")
		return nil, err
	}
	r.log.Trace().Str("database", "irc.checkExistingNetwork").Msgf("query: '%v', args: '%v'", query, args)

	row := r.db.handler.QueryRowContext(ctx, query, args...)

	var net domain.IrcNetwork

	var pass, inviteCmd, nickPass sql.NullString
	var tls sql.NullBool

	err = row.Scan(&net.ID, &net.Enabled, &net.Name, &net.Server, &net.Port, &tls, &pass, &inviteCmd, &net.NickServ.Account, &nickPass)
	if err == sql.ErrNoRows {
		// no result is not an error in our case
		return nil, nil
	} else if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.checkExistingNetwork: error scanning data to struct")
		return nil, err
	}

	net.TLS = tls.Bool
	net.Pass = pass.String
	net.InviteCommand = inviteCmd.String
	net.NickServ.Password = nickPass.String

	return &net, nil
}

func (r *IrcRepo) StoreNetwork(network *domain.IrcNetwork) error {
	netName := toNullString(network.Name)
	pass := toNullString(network.Pass)
	inviteCmd := toNullString(network.InviteCommand)

	nsAccount := toNullString(network.NickServ.Account)
	nsPassword := toNullString(network.NickServ.Password)

	var err error
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
			"invite_command",
			"nickserv_account",
			"nickserv_password",
		).
		Values(
			network.Enabled,
			netName,
			network.Server,
			network.Port,
			network.TLS,
			pass,
			inviteCmd,
			nsAccount,
			nsPassword,
		).
		Suffix("RETURNING id").
		RunWith(r.db.handler)

	err = queryBuilder.QueryRow().Scan(&retID)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.storeNetwork: error executing query")
		return errors.Wrap(err, "error executing query")
	}

	network.ID = retID

	return err
}

func (r *IrcRepo) UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error {
	netName := toNullString(network.Name)
	pass := toNullString(network.Pass)
	inviteCmd := toNullString(network.InviteCommand)

	nsAccount := toNullString(network.NickServ.Account)
	nsPassword := toNullString(network.NickServ.Password)

	var err error

	queryBuilder := r.db.squirrel.
		Update("irc_network").
		Set("enabled", network.Enabled).
		Set("name", netName).
		Set("server", network.Server).
		Set("port", network.Port).
		Set("tls", network.TLS).
		Set("pass", pass).
		Set("invite_command", inviteCmd).
		Set("nickserv_account", nsAccount).
		Set("nickserv_password", nsPassword).
		Set("updated_at", time.Now().Format(time.RFC3339)).
		Where("id = ?", network.ID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.updateNetwork: error building query")
		return err
	}

	// update record
	_, err = r.db.handler.ExecContext(ctx, query, args...)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.updateNetwork: error executing query")
		return err
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
		Where("network_id = ?", networkID)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.storeNetworkChannels: error building query")
		return err
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.storeNetworkChannels: error executing query")
		return err
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

		err = channelQueryBuilder.QueryRowContext(ctx).Scan(&retID)
		if err != nil {
			r.log.Error().Stack().Err(err).Msg("irc.storeNetworkChannels: error executing query")
			return errors.Wrap(err, "error executing query")
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

	err = tx.Commit()
	if err != nil {
		r.log.Error().Stack().Err(err).Msgf("irc.storeNetworkChannels: error deleting network: %v", networkID)
		return err
	}

	return nil
}

func (r *IrcRepo) StoreChannel(networkID int64, channel *domain.IrcChannel) error {
	pass := toNullString(channel.Password)

	var err error
	if channel.ID != 0 {
		// update record
		channelQueryBuilder := r.db.squirrel.
			Update("irc_channel").
			Set("enabled", channel.Enabled).
			Set("detached", channel.Detached).
			Set("name", channel.Name).
			Set("pass", pass).
			Where("id = ?", channel.ID)

		query, args, err := channelQueryBuilder.ToSql()
		if err != nil {
			r.log.Error().Stack().Err(err).Msg("irc.storeChannel: error building query")
			return err
		}

		_, err = r.db.handler.Exec(query, args...)
		if err != nil {
			r.log.Error().Stack().Err(err).Msg("irc.storeChannel: error executing query")
			return err
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

		err = queryBuilder.QueryRow().Scan(&retID)
		if err != nil {
			r.log.Error().Stack().Err(err).Msg("irc.storeChannels: error executing query")
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

	return err
}

func (r *IrcRepo) UpdateChannel(channel *domain.IrcChannel) error {
	pass := toNullString(channel.Password)

	// update record
	channelQueryBuilder := r.db.squirrel.
		Update("irc_channel").
		Set("enabled", channel.Enabled).
		Set("detached", channel.Detached).
		Set("name", channel.Name).
		Set("pass", pass).
		Where("id = ?", channel.ID)

	query, args, err := channelQueryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.updateChannel: error building query")
		return err
	}

	_, err = r.db.handler.Exec(query, args...)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.updateChannel: error executing query")
		return err
	}

	return err
}

func (r *IrcRepo) UpdateInviteCommand(networkID int64, invite string) error {

	// update record
	channelQueryBuilder := r.db.squirrel.
		Update("irc_network").
		Set("invite_command", invite).
		Where("id = ?", networkID)

	query, args, err := channelQueryBuilder.ToSql()
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.UpdateInviteCommand: error building query")
		return err
	}

	_, err = r.db.handler.Exec(query, args...)
	if err != nil {
		r.log.Error().Stack().Err(err).Msg("irc.UpdateInviteCommand: error executing query")
		return err
	}

	return err
}
