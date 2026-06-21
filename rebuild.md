#!/bin/bash
# Sample Commands to build it from source code & replace with existing binary 
# Works with Debian 12, Arch Linux (Both tested, ARM64 & AMD64 both working)
# If you face any issue like package missing or go not found, install it, just chatgpt it ! 


make clean
make build
make PREFIX=/usr install
make build/ctl
sudo install -Dm755 bin/autobrrctl /usr/bin/autobrrctl

sudo systemctl daemon-reload
sudo systemctl restart autobrr@nihal.service

# Below sqlite3 command may be needed or not sure, need to cross verify
# If you face any issue, create issue under github repo. 

sqlite3 ~/.config/autobrr/autobrr.db "ALTER TABLE feed ADD COLUMN user_agent TEXT;"

sudo systemctl restart autobrr@nihal.service

sudo systemctl status autobrr@nihal.service

# Autobrr, RSS Feed -> User agent update by @nihalxx3 
