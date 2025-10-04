ALTER TABLE irc_network
    RENAME COLUMN nickserv_account TO auth_account;

ALTER TABLE irc_network
    RENAME COLUMN nickserv_password TO auth_password;

ALTER TABLE irc_network
    ADD nick TEXT;

ALTER TABLE irc_network
    ADD auth_mechanism TEXT DEFAULT 'SASL_PLAIN';

ALTER TABLE irc_network
    DROP CONSTRAINT irc_network_server_port_nickserv_account_key;

ALTER TABLE irc_network
    ADD CONSTRAINT irc_network_server_port_nick_key
        UNIQUE (server, port, nick);

UPDATE irc_network
SET nick = irc_network.auth_account;

UPDATE irc_network
SET auth_mechanism = 'SASL_PLAIN';
