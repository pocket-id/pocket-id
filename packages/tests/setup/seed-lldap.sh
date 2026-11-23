#!/bin/sh
set -eu

LLDAP_HTTP_URL="http://localhost:17170"
LLDAP_ADMIN_USERNAME="admin"
LLDAP_ADMIN_PASSWORD="admin_password"
LLDAP_TOKEN=""

login() {
  response="$(
    jq -n \
      --arg username "$LLDAP_ADMIN_USERNAME" \
      --arg password "$LLDAP_ADMIN_PASSWORD" \
      '{username: $username, password: $password}' |
      curl -fsS \
        -X POST \
        -H 'Content-Type: application/json' \
        --data-binary @- \
        "$LLDAP_HTTP_URL/auth/simple/login"
  )"

  LLDAP_TOKEN="$(printf '%s' "$response" | jq -r '.token // empty')"
  if [ -z "$LLDAP_TOKEN" ]; then
    echo "Failed to authenticate to LLDAP" >&2
    exit 1
  fi
}

graphql() {
  query="$1"
  if [ "$#" -ge 2 ]; then
    variables="$2"
  else
    variables="{}"
  fi

  response="$(
    jq -cn \
      --arg query "$query" \
      --argjson variables "$variables" \
      '{query: $query, variables: $variables}' |
      curl -fsS \
        -H 'Content-Type: application/json' \
        -H "Authorization: Bearer $LLDAP_TOKEN" \
        --data-binary @- \
        "$LLDAP_HTTP_URL/api/graphql"
  )"

  errors="$(printf '%s' "$response" | jq -r '.errors[]?.message')"
  if [ -n "$errors" ]; then
    echo "$errors" >&2
    return 1
  fi

  printf '%s\n' "$response"
}

user_exists() {
  response="$(graphql '{users{id}}' '{}')"
  printf '%s' "$response" | jq -e --arg id "$1" '.data.users[]? | select(.id == $id)' >/dev/null
}

create_user() {
  id="$1"
  email="$2"
  display_name="$3"
  first_name="$4"
  last_name="$5"
  password="$6"

  variables="$(
    jq -cn \
      --arg id "$id" \
      --arg email "$email" \
      --arg displayName "$display_name" \
      --arg firstName "$first_name" \
      --arg lastName "$last_name" \
      '{user: {id: $id, email: $email, displayName: $displayName, firstName: $firstName, lastName: $lastName, avatar: ""}}'
  )"

  graphql 'mutation createUser($user:CreateUserInput!){createUser(user:$user){id}}' "$variables" >/dev/null
  lldap_set_password -b "$LLDAP_HTTP_URL" --token="$LLDAP_TOKEN" -u "$id" -p "$password"
  echo "Created user: $id"
}

create_group() {
  name="$1"
  variables="$(jq -cn --arg name "$name" '{group: $name}')"

  graphql 'mutation createGroup($group:String!){createGroup(name:$group){id}}' "$variables" >/dev/null
  echo "Created group: $name"
}

get_group_id() {
  name="$1"
  response="$(graphql '{groups{id displayName}}' '{}')"
  group_id="$(printf '%s' "$response" | jq -r --arg name "$name" '.data.groups[]? | select(.displayName == $name) | .id' | head -n 1)"

  if [ -z "$group_id" ]; then
    echo "Failed to retrieve group ID for group: $name" >&2
    return 1
  fi

  printf '%s\n' "$group_id"
}

update_group_display_name() {
  name="$1"
  display_name="$2"
  group_id="$(get_group_id "$name")"
  variables="$(
    jq -cn \
      --argjson id "$group_id" \
      --arg displayName "$display_name" \
      '{group: {id: $id, insertAttributes: {name: "display_name", value: $displayName}}}'
  )"

  graphql 'mutation updateGroup($group:UpdateGroupInput!){updateGroup(group:$group){ok}}' "$variables" >/dev/null
  echo "Attribute set for group: $name, attribute: display_name, value: $display_name"
}

add_user_to_group() {
  user_id="$1"
  group_name="$2"
  group_id="$(get_group_id "$group_name")"
  variables="$(
    jq -cn \
      --arg userId "$user_id" \
      --argjson groupId "$group_id" \
      '{userId: $userId, groupId: $groupId}'
  )"

  graphql 'mutation addUserToGroup($userId:String!,$groupId:Int!){addUserToGroup(userId:$userId,groupId:$groupId){ok}}' "$variables" >/dev/null
}

add_user_to_group_with_retry() {
  user_id="$1"
  group_name="$2"
  i=1

  while [ "$i" -le 3 ]; do
    echo "Attempt $i to add $user_id to $group_name"
    if add_user_to_group "$user_id" "$group_name"; then
      echo "Successfully added $user_id to $group_name"
      return 0
    fi

    if [ "$i" -eq 3 ]; then
      echo "Warning: Could not add $user_id to $group_name after 3 attempts"
      return 0
    fi

    echo "Failed to add $user_id to $group_name, retrying in 2 seconds..."
    sleep 2
    i=$((i + 1))
  done
}

# Wait for LLDAP to start
i=1
while [ "$i" -le 15 ]; do
  if curl -s --fail "$LLDAP_HTTP_URL/api/healthcheck" >/dev/null; then
    echo "LLDAP is ready"
    break
  fi
  if [ "$i" -eq 15 ]; then
    echo "LLDAP failed to start in time"
    exit 1
  fi
  echo "Waiting for LLDAP... ($i/15)"
  sleep 3
  i=$((i + 1))
done

login

echo "Checking if data is already seeded..."
if user_exists "testuser1"; then
  echo "Data already seeded, skipping setup."
  exit 0
fi

echo "Setting up LLDAP test data..."

echo "Creating test users..."
create_user "testuser1" "testuser1@pocket-id.org" "Test User 1" "Test" "User" "password123"
create_user "testuser2" "testuser2@pocket-id.org" "Test User 2" "Test2" "User2" "password123"

echo "Creating test groups..."
create_group "test_group"
sleep 1
update_group_display_name "test_group" "test_group"

create_group "admin_group"
sleep 1
update_group_display_name "admin_group" "admin_group"

echo "Adding users to groups..."
add_user_to_group_with_retry "testuser1" "test_group"
add_user_to_group_with_retry "testuser2" "admin_group"

echo "LLDAP test data setup complete"
