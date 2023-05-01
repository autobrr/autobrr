<h1 align="center">
  <img alt="autobrr logo" src=".github/images/logo.png" width="160px"/><br/>
  autobrr
</h1>

<p align="center">autobrr is the modern download automation tool for torrents and usenet.
With inspiration and ideas from tools like trackarr, autodl-irssi and flexget we built one tool that can do it all, and then some.</p>

<p align="center"><img alt="GitHub release (latest by date)" src="https://img.shields.io/github/v/release/autobrr/autobrr?style=for-the-badge">&nbsp;<img alt="GitHub all releases" src="https://img.shields.io/github/downloads/autobrr/autobrr/total?style=for-the-badge">&nbsp;<img alt="GitHub Workflow Status" src="https://img.shields.io/github/actions/workflow/status/autobrr/autobrr/release.yml?style=for-the-badge"></p>

<img alt="autobrr ui" src=".github/images/autobrr-front.png"/><br/>

## Documentation

Installation guide and documentation can be found at https://autobrr.com

## Key features

- Torrents and usenet support
- Support for 70+ torrent trackers with IRC announces
- Newznab, Torznab and RSS support to easily get access to hundreds of torrent and usenet indexers
- Torrent Magnet support
- Powerful but simple filtering with RegEx support (like in autodl-irssi)
- Easy to use and mobile friendly web UI (with dark mode!) to manage everything
- Built on Go and React making autobrr lightweight and perfect for supporting multiple platforms (Linux, FreeBSD,
  Windows, macOS) on different architectures (e.g. x86, ARM)
- Great container support (Docker, k8s/Kubernetes)
- Database engine supporting both PostgreSQL and SQLite
- Notifications (Discord, Telegram, Notifiarr, Pushover)
- One autobrr instance can communicate with multiple clients (torrent, usenet and \*arr) on remote servers
- Base path / Subfolder (and subdomain) support for convenient reverse-proxy support

Available download clients and actions

- qBittorrent (with built in re-announce, categories, rules, max active downloads, etc)
- Deluge v1+ and v2+
- rTorrent
- Transmission
- Sonarr, Radarr, Lidarr, Whisparr and Readarr (pushes releases directly to them and gets in the early swarm, instead of
  getting them via RSS when it's already over)
- SABnzbd (usenet)
- Watch folder
- Exec custom scripts
- Webhook

## What is autobrr and how does it fit into the ecosystem?

We can start by talking about torrent trackers (hereby referred to as indexers) and maintaining ratio. You are required
to maintain a ratio with most indexers. Ratio is built by seeding your torrents. The earlier you're seeding a torrent,
the more peers you make yourself available to on that torrent.

Software like Radarr and Sonarr utilizes RSS to look for new torrents. RSS feeds are updated regularly, but too slow to
let you be a part of what we call the initial swarm of a torrent. This is were autobrr comes into play.

Many indexers announce new torrents on their IRC channels the second it is uploaded to the site. autobrr monitors such
channels in real time and grabs the torrent file as soon as it's uploaded based on certain conditions (hereby referred
to as filters) that you set up within autobrr. It then sends that torrent file to a download client of your choice via
an action set within the filter. A download client can be anything from qBittorrent and Deluge, to Radarr and Sonarr, or
a watch folder.

When your autobrr filter is set to send the torrent files to Radarr and Sonarr, they will decide if it's something they
want, and then forward it to the torrent client they are set up with.

autobrr can also send matches (torrent files that meets your filter's criteria) directly to torrent clients like
qBittorrent, Deluge, r(u)Torrent and Transmission. You don't need to use the *arr suite to make use of autobrr.

### RSS support for indexers without an IRC announcer

A lot of indexers do not announce new torrents in an IRC channel. You can still make use of these indexers with autobrr
since it has built in support for feeds as well. Both torznab and regular RSS is supported. RSS indexers are treated the
same way as regular indexers within autobrr.

This isn't needed if your usecase is feeding the *arrs only. Since they have RSS support already.

### Usenet support

Usenet support via Newzbab feeds allows you to easily manage everything in a single application. While there is a lot of
applications that handles RSS well, we think autobrr offers very easy to use filtering to help you get the content you
want.

You can use Usenet feeds and send to arrs or send directly to SABnzbd.

## Installation

Full installation guide and documentation can be found at https://autobrr.com

Remember to head over to our [Configuration Guide](https://autobrr.com/configuration/autobrr) to learn how to set up
your indexers, IRC, and download clients after you're done installing.

### Swizzin

[Swizzin](https://swizzin.ltd/) users can simply run:

```
sudo box install autobrr
```

### Saltbox

[Saltbox](https://saltbox.dev/) users can simply run:

```
sb install sandbox-autobrr
```

For more info check the [docs](https://docs.saltbox.dev/sandbox/apps/autobrr/)

### QuickBox (v3)

[QuickBox](https://quickbox.io/) users can simply run:

```
qb install autobrr -u ${username}
```

For more info check
the [docs](https://quickbox.io/knowledge-base/v3/applications-v3/autobrr-applications-v3/autobrr-quick-reference/)

### Shared seedbox

We have support for a couple of providers out of the box and if yours are missing then please write on Discord so we add
support.

The scripts require some input but does most of the work.

#### HostingByDesign (former Seedbox.io)

    wget https://gobrr.sh/install_sbio && bash install_sbio

#### Swizzin.net

    wget https://gobrr.sh/install_sbio && bash install_sbio

#### Ultra.cc

Use their official one-click installer or ours:

    wget https://gobrr.sh/install_ultra && bash install_ultra

#### WhatBox

    wget https://gobrr.sh/install_whatbox && bash install_whatbox

#### Feralhosting

    wget https://gobrr.sh/install_feral && bash install_feral

#### Bytesized hosting

    wget https://gobrr.sh/install_bytesized && bash install_bytesized

#### Other providers

For other providers the Seedbox.io installer should work. If not, open an issue or contact us
on [Discord](https://discord.gg/WQ2eUycxyT)

    wget https://gobrr.sh/install_sbio && bash install_sbio

##### One-click installers

- Ultra.cc
- Seedit4.me

### Docker compose

docker-compose for autobrr. Modify accordingly if running with unRAID or setting up with Portainer.

* Logging is optional
* Host port mapping might need to be changed to not collide with other apps
* Change `BASE_DOCKER_DATA_PATH` to match your setup. Can be simply `./data`
* Set custom network if needed

Create `docker-compose.yml` and add the following. If you have a existing setup change to fit that.

```yml
version: "3.7"

services:
  autobrr:
    container_name: autobrr
    image: ghcr.io/autobrr/autobrr:latest
    restart: unless-stopped
    environment:
      - TZ=${TZ}
    user: 1000:1000
    volumes:
      - ${BASE_DOCKER_DATA_PATH}/autobrr/config:/config
    ports:
      - 7474:7474
```

Then start with

    docker compose up -d

### Windows

Check the windows setup guide [here](https://autobrr.com/installation/windows)

### Linux generic

Download the latest release, or download the [source code](https://github.com/autobrr/autobrr/releases/latest) and build
it yourself using `make build`.

```bash
wget $(curl -s https://api.github.com/repos/autobrr/autobrr/releases/latest | grep download | grep linux_x86_64 | cut -d\" -f4)
```

#### Unpack

Run with `root` or `sudo`. If you do not have root, or are on a shared system, place the binaries somewhere in your home
directory like `~/.bin`.

```bash
tar -C /usr/local/bin -xzf autobrr*.tar.gz
```

This will extract both `autobrr` and `autobrrctl` to `/usr/local/bin`.
Note: If the command fails, prefix it with `sudo ` and re-run again.

#### Systemd (Recommended)

On Linux-based systems, it is recommended to run autobrr as sort of a service with auto-restarting capabilities, in
order to account for potential downtime. The most common way is to do it via systemd.

You will need to create a service file in `/etc/systemd/system/` called `autobrr.service`.

```bash
touch /etc/systemd/system/autobrr@.service
```

Then place the following content inside the file (e.g. via nano/vim/ed):

```systemd title="/etc/systemd/system/autobrr@.service"
[Unit]
Description=autobrr service for %i
After=syslog.target network-online.target

[Service]
Type=simple
User=%i
Group=%i
ExecStart=/usr/bin/autobrr --config=/home/%i/.config/autobrr/

[Install]
WantedBy=multi-user.target
```

Start the service. Enable will make it startup on reboot.

```bash
systemctl enable -q --now --user autobrr@$USER
```

By default, the configuration is set to listen on `127.0.0.1`. While autobrr works fine as is exposed to the internet,
it is recommended to use a reverse proxy
like [nginx](https://autobrr.com/installation/linux#nginx), [caddy](https://autobrr.com/installation/linux#caddy)
or [traefik](https://autobrr.com/installation/docker#traefik).

If you are not running a reverse proxy change `host` in the `config.toml` to `0.0.0.0`.

## Community

Come join us on [Discord](https://discord.gg/WQ2eUycxyT)!

## License

* [GNU GPL v2 or later](https://www.gnu.org/licenses/old-licenses/gpl-2.0-standalone.html)
* Copyright 2021-2023