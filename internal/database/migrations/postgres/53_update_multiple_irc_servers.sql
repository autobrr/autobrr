UPDATE irc_network
SET server = 'irc.scenehd.org'
WHERE server = 'irc.scenehd.eu';

UPDATE irc_network
SET server = 'irc.p2p-network.net',
    name   = 'P2P-Network',
    nick   = nick || '_0'
WHERE server = 'irc.librairc.net';

UPDATE irc_network
SET server = 'irc.atw-inter.net',
    name   = 'ATW-Inter'
WHERE server = 'irc.ircnet.com';
