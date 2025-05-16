#!/bin/sh

DB_PATH="./backend/data/pocket-id.db"
DB_PROVIDER="${DB_PROVIDER:=sqlite}"

# Parse command-line arguments for the -d flag (database path)
while getopts ":d:" opt; do
    case $opt in
    d)
        DB_PATH="$OPTARG"
        ;;
    \?)
        echo "Invalid option -$OPTARG" >&2
        exit 1
        ;;
    esac
done

# Shift past the processed options
shift $((OPTIND - 1))

# Ensure username or email is provided as a parameter
USER_IDENTIFIER="$1"
if [ -z "$USER_IDENTIFIER" ]; then
    echo "Usage: $0 [-d <database_path>] <username or email>"
    if [ "$DB_PROVIDER" == "sqlite" ]; then
        echo "-d <database_path> (optional): Path to the SQLite database file. Default: $DB_PATH"
    fi
    exit 1
fi

# Check and try to install the required commands
check_and_install() {
    local cmd=$1
    local pkg=$2

    if
        ! command -v "$cmd" &
        >/dev/null
    then
        if
            command -v apk &
            >/dev/null
        then
            echo "$cmd not found. Installing..."
            apk add "$pkg" --no-cache
        else
            echo "$cmd is not installed, please install it manually."
            exit 1
        fi
    fi
}

check_and_install uuidgen uuidgen
if [ "$DB_PROVIDER" == "postgres" ]; then
    check_and_install psql postgresql-client
elif [ "$DB_PROVIDER" == "sqlite" ]; then
    check_and_install sqlite3 sqlite
fi

# Generate a 16-character alphanumeric secret token
SECRET_TOKEN=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 16)

# Get the current Unix timestamp for creation and expiration (1 hour from now)
CREATED_AT=$(date +%s)
EXPIRES_AT=$((CREATED_AT + 3600))

# Retrieve user_id based on username or email and insert token
if [ "$DB_PROVIDER" == "postgres" ]; then
    if [ -z "$POSTGRES_CONNECTION_STRING" ]; then
        echo "Error: POSTGRES_CONNECTION_STRING must be set when using PostgreSQL."
        exit 1
    fi

    # Retrieve user_id
    USER_ID=$(psql "$POSTGRES_CONNECTION_STRING" -Atc "SELECT id FROM users WHERE username='$USER_IDENTIFIER' OR email='$USER_IDENTIFIER';")

    if [ -z "$USER_ID" ]; then
        echo "User not found for username/email: $USER_IDENTIFIER"
        exit 1
    fi

    # Insert the one-time token
    psql "$POSTGRES_CONNECTION_STRING" <<EOF
INSERT INTO one_time_access_tokens (id, created_at, token, expires_at, user_id)
VALUES ('$(uuidgen)', to_timestamp('$CREATED_AT'), '$SECRET_TOKEN', to_timestamp('$EXPIRES_AT'), '$USER_ID');
EOF

elif [ "$DB_PROVIDER" == "sqlite" ]; then
    # Retrieve user_id
    USER_ID=$(sqlite3 "$DB_PATH" "SELECT id FROM users WHERE username='$USER_IDENTIFIER' OR email='$USER_IDENTIFIER';")

    if [ -z "$USER_ID" ]; then
        echo "User not found for username/email: $USER_IDENTIFIER"
        exit 1
    fi

    # Insert the one-time token
    sqlite3 "$DB_PATH" <<EOF
INSERT INTO one_time_access_tokens (id, created_at, token, expires_at, user_id)
VALUES ('$(uuidgen)', '$CREATED_AT', '$SECRET_TOKEN', '$EXPIRES_AT', '$USER_ID');
EOF
else
    echo "Error: Invalid DB_PROVIDER. Must be 'postgres' or 'sqlite'."
    exit 1
fi

echo "================================================="
if [ $? -eq 0 ]; then
    echo "A one-time access token valid for 1 hour has been created for \"$USER_IDENTIFIER\"."
    echo "Use the following URL to sign in once: ${APP_URL:=https://<your-pocket-id-domain>}/lc/$SECRET_TOKEN"
else
    echo "Error creating access token."
    exit 1
fi
echo "================================================="
