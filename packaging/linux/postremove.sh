#!/bin/sh

printf "Stopping, disabling, and removing Navidrome service\n"
systemctl disable --now navidrome ||:
rm -rf /etc/systemd/system/navidrome.service

printf "The following may still be present (especially if you have not done a purge):\n"
printf "1. /etc/navidrome/navidrome.toml (configuration file)\n"
printf "2. /var/lib/navidrome (database/cache)\n"
printf "3. /opt/navidrome (default location for music)\n"
printf "4. The Navidrome user (user name navidrome)\n"
