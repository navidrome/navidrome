#!/bin/sh

# It is possible for a user to delete the configuration file in such a way that
# the package manager (in particular, deb) thinks that the file exists, while it is 
# no longer on disk. Specifically, doing a `rm /etc/navidrome/navidrome.toml` 
# without something like `apt purge navidrome` will result in the system believing that
# the file still exists. In this case, during isntall it will NOT extract the configuration 
# file (as to not override it). Since `navidrome service install` depends on this file existing,
# we will create it with the defaults anyway.
if [ ! -f /etc/navidrome/navidrome.toml ]; then
    printf "No navidrome.toml detected, creating in postinstall\n"
    printf "DataFolder = \"/var/lib/navidrome\"\n" > /etc/navidrome/navidrome.toml
    printf "MusicFolder = \"/opt/navidrome/music\"\n" >> /etc/navidrome/navidrome.toml
fi

# `navidrome service install`` will just put the service file here. Only do this if it is the first time
if [ ! -f /etc/systemd/system/navidrome.service ]; then
    navidrome service install --user navidrome --working-directory /var/lib/navidrome --configfile /etc/navidrome/navidrome.toml
    # Generally good to run after adding a new service file
    systemctl daemon-reload ||:
fi
