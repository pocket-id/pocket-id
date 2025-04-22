echo "Starting frontend..."
node frontend/build &

echo "Starting backend..."
cd backend && ./pocket-id-backend &

if [ "$CADDY_DISABLED" != "true" ]; then
  echo "Starting Caddy..."

  # https://caddyserver.com/docs/conventions#data-directory
  export XDG_DATA_HOME=${XDG_DATA_HOME:-/app/backend/data/.local/share}
  # https://caddyserver.com/docs/conventions#configuration-directory
  export XDG_CONFIG_HOME=${XDG_CONFIG_HOME:-/app/backend/data/.config}

  # Check if TRUST_PROXY is set to true and use the appropriate Caddyfile
  if [ "$TRUST_PROXY" = "true" ]; then
    caddy run --adapter caddyfile --config /etc/caddy/Caddyfile.trust-proxy &
  else
    caddy run --adapter caddyfile --config /etc/caddy/Caddyfile &
  fi
else
  echo "Caddy is disabled. Skipping..."
fi

# Set up trap to catch child process terminations
trap 'exit 1' SIGCHLD

wait
