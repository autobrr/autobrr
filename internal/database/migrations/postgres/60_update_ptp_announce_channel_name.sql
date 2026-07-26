UPDATE irc_channel
SET name = '#ptp-announce'
WHERE name = '#ptp-announce-dev'
  AND NOT EXISTS (SELECT 1 FROM irc_channel WHERE name = '#ptp-announce');

UPDATE irc_network
SET invite_command = REPLACE(invite_command, '#ptp-announce-dev', '#ptp-announce')
WHERE invite_command LIKE '%#ptp-announce-dev%';
