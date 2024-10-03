#!/bin/bash

if [ ! -f /etc/systemd/system/navidrome.service ]; then
    navidrome service install --user navidrome --working-directory /var/lib/navidrome --configfile /etc/navidrome/navidrome.toml
fi
