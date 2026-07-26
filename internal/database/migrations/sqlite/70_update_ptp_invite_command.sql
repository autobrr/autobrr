UPDATE irc_network
SET invite_command = REPLACE(invite_command, '#ptp-announce-dev', '#ptp-announce')
WHERE invite_command LIKE '%#ptp-announce-dev%';
