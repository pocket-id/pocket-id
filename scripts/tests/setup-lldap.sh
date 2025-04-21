#!/bin/bash
set -e

echo 'deb http://download.opensuse.org/repositories/home:/Masgalor:/LLDAP/xUbuntu_24.04/ /' | sudo tee /etc/apt/sources.list.d/home:Masgalor:LLDAP.list
curl -fsSL https://download.opensuse.org/repositories/home:Masgalor:LLDAP/xUbuntu_24.04/Release.key | gpg --dearmor | sudo tee /etc/apt/trusted.gpg.d/home_Masgalor_LLDAP.gpg > /dev/null
sudo apt-get update
sudo apt-get install -y lldap-cli lldap-set-password

echo "Setting up LLDAP container..."

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

echo "Setting up LLDAP test data..."

# Configure LLDAP CLI connection via environment variables
export LLDAP_HTTPURL="http://localhost:17170"
export LLDAP_USERNAME="admin"
export LLDAP_PASSWORD="admin_password"

# Create test users using the user add command
echo "Creating test users..."
lldap-cli user add "testuser1" "testuser1@pocket-id.org" \
  -p "password123" \
  -d "Test User 1" \
  -f "Test" \
  -l "User"
  
lldap-cli user add "testuser2" "testuser2@pocket-id.org" \
  -p "password123" \
  -d "Test User 2" \
  -f "Test2" \
  -l "User2"

# Create test groups
echo "Creating test groups..."
lldap-cli group add "test_group"
sleep 1
lldap-cli group update set "test_group" "display_name" "test_group"

lldap-cli group add "admin_group"
sleep 1
lldap-cli group update set "admin_group" "display_name" "admin_group"

# Add users to groups with retry logic
echo "Adding users to groups..."
for i in {1..3}; do
  echo "Attempt $i to add testuser1 to test_group"
  if lldap-cli user group add "testuser1" "test_group"; then
    echo "Successfully added testuser1 to test_group"
    break
  else
    echo "Failed to add testuser1 to test_group, retrying in 2 seconds..."
    sleep 2
  fi
  
  if [ $i -eq 3 ]; then
    echo "Warning: Could not add testuser1 to test_group after 3 attempts"
  fi
done

for i in {1..3}; do
  echo "Attempt $i to add testuser2 to admin_group"
  if lldap-cli user group add "testuser2" "admin_group"; then
    echo "Successfully added testuser2 to admin_group"
    break
  else
    echo "Failed to add testuser2 to admin_group, retrying in 2 seconds..."
    sleep 2
  fi
  
  if [ $i -eq 3 ]; then
    echo "Warning: Could not add testuser2 to admin_group after 3 attempts"
  fi
done

echo "LLDAP test data setup complete"