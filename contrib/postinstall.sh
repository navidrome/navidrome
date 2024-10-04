#!/bin/sh

if [ ! -f /etc/navidrome/navidrome.toml ]; then
    printf "No navidrome.toml detected, creating in postinstall\n"
    printf "DataFolder = \"/var/lib/navidrome\"\n" > /etc/navidrome/navidrome.toml
    printf "MusicFolder = \"/opt/navidrome/music\"\n" >> /etc/navidrome/navidrome.toml
fi

if [ ! -f /etc/systemd/system/navidrome.service ]; then
    navidrome service install --user navidrome --working-directory /var/lib/navidrome --configfile /etc/navidrome/navidrome.toml
fi
