package database

import (
	"context"
	"database/sql"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog/log"
)

type IrcRepo struct {
	db *sql.DB
}

func NewIrcRepo(db *sql.DB) domain.IrcRepo {
	return &IrcRepo{db: db}
}

func (ir *IrcRepo) Store(announce domain.Announce) error {
	return nil
}

func (ir *IrcRepo) GetNetworkByID(id int64) (*domain.IrcNetwork, error) {

	row := ir.db.QueryRow("SELECT id, enabled, name, addr, tls, nick, pass, connect_commands, sasl_mechanism, sasl_plain_username, sasl_plain_password FROM irc_network WHERE id = ?", id)
	if err := row.Err(); err != nil {
		log.Fatal().Err(err)
		return nil, err
	}

	var n domain.IrcNetwork

	var pass, connectCommands sql.NullString
	var saslMechanism, saslPlainUsername, saslPlainPassword sql.NullString
	var tls sql.NullBool

	if err := row.Scan(&n.ID, &n.Enabled, &n.Name, &n.Addr, &tls, &n.Nick, &pass, &connectCommands, &saslMechanism, &saslPlainUsername, &saslPlainPassword); err != nil {
		log.Fatal().Err(err)
	}

	n.TLS = tls.Bool
	n.Pass = pass.String
	if connectCommands.Valid {
		n.ConnectCommands = strings.Split(connectCommands.String, "\r\n")
	}
	n.SASL.Mechanism = saslMechanism.String
	n.SASL.Plain.Username = saslPlainUsername.String
	n.SASL.Plain.Password = saslPlainPassword.String

	return &n, nil
}

func (ir *IrcRepo) DeleteNetwork(ctx context.Context, id int64) error {
	tx, err := ir.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `DELETE FROM irc_network WHERE id = ?`, id)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error deleting network: %v", id)
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM irc_channel WHERE network_id = ?`, id)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error deleting channels for network: %v", id)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error deleting network: %v", id)
		return err

	}

	return nil
}

func (ir *IrcRepo) ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error) {

	rows, err := ir.db.QueryContext(ctx, "SELECT id, enabled, name, addr, tls, nick, pass, connect_commands FROM irc_network")
	if err != nil {
		log.Fatal().Err(err)
	}

	defer rows.Close()

	var networks []domain.IrcNetwork
	for rows.Next() {
		var net domain.IrcNetwork

		//var username, realname, pass, connectCommands sql.NullString
		var pass, connectCommands sql.NullString
		var tls sql.NullBool

		if err := rows.Scan(&net.ID, &net.Enabled, &net.Name, &net.Addr, &tls, &net.Nick, &pass, &connectCommands); err != nil {
			log.Fatal().Err(err)
		}

		net.TLS = tls.Bool
		net.Pass = pass.String

		if connectCommands.Valid {
			net.ConnectCommands = strings.Split(connectCommands.String, "\r\n")
		}

		networks = append(networks, net)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return networks, nil
}

func (ir *IrcRepo) ListChannels(networkID int64) ([]domain.IrcChannel, error) {

	rows, err := ir.db.Query("SELECT id, name, enabled FROM irc_channel WHERE network_id = ?", networkID)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer rows.Close()

	var channels []domain.IrcChannel
	for rows.Next() {
		var ch domain.IrcChannel

		//if err := rows.Scan(&ch.ID, &ch.Name, &ch.Enabled, &ch.Pass, &ch.InviteCommand, &ch.InviteHTTPURL, &ch.InviteHTTPHeader, &ch.InviteHTTPData); err != nil {
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Enabled); err != nil {
			log.Fatal().Err(err)
		}

		channels = append(channels, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return channels, nil
}

func (ir *IrcRepo) StoreNetwork(network *domain.IrcNetwork) error {

	netName := toNullString(network.Name)
	pass := toNullString(network.Pass)
	connectCommands := toNullString(strings.Join(network.ConnectCommands, "\r\n"))

	var saslMechanism, saslPlainUsername, saslPlainPassword sql.NullString
	if network.SASL.Mechanism != "" {
		saslMechanism = toNullString(network.SASL.Mechanism)
		switch network.SASL.Mechanism {
		case "PLAIN":
			saslPlainUsername = toNullString(network.SASL.Plain.Username)
			saslPlainPassword = toNullString(network.SASL.Plain.Password)
		default:
			log.Warn().Msgf("unsupported SASL mechanism: %q", network.SASL.Mechanism)
			//return fmt.Errorf("cannot store network: unsupported SASL mechanism %q", network.SASL.Mechanism)
		}
	}

	var err error
	if network.ID != 0 {
		// update record
		_, err = ir.db.Exec(`UPDATE irc_network
			SET enabled = ?,
			    name = ?,
			    addr = ?,
			    tls = ?,
			    nick = ?,
			    pass = ?,
			    connect_commands = ?,
			    sasl_mechanism = ?,
			    sasl_plain_username = ?,
			    sasl_plain_password = ?,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			network.Enabled,
			netName,
			network.Addr,
			network.TLS,
			network.Nick,
			pass,
			connectCommands,
			saslMechanism,
			saslPlainUsername,
			saslPlainPassword,
			network.ID,
		)
	} else {
		var res sql.Result

		res, err = ir.db.Exec(`INSERT INTO irc_network (
                         enabled,
                         name,
                         addr,
                         tls,
                         nick,
                         pass,
                         connect_commands,
                         sasl_mechanism,
                         sasl_plain_username,
                         sasl_plain_password
                         ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			network.Enabled,
			netName,
			network.Addr,
			network.TLS,
			network.Nick,
			pass,
			connectCommands,
			saslMechanism,
			saslPlainUsername,
			saslPlainPassword,
		)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error executing query")
			return err
		}

		network.ID, err = res.LastInsertId()
	}

	return err
}

func (ir *IrcRepo) StoreChannel(networkID int64, channel *domain.IrcChannel) error {
	pass := toNullString(channel.Password)

	var err error
	if channel.ID != 0 {
		// update record
		_, err = ir.db.Exec(`UPDATE irc_channel
			SET 
			    enabled = ?,
				detached = ?,
				name = ?,
				password = ?
			WHERE 
			      id = ?`,
			channel.Enabled,
			channel.Detached,
			channel.Name,
			pass,
			channel.ID,
		)
	} else {
		var res sql.Result

		res, err = ir.db.Exec(`INSERT INTO irc_channel (
                         enabled,
                         detached,
                         name,
                         password,
                         network_id
                         ) VALUES (?, ?, ?, ?, ?)`,
			channel.Enabled,
			true,
			channel.Name,
			pass,
			networkID,
		)
		if err != nil {
			log.Error().Stack().Err(err).Msg("error executing query")
			return err
		}

		channel.ID, err = res.LastInsertId()
	}

	return err
}
