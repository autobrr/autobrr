INSERT INTO irc_network (enabled, name, server, port, tls, pass, nick,
                         auth_mechanism, auth_account, auth_password,
                         invite_command, use_bouncer, bouncer_addr, bot_mode,
                         connected, connected_since, use_proxy, proxy_id,
                         created_at, updated_at)
SELECT enabled,
       'ULCX',
       'irc.upload.cx',
       port,
       tls,
       pass,
       nick,
       auth_mechanism,
       auth_account,
       auth_password,
       invite_command,
       use_bouncer,
       bouncer_addr,
       bot_mode,
       connected,
       connected_since,
       use_proxy,
       proxy_id,
       CURRENT_TIMESTAMP,
       CURRENT_TIMESTAMP
FROM irc_network
WHERE id IN (SELECT network_id
             FROM irc_channel
             WHERE name = '#ulcx-announce');

INSERT INTO irc_channel (enabled, name, password, detached, network_id)
SELECT c.enabled,
       '#announce',
       c.password,
       c.detached,
       (SELECT MAX(id) FROM irc_network WHERE name = 'ULCX' AND server = 'irc.upload.cx')
FROM irc_channel c
WHERE c.name = '#ulcx-announce';

DELETE
FROM irc_channel
WHERE name = '#ulcx-announce';
