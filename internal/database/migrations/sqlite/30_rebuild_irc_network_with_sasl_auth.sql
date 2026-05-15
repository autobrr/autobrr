CREATE TABLE irc_network_dg_tmp
(
    id              INTEGER
        primary key,
    enabled         BOOLEAN,
    name            TEXT    not null,
    server          TEXT    not null,
    port            INTEGER not null,
    tls             BOOLEAN,
    pass            TEXT,
    nick            TEXT,
    auth_mechanism  TEXT,
    auth_account    TEXT,
    auth_password   TEXT,
    invite_command  TEXT,
    connected       BOOLEAN,
    connected_since TIMESTAMP,
    created_at      TIMESTAMP default CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP default CURRENT_TIMESTAMP,
    unique (server, port, nick)
);

INSERT INTO irc_network_dg_tmp(id, enabled, name, server, port, tls, pass, nick, auth_mechanism, auth_account,
                               auth_password, invite_command,
                               connected, connected_since, created_at, updated_at)
SELECT id,
       enabled,
       name,
       server,
       port,
       tls,
       pass,
       nickserv_account,
       'SASL_PLAIN',
       nickserv_account,
       nickserv_password,
       invite_command,
       connected,
       connected_since,
       created_at,
       updated_at
FROM irc_network;

DROP TABLE irc_network;

ALTER TABLE irc_network_dg_tmp
    RENAME TO irc_network;
