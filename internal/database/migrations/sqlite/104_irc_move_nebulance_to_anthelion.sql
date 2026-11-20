-- Nebulance moved onto Anthelion's irc.anthelion.me along with its services, so accounts and
-- credentials still work there. Every row left on irc.nebulance.io is moved: merged into the Anthelion
-- row for the same nick when there is one, changed in place otherwise. Merging also fills in the auth
-- the Anthelion network migration left off rows whose Nebulance row is still around.
--
-- A merge never starts or stops a channel and never weakens a connection: the Anthelion row takes TLS
-- and a proxy from the rows merged into it. Bouncer rows keep only NickServ auth: their server
-- password, and possibly their SASL login, belong to the bouncer and must not be sent to Anthelion.

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
--    The pick also guarantees no other Anthelion row for the nick sits on 6697 when the target does not.
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

-- 3. Keep the stronger transport: TLS when a merged row used it, and a merged row's proxy when the
--    Anthelion row connects directly.
UPDATE irc_network
SET tls        = true,
    port       = 6697,
    updated_at = CURRENT_TIMESTAMP
WHERE NOT tls
  AND id IN (
      SELECT t.target_id FROM nebulance_target t
      JOIN irc_network n ON n.id = t.network_id
      WHERE n.tls
  );

UPDATE irc_network
SET use_proxy  = true,
    proxy_id   = (
        SELECT n.proxy_id FROM nebulance_target t
        JOIN irc_network n ON n.id = t.network_id
        WHERE t.target_id = irc_network.id AND n.use_proxy AND n.proxy_id IS NOT NULL
        ORDER BY CASE WHEN n.enabled THEN 0 ELSE 1 END, n.id
        LIMIT 1
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE NOT use_proxy
  AND id IN (
      SELECT t.target_id FROM nebulance_target t
      JOIN irc_network n ON n.id = t.network_id
      WHERE n.use_proxy AND n.proxy_id IS NOT NULL
  );

-- 4. Fill in auth and server password on Anthelion rows that have neither, from an enabled source first.
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

-- 5. Pick one channel per name to move, skipping names the Anthelion row already has, and record
--    whether it is running now.
CREATE TEMPORARY TABLE nebulance_channel AS
SELECT
    c.id AS channel_id,
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

-- 6. Channels the Anthelion row already has, whose nebulance copy is running. The Anthelion copy has
--    to run after the merge.
CREATE TEMPORARY TABLE nebulance_duplicate AS
SELECT DISTINCT ac.id AS channel_id, t.target_id
FROM nebulance_target t
JOIN irc_network n ON n.id = t.network_id
JOIN irc_channel c ON c.network_id = n.id
JOIN irc_channel ac ON ac.network_id = t.target_id AND LOWER(ac.name) = LOWER(c.name)
WHERE n.enabled AND c.enabled;

-- 7. A disabled Anthelion row that has to run a channel is enabled, and the channels it already had
--    are disabled instead so they stay off. The running duplicates are turned back on right after.
CREATE TEMPORARY TABLE nebulance_wake AS
SELECT target_id
FROM nebulance_channel
WHERE target_enabled = 0 AND network_enabled = 1 AND channel_enabled = 1
UNION
SELECT d.target_id
FROM nebulance_duplicate d
JOIN irc_network a ON a.id = d.target_id
WHERE NOT a.enabled;

UPDATE irc_channel
SET enabled = false
WHERE network_id IN (SELECT target_id FROM nebulance_wake);

UPDATE irc_network
SET enabled    = true,
    updated_at = CURRENT_TIMESTAMP
WHERE id IN (SELECT target_id FROM nebulance_wake);

UPDATE irc_channel
SET enabled = true
WHERE id IN (SELECT channel_id FROM nebulance_duplicate);

-- 8. Move the channels. One from a disabled nebulance row landing on an enabled Anthelion row is
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

-- 9. Give the Anthelion row an invite command for every bot it has none for, taken from the merged
--    rows, enabled first. Commands are split the way the IRC handler reads them: on commas, without
--    /msg, and keyed by the bot, the first word.
CREATE TEMPORARY TABLE nebulance_invite AS
WITH RECURSIVE split (network_id, rest, cmd, pos) AS (
    SELECT id, COALESCE(invite_command, '') || ',', CAST('' AS TEXT), 0
    FROM irc_network
    WHERE id IN (SELECT network_id FROM nebulance_target)
       OR id IN (SELECT target_id FROM nebulance_target)
    UNION ALL
    SELECT
        network_id,
        substr(rest, instr(rest, ',') + 1),
        TRIM(REPLACE(substr(rest, 1, instr(rest, ',') - 1), '/msg', '')),
        pos + 1
    FROM split
    WHERE rest <> ''
),
command AS (
    SELECT network_id, cmd, pos, LOWER(substr(cmd, 1, instr(cmd || ' ', ' ') - 1)) AS bot
    FROM split
    WHERE cmd <> ''
)
SELECT
    t.target_id,
    c.cmd,
    CASE WHEN n.enabled THEN 0 ELSE 1 END AS source_rank,
    n.id AS network_id,
    c.pos,
    ROW_NUMBER() OVER (
        PARTITION BY t.target_id, c.bot
        ORDER BY CASE WHEN n.enabled THEN 0 ELSE 1 END, n.id, c.pos
    ) AS rn
FROM nebulance_target t
JOIN irc_network n ON n.id = t.network_id
JOIN command c ON c.network_id = t.network_id
WHERE NOT EXISTS (
    SELECT 1 FROM command tc
    WHERE tc.network_id = t.target_id AND tc.bot = c.bot
);

DELETE FROM nebulance_invite WHERE rn > 1;

UPDATE irc_network
SET invite_command = CASE WHEN TRIM(COALESCE(invite_command, '')) = '' THEN '' ELSE invite_command || ',' END
        || (
            SELECT string_agg(m.cmd, ',' ORDER BY m.source_rank, m.network_id, m.pos)
            FROM nebulance_invite m
            WHERE m.target_id = irc_network.id
        ),
    updated_at     = CURRENT_TIMESTAMP
WHERE id IN (SELECT target_id FROM nebulance_invite);

-- 10. Drop the merged nebulance rows and the duplicate channels left on them.
DELETE FROM irc_channel
WHERE network_id IN (SELECT network_id FROM nebulance_target);

DELETE FROM irc_network
WHERE id IN (SELECT network_id FROM nebulance_target);

DROP TABLE nebulance_invite;
DROP TABLE nebulance_wake;
DROP TABLE nebulance_duplicate;
DROP TABLE nebulance_channel;
DROP TABLE nebulance_auth;
DROP TABLE nebulance_target;
DROP TABLE nebulance_network;
