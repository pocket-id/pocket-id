#!/bin/bash

# Check if the script is being run from the root of the project
if [ ! -f .version ] || [ ! -f frontend/package.json ] || [ ! -f CHANGELOG.md ]; then
    echo "Error: This script must be run from the root of the project."
    exit 1
fi

# Check if git cliff is installed
if ! command -v git-cliff &>/dev/null; then
    echo "Error: git cliff is not installed. Please install it from https://git-cliff.org/docs/installation."
    exit 1
fi

# Check if GitHub CLI is installed
if ! command -v gh &>/dev/null; then
    echo "Error: GitHub CLI (gh) is not installed. Please install it and authenticate using 'gh auth login'."
    exit 1
fi

# Check if Snyk CLI is installed
if ! command -v snyk &>/dev/null; then
    echo "Error: Snyk CLI is not installed. Please install it and authenticate using 'snyk auth'."
    exit 1
fi

# Check if we're on the main branch
if [ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]; then
    echo "Error: This script must be run on the main branch."
    exit 1
fi

# Parse command line arguments
FORCE_MAJOR=false
for arg in "$@"; do
    case $arg in
    --major)
        FORCE_MAJOR=true
        shift
        ;;
    *)
        # Unknown option
        ;;
    esac
done

BUMP_ARGUMENTS=(--bumped-version --unreleased --offline)
if [ "$FORCE_MAJOR" == true ]; then
    BUMP_ARGUMENTS+=(--bump major)
fi

# Calculate the next version from the unreleased conventional commits
if ! NEW_VERSION=$(git cliff "${BUMP_ARGUMENTS[@]}"); then
    echo "Error: Could not calculate the next version."
    exit 1
fi
NEW_VERSION=${NEW_VERSION#v}

if [ "$NEW_VERSION" == "$(cat .version)" ]; then
    echo "No commits requiring a version bump found since the latest release. No new release will be created."
    exit 0
fi

echo "Running Snyk dependency scan..."
if ! snyk test --all-projects --dev --detection-depth=3 --strict-out-of-sync=false --severity-threshold=high; then
    echo "Error: Snyk detected high-severity vulnerable dependencies. Release creation aborted."
    exit 1
fi

# Confirm release creation
read -p "This will create a new release with version $NEW_VERSION. Do you want to proceed? (y/n) " CONFIRM
if [[ "$CONFIRM" != "y" ]]; then
    echo "Release process canceled."
    exit 1
fi

# Update the .version file with the new version
echo $NEW_VERSION >.version
git add .version

# Update version in frontend/package.json
jq --arg new_version "$NEW_VERSION" '.version = $new_version' frontend/package.json >frontend/package_tmp.json && mv frontend/package_tmp.json frontend/package.json
pnpm --dir frontend exec prettier --write package.json
git add frontend/package.json

# Generate changelog
echo "Generating changelog..."
git cliff --github-token=$(gh auth token) --prepend CHANGELOG.md --tag "v$NEW_VERSION" --unreleased
git add CHANGELOG.md

# Commit the changes with the new version
git commit -m "release: $NEW_VERSION"

# Create a Git tag with the new version
git tag "v$NEW_VERSION"

# Push the commit and the tag to the repository
git push
git push --tags

# Extract the changelog content for the latest release
echo "Extracting changelog content for version $NEW_VERSION..."
CHANGELOG=$(awk '/^## v[0-9]/ { if (found) exit; found=1; next } found' CHANGELOG.md)

if [ -z "$CHANGELOG" ]; then
    echo "Error: Could not extract changelog for version $NEW_VERSION."
    exit 1
fi

# Create the release on GitHub
echo "Creating GitHub release..."
gh release create "v$NEW_VERSION" --title "v$NEW_VERSION" --notes "$CHANGELOG" --draft

if [ $? -eq 0 ]; then
    echo "GitHub release created successfully."
else
    echo "Error: Failed to create GitHub release."
    exit 1
fi

echo "Release process complete. New version: $NEW_VERSION"
