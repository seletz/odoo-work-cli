#!/usr/bin/env bash
set -euo pipefail

# Get the latest version tag, default to v0.0.0
LATEST=$(git tag -l 'v*' --sort=-v:refname | head -n1)
if [ -z "$LATEST" ]; then
    LATEST="v0.0.0"
fi

# The release workflow refuses stale generated docs (screenshots, command
# reference, man pages); catch that before the tag exists.
if ! mise run docs:check; then
    echo "" >&2
    echo "Generated docs are stale; the release would fail. Run 'mise run docs:generate'," >&2
    echo "commit the result, and run the release again." >&2
    exit 1
fi

# A release needs release notes: the Unreleased section of the changelog is
# the text of the GitHub release and the "Release notes" page of the site.
if ! go run ./tools/changelog check; then
    echo "" >&2
    echo "Describe the changes in the [Unreleased] section of CHANGELOG.md, merge that," >&2
    echo "and run the release again." >&2
    exit 1
fi

echo "Current version: $LATEST"
echo ""
echo "Select bump type:"
echo "  1) patch"
echo "  2) minor"
echo "  3) major"
echo ""
read -rp "Choice [1/2/3]: " CHOICE

# Strip leading 'v' for arithmetic
VERSION="${LATEST#v}"
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

case "$CHOICE" in
    1|patch)
        PATCH=$((PATCH + 1))
        ;;
    2|minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    3|major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    *)
        echo "Invalid choice"
        exit 1
        ;;
esac

NEXT="v${MAJOR}.${MINOR}.${PATCH}"

echo ""
read -rp "Create release $NEXT? [y/N]: " CONFIRM
if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
    echo "Aborted."
    exit 0
fi

# Cut the release into the changelog before tagging, so the tag contains
# the section the release workflow publishes.
echo "Updating CHANGELOG.md for $NEXT..."
go run ./tools/changelog release "$NEXT" "$(date +%Y-%m-%d)"
git add CHANGELOG.md
git commit -S -m "docs: changelog for $NEXT"

echo "Tagging $NEXT..."
git tag -s "$NEXT" -m "Release $NEXT"

echo "Pushing the changelog commit and the tag to origin..."
git push origin HEAD
git push origin "$NEXT"

echo ""
echo "Done! Tag $NEXT pushed."
echo "The Release workflow now verifies (tests, lint, docs), runs GoReleaser (artifacts,"
echo "GitHub release, Homebrew tap) and deploys the docs site."
