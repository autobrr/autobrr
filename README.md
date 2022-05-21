# autobrr

> :warning: Work in progress. Expect bugs and breaking changes. Features may be broken or incomplete.

autobrr is a modern single binary replacement for the autodl-irssi+rutorrent plugin.  
autobrr monitors IRC announce channels and torznab RSS feeds to get releases as soon as they are available, with good filtering, and regex support.

Built on Go/React to be resource friendly.

## Documentation

Installation guide and documentation can be found at https://autobrr.com

## Features:

- Single binary + config for easy setup
- Easy to use UI
  - Mobile friendly
- Powerful filtering
  - Regex support
- Available torrent actions:
  - qBittorrent
    - With built in reannounce
  - Deluge
    - v1+ and v2 support
  - Radarr
  - Sonarr
  - Lidarr
  - Whisparr
  - Save to watch folder
  - Run custom commands
- 30+ supported indexers
- Torznab RSS feeds
- Postgres support
- Notifications
  - Discord
  - Notifiarr
