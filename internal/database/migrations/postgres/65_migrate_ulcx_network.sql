WITH original_network AS (SELECT *
                          FROM irc_network
                          WHERE id IN (SELECT network_id
                                       FROM irc_channel
                                       WHERE name = '#ulcx-announce')),
     new_network AS (
         INSERT INTO irc_network (
                                  enabled, name, server, port, tls, pass, nick,
                                  auth_mechanism, auth_account, auth_password,
                                  invite_command, use_bouncer, bouncer_addr, bot_mode,
                                  connected, connected_since, use_proxy, proxy_id,
                                  created_at, updated_at
             )
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
             FROM original_network
             RETURNING id)
INSERT
INTO irc_channel (enabled, name, password, detached, network_id)
SELECT c.enabled, '#announce', c.password, c.detached, n.id
FROM irc_channel c
         CROSS JOIN new_network n
WHERE c.name = '#ulcx-announce';

DELETE
FROM irc_channel
WHERE name = '#ulcx-announce';
