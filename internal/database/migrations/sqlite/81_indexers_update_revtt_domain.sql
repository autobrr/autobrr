UPDATE indexer
SET base_url = 'https://revott.me/'
WHERE base_url = 'https://www.revolutiontt.me/' OR base_url = 'https://revolutiontt.me/';

UPDATE irc_network
SET server = 'irc.revott.me'
WHERE server = 'irc.revolutiontt.me';
