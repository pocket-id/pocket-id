# Pocket-ID CLI Usage Guide

This guide covers the new CLI commands for interacting with Pocket-ID via its REST API.

## Table of Contents
- [Authentication](#authentication)
- [Global Flags](#global-flags)
- [Initial Setup](#initial-setup)
- [User Management](#user-management)
- [OIDC Client Management](#oidc-client-management)
- [User Group Management](#user-group-management)
- [API Key Management](#api-key-management)
- [Application Configuration](#application-configuration)
- [Application Images](#application-images)
- [Custom Claims Management](#custom-claims-management)
- [SCIM Service Providers](#scim-service-providers)
- [Examples](#examples)
- [Environment Variables](#environment-variables)

## Authentication

Most CLI commands require authentication via API key. You can provide the API key in two ways:

1. **Command-line flag**: `--api-key` or `-k`
2. **Environment variable**: `POCKET_ID_API_KEY`

```bash
# Using flag
pocket-id --api-key "your-api-key-here" users list

# Using environment variable
export POCKET_ID_API_KEY="your-api-key-here"
pocket-id users list
```

## Global Flags

These flags are available for all commands:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--endpoint` | `-e` | API endpoint URL | `http://localhost:1411` |
| `--api-key` | `-k` | API key for authentication | (none) |
| `--format` | `-f` | Output format: `json`, `yaml`, or `table` | `json` |

## Initial Setup

### Create Initial Admin User
You have two options for creating the initial admin user:

#### Option 1: Using CLI Command
This command creates the first admin user for a fresh Pocket ID installation. It only works when no users exist in the database yet.

```bash
# Create initial admin user
pocket-id setup create-admin \
  --username "admin" \
  --first-name "Admin" \
  --email "admin@example.com" \
  --last-name "User"

# With all required fields
pocket-id setup create-admin --username "admin" --first-name "Admin" --email "admin@example.com"
```

After running this command, you'll receive an access token that can be used to:
1. Log in to the web interface
2. Generate API keys for automation

#### Option 2: Declarative via Environment Variables
For automated deployments, you can create the initial admin user declaratively using environment variables. Set these variables before starting Pocket ID:

```bash
# Required fields (triggers automatic admin creation)
export INITIAL_ADMIN_USERNAME="admin"
export INITIAL_ADMIN_FIRST_NAME="Admin"
export INITIAL_ADMIN_EMAIL="admin@example.com"

# Optional field
export INITIAL_ADMIN_LAST_NAME="User"
```

When Pocket ID starts with `INITIAL_ADMIN_USERNAME` set, it will automatically:
1. Check if any users exist in the database
2. Create the admin user if the database is empty (requires `INITIAL_ADMIN_FIRST_NAME` and `INITIAL_ADMIN_EMAIL` to also be set)
3. Log the access token (save this securely!)
4. Continue with normal startup

Note: Email is required for admin user creation. If `INITIAL_ADMIN_EMAIL` is not set, the admin user will not be created.

This is especially useful for:
- Containerized deployments (Docker, Kubernetes)
- Infrastructure as Code (Terraform, Ansible)
- Automated provisioning scripts

## User Management

### List Users
```bash
# List all users
pocket-id users list

# List users with pagination and search
pocket-id users list --page 2 --limit 50 --search "john"

# List users in table format
pocket-id users list --format table
```

### Get User
```bash
# Get user by ID
pocket-id users get <user-id>

# Get current authenticated user
pocket-id users me
```

### Create User
```bash
# Create a new user
pocket-id users create \
  --username "johndoe" \
  --first-name "John" \
  --last-name "Doe" \
  --display-name "John Doe" \
  --email "john@example.com" \
  --admin true \
  --locale "en-US"
```

### Update User
```bash
# Update user by ID
pocket-id users update <user-id> \
  --email "newemail@example.com" \
  --display-name "John Smith"

# Update current user
pocket-id users update-me \
  --first-name "Jonathan" \
  --locale "en-GB"
```

### Delete User
```bash
# Delete user by ID (with confirmation)
pocket-id users delete <user-id>

# Force delete without confirmation
pocket-id users delete <user-id> --force
```

## OIDC Client Management

### List OIDC Clients
```bash
# List all OIDC clients
pocket-id oidc-clients list --format table

# List with pagination and search
pocket-id oidc-clients list --page 1 --limit 20 --search "myapp" --format table
```

### Get OIDC Client
```bash
# Get client details by ID
pocket-id oidc-clients get <client-id>

# Get with JSON output
pocket-id oidc-clients get <client-id> --format json
```

### Create OIDC Client
```bash
# Create a new OIDC client
pocket-id oidc-clients create \
  --name "My Application" \
  --callback-urls "https://app.example.com/callback" \
  --logout-callback-urls "https://app.example.com/logout" \
  --pkce-enabled true \
  --public false

# Create with custom ID and launch URL
pocket-id oidc-clients create \
  --id "my-custom-client-id" \
  --name "Production App" \
  --callback-urls "https://prod.example.com/callback" \
  --launch-url "https://prod.example.com" \
  --group-restricted true
```

### Update OIDC Client
```bash
# Update client details
pocket-id oidc-clients update <client-id> \
  --name "Updated App Name" \
  --callback-urls "https://new.example.com/callback"

# Make client public
pocket-id oidc-clients update <client-id> --public true
```

### Delete OIDC Client
```bash
# Delete client by ID (with confirmation)
pocket-id oidc-clients delete <client-id>

# Force delete without confirmation
pocket-id oidc-clients delete <client-id> --force
```

### Manage Client Secrets
```bash
# Generate new client secret
pocket-id oidc-clients create-secret <client-id>

# Output secret in table format (recommended for copying)
pocket-id oidc-clients create-secret <client-id> --format table
```

### Manage Allowed User Groups
```bash
# Update allowed user groups for a client
pocket-id oidc-clients update-allowed-groups <client-id> \
  --group-ids "group-id-1,group-id-2,group-id-3"
```

## User Group Management

### List User Groups
```bash
# List all user groups
pocket-id user-groups list --format table

# List with pagination
pocket-id user-groups list --page 2 --limit 10 --format table
```

### Get User Group
```bash
# Get group details by ID
pocket-id user-groups get <group-id>

# Get with JSON output
pocket-id user-groups get <group-id> --format json
```

### Create User Group
```bash
# Create a new user group
pocket-id user-groups create \
  --name "developers" \
  --friendly-name "Development Team"

# Create another group
pocket-id user-groups create \
  --name "admins" \
  --friendly-name "Administrators"
```

### Update User Group
```bash
# Update group details
pocket-id user-groups update <group-id> \
  --name "senior-developers" \
  --friendly-name "Senior Development Team"
```

### Delete User Group
```bash
# Delete group by ID (with confirmation)
pocket-id user-groups delete <group-id>

# Force delete without confirmation
pocket-id user-groups delete <group-id> --force
```

### Manage Group Members
```bash
# Update users in a group
pocket-id user-groups update-users <group-id> \
  --user-ids "user-id-1,user-id-2,user-id-3"
```

### Manage Allowed OIDC Clients
```bash
# Update allowed OIDC clients for a group
pocket-id user-groups update-allowed-clients <group-id> \
  --client-ids "client-id-1,client-id-2"
```

## API Key Management

### Generate API Key (Direct Database Access)
This command works without a running server and creates API keys directly in the database.

```bash
# Generate API key for a user
pocket-id api-key generate "username-or-email" \
  --name "My CLI Key" \
  --description "For automation scripts" \
  --expires-in "720h" \  # 30 days
  --show-token

# Common duration formats:
# --expires-in "24h"    # 24 hours
# --expires-in "7d"     # 7 days
# --expires-in "30d"    # 30 days
# --expires-in "1y"     # 1 year
```

### Create API Key (Via API)
This command requires a running server and uses an existing API key for authentication.

```bash
# Create API key for current user
pocket-id api-key create \
  --name "New API Key" \
  --description "For webhooks" \
  --expires-in "90d"
```

### List API Keys
```bash
# List API keys for current user
pocket-id api-key list

# List with pagination
pocket-id api-key list --page 1 --limit 20 --format table
```

### Revoke API Key
```bash
# Revoke API key by ID
pocket-id api-key revoke <key-id>

### Force revoke without confirmation
pocket-id api-key revoke <key-id> --force
```

## Application Configuration

### Get Public Configuration
```bash
# Get public application configuration
pocket-id app-config get --format table

# Get as JSON
pocket-id app-config get --format json
```

### Get All Configuration (Including Private)
```bash
# Get all configuration including private settings
pocket-id app-config get-all --format table

# Get as YAML
pocket-id app-config get-all --format yaml
```

### Update Configuration
```bash
# Update configuration from JSON file
pocket-id app-config update --file config.json

# Update configuration from JSON string
pocket-id app-config update --config '{"appName": "My App", "sessionDuration": "24h"}'

# Update configuration from stdin
echo '{"appName": "My App", "sessionDuration": "24h"}' | pocket-id app-config update
```

### Test Email Configuration
```bash
# Send test email to verify email configuration
pocket-id app-config test-email
```

### Sync LDAP
```bash
# Manually trigger LDAP synchronization
pocket-id app-config sync-ldap
```

## Application Images



### Manage Application Images
```bash
# Update light mode logo
pocket-id app-images update logo logo-light.png

# Update dark mode logo
pocket-id app-images update logo logo-dark.png --light=false

# Update email logo
pocket-id app-images update email email-logo.png

# Update background image
pocket-id app-images update background background-image.png

# Update favicon (.ico file required)
pocket-id app-images update favicon favicon.ico

# Update default profile picture
pocket-id app-images update default-profile-picture avatar.png
```

### Delete Default Profile Picture
```bash
# Delete default profile picture (restores to default)
pocket-id app-images delete default-profile-picture
```

## Custom Claims Management

### Get Custom Claim Suggestions
```bash
# Get suggested custom claim names
pocket-id custom-claims suggestions --format table

# Get as JSON
pocket-id custom-claims suggestions --format json
```

### Update Custom Claims for a User
```bash
# Update custom claims for a user from JSON file
pocket-id custom-claims update-user <user-id> --file claims.json

# Update custom claims for a user from JSON string
pocket-id custom-claims update-user <user-id> --claims '[{"key": "department", "value": "Engineering"}, {"key": "role", "value": "Developer"}]'

# Update custom claims for a user from stdin
echo '[{"key": "department", "value": "Engineering"}]' | pocket-id custom-claims update-user <user-id>
```

### Update Custom Claims for a User Group
```bash
# Update custom claims for a user group from JSON file
pocket-id custom-claims update-user-group <group-id> --file claims.json

# Update custom claims for a user group from JSON string
pocket-id custom-claims update-user-group <group-id> --claims '[{"key": "department", "value": "Marketing"}, {"key": "location", "value": "Remote"}]'

# Update custom claims for a user group from stdin
echo '[{"key": "location", "value": "Remote"}]' | pocket-id custom-claims update-user-group <group-id>
```

## SCIM Service Providers

### Create SCIM Service Provider
```bash
# Create SCIM service provider with required parameters
pocket-id scim create \
  --endpoint "https://scim.example.com/v2" \
  --oidc-client-id "<oidc-client-id>" \
  --token "scim-auth-token"

# Create SCIM service provider from JSON file
pocket-id scim create --file scim-config.json
```

### Update SCIM Service Provider
```bash
# Update SCIM service provider with new endpoint
pocket-id scim update <provider-id> --endpoint "https://new-scim.example.com/v2"

# Update SCIM service provider with new token
pocket-id scim update <provider-id> --token "new-scim-auth-token"

# Update SCIM service provider from JSON file
pocket-id scim update <provider-id> --file updated-config.json

# Update SCIM service provider from stdin
echo '{"endpoint": "https://updated.example.com/v2", "oidcClientId": "<client-id>"}' | pocket-id scim update <provider-id>
```

### Sync SCIM Service Provider
```bash
# Trigger synchronization for a SCIM service provider
pocket-id scim sync <provider-id>
```

### Delete SCIM Service Provider
```bash
# Delete SCIM service provider (with confirmation)
pocket-id scim delete <provider-id>

# Force delete without confirmation
pocket-id scim delete <provider-id> --force
```

## Examples

### Complete Workflow Example
```bash
# 1. First, create the initial admin user (if starting fresh)
pocket-id setup create-admin \
  --username "admin" \
  --first-name "Admin" \
  --email "admin@example.com"

# 2. Generate an API key for the admin user (direct database access)
pocket-id api-key generate "admin" \
  --name "Admin CLI Key" \
  --expires-in "1y" \
  --show-token

# 3. Save the generated token as environment variable
export POCKET_ID_API_KEY="generated-token-here"

# 4. List all users
pocket-id users list --format table

# 5. Create a new user
pocket-id users create \
  --username "newuser" \
  --first-name "Alice" \
  --display-name "Alice Smith" \
  --email "alice@example.com"

# 6. Generate an API key for the new user
pocket-id api-key generate "newuser" \
  --name "Alice's Automation Key" \
  --expires-in "30d" \
  --show-token

# 7. Create user groups
pocket-id user-groups create \
  --name "developers" \
  --friendly-name "Development Team"

pocket-id user-groups create \
  --name "admins" \
  --friendly-name "Administrators"

# 8. Add users to groups
pocket-id user-groups update-users <developers-group-id> \
  --user-ids "newuser-id"

# 9. Create OIDC clients
pocket-id oidc-clients create \
  --name "Internal Tools" \
  --callback-urls "https://tools.example.com/callback" \
  --group-restricted true

# 10. Configure group access to OIDC clients
pocket-id oidc-clients update-allowed-groups <internal-tools-client-id> \
  --group-ids "developers-group-id,admins-group-id"

# 11. Generate client secret
pocket-id oidc-clients create-secret <internal-tools-client-id> --format table
```

### Automation Script Example
```bash
#!/bin/bash
# automation.sh

export POCKET_ID_API_KEY="${POCKET_ID_API_KEY}"
ENDPOINT="http://localhost:1411"

# Get user count
echo "Getting user statistics..."
pocket-id --endpoint "$ENDPOINT" users list --format json | \
  jq '.pagination.totalItems'

# Get OIDC client count
echo "Getting OIDC client statistics..."
pocket-id --endpoint "$ENDPOINT" oidc-clients list --format json | \
  jq '.pagination.totalItems'

# Get user group count
echo "Getting user group statistics..."
pocket-id --endpoint "$ENDPOINT" user-groups list --format json | \
  jq '.pagination.totalItems'

# Create backup user
echo "Creating backup admin user..."
pocket-id --endpoint "$ENDPOINT" users create \
  --username "backupadmin" \
  --first-name "Backup" \
  --display-name "Backup Admin" \
  --admin true \
  --format json | jq '.id'
```

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `POCKET_ID_API_KEY` | API key for authentication | `export POCKET_ID_API_KEY="abc123..."` |
| `ENCRYPTION_KEY` | Required for direct database commands | `export ENCRYPTION_KEY="0123456789abcdef"` |
| `INITIAL_ADMIN_USERNAME` | Username for the initial admin user (triggers automatic creation if set) | `export INITIAL_ADMIN_USERNAME="admin"` |
| `INITIAL_ADMIN_FIRST_NAME` | First name for the initial admin user (required when `INITIAL_ADMIN_USERNAME` is set) | `export INITIAL_ADMIN_FIRST_NAME="Admin"` |
| `INITIAL_ADMIN_EMAIL` | Email for the initial admin user (required when `INITIAL_ADMIN_USERNAME` is set) | `export INITIAL_ADMIN_EMAIL="admin@example.com"` |
| `INITIAL_ADMIN_LAST_NAME` | Last name for the initial admin user (optional) | `export INITIAL_ADMIN_LAST_NAME="User"` |

## Hybrid Approach Notes

### Setup Commands
- `setup create-admin`: Create initial admin user (API-based, no auth required)
- **Declarative Initial Admin**: You can also create the initial admin user declaratively using environment variables. Set `INITIAL_ADMIN_USERNAME`, `INITIAL_ADMIN_FIRST_NAME`, and `INITIAL_ADMIN_EMAIL` environment variables, and Pocket ID will automatically create the admin user on startup if no users exist in the database. Email is required for admin user creation.

Pocket-ID uses a hybrid approach for CLI commands:

1. **API-based commands**: User management, API key management (via API)
   - Require running server
   - Use REST API with authentication
   - Consistent with web interface behavior

2. **Direct-access commands**: `api-key generate`, `export`, `import`, `key-rotate`
3. **Setup commands**: `setup create-admin` (special case for initial configuration)
   - Work without running server
   - Direct database access
   - Useful for administration and recovery
4. **Declarative setup**: Initial admin creation via environment variables
   - Automatic creation on application startup
   - Only works when database has no users
   - Useful for containerized deployments and automation

## Troubleshooting

### Common Issues

1. **"API key is required" error**
   ```bash
   # Solution: Set API key
   export POCKET_ID_API_KEY="your-key"
   # or
   pocket-id --api-key "your-key" users list
   ```

2. **"Connection refused" error**
   ```bash
   # Solution: Check if server is running
   pocket-id healthcheck --endpoint "http://localhost:1411"
   
   # Or use direct-access commands
   pocket-id api-key generate "admin" --name "Recovery Key"
   ```

3. **"ENCRYPTION_KEY must be at least 16 bytes long"**
   ```bash
   # Solution: Set encryption key
   export ENCRYPTION_KEY="0123456789abcdef0123456789abcdef"
   ```

4. **"Setup already completed" error**
   ```bash
   # Solution: Initial admin already exists, use existing credentials
   pocket-id api-key generate "existing-admin-username" --name "New Key"
   ```

### Getting Help
```bash
# General help
pocket-id --help

# Command-specific help
pocket-id users --help
pocket-id api-key --help
pocket-id setup --help
pocket-id api-key generate --help
pocket-id setup create-admin --help
```

## Next Steps

The CLI currently supports user and API key management. Future enhancements may include:
- OIDC client management
- User group management
- Audit log viewing
- Application configuration
- Bulk operations

For feature requests or issues, please refer to the Pocket-ID documentation or GitHub repository.