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

postinstall_flag="/var/lib/navidrome/.installed"

if [ ! -f "$postinstall_flag" ]; then
    # The primary reason why this would fail is if the service was already installed AND
    # someone manually removed the .installed flag. In this case, ignore the error
    navidrome service install --user navidrome --working-directory /var/lib/navidrome --configfile /etc/navidrome/navidrome.toml || :
    # Any `navidrome` command will make a cache. Make sure that this is properly owned by the Navidrome user
    # and not by root
    chown navidrome:navidrome /var/lib/navidrome/cache
    touch "$postinstall_flag"
fi


