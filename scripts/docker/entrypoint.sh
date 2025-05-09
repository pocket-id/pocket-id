# For backwards compatibility we have to switch into the backend directory
cd /app/backend

# If we aren't running as root, just exec the CMD
[ "$(id -u)" -ne 0 ] && exec "$@"


echo "Creating user and group..."

PUID=${PUID:-1000}
PGID=${PGID:-1000}

# Check if the group with PGID exists; if not, create it
if ! getent group pocket-id-group > /dev/null 2>&1; then
    addgroup -g "$PGID" pocket-id-group
fi

# Check if a user with PUID exists; if not, create it
if ! id -u pocket-id > /dev/null 2>&1; then
    if ! getent passwd "$PUID" > /dev/null 2>&1; then
        adduser -u "$PUID" -G pocket-id-group pocket-id
    else
        # If a user with the PUID already exists, use that user
        existing_user=$(getent passwd "$PUID" | cut -d: -f1)
        echo "Using existing user: $existing_user"
    fi
fi

# Change ownership of the /app directory
mkdir -p /app/backend/data
find /app/backend/data \( ! -group "${PGID}" -o ! -user "${PUID}" \) -exec chown "${PUID}:${PGID}" {} +

# Switch to the non-root user
exec su-exec "$PUID:$PGID" "$@"