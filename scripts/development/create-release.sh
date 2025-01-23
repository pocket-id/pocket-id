# Check if the script is being run from the root of the project
if [ ! -f .version ] || [ ! -f frontend/package.json ] || [ ! -f CHANGELOG.md ]; then
    echo "Error: This script must be run from the root of the project."
    exit 1
fi

# Read the current version from .version
VERSION=$(cat .version)

# Function to increment the version
increment_version() {
    local version=$1
    local part=$2

    IFS='.' read -r -a parts <<<"$version"
    if [ "$part" == "minor" ]; then
        parts[1]=$((parts[1] + 1))
        parts[2]=0
    elif [ "$part" == "patch" ]; then
        parts[2]=$((parts[2] + 1))
    fi
    echo "${parts[0]}.${parts[1]}.${parts[2]}"
}

RELEASE_TYPE=$1

if [ "$RELEASE_TYPE" == "minor" ]; then
    echo "Performing minor release..."
    NEW_VERSION=$(increment_version $VERSION minor)
elif [ "$RELEASE_TYPE" == "patch" ]; then
    echo "Performing patch release..."
    NEW_VERSION=$(increment_version $VERSION patch)
else
    echo "Invalid release type. Please enter either 'minor' or 'patch'."
    exit 1
fi

# Confirm release creation
read -p "This will create a new $RELEASE_TYPE release with version $NEW_VERSION. Do you want to proceed? (y/n) " CONFIRM
if [[ "$CONFIRM" != "y" ]]; then
    echo "Release process canceled."
    exit 1
fi

# Update the .version file with the new version
echo $NEW_VERSION >.version
git add .version

# Update version in frontend/package.json
jq --arg new_version "$NEW_VERSION" '.version = $new_version' frontend/package.json >frontend/package_tmp.json && mv frontend/package_tmp.json frontend/package.json
git add frontend/package.json

# Check if conventional-changelog is installed, if not install it
if ! command -v conventional-changelog &>/dev/null; then
    echo "conventional-changelog not found, installing..."
    npm install -g conventional-changelog-cli
fi

# Generate changelog
echo "Generating changelog..."
conventional-changelog -p conventionalcommits -i CHANGELOG.md -s
git add CHANGELOG.md

# Commit the changes with the new version
git commit -m "release: $NEW_VERSION"

# Create a Git tag with the new version
git tag "v$NEW_VERSION"

# Push the commit and the tag to the repository
git push
git push --tags

# Check if GitHub CLI is installed
if ! command -v gh &>/dev/null; then
    echo "GitHub CLI (gh) is not installed. Please install it and authenticate using 'gh auth login'."
    exit 1
fi

# Extract the changelog content for the latest release
echo "Extracting changelog content for version $NEW_VERSION..."
CHANGELOG=$(awk '/^## / {if (NR > 1) exit} NR > 1 {print}' CHANGELOG.md | awk 'NR > 2 || NF {print}')

if [ -z "$CHANGELOG" ]; then
    echo "Error: Could not extract changelog for version $NEW_VERSION."
    exit 1
fi

# Create the release on GitHub
echo "Creating GitHub release..."
gh release create "v$NEW_VERSION" --title "v$NEW_VERSION" --notes "$CHANGELOG"

if [ $? -eq 0 ]; then
    echo "GitHub release created successfully."
else
    echo "Error: Failed to create GitHub release."
    exit 1
fi

echo "Release process complete. New version: $NEW_VERSION"
