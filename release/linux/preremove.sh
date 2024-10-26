#!/bin/sh

action=$1

remove() {
    postinstall_flag="/var/lib/navidrome/.installed"

    if [ -f "$postinstall_flag" ]; then
        # If this fails, ignore it
        navidrome service uninstall || :
        rm "$postinstall_flag"

        printf "The following may still be present (especially if you have not done a purge):\n"
        printf "1. /etc/navidrome/navidrome.toml (configuration file)\n"
        printf "2. /var/lib/navidrome (database/cache)\n"
        printf "3. /opt/navidrome (default location for music)\n"
        printf "4. The Navidrome user (user name navidrome)\n"
    fi
}

case "$action" in 
    "1" | "upgrade")
        # For an upgrade, do nothing
        # Leave the service file untouched
        # This is relevant for RPM/DEB-based installs
        ;;
    *)
        remove
        ;;
esac
