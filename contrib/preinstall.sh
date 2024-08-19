#!/bin/bash

if ! getent passwd navidrome &> /dev/null; then
    useradd --home-dir /var/lib/navidrome --create-home --system --user-group navidrome
fi
