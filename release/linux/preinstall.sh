#!/bin/sh

if ! getent passwd navidrome > /dev/null 2>&1; then
    printf "Creating default Navidrome user\n"
    # SELinux denies useradd's /etc/skel copy into /var/lib (var_lib_t), so
    # --create-home logs an AVC on enforcing distros. Create the dir ourselves,
    # with the same 0700 mode useradd --create-home would have set.
    useradd --home-dir /var/lib/navidrome --no-create-home --system --user-group navidrome
    install -d -m 700 -o navidrome -g navidrome /var/lib/navidrome
fi
