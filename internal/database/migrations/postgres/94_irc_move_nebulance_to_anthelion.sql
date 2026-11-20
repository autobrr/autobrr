-- Nebulance moved onto Anthelion's irc.anthelion.me along with its services, so accounts and
-- credentials still work there. Every row left on irc.nebulance.io is moved: merged into the Anthelion
-- row for the same nick when there is one, changed in place otherwise. Merging also fills in the auth
-- the Anthelion network migration left off rows whose Nebulance row is still around.
--
-- A merge never starts or stops a channel and never weakens a connection: the Anthelion row takes TLS
-- and a proxy from the rows merged into it. Bouncer rows drop their server password and SASL login,
-- which may belong to the bouncer, and are not sent to Anthelion. NickServ auth is kept.

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
--    since it still points at Nebulance; the address is kept for the user to re-point it. TLS is
--    verified again, since a skip was likely for the bouncer's certificate.
UPDATE irc_network
SET server         = 'irc.anthelion.me',
    port           = 6697,
    tls            = true,
    tls_skip_verify = CASE WHEN use_bouncer THEN false ELSE tls_skip_verify END,
    pass           = CASE WHEN use_bouncer THEN '' ELSE pass END,
    auth_mechanism = CASE WHEN use_bouncer AND auth_mechanism = 'SASL_PLAIN' THEN 'NONE' ELSE auth_mechanism END,
    auth_account   = CASE WHEN use_bouncer AND auth_mechanism = 'SASL_PLAIN' THEN '' ELSE auth_account END,
    auth_password  = CASE WHEN use_bouncer AND auth_mechanism = 'SASL_PLAIN' THEN '' ELSE auth_password END,
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

-- 3. Keep the stronger transport on an Anthelion row that connects directly: TLS when a merged row
--    used it, and an enabled proxy of a merged row when it has none. A bouncer row keeps its own
--    transport, since it dials the bouncer and not Anthelion.
UPDATE irc_network
SET tls        = true,
    port       = 6697,
    updated_at = CURRENT_TIMESTAMP
WHERE NOT COALESCE(tls, false)
  AND NOT COALESCE(use_bouncer, false)
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
        JOIN proxy p ON p.id = n.proxy_id
        WHERE t.target_id = irc_network.id AND n.use_proxy AND p.enabled
        ORDER BY CASE WHEN n.enabled THEN 0 ELSE 1 END, n.id
        LIMIT 1
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE NOT COALESCE(use_proxy, false)
  AND NOT COALESCE(use_bouncer, false)
  AND id IN (
      SELECT t.target_id FROM nebulance_target t
      JOIN irc_network n ON n.id = t.network_id
      JOIN proxy p ON p.id = n.proxy_id
      WHERE n.use_proxy AND p.enabled
  );

-- 4. Fill in auth and server password on Anthelion rows that have neither usable auth nor a server
--    password, from an enabled source with usable credentials first. Usable follows the handler: SASL
--    needs account and password, NickServ or an empty mechanism needs a password. A bouncer on either
--    side keeps the server password and SASL login from crossing.
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
        CASE WHEN n.use_bouncer OR COALESCE(a.use_bouncer, false) THEN '' ELSE COALESCE(n.pass, '') END AS pass,
        CASE WHEN (n.use_bouncer OR COALESCE(a.use_bouncer, false)) AND n.auth_mechanism = 'SASL_PLAIN' THEN 'NONE' ELSE COALESCE(n.auth_mechanism, 'NONE') END AS auth_mechanism,
        CASE WHEN (n.use_bouncer OR COALESCE(a.use_bouncer, false)) AND n.auth_mechanism = 'SASL_PLAIN' THEN '' ELSE COALESCE(n.auth_account, '') END AS auth_account,
        CASE WHEN (n.use_bouncer OR COALESCE(a.use_bouncer, false)) AND n.auth_mechanism = 'SASL_PLAIN' THEN '' ELSE COALESCE(n.auth_password, '') END AS auth_password
    FROM nebulance_target t
    JOIN irc_network n ON n.id = t.network_id
    JOIN irc_network a ON a.id = t.target_id
    WHERE COALESCE(a.pass, '') = ''
      AND NOT (
          (COALESCE(a.auth_mechanism, '') IN ('NICKSERV', '') AND COALESCE(a.auth_password, '') <> '')
          OR (a.auth_mechanism = 'SASL_PLAIN' AND COALESCE(a.auth_account, '') <> '' AND COALESCE(a.auth_password, '') <> '')
      )
) s
WHERE (s.auth_mechanism IN ('NICKSERV', '') AND s.auth_password <> '')
   OR (s.auth_mechanism = 'SASL_PLAIN' AND s.auth_account <> '' AND s.auth_password <> '')
   OR s.pass <> '';

DELETE FROM nebulance_auth WHERE rn > 1;

UPDATE irc_network
SET pass           = (SELECT m.pass FROM nebulance_auth m WHERE m.target_id = irc_network.id),
    auth_mechanism = (SELECT m.auth_mechanism FROM nebulance_auth m WHERE m.target_id = irc_network.id),
    auth_account   = (SELECT m.auth_account FROM nebulance_auth m WHERE m.target_id = irc_network.id),
    auth_password  = (SELECT m.auth_password FROM nebulance_auth m WHERE m.target_id = irc_network.id),
    updated_at     = CURRENT_TIMESTAMP
WHERE id IN (SELECT target_id FROM nebulance_auth);

-- 5. Remember, for the invite merge, which Anthelion rows already run their own #nbl-announce and
--    which nebulance rows run a channel, before channels move.
CREATE TEMPORARY TABLE nebulance_running_source AS
SELECT DISTINCT n.id AS network_id
FROM nebulance_target t
JOIN irc_network n ON n.id = t.network_id
JOIN irc_channel c ON c.network_id = n.id
WHERE n.enabled AND c.enabled;

CREATE TEMPORARY TABLE nebulance_target_nbl AS
SELECT DISTINCT a.id AS target_id
FROM irc_network a
JOIN irc_channel ac ON ac.network_id = a.id
WHERE a.id IN (SELECT target_id FROM nebulance_target)
  AND LOWER(ac.name) = '#nbl-announce'
  AND a.enabled
  AND ac.enabled;

-- 6. Pick one channel per name to move, skipping names the Anthelion row already has, and record
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

-- 7. Channels the Anthelion row already has, whose nebulance copy is running. The Anthelion copy has
--    to run after the merge, with the nebulance copy's key unless it was running already.
CREATE TEMPORARY TABLE nebulance_duplicate AS
SELECT d.channel_id, d.target_id, d.target_running, d.password
FROM (
    SELECT
        ac.id AS channel_id,
        t.target_id,
        CASE WHEN a.enabled AND ac.enabled THEN 1 ELSE 0 END AS target_running,
        c.password,
        ROW_NUMBER() OVER (PARTITION BY ac.id ORDER BY c.id) AS rn
    FROM nebulance_target t
    JOIN irc_network n ON n.id = t.network_id
    JOIN irc_network a ON a.id = t.target_id
    JOIN irc_channel c ON c.network_id = n.id
    JOIN irc_channel ac ON ac.network_id = t.target_id AND LOWER(ac.name) = LOWER(c.name)
    WHERE n.enabled AND c.enabled
) d
WHERE d.rn = 1;

UPDATE irc_channel
SET password = (SELECT d.password FROM nebulance_duplicate d WHERE d.channel_id = irc_channel.id)
WHERE id IN (SELECT channel_id FROM nebulance_duplicate WHERE target_running = 0);

-- 8. A disabled Anthelion row that has to run a channel is enabled, and the channels it already had
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

-- 9. Move the channels. One from a disabled nebulance row landing on an enabled Anthelion row is
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

-- 10. Merge the invite commands per bot, split the way the IRC handler reads them: on commas, without
--     /msg, keyed by the bot, the first word. A command without a message is ignored, as the handler
--     does. The Anthelion row gets a command for every bot it has none for, from a merged row with a
--     running channel first, then an enabled one. Muffit invites to #nbl-announce, so the merged row's
--     Muffit command replaces the Anthelion row's, unless that row already ran its own #nbl-announce.
CREATE TEMPORARY TABLE nebulance_command AS
WITH RECURSIVE split (network_id, rest, cmd, pos) AS (
    SELECT id, COALESCE(invite_command, '') || ',', CAST('' AS TEXT), 0
    FROM irc_network
    WHERE id IN (SELECT network_id FROM nebulance_target)
       OR id IN (SELECT target_id FROM nebulance_target)
    UNION ALL
    SELECT
        network_id,
        substr(rest, strpos(rest, ',') + 1),
        TRIM(REPLACE(substr(rest, 1, strpos(rest, ',') - 1), '/msg', '')),
        pos + 1
    FROM split
    WHERE rest <> ''
)
SELECT
    network_id,
    cmd,
    pos,
    LOWER(substr(cmd, 1, strpos(cmd || ' ', ' ') - 1)) AS bot,
    CASE WHEN strpos(cmd, ' ') > 0 THEN 1 ELSE 0 END AS usable
FROM split
WHERE cmd <> '';

CREATE TEMPORARY TABLE nebulance_invite AS
SELECT
    t.target_id,
    c.cmd,
    c.bot,
    CASE WHEN n.enabled THEN 0 ELSE 1 END AS source_rank,
    n.id AS network_id,
    c.pos,
    ROW_NUMBER() OVER (
        PARTITION BY t.target_id, c.bot
        ORDER BY
            CASE WHEN n.id IN (SELECT network_id FROM nebulance_running_source) THEN 0 ELSE 1 END,
            CASE WHEN n.enabled THEN 0 ELSE 1 END,
            n.id,
            c.pos
    ) AS rn
FROM nebulance_target t
JOIN irc_network n ON n.id = t.network_id
JOIN nebulance_command c ON c.network_id = t.network_id
WHERE c.usable = 1
  AND (
      NOT EXISTS (
          SELECT 1 FROM nebulance_command tc
          WHERE tc.network_id = t.target_id AND tc.bot = c.bot AND tc.usable = 1
      )
      OR (c.bot = 'muffit' AND t.target_id NOT IN (SELECT target_id FROM nebulance_target_nbl))
  );

DELETE FROM nebulance_invite WHERE rn > 1;

CREATE TEMPORARY TABLE nebulance_invite_merged AS
SELECT x.target_id, string_agg(x.cmd, ',' ORDER BY x.grp, x.source_rank, x.network_id, x.pos) AS invite_command
FROM (
    SELECT t.target_id, c.cmd, 0 AS grp, 0 AS source_rank, 0 AS network_id, c.pos
    FROM (SELECT DISTINCT target_id FROM nebulance_invite) t
    JOIN nebulance_command c ON c.network_id = t.target_id
    WHERE NOT EXISTS (
        SELECT 1 FROM nebulance_invite i
        WHERE i.target_id = t.target_id AND i.bot = c.bot
    )
    UNION ALL
    SELECT target_id, cmd, 1, source_rank, network_id, pos
    FROM nebulance_invite
) x
GROUP BY x.target_id;

UPDATE irc_network
SET invite_command = (SELECT m.invite_command FROM nebulance_invite_merged m WHERE m.target_id = irc_network.id),
    updated_at     = CURRENT_TIMESTAMP
WHERE id IN (SELECT target_id FROM nebulance_invite_merged);

-- 11. Drop the merged nebulance rows and the duplicate channels left on them.
DELETE FROM irc_channel
WHERE network_id IN (SELECT network_id FROM nebulance_target);

DELETE FROM irc_network
WHERE id IN (SELECT network_id FROM nebulance_target);

DROP TABLE nebulance_invite_merged;
DROP TABLE nebulance_invite;
DROP TABLE nebulance_command;
DROP TABLE nebulance_wake;
DROP TABLE nebulance_duplicate;
DROP TABLE nebulance_channel;
DROP TABLE nebulance_target_nbl;
DROP TABLE nebulance_running_source;
DROP TABLE nebulance_auth;
DROP TABLE nebulance_target;
DROP TABLE nebulance_network;
