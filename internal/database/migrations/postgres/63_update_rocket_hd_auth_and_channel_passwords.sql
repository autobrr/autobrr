UPDATE irc_network
SET auth_mechanism = 'NONE',
    auth_account   = '',
    auth_password  = ''
WHERE server = 'irc.rocket-hd.cc'
  AND auth_mechanism != 'NONE';

UPDATE irc_channel
SET password = NULL
WHERE password IS NOT NULL
  AND network_id IN (SELECT id
                     FROM irc_network
                     WHERE server = 'irc.rocket-hd.cc');
