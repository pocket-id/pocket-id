#!/bin/bash
set -e

echo "Setting up LLDAP container..."

# Create network if needed (will fail gracefully if exists)
docker network create pocket-id-network || true

# Run LLDAP container
docker run -d --name lldap \
  --network pocket-id-network \
  -p 3890:3890 \
  -p 17170:17170 \
  -e LLDAP_JWT_SECRET=secret \
  -e LLDAP_LDAP_USER_PASS=admin_password \
  -e LLDAP_LDAP_BASE_DN="dc=pocket-id,dc=org" \
  nitnelave/lldap:stable

# Wait for LLDAP to start
for i in {1..15}; do
  if curl -s --fail http://localhost:17170/api/healthcheck > /dev/null; then
    echo "LLDAP is ready"
    break
  fi
  if [ $i -eq 15 ]; then
    echo "LLDAP failed to start in time"
    exit 1
  fi
  echo "Waiting for LLDAP... ($i/15)"
  sleep 3
done

echo "LLDAP container setup complete"