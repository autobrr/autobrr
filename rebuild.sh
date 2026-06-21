#!/bin/bash

# set -e

# cd /home/nihal/Downloads/xx/auto/nihal/autobrr

sudo systemctl stop autobrr@nihal.service || true

sudo systemctl disable autobrr@nihal.service || true

# sudo rm -f /usr/bin/autobrr /usr/bin/autobrrctl

sudo rm -f /etc/systemd/system/autobrr@nihal.service

rm -rf ~/.config/autobrr

# make clean || true

# make build

# sudo install -Dm755 bin/autobrr /usr/bin/autobrr

# sudo install -Dm755 bin/autobrrctl /usr/bin/autobrrctl

sudo tee /etc/systemd/system/autobrr@nihal.service > /dev/null << 'EOF'
[Unit]
Description=autobrr service for %i
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%i
ExecStart=/usr/bin/autobrr
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload

sudo systemctl enable autobrr@nihal.service

sudo pnpm --dir web run build

sudo systemctl start autobrr@nihal.service

sleep 2

sudo systemctl status autobrr@nihal.service
