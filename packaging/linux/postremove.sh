#!/bin/sh

postinstall_flag="/var/lib/navidrome/.installed"

if [ -f "$postinstall_flag" ]; then
    navidrome service uninstall
    rm "$postinstall_flag"

    printf "The following may still be present (especially if you have not done a purge):\n"
    printf "1. /etc/navidrome/navidrome.toml (configuration file)\n"
    printf "2. /var/lib/navidrome (database/cache)\n"
    printf "3. /opt/navidrome (default location for music)\n"
    printf "4. The Navidrome user (user name navidrome)\n"
fi
