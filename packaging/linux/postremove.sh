#!/bin/sh

# Adapted from https://github.com/kardianos/service/blob/becf2eb62b83ed01f5e782cb8da7bb739ded2bb5/service_systemd_linux.go#L22
is_systemd() {
    if [ -e /run/systemd/system ]; then
        return 0
    elif type systemctl > /dev/null 2>&1; then
        return 0
    elif [ "$(cat /proc/1/comm)" = "systemd" ]; then
        return 0
    else
        return 1
    fi
}

# Adapted from https://github.com/kardianos/service/blob/becf2eb62b83ed01f5e782cb8da7bb739ded2bb5/service_openrc_linux.go#L16
is_openrc() {
    if type openrc-init > /dev/null 2>&1; then
        return 0
    elif expr "$(cat /proc/1/comm)" : "::sysinit:.*openrc.*sysinit" > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

if is_systemd; then
    printf "Stopping, disabling, and removing Navidrome service (systemd)\n"
    systemctl disable --now navidrome ||:
    rm -rf /etc/systemd/system/navidrome.service
elif is_openrc; then
    printf "Openrc-specific cleanup\n"
else
    printf "Systemv init specific cleanup\n"
fi

rm  -f /var/lib/navidrome/.installed

printf "The following may still be present (especially if you have not done a purge):\n"
printf "1. /etc/navidrome/navidrome.toml (configuration file)\n"
printf "2. /var/lib/navidrome (database/cache)\n"
printf "3. /opt/navidrome (default location for music)\n"
printf "4. The Navidrome user (user name navidrome)\n"
