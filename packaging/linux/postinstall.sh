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

if [ ! -f "$postinstall_flag" ]; then
    navidrome service install --user navidrome --working-directory /var/lib/navidrome --configfile /etc/navidrome/navidrome.toml
    touch "$postinstall_flag"

    if is_systemd; then
        # Generally good to run after adding a new service file
        systemctl daemon-reload ||:
    elif is_openrc; then
        printf "openrc init\n"
    else
        printf "systemv init\n" 
    fi
fi


