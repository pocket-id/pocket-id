#!/bin/bash
set -e

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

# Verify setup
echo "Listing users:"
lldap-cli user list
echo "Listing groups:"
lldap-cli group list