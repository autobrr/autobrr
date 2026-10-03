-- NordicBytes moved from irc.p2p-network.net to irc.nordicbytes.org (6697/TLS), channel #nordicbytes is now #announce.
-- Only rows carrying #nordicbytes are touched so other indexers sharing irc.p2p-network.net keep working.
-- The source is the lowest-id p2p-network row per nick, so two rows with the same nick on different
-- ports cannot both map onto the single (irc.nordicbytes.org, 6697, nick) row allowed by UNIQUE.

-- 1. Mirror the p2p-network row as a NordicBytes row, preserving auth and bouncer/proxy settings.
--    Skipped when any irc.nordicbytes.org row exists: the user already moved over by hand or with a
--    custom definition, and a second row would open a duplicate connection.
INSERT INTO irc_network (
    enabled, name, server, port, tls, tls_skip_verify, pass, nick,
    auth_mechanism, auth_account, auth_password, invite_command,
    use_bouncer, bouncer_addr, bot_mode, use_proxy, proxy_id,
    created_at, updated_at
)
SELECT
    n.enabled, 'NordicBytes', 'irc.nordicbytes.org', 6697, true, n.tls_skip_verify, n.pass, n.nick,
    n.auth_mechanism, n.auth_account, n.auth_password, n.invite_command,
    n.use_bouncer, n.bouncer_addr, n.bot_mode, n.use_proxy, n.proxy_id,
    CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM irc_network n
WHERE n.id IN (
      SELECT MIN(o.id)
      FROM irc_network o
      JOIN irc_channel c ON c.network_id = o.id
      WHERE o.server = 'irc.p2p-network.net'
        AND LOWER(c.name) = '#nordicbytes'
      GROUP BY COALESCE(o.nick, '')
  )
  AND NOT EXISTS (
      SELECT 1 FROM irc_network n2 WHERE n2.server = 'irc.nordicbytes.org'
  );

-- 2. Copy the channel onto the matching NordicBytes row as #announce, keeping enabled/password/detached.
INSERT INTO irc_channel (enabled, name, password, detached, network_id)
SELECT c.enabled, '#announce', c.password, c.detached, new_n.id
FROM irc_network old_n
JOIN irc_channel c ON c.network_id = old_n.id
JOIN irc_network new_n
     ON new_n.server = 'irc.nordicbytes.org'
    AND new_n.port = 6697
    AND COALESCE(new_n.nick, '') = COALESCE(old_n.nick, '')
WHERE old_n.id IN (
      SELECT MIN(o.id)
      FROM irc_network o
      JOIN irc_channel c3 ON c3.network_id = o.id
      WHERE o.server = 'irc.p2p-network.net'
        AND LOWER(c3.name) = '#nordicbytes'
      GROUP BY COALESCE(o.nick, '')
  )
  AND c.id = (
      SELECT MIN(c4.id) FROM irc_channel c4
      WHERE c4.network_id = old_n.id
        AND LOWER(c4.name) = '#nordicbytes'
  )
  AND NOT EXISTS (
      SELECT 1 FROM irc_channel c2
      WHERE c2.network_id = new_n.id
        AND LOWER(c2.name) = '#announce'
  );

-- 3. Drop the stale #nordicbytes channel from the p2p-network row(s), but only once a NordicBytes
--    row carries #announce, so a user is never left without the channel.
DELETE FROM irc_channel
WHERE LOWER(name) = '#nordicbytes'
  AND network_id IN (
      SELECT id FROM irc_network WHERE server = 'irc.p2p-network.net'
  )
  AND EXISTS (
      SELECT 1 FROM irc_network n
      JOIN irc_channel c ON c.network_id = n.id
      WHERE n.server = 'irc.nordicbytes.org'
        AND LOWER(c.name) = '#announce'
  );

-- 4. Remove p2p-network rows left without channels, i.e. the ones that existed solely for NordicBytes.
DELETE FROM irc_network
WHERE server = 'irc.p2p-network.net'
  AND NOT EXISTS (
      SELECT 1 FROM irc_channel c WHERE c.network_id = irc_network.id
  )
  AND EXISTS (
      SELECT 1 FROM irc_network n
      JOIN irc_channel c ON c.network_id = n.id
      WHERE n.server = 'irc.nordicbytes.org'
        AND LOWER(c.name) = '#announce'
  );
