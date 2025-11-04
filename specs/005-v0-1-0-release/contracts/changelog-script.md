# CHANGELOG Generation Script Contract

**Script**: `scripts/generate-changelog.sh`
**Purpose**: Generate CHANGELOG.md from git commit history following Keep a Changelog format

## Script Interface

### Invocation
```bash
./scripts/generate-changelog.sh [OPTIONS]
```

### Options
- `--from-commit <SHA>`: Start commit (default: initial commit)
- `--to-commit <SHA>`: End commit (default: HEAD)
- `--output <PATH>`: Output file (default: CHANGELOG.md)
- `--version <VERSION>`: Release version for header (default: "Unreleased")
- `--dry-run`: Print to stdout instead of writing file
- `--help`: Show usage help

### Exit Codes
- `0`: Success (changelog generated)
- `1`: Error (missing git, invalid commit range, write failure)

## Input

### Git Commit Log
```bash
# Example git log output parsed by script
git log --pretty=format:"%H|%h|%an|%aI|%s" <from-commit>..<to-commit>

# Format:
# Full SHA | Short SHA | Author | ISO Date | Subject
a1b2c3d4e5f6|a1b2c3d|Brandon Williams|2025-11-04T12:00:00Z|feat(cli): add version command
```

### Conventional Commit Format
```
<type>(<scope>): <description>
<type>: <description>
<type>!: <description> (breaking change)
```

**Supported Types**:
- `feat` → "Features"
- `fix` → "Bug Fixes"
- `perf` → "Performance"
- `test` → "Tests"
- `docs` → "Documentation"
- `refactor` → "Refactoring"
- `chore` → "Chores"
- (others) → "Uncategorized"

## Output

### CHANGELOG.md Format (Keep a Changelog 1.1.0)

```markdown
# Changelog

All notable changes to gonuget will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-11-04

### Features
- Add version command for CLI ([a1b2c3d](https://github.com/willibrandon/gonuget/commit/a1b2c3d))
- Implement solution file support ([b2c3d4e](https://github.com/willibrandon/gonuget/commit/b2c3d4e))

### Bug Fixes
- Fix race condition in resolver cache ([c3d4e5f](https://github.com/willibrandon/gonuget/commit/c3d4e5f))

### Performance
- Optimize version comparison with zero allocations ([d4e5f6g](https://github.com/willibrandon/gonuget/commit/d4e5f6g))

### Tests
- Add resolver advanced interop tests ([e5f6g7h](https://github.com/willibrandon/gonuget/commit/e5f6g7h))

### Documentation
- Add CLI user guide ([f6g7h8i](https://github.com/willibrandon/gonuget/commit/f6g7h8i))

### Refactoring
- Extract solution parsing to library package ([g7h8i9j](https://github.com/willibrandon/gonuget/commit/g7h8i9j))

### Chores
- Update CI workflow to Go 1.23 ([h8i9j0k](https://github.com/willibrandon/gonuget/commit/h8i9j0k))

### Uncategorized
- Update README ([i9j0k1l](https://github.com/willibrandon/gonuget/commit/i9j0k1l))

[Unreleased]: https://github.com/willibrandon/gonuget/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/willibrandon/gonuget/releases/tag/v0.1.0
```

## Processing Logic

### 1. Parse Conventional Commits
```bash
# Regex pattern for conventional commit parsing
REGEX='^([a-z]+)(\(([a-zA-Z0-9_-]+)\))?(!)?: (.+)$'

# Parse commit subject
if [[ "$subject" =~ $REGEX ]]; then
    type="${BASH_REMATCH[1]}"
    scope="${BASH_REMATCH[3]}"
    breaking="${BASH_REMATCH[4]}"
    description="${BASH_REMATCH[5]}"
else
    # Not conventional format → Uncategorized
    type="uncategorized"
    description="$subject"
fi
```

### 2. Categorize Commits
```bash
# Map commit type to CHANGELOG category
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
```

### 3. Format Entry
```bash
# Format: - Description ([short-sha](commit-url))
entry="- ${description} ([${short_sha}](${commit_url}))"
```

### 4. Group by Category
```bash
# Store entries in associative arrays by category
declare -A entries
entries["Features"]+="${entry}\n"
entries["Bug Fixes"]+="${entry}\n"
# ... etc
```

### 5. Generate Markdown
```bash
# Write header
echo "# Changelog" > CHANGELOG.md
echo "" >> CHANGELOG.md
echo "All notable changes to gonuget will be documented in this file." >> CHANGELOG.md
echo "" >> CHANGELOG.md
echo "The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)," >> CHANGELOG.md
echo "and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)." >> CHANGELOG.md
echo "" >> CHANGELOG.md
echo "## [Unreleased]" >> CHANGELOG.md
echo "" >> CHANGELOG.md
echo "## [${VERSION}] - ${DATE}" >> CHANGELOG.md

# Write sections
for category in "Features" "Bug Fixes" "Performance" "Tests" "Documentation" "Refactoring" "Chores" "Uncategorized"; do
    if [[ -n "${entries[$category]}" ]]; then
        echo "" >> CHANGELOG.md
        echo "### $category" >> CHANGELOG.md
        echo -e "${entries[$category]}" >> CHANGELOG.md
    fi
done

# Write version links
echo "" >> CHANGELOG.md
echo "[Unreleased]: https://github.com/willibrandon/gonuget/compare/v${VERSION}...HEAD" >> CHANGELOG.md
echo "[${VERSION}]: https://github.com/willibrandon/gonuget/releases/tag/v${VERSION}" >> CHANGELOG.md
```

## Edge Cases

### 1. Malformed Commit Messages
```bash
# Example: Commit with invalid characters or missing colon
# "feat add version command" (missing colon)
# Result: Categorized as "Uncategorized"
if [[ "$subject" =~ $REGEX ]]; then
    # Conventional format
else
    # Fallback to uncategorized
    type="uncategorized"
    description="$subject"  # Use entire subject as description
fi
```

### 2. Merge Commits
```bash
# Example: "Merge pull request #123 from user/feature"
# Result: Skip merge commits (no-op)
if [[ "$subject" =~ ^Merge ]]; then
    continue  # Skip to next commit
fi
```

### 3. Empty Commit Range
```bash
# Example: No commits between from-commit and to-commit
# Result: Generate CHANGELOG with "No changes" message
if [[ $(git log --oneline $from_commit..$to_commit | wc -l) -eq 0 ]]; then
    echo "## [${VERSION}] - ${DATE}" >> CHANGELOG.md
    echo "" >> CHANGELOG.md
    echo "No changes." >> CHANGELOG.md
    exit 0
fi
```

### 4. No Git Repository
```bash
# Example: Script run outside git repository
# Result: Error exit code 1
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "Error: Not a git repository" >&2
    exit 1
fi
```

## Validation

### Pre-conditions
1. Must be run from repository root (where .git directory exists)
2. Git must be installed and accessible in PATH
3. Write permissions for output file location

### Post-conditions
1. CHANGELOG.md exists at specified output path
2. File contains valid markdown syntax
3. All commits in range are categorized
4. Version links at footer are correct

### Testing
```bash
# Test 1: Generate changelog for v0.1.0
./scripts/generate-changelog.sh --version "0.1.0" --from-commit $(git rev-list --max-parents=0 HEAD) --to-commit HEAD

# Test 2: Dry-run mode (stdout)
./scripts/generate-changelog.sh --dry-run | grep -q "## \[0.1.0\]"

# Test 3: Custom output path
./scripts/generate-changelog.sh --output /tmp/CHANGELOG-test.md
test -f /tmp/CHANGELOG-test.md

# Test 4: Verify conventional commit parsing
echo "feat: test commit" | ./scripts/generate-changelog.sh --dry-run | grep -q "Features"
echo "Random commit message" | ./scripts/generate-changelog.sh --dry-run | grep -q "Uncategorized"
```

## Integration

### Manual Execution
```bash
# Generate CHANGELOG for v0.1.0 from all commits
cd /Users/brandon/src/gonuget
./scripts/generate-changelog.sh --version "0.1.0"
```

### Pre-Release Checklist
```bash
# Step 1: Generate CHANGELOG
./scripts/generate-changelog.sh --version "0.1.0"

# Step 2: Review Uncategorized section
grep -A 100 "### Uncategorized" CHANGELOG.md

# Step 3: Manually edit if needed (fix typos, reword entries)
$EDITOR CHANGELOG.md

# Step 4: Commit CHANGELOG
git add CHANGELOG.md
git commit -m "docs: add CHANGELOG for v0.1.0"
```

### CI Integration (Future)
```yaml
# .github/workflows/release.yml
- name: Generate CHANGELOG
  run: |
    ./scripts/generate-changelog.sh --version "${{ github.ref_name }}" --from-commit $(git describe --tags --abbrev=0 2>/dev/null || git rev-list --max-parents=0 HEAD)
```

## Dependencies

- `git` command (version 2.0+)
- `bash` shell (version 4.0+ for associative arrays)
- POSIX utilities: `date`, `grep`, `sed`, `awk`

## Security

- No network access required (local git operations only)
- No sensitive data in output (public commit history)
- No code execution from commit messages (sanitized markdown)
