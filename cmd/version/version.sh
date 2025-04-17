#!/bin/bash

set -e  # Exit immediately if a command exits with a non-zero status

# Enable case-insensitive matching
shopt -s nocasematch

# Function to display usage information
usage() {
    echo "Usage: $0 -v <version> -t <title>"
    echo "  -v, --version    Current version in semantic versioning format (e.g., 1.2.3). Defaults to 0.0.0"
    echo "  -t, --title      Pull Request title containing [major] or [minor] to indicate version increment (required)"
    exit 1
}

# Function to determine which part of the version to increment
determine_part() {
    local title="$1"
    if [[ "$title" =~ \[major\] ]]; then
        echo "major"
    elif [[ "$title" =~ \[minor\] ]]; then
        echo "minor"
    else
        echo "patch"
    fi
}

# Function to increment the version based on the specified part
increment_version() {
    local version="$1"
    local part="$2"

    IFS='.' read -r major minor patch <<< "$version"

    # Validate that version parts are numeric
    if ! [[ "$major" =~ ^[0-9]+$ && "$minor" =~ ^[0-9]+$ && "$patch" =~ ^[0-9]+$ ]]; then
        echo "Error: Invalid version format: $version" >&2
        exit 1
    fi

    case "$part" in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            echo "Error: Unknown part to increment: $part" >&2
            exit 1
            ;;
    esac

    echo "${major}.${minor}.${patch}"
}

# Default version if not provided
VERSION="0.0.0"

# Parse command-line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -v|--version)
            if [[ -n "$2" && ! "$2" =~ ^- ]]; then
                VERSION="$2"
                shift 2
            else
                echo "Error: --version requires a non-empty argument." >&2
                usage
            fi
            ;;
        -t|--title)
            if [[ -n "$2" && ! "$2" =~ ^- ]]; then
                TITLE="$2"
                shift 2
            else
                echo "Error: --title requires a non-empty argument." >&2
                usage
            fi
            ;;
        -*|--*)
            echo "Unknown option: $1" >&2
            usage
            ;;
        *)
            echo "Unknown argument: $1" >&2
            usage
            ;;
    esac
done

# Ensure that the title is provided
if [[ -z "$TITLE" ]]; then
    echo "Error: -t|--title flag is required" >&2
    usage
fi

# Determine which part to increment based on the title
PART=$(determine_part "$TITLE")

# Increment the version accordingly
NEW_VERSION=$(increment_version "$VERSION" "$PART")

# Output the new version
echo "$NEW_VERSION"

# If running in GitHub Actions, write the new version to GITHUB_OUTPUT
if [[ -n "$GITHUB_OUTPUT" ]]; then
    echo "new_version=$NEW_VERSION" >> "$GITHUB_OUTPUT"
fi

