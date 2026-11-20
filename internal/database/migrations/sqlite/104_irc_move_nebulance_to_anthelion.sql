-- Nebulance moved onto Anthelion's irc.anthelion.me along with its services, so accounts and
-- credentials still work there. Every row left on irc.nebulance.io is moved: merged into the Anthelion
-- row for the same nick when there is one, changed in place otherwise. Merging also fills in the auth
-- the Anthelion network migration left off rows whose Nebulance row is still around.
--
-- Bouncer rows keep only NickServ auth: their server password, and possibly their SASL login, belong
-- to the bouncer and must not be sent to Anthelion directly.

-- 0. Record the nebulance rows, ranked per nick so the row moved in place is an enabled one.
CREATE TEMPORARY TABLE nebulance_network AS
SELECT
    n.id AS network_id,
    COALESCE(n.nick, '') AS nick,
    ROW_NUMBER() OVER (
        PARTITION BY COALESCE(n.nick, '')
        ORDER BY CASE WHEN n.enabled THEN 0 ELSE 1 END, n.id
    ) AS rn,
    CASE WHEN EXISTS (
        SELECT 1 FROM irc_network a
        WHERE a.server = 'irc.anthelion.me' AND COALESCE(a.nick, '') = COALESCE(n.nick, '')
    ) THEN 1 ELSE 0 END AS has_anthelion
FROM irc_network n
WHERE n.server = 'irc.nebulance.io';

-- 1. Move one row per nick in place when the nick has no Anthelion row yet. The bouncer is turned off
--    since it still points at Nebulance; the address is kept for the user to re-point it.
UPDATE irc_network
SET server         = 'irc.anthelion.me',
    port           = 6697,
    tls            = true,
    pass           = CASE WHEN use_bouncer THEN '' ELSE pass END,
    auth_mechanism = CASE WHEN use_bouncer AND COALESCE(auth_mechanism, 'NONE') <> 'NICKSERV' THEN 'NONE' ELSE auth_mechanism END,
    auth_account   = CASE WHEN use_bouncer AND COALESCE(auth_mechanism, 'NONE') <> 'NICKSERV' THEN '' ELSE auth_account END,
    auth_password  = CASE WHEN use_bouncer AND COALESCE(auth_mechanism, 'NONE') <> 'NICKSERV' THEN '' ELSE auth_password END,
    use_bouncer    = false,
    updated_at     = CURRENT_TIMESTAMP
WHERE id IN (SELECT network_id FROM nebulance_network WHERE rn = 1 AND has_anthelion = 0);

-- 2. Pair every remaining nebulance row with the Anthelion row it merges into, preferring port 6697.
CREATE TEMPORARY TABLE nebulance_target AS
SELECT
    m.network_id,
    (
        SELECT a.id FROM irc_network a
        WHERE a.server = 'irc.anthelion.me' AND COALESCE(a.nick, '') = m.nick
        ORDER BY CASE WHEN a.port = 6697 THEN 0 ELSE 1 END, a.id
        LIMIT 1
    ) AS target_id
FROM nebulance_network m
JOIN irc_network n ON n.id = m.network_id
WHERE n.server = 'irc.nebulance.io';

-- 3. Fill in auth and server password on Anthelion rows that have neither, from an enabled source first.
CREATE TEMPORARY TABLE nebulance_auth AS
SELECT
    s.target_id,
    s.pass,
    s.auth_mechanism,
    s.auth_account,
    s.auth_password,
    ROW_NUMBER() OVER (
        PARTITION BY s.target_id
        ORDER BY CASE WHEN s.enabled THEN 0 ELSE 1 END, s.network_id
    ) AS rn
FROM (
    SELECT
        t.target_id,
        n.id AS network_id,
        n.enabled,
        CASE WHEN n.use_bouncer THEN '' ELSE COALESCE(n.pass, '') END AS pass,
        CASE WHEN n.use_bouncer AND COALESCE(n.auth_mechanism, 'NONE') <> 'NICKSERV' THEN 'NONE' ELSE COALESCE(n.auth_mechanism, 'NONE') END AS auth_mechanism,
        CASE WHEN n.use_bouncer AND COALESCE(n.auth_mechanism, 'NONE') <> 'NICKSERV' THEN '' ELSE COALESCE(n.auth_account, '') END AS auth_account,
        CASE WHEN n.use_bouncer AND COALESCE(n.auth_mechanism, 'NONE') <> 'NICKSERV' THEN '' ELSE COALESCE(n.auth_password, '') END AS auth_password
    FROM nebulance_target t
    JOIN irc_network n ON n.id = t.network_id
    JOIN irc_network a ON a.id = t.target_id
    WHERE COALESCE(a.auth_mechanism, 'NONE') = 'NONE'
      AND COALESCE(a.auth_account, '') = ''
      AND COALESCE(a.auth_password, '') = ''
      AND COALESCE(a.pass, '') = ''
) s
WHERE s.auth_mechanism <> 'NONE' OR s.pass <> '';

DELETE FROM nebulance_auth WHERE rn > 1;

UPDATE irc_network
SET pass           = (SELECT m.pass FROM nebulance_auth m WHERE m.target_id = irc_network.id),
    auth_mechanism = (SELECT m.auth_mechanism FROM nebulance_auth m WHERE m.target_id = irc_network.id),
    auth_account   = (SELECT m.auth_account FROM nebulance_auth m WHERE m.target_id = irc_network.id),
    auth_password  = (SELECT m.auth_password FROM nebulance_auth m WHERE m.target_id = irc_network.id),
    updated_at     = CURRENT_TIMESTAMP
WHERE id IN (SELECT target_id FROM nebulance_auth);

-- 4. Pick one channel per name to move, skipping names the Anthelion row already has, and record
--    whether it is running now so the merge neither stops nor starts anything.
CREATE TEMPORARY TABLE nebulance_channel AS
SELECT
    c.id AS channel_id,
    n.id AS network_id,
    t.target_id,
    CASE WHEN n.enabled THEN 1 ELSE 0 END AS network_enabled,
    CASE WHEN c.enabled THEN 1 ELSE 0 END AS channel_enabled,
    CASE WHEN a.enabled THEN 1 ELSE 0 END AS target_enabled,
    ROW_NUMBER() OVER (
        PARTITION BY t.target_id, LOWER(c.name)
        ORDER BY CASE WHEN n.enabled AND c.enabled THEN 0 ELSE 1 END, c.id
    ) AS rn
FROM nebulance_target t
JOIN irc_network n ON n.id = t.network_id
JOIN irc_network a ON a.id = t.target_id
JOIN irc_channel c ON c.network_id = n.id
WHERE NOT EXISTS (
    SELECT 1 FROM irc_channel ac
    WHERE ac.network_id = t.target_id AND LOWER(ac.name) = LOWER(c.name)
);

DELETE FROM nebulance_channel WHERE rn > 1;

-- 5. A disabled Anthelion row receiving a running channel is enabled, and the channels it already had
--    are disabled instead so they stay off.
CREATE TEMPORARY TABLE nebulance_wake AS
SELECT DISTINCT target_id
FROM nebulance_channel
WHERE target_enabled = 0 AND network_enabled = 1 AND channel_enabled = 1;

UPDATE irc_channel
SET enabled = false
WHERE network_id IN (SELECT target_id FROM nebulance_wake);

UPDATE irc_network
SET enabled    = true,
    updated_at = CURRENT_TIMESTAMP
WHERE id IN (SELECT target_id FROM nebulance_wake);

-- 6. Move the channels. One from a disabled nebulance row landing on an enabled Anthelion row is
--    disabled so it does not start.
UPDATE irc_channel
SET enabled    = CASE WHEN EXISTS (
        SELECT 1 FROM nebulance_channel m
        WHERE m.channel_id = irc_channel.id
          AND m.network_enabled = 0
          AND (m.target_enabled = 1 OR m.target_id IN (SELECT target_id FROM nebulance_wake))
    ) THEN false ELSE enabled END,
    network_id = (SELECT m.target_id FROM nebulance_channel m WHERE m.channel_id = irc_channel.id)
WHERE id IN (SELECT channel_id FROM nebulance_channel);

-- 7. Append the invite commands of the rows that gave up a channel, so the moved channels still get
--    invited. A command string the Anthelion row already carries is not added twice.
CREATE TEMPORARY TABLE nebulance_invite AS
SELECT m.target_id, n.invite_command, MIN(n.id) AS first_id
FROM (SELECT DISTINCT network_id, target_id FROM nebulance_channel) m
JOIN irc_network n ON n.id = m.network_id
JOIN irc_network a ON a.id = m.target_id
WHERE COALESCE(n.invite_command, '') <> ''
  AND REPLACE(',' || COALESCE(a.invite_command, '') || ',', ',' || n.invite_command || ',', '') = ',' || COALESCE(a.invite_command, '') || ','
GROUP BY m.target_id, n.invite_command;

UPDATE irc_network
SET invite_command = CASE WHEN COALESCE(invite_command, '') = '' THEN '' ELSE invite_command || ',' END
        || (SELECT string_agg(m.invite_command, ',' ORDER BY m.first_id) FROM nebulance_invite m WHERE m.target_id = irc_network.id),
    updated_at     = CURRENT_TIMESTAMP
WHERE id IN (SELECT target_id FROM nebulance_invite);

-- 8. Drop the merged nebulance rows and the duplicate channels left on them.
DELETE FROM irc_channel
WHERE network_id IN (SELECT network_id FROM nebulance_target);

DELETE FROM irc_network
WHERE id IN (SELECT network_id FROM nebulance_target);

DROP TABLE nebulance_invite;
DROP TABLE nebulance_wake;
DROP TABLE nebulance_channel;
DROP TABLE nebulance_auth;
DROP TABLE nebulance_target;
DROP TABLE nebulance_network;
