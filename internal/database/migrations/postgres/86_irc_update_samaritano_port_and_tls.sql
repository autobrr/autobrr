-- Samaritano moved its announce server to TLS on port 6697.
-- Skip rows that would collide with an existing 6697 row for the same nick (server, port, nick is UNIQUE).
UPDATE irc_network
SET port       = 6697,
    tls        = true,
    updated_at = CURRENT_TIMESTAMP
WHERE server = 'irc.samaritano.cc'
  AND (port != 6697 OR tls IS NULL OR tls = false)
  AND NOT EXISTS (SELECT 1
                  FROM irc_network n2
                  WHERE n2.server = 'irc.samaritano.cc'
                    AND n2.port = 6697
                    AND n2.id != irc_network.id
                    AND ((n2.nick IS NULL AND irc_network.nick IS NULL) OR n2.nick = irc_network.nick));
