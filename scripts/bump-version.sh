#!/bin/bash

# Version bump script for EntityStore
# Usage: ./bump-version.sh [patch|minor|major]

set -e

# Check if type is provided
if [ -z "$1" ]; then
    echo "Usage: $0 [patch|minor|major]"
    exit 1
fi

TYPE=$1
VERSION_FILE="VERSION"

# Check if VERSION file exists
if [ ! -f "$VERSION_FILE" ]; then
    echo "VERSION file not found!"
    exit 1
fi

# Read current version
CURRENT_VERSION=$(cat $VERSION_FILE)
echo "Current version: $CURRENT_VERSION"

# Parse version components
IFS='.' read -r -a VERSION_PARTS <<< "$CURRENT_VERSION"
MAJOR="${VERSION_PARTS[0]}"
MINOR="${VERSION_PARTS[1]}"
PATCH="${VERSION_PARTS[2]}"

# Bump version based on type
case $TYPE in
    patch)
        PATCH=$((PATCH + 1))
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    *)
        echo "Invalid version type. Use: patch, minor, or major"
        exit 1
        ;;
esac

# Create new version
NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
echo "New version: $NEW_VERSION"

# Update VERSION file
echo "$NEW_VERSION" > $VERSION_FILE

# Update version.go
sed -i.bak "s/Version = \".*\"/Version = \"$NEW_VERSION\"/" version.go && rm version.go.bak

# Update CHANGELOG.md
TODAY=$(date +%Y-%m-%d)
sed -i.bak "s/## \[Unreleased\]/## [Unreleased]\n\n## [$NEW_VERSION] - $TODAY/" CHANGELOG.md && rm CHANGELOG.md.bak

echo "Version bumped to $NEW_VERSION"
echo ""
echo "Next steps:"
echo "1. Review and update CHANGELOG.md with release notes"
echo "2. Commit changes: git add -A && git commit -m \"Bump version to $NEW_VERSION\""
echo "3. Create release: make release"