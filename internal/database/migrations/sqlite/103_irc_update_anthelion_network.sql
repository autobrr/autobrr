-- Anthelion moved #ant-announce from irc.nebulance.io to its own network on irc.anthelion.me (6697/TLS).
-- Only rows carrying #ant-announce are touched so Nebulance (#nbl-announce) on the same rows keeps working.

-- 0. Record every nebulance row carrying #ant-announce, ranked per nick so the copy prefers an enabled
--    network and channel. One row per nick is copied because UNIQUE (server, port, nick) allows only
--    one irc.anthelion.me:6697 row per nick. Left empty when any irc.anthelion.me row exists: the user
--    already moved over by hand or with a custom definition, and the whole migration is skipped.
CREATE TEMPORARY TABLE anthelion_migration AS
SELECT
    n.id AS network_id,
    c.id AS channel_id,
    ROW_NUMBER() OVER (
        PARTITION BY COALESCE(n.nick, '')
        ORDER BY CASE WHEN n.enabled AND c.enabled THEN 0 ELSE 1 END, n.id, c.id
    ) AS rn
FROM irc_network n
JOIN irc_channel c ON c.network_id = n.id
WHERE n.server = 'irc.nebulance.io'
  AND LOWER(c.name) = '#ant-announce'
  AND NOT EXISTS (
      SELECT 1 FROM irc_network n2 WHERE n2.server = 'irc.anthelion.me'
  );

-- 1. Mirror the source row as an Anthelion row. Auth and server password carry over since Anthelion
--    moved the Nebulance services over with their accounts. The bouncer is turned off since it still
--    points at Nebulance; the address is kept for the user to re-point it. Bouncer rows drop their
--    server password and SASL login, which may belong to the bouncer, and verify TLS again since a
--    skip was likely for the bouncer's certificate. NickServ auth is kept.
INSERT INTO irc_network (
    enabled, name, server, port, tls, tls_skip_verify, pass, nick,
    auth_mechanism, auth_account, auth_password, invite_command,
    use_bouncer, bouncer_addr, bot_mode, use_proxy, proxy_id,
    created_at, updated_at
)
SELECT
    n.enabled, 'Anthelion', 'irc.anthelion.me', 6697, true,
    CASE WHEN n.use_bouncer THEN false ELSE n.tls_skip_verify END,
    CASE WHEN n.use_bouncer THEN '' ELSE n.pass END,
    n.nick,
    CASE WHEN n.use_bouncer AND n.auth_mechanism = 'SASL_PLAIN' THEN 'NONE' ELSE n.auth_mechanism END,
    CASE WHEN n.use_bouncer AND n.auth_mechanism = 'SASL_PLAIN' THEN '' ELSE n.auth_account END,
    CASE WHEN n.use_bouncer AND n.auth_mechanism = 'SASL_PLAIN' THEN '' ELSE n.auth_password END,
    n.invite_command,
    false, n.bouncer_addr, n.bot_mode, n.use_proxy, n.proxy_id,
    CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM irc_network n
JOIN anthelion_migration m ON m.network_id = n.id AND m.rn = 1;

-- 2. Copy the channel onto the matching Anthelion row, keeping enabled/password/detached.
INSERT INTO irc_channel (enabled, name, password, detached, network_id)
SELECT c.enabled, '#ant-announce', c.password, c.detached, new_n.id
FROM anthelion_migration m
JOIN irc_channel c ON c.id = m.channel_id
JOIN irc_network old_n ON old_n.id = m.network_id
JOIN irc_network new_n
     ON new_n.server = 'irc.anthelion.me'
    AND new_n.port = 6697
    AND COALESCE(new_n.nick, '') = COALESCE(old_n.nick, '')
WHERE m.rn = 1;

-- 3. Drop the stale #ant-announce channels from the nebulance rows.
DELETE FROM irc_channel
WHERE id IN (SELECT channel_id FROM anthelion_migration);

-- 4. Remove the nebulance rows this migration left without channels, i.e. the ones that existed
--    solely for Anthelion. Rows that were already empty are kept.
DELETE FROM irc_network
WHERE id IN (SELECT network_id FROM anthelion_migration)
  AND NOT EXISTS (
      SELECT 1 FROM irc_channel c WHERE c.network_id = irc_network.id
  );

DROP TABLE anthelion_migration;
