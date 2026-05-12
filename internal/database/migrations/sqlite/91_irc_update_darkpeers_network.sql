-- DarkPeers moved from irc.p2p-network.net (#dpannounce) to irc.darkpeers.org (#announce).
-- Carry over only the row(s) that actually carried the #dpannounce channel so we don't
-- disturb other indexers (bit-hdtv, fearnopeer, ncore, etc.) sharing irc.p2p-network.net.

-- 1. Mirror each affected p2p-network row as a new DarkPeers row, preserving connection/auth.
INSERT INTO irc_network (
    enabled, name, server, port, tls, tls_skip_verify, pass, nick,
    auth_mechanism, auth_account, auth_password, invite_command,
    use_bouncer, bouncer_addr, bot_mode, use_proxy, proxy_id,
    created_at, updated_at
)
SELECT
    n.enabled, 'DarkPeers', 'irc.darkpeers.org', n.port, n.tls, n.tls_skip_verify, n.pass, n.nick,
    n.auth_mechanism, n.auth_account, n.auth_password, n.invite_command,
    n.use_bouncer, n.bouncer_addr, n.bot_mode, n.use_proxy, n.proxy_id,
    CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM irc_network n
WHERE n.server = 'irc.p2p-network.net'
  AND EXISTS (
      SELECT 1 FROM irc_channel c
      WHERE c.network_id = n.id
        AND LOWER(c.name) = '#dpannounce'
  )
  AND NOT EXISTS (
      SELECT 1 FROM irc_network n2
      WHERE n2.server = 'irc.darkpeers.org'
        AND n2.port = n.port
        AND ((n2.nick IS NULL AND n.nick IS NULL) OR n2.nick = n.nick)
  );

-- 2. Copy the channel over as #announce, keeping enabled/password/detached.
INSERT INTO irc_channel (enabled, name, password, detached, network_id)
SELECT c.enabled, '#announce', c.password, c.detached, new_n.id
FROM irc_channel c
JOIN irc_network old_n ON old_n.id = c.network_id
JOIN irc_network new_n
     ON new_n.server = 'irc.darkpeers.org'
    AND new_n.port = old_n.port
    AND ((new_n.nick IS NULL AND old_n.nick IS NULL) OR new_n.nick = old_n.nick)
WHERE old_n.server = 'irc.p2p-network.net'
  AND LOWER(c.name) = '#dpannounce'
  AND NOT EXISTS (
      SELECT 1 FROM irc_channel c2
      WHERE c2.network_id = new_n.id
        AND LOWER(c2.name) = '#announce'
  );

-- 3. Drop the obsolete #dpannounce channel from the old p2p-network row(s).
DELETE FROM irc_channel
WHERE LOWER(name) = '#dpannounce'
  AND network_id IN (
      SELECT id FROM irc_network WHERE server = 'irc.p2p-network.net'
  );

-- 4. Remove the old p2p-network row only when it has no channels left AND we created
--    a matching DarkPeers replacement (i.e. it existed solely for DarkPeers).
DELETE FROM irc_network
WHERE server = 'irc.p2p-network.net'
  AND NOT EXISTS (
      SELECT 1 FROM irc_channel c WHERE c.network_id = irc_network.id
  )
  AND EXISTS (
      SELECT 1 FROM irc_network new_n
      WHERE new_n.server = 'irc.darkpeers.org'
        AND new_n.port = irc_network.port
        AND ((new_n.nick IS NULL AND irc_network.nick IS NULL) OR new_n.nick = irc_network.nick)
  );
