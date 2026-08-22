UPDATE indexer
SET identifier          = 'seedcore',
    identifier_external = 'SeedCore',
    base_url            = 'https://seedcore.net/',
    name                = 'SeedCore'
WHERE identifier = 'rotorrent';

UPDATE irc_channel
SET name = '#SeedCore.net'
WHERE name = '#Rotorrent.info'
  AND NOT EXISTS (SELECT 1 FROM irc_channel WHERE name = '#SeedCore.net');

UPDATE "release"
SET indexer = 'seedcore'
WHERE indexer = 'rotorrent';
