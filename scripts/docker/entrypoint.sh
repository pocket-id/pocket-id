#!/bin/sh

# Ensure we are in the /app folder
cd /app

# If we aren't running as root, just exec the CMD
if [ "$(id -u)" -ne 0 ] ; then
    exec "$@"
    exit 0
fi

PUID=${PUID:-1000}
PGID=${PGID:-1000}

# Check if the group with PGID exists; if not, create it
if ! getent group pocket-id-group > /dev/null 2>&1; then
    echo "Creating group $PGID..."
    addgroup -g "$PGID" pocket-id-group
fi

# Check if a user with PUID exists; if not, create it
if ! id -u pocket-id > /dev/null 2>&1; then
    if ! getent passwd "$PUID" > /dev/null 2>&1; then
        echo "Creating user $PUID..."
        adduser -u "$PUID" -G pocket-id-group pocket-id > /dev/null 2>&1
    else
        # If a user with the PUID already exists, use that user
        existing_user=$(getent passwd "$PUID" | cut -d: -f1)
        echo "Using existing user: $existing_user"
    fi
fi

# Change ownership of the /app/data directory
mkdir -p /app/data
find /app/data \( ! -group "${PGID}" -o ! -user "${PUID}" \) -exec chown "${PUID}:${PGID}" {} +

# Switch to the non-root user
exec su-exec "$PUID:$PGID" "$@"
