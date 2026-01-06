# <div align="center"><img  src="https://github.com/user-attachments/assets/4ceb2708-9f29-4694-b797-be833efce17d" width="100"/> </br>Pocket ID</div>

Pocket ID is a simple OIDC provider that allows users to authenticate with their passkeys to your services.

→ Try out the [Demo](https://demo.pocket-id.org)

<img src="https://github.com/user-attachments/assets/1e99ba44-76da-4b47-9b8a-dbe9b7f84512" width="1200"/>

The goal of Pocket ID is to be a simple and easy-to-use. There are other self-hosted OIDC providers like [Keycloak](https://www.keycloak.org/) or [ORY Hydra](https://www.ory.sh/hydra/) but they are often too complex for simple use cases.

Additionally, what makes Pocket ID special is that it only supports [passkey](https://www.passkeys.io/) authentication, which means you don’t need a password. Some people might not like this idea at first, but I believe passkeys are the future, and once you try them, you’ll love them. For example, you can now use a physical Yubikey to sign in to all your self-hosted services easily and securely.

## Setup

Pocket ID can be set up in multiple ways. The easiest and recommended way is to use Docker.

Visit the [documentation](https://docs.pocket-id.org) for the setup guide and more information.

## CLI Commands

Pocket ID includes a comprehensive CLI for administration and automation. The CLI supports both direct database access and API-based operations.

### Key Features:
- **Initial Setup**: Create first admin user for fresh installations
- **User Management**: Create, list, update, and delete users
- **OIDC Client Management**: Create, list, update, and delete OIDC clients
- **User Group Management**: Create, list, update, and delete user groups
- **API Key Management**: Generate and manage API keys
- **Application Configuration**: View and update application settings
- **Application Images**: Manage logos, backgrounds, favicons, and profile pictures
- **Custom Claims Management**: Manage custom claims for users and user groups
- **SCIM Service Providers**: Create, update, delete, and sync SCIM service providers
- **Hybrid Approach**: Some commands work directly with database, others use REST API
- **Multiple Output Formats**: JSON, YAML, or table format
- **Environment Variable Support**: Use `POCKET_ID_API_KEY` env var for authentication

### Quick Start:
```bash
# Option 1: Using CLI command (for fresh installations)
pocket-id setup create-admin --username "admin" --first-name "Admin" --email "admin@example.com"

# Option 2: Declarative via environment variables (for automated deployments)
export INITIAL_ADMIN_USERNAME="admin"
export INITIAL_ADMIN_FIRST_NAME="Admin"
export INITIAL_ADMIN_EMAIL="admin@example.com"
# Then start Pocket ID - it will automatically create the admin user

# Generate an API key for a user (direct database access)
pocket-id api-key generate "admin" --name "CLI Key" --show-token

# Set the API key as environment variable
export POCKET_ID_API_KEY="generated-token-here"

# List users via API
pocket-id users list --format table

# Manage OIDC clients
pocket-id oidc-clients list --format table
pocket-id oidc-clients create --name "My App" --callback-urls "https://app.example.com/callback"

# Manage user groups
pocket-id user-groups list --format table
pocket-id user-groups create --name "developers" --friendly-name "Development Team"

# Create a new user
pocket-id users create --username "newuser" --first-name "John" --display-name "John Doe"
```

For complete CLI documentation with examples, see [CLI_USAGE.md](CLI_USAGE.md).

### Available Command Groups:
- `pocket-id setup` - Initial setup commands
- `pocket-id users` - User management (list, get, create, update, delete, me, update-me)
- `pocket-id oidc-clients` - OIDC client management (list, get, create, update, delete, create-secret, update-allowed-groups)
- `pocket-id user-groups` - User group management (list, get, create, update, delete, update-users, update-allowed-clients)
- `pocket-id api-key` - API key management (generate, create, list, revoke)
- `pocket-id app-config` - Application configuration (get, get-all, update, test-email, sync-ldap)
- `pocket-id app-images` - Application images (update, delete for logos, backgrounds, favicons, profile pictures)
- `pocket-id custom-claims` - Custom claims management (suggestions, update-user, update-user-group)
- `pocket-id scim` - SCIM service providers (create, update, delete, sync)
- `pocket-id export` - Export database
- `pocket-id import` - Import database
- `pocket-id healthcheck` - Check server health
- `pocket-id version` - Show version
- `pocket-id key-rotate` - Rotate encryption keys
- `pocket-id one-time-access-token` - Generate one-time access tokens

## Declarative Initial Admin Creation

Pocket ID supports declarative initial admin creation through environment variables, making it ideal for automated deployments:

```bash
# Required fields (triggers automatic admin creation)
export INITIAL_ADMIN_USERNAME="admin"
export INITIAL_ADMIN_FIRST_NAME="Admin"
export INITIAL_ADMIN_EMAIL="admin@example.com"

# Optional field
export INITIAL_ADMIN_LAST_NAME="User"
```

When `INITIAL_ADMIN_USERNAME` is set, Pocket ID will automatically create the initial admin user on startup if the database is empty (requires `INITIAL_ADMIN_FIRST_NAME` and `INITIAL_ADMIN_EMAIL` to also be set). The access token will be logged for you to save securely. Email is required for admin user creation.

## Contribute

You're very welcome to contribute to Pocket ID! Please follow the [contribution guide](/CONTRIBUTING.md) to get started.
