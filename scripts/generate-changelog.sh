#!/bin/bash
#
# generate-changelog.sh - Generate CHANGELOG.md from git commit history
#
# Usage:
#   ./scripts/generate-changelog.sh [OPTIONS]
#
# Options:
#   --from-commit <SHA>    Start commit (default: initial commit)
#   --to-commit <SHA>      End commit (default: HEAD)
#   --output <PATH>        Output file (default: CHANGELOG.md)
#   --version <VERSION>    Release version for header (default: "Unreleased")
#   --dry-run              Print to stdout instead of writing file
#   --help                 Show usage help
#

set -e

# Default values
FROM_COMMIT=""
TO_COMMIT="HEAD"
OUTPUT_FILE="CHANGELOG.md"
VERSION="Unreleased"
DRY_RUN=false
REPO_OWNER="willibrandon"
REPO_NAME="gonuget"

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --from-commit)
            FROM_COMMIT="$2"
            shift 2
            ;;
        --to-commit)
            TO_COMMIT="$2"
            shift 2
            ;;
        --output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --from-commit <SHA>    Start commit (default: initial commit)"
            echo "  --to-commit <SHA>      End commit (default: HEAD)"
            echo "  --output <PATH>        Output file (default: CHANGELOG.md)"
            echo "  --version <VERSION>    Release version for header (default: \"Unreleased\")"
            echo "  --dry-run              Print to stdout instead of writing file"
            echo "  --help                 Show usage help"
            exit 0
            ;;
        *)
            echo "Error: Unknown option: $1" >&2
            echo "Run '$0 --help' for usage information." >&2
            exit 1
            ;;
    esac
done

# Verify we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "Error: Not a git repository" >&2
    exit 1
fi

# If FROM_COMMIT not specified, use initial commit
if [ -z "$FROM_COMMIT" ]; then
    FROM_COMMIT=$(git rev-list --max-parents=0 HEAD)
fi

# Get commit range
if [ "$FROM_COMMIT" = "$TO_COMMIT" ]; then
    COMMIT_RANGE="$FROM_COMMIT"
else
    COMMIT_RANGE="$FROM_COMMIT..$TO_COMMIT"
fi

# Check if there are commits in range
COMMIT_COUNT=$(git log --oneline "$COMMIT_RANGE" 2>/dev/null | wc -l | tr -d ' ')
if [ "$COMMIT_COUNT" -eq 0 ]; then
    echo "Error: No commits in range $COMMIT_RANGE" >&2
    exit 1
fi

# Regex pattern for conventional commit parsing
REGEX='^([a-z]+)(\(([a-zA-Z0-9_-]+)\))?(!)?: (.+)$'

# Temporary files to store entries by category
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

# Category order for output
CATEGORIES=("Features" "Bug Fixes" "Performance" "Tests" "Documentation" "Refactoring" "Chores" "Uncategorized")

# Create temp files for each category
for category in "${CATEGORIES[@]}"; do
    touch "$TMPDIR/$category.txt"
done

# Process each commit
git log --pretty=format:"%H|%h|%an|%aI|%s" "$COMMIT_RANGE" | while IFS='|' read -r full_sha short_sha author date subject; do
    # Skip merge commits
    if echo "$subject" | grep -q "^Merge"; then
        continue
    fi

    # Parse conventional commit format using grep and sed
    if echo "$subject" | grep -qE "$REGEX"; then
        # Extract type (first capture group)
        type=$(echo "$subject" | sed -E 's/^([a-z]+)(\(([a-zA-Z0-9_-]+)\))?(!)?: (.+)$/\1/')
        description=$(echo "$subject" | sed -E 's/^([a-z]+)(\(([a-zA-Z0-9_-]+)\))?(!)?: (.+)$/\5/')

        # Map commit type to category
        case "$type" in
            feat)      category="Features" ;;
            fix)       category="Bug Fixes" ;;
            perf)      category="Performance" ;;
            test)      category="Tests" ;;
            docs)      category="Documentation" ;;
            refactor)  category="Refactoring" ;;
            chore)     category="Chores" ;;
            *)         category="Uncategorized" ;;
        esac
    else
        # Not conventional format → Uncategorized
        category="Uncategorized"
        description="$subject"
    fi

    # Format entry with commit link
    commit_url="https://github.com/$REPO_OWNER/$REPO_NAME/commit/$short_sha"
    entry="- $description ([$short_sha]($commit_url))"

    # Append to category file
    echo "$entry" >> "$TMPDIR/$category.txt"
done

# Generate current date for version
DATE=$(date -u +%Y-%m-%d)

# Generate CHANGELOG content
generate_changelog() {
    echo "# Changelog"
    echo ""
    echo "All notable changes to gonuget will be documented in this file."
    echo ""
    echo "The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),"
    echo "and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)."
    echo ""
    echo "## [Unreleased]"
    echo ""

    if [ "$VERSION" != "Unreleased" ]; then
        echo "## [$VERSION] - $DATE"

        # Write sections in order
        for category in "${CATEGORIES[@]}"; do
            if [ -s "$TMPDIR/$category.txt" ]; then
                echo ""
                echo "### $category"
                cat "$TMPDIR/$category.txt"
            fi
        done

        # Write version links
        echo ""
        echo "[Unreleased]: https://github.com/$REPO_OWNER/$REPO_NAME/compare/v$VERSION...HEAD"
        echo "[$VERSION]: https://github.com/$REPO_OWNER/$REPO_NAME/releases/tag/v$VERSION"
    else
        # Just Unreleased section
        for category in "${CATEGORIES[@]}"; do
            if [ -s "$TMPDIR/$category.txt" ]; then
                echo ""
                echo "### $category"
                cat "$TMPDIR/$category.txt"
            fi
        done

        echo ""
        echo "[Unreleased]: https://github.com/$REPO_OWNER/$REPO_NAME/compare/HEAD...HEAD"
    fi
}

# Output CHANGELOG
if [ "$DRY_RUN" = true ]; then
    generate_changelog
else
    generate_changelog > "$OUTPUT_FILE"
    echo "Generated $OUTPUT_FILE with $COMMIT_COUNT commits"
fi
