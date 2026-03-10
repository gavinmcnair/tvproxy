#!/bin/sh
set -e

PUID=${PUID:-1000}
PGID=${PGID:-1000}

# Update tvproxy group/user to match requested IDs
if [ "$(id -g tvproxy)" != "$PGID" ]; then
  groupmod -o -g "$PGID" tvproxy
fi
if [ "$(id -u tvproxy)" != "$PUID" ]; then
  usermod -o -u "$PUID" tvproxy
fi

# Ensure /data is writable
chown "$PUID:$PGID" /data

exec gosu tvproxy tvproxy "$@"
