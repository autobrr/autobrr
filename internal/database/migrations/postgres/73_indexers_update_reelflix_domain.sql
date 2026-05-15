UPDATE indexer
SET base_url = 'https://reelflix.cc/'
WHERE base_url = 'https://reelflix.xyz/';

UPDATE irc_network
SET server = 'irc.reelflix.cc'
WHERE server = 'irc.reelflix.xyz';
