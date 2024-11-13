#!/bin/sh

if ! getent passwd navidrome > /dev/null 2>&1; then
    printf "Creating default Navidrome user\n"
    useradd --home-dir /var/lib/navidrome --create-home --system --user-group navidrome
fi
