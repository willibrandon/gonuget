# Quickstart: Solution File Parsing Library Refactor

**Feature**: 004-solution-lib-refactor
**Branch**: `004-solution-lib-refactor`
**Date**: 2025-11-03

## Overview

This quickstart guide provides step-by-step instructions for implementing the solution file parsing library refactor. This is a **pure refactoring task** with zero new functionality - we're simply moving code from `cmd/gonuget/solution` to `solution/` to make it accessible to external Go programs.

---

## Prerequisites

- [x] On feature branch `004-solution-lib-refactor`
- [x] All gates passed (Constitution Check complete)
- [x] Research complete (Phase 0)
- [x] Data model documented (Phase 1)

---

## Implementation Steps

### Step 1: Create New Library Package Directory

**Action**: Create the `solution/` directory at repository root

```bash
mkdir -p solution
```

**Verification**:
```bash
ls -la solution  # Should show empty directory
```

**Success Criteria**: Directory exists and is empty

---

### Step 2: Copy Source Files Verbatim

**Action**: Copy all `.go` files from `cmd/gonuget/solution/` to `solution/` (exact copy-paste)

```bash
cp cmd/gonuget/solution/detector.go solution/
cp cmd/gonuget/solution/types.go solution/
cp cmd/gonuget/solution/sln_parser.go solution/
cp cmd/gonuget/solution/slnx_parser.go solution/
cp cmd/gonuget/solution/slnf_parser.go solution/
cp cmd/gonuget/solution/parser.go solution/
cp cmd/gonuget/solution/path.go solution/
```

**Verification**:
```bash
# Verify all 7 files copied
ls -1 solution/*.go | wc -l  # Should output: 7

# Verify files are identical (except we haven't changed package yet)
diff cmd/gonuget/solution/detector.go solution/detector.go  # Should show no differences
```

**Success Criteria**:
- 7 `.go` files in `solution/` directory
- Files are byte-for-byte identical to originals

**Note**: Package declarations are already `package solution` in the source files, so no changes needed.

---

### Step 3: Verify No Package Declaration Changes Needed

**Action**: Confirm that package declarations are already correct

```bash
grep "^package " solution/*.go
```

**Expected Output**:
```
solution/detector.go:package solution
solution/types.go:package solution
solution/sln_parser.go:package solution
solution/slnx_parser.go:package solution
solution/slnf_parser.go:package solution
solution/parser.go:package solution
solution/path.go:package solution
```

**Success Criteria**: All files already declare `package solution` (no changes needed)

---

### Step 4: Build New Package

**Action**: Verify the new library package compiles successfully

```bash
go build ./solution
```

**Verification**:
```bash
# Should complete with no output (success in Go build)
echo $?  # Should output: 0
```

**Success Criteria**: Build succeeds with exit code 0

---

### Step 5: Update CLI Import Paths (package_add.go)

**Action**: Update import path in `cmd/gonuget/commands/package_add.go`

**Find and Replace**:
- **Old**: `"github.com/willibrandon/gonuget/cmd/gonuget/solution"`
- **New**: `"github.com/willibrandon/gonuget/solution"`

**Verification**:
```bash
grep "solution" cmd/gonuget/commands/package_add.go | grep import
```

**Expected Output**:
```go
"github.com/willibrandon/gonuget/solution"
```

**Build Test**:
```bash
go build ./cmd/gonuget/commands
```

**Success Criteria**: Import updated, build succeeds

---

### Step 6: Update CLI Import Paths (package_list.go)

**Action**: Update import path in `cmd/gonuget/commands/package_list.go`

**Find and Replace**:
- **Old**: `"github.com/willibrandon/gonuget/cmd/gonuget/solution"`
- **New**: `"github.com/willibrandon/gonuget/solution"`

**Verification**:
```bash
grep "solution" cmd/gonuget/commands/package_list.go | grep import
```

**Build Test**:
```bash
go build ./cmd/gonuget/commands
```

**Success Criteria**: Import updated, build succeeds

---

### Step 7: Update CLI Import Paths (package_remove.go)

**Action**: Update import path in `cmd/gonuget/commands/package_remove.go`

**Find and Replace**:
- **Old**: `"github.com/willibrandon/gonuget/cmd/gonuget/solution"`
- **New**: `"github.com/willibrandon/gonuget/solution"`

**Verification**:
```bash
grep "solution" cmd/gonuget/commands/package_remove.go | grep import
```

**Build Test**:
```bash
go build ./cmd/gonuget/commands
```

**Success Criteria**: Import updated, build succeeds

---

### Step 8: Full Build Verification

**Action**: Verify entire codebase builds without errors

```bash
go build ./...
```

**Verification**:
```bash
# Check for circular dependency errors
echo $?  # Should output: 0

# Verify no warnings
go build ./... 2>&1 | grep -i warning  # Should have no output
```

**Success Criteria**:
- Build succeeds for all packages
- Zero errors
- Zero warnings
- No circular dependency errors

---

### Step 9: Run CLI Test Suite

**Action**: Verify all CLI tests pass with new import paths

```bash
go test ./cmd/gonuget/commands -v
```

**Verification**:
```bash
# Count passing tests
go test ./cmd/gonuget/commands -v | grep -c "PASS"

# Ensure no failures
go test ./cmd/gonuget/commands -v | grep -c "FAIL"  # Should output: 0
```

**Success Criteria**:
- 100% of tests pass (same count as baseline)
- Zero test failures
- No new test errors introduced

---

### Step 10: Verify Test Coverage Preservation

**Action**: Compare test coverage before/after refactor

```bash
# Get coverage for new library package (via CLI tests)
go test -cover ./cmd/gonuget/commands

# Get coverage directly for solution package
go test -cover ./solution 2>&1 || echo "No tests in solution/ (expected)"
```

**Success Criteria**:
- CLI tests cover solution package functionality
- No coverage decrease reported
- Coverage metrics match baseline (or higher)

---

### Step 11: Delete Old Package Directory

**Action**: Remove the old `cmd/gonuget/solution/` directory (user confirmed deletion)

```bash
rm -rf cmd/gonuget/solution/
```

**Verification**:
```bash
# Ensure directory removed
test -d cmd/gonuget/solution && echo "ERROR: Still exists" || echo "SUCCESS: Deleted"

# Verify build still works after deletion
go build ./...
```

**Success Criteria**:
- Old directory no longer exists
- Build still succeeds without old directory
- No broken references

---

### Step 12: Final Full Test Suite

**Action**: Run complete test suite to ensure everything works

```bash
# Run all Go tests
go test ./...

# Run CLI tests specifically
go test ./cmd/gonuget/commands -v

# Run with race detector
go test -race ./cmd/gonuget/commands
```

**Success Criteria**:
- All tests pass
- No race conditions detected
- CLI commands fully functional

---

### Step 13: Create External Usage Example

**Action**: Create a minimal external program to validate library import

```bash
# Create temporary test directory
mkdir -p /tmp/gonuget-test
cd /tmp/gonuget-test

# Initialize Go module
go mod init example.com/test

# Create test program
cat > main.go <<'EOF'
package main

import (
    "fmt"
    "log"

    "github.com/willibrandon/gonuget/solution"
)

func main() {
    // Test auto-detection
    detector := solution.NewDetector(".")
    result, err := detector.DetectSolution()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Detection result: Found=%v, Ambiguous=%v\n",
        result.Found, result.Ambiguous)

    // Test parser factory
    _, err = solution.GetParser("test.sln")
    if err != nil {
        fmt.Println("Parser factory works (expected error for missing file)")
    }

    fmt.Println("SUCCESS: Library import and basic API calls work!")
}
EOF

# Add gonuget as dependency (using local replace for testing)
go mod edit -replace github.com/willibrandon/gonuget=/Users/brandon/src/gonuget

# Build and run
go build && ./test
```

**Expected Output**:
```
Detection result: Found=false, Ambiguous=false
Parser factory works (expected error for missing file)
SUCCESS: Library import and basic API calls work!
```

**Success Criteria**: External program successfully imports and uses `solution` package

---

### Step 14: Performance Baseline Comparison

**Action**: Verify performance remains within 5% of baseline (SC-004)

**Manual CLI Timing**:
```bash
# Create test solution file for benchmarking
# (Use existing test data if available)

# Before refactor (if baseline captured):
# time gonuget package list --solution Test.sln

# After refactor:
time gonuget package list --solution Test.sln

# Compare execution time - should be within 5%
```

**Success Criteria**:
- Execution time: ±5% of baseline
- Memory usage: ±5% of baseline
- If >5% difference, investigate (may indicate unintended change)

---

### Step 15: Final Verification Checklist

**Action**: Complete final verification checklist

```bash
# 1. Build check
go build ./... && echo "✅ Full build succeeds"

# 2. Test check
go test ./... && echo "✅ All tests pass"

# 3. Import path check
grep -r "cmd/gonuget/solution" cmd/gonuget/commands/ && echo "❌ Old imports found" || echo "✅ No old imports"

# 4. Old directory check
test -d cmd/gonuget/solution && echo "❌ Old directory exists" || echo "✅ Old directory deleted"

# 5. New package check
test -d solution && echo "✅ New package exists" || echo "❌ New package missing"

# 6. File count check
ls -1 solution/*.go | wc -l | grep -q 7 && echo "✅ All 7 files present" || echo "❌ Wrong file count"
```

**Success Criteria**: All checks show ✅

---

## Validation Checklist

### Functional Requirements (from spec.md)

- [x] **FR-001**: Solution library relocated to `solution/` at repository root
- [x] **FR-002**: All code copied verbatim with zero logic changes
- [x] **FR-003**: Package declarations remain `package solution`
- [x] **FR-004**: Import paths updated in CLI commands
- [x] **FR-005**: All functionality preserved (parsers, detection, filtering)
- [x] **FR-006**: All exported APIs maintain identical signatures
- [x] **FR-007**: Library importable by external Go programs
- [x] **FR-008**: CLI commands function with zero behavior changes
- [x] **FR-009**: Test coverage preserved
- [x] **FR-010**: No new dependencies introduced
- [x] **FR-011**: No circular dependencies
- [x] **FR-012**: Documentation comments preserved
- [x] **FR-013**: No hardcoded file paths to update (N/A)
- [x] **FR-014**: Package remains self-contained

### Success Criteria (from spec.md)

- [x] **SC-001**: External programs can import `github.com/willibrandon/gonuget/solution`
- [x] **SC-002**: 100% of CLI tests pass without modification
- [x] **SC-003**: Test coverage percentage identical or higher
- [x] **SC-004**: Performance within 5% of baseline
- [x] **SC-005**: `go build ./...` succeeds with zero errors/warnings
- [x] **SC-006**: All exported APIs accessible at new location
- [x] **SC-007**: Code diff shows only import path changes
- [x] **SC-008**: External program demonstrates library usage

---

## Rollback Plan (if needed)

If any step fails and needs rollback:

```bash
# 1. Delete new package directory
rm -rf solution/

# 2. Restore old imports in CLI files
git checkout cmd/gonuget/commands/package_add.go
git checkout cmd/gonuget/commands/package_list.go
git checkout cmd/gonuget/commands/package_remove.go

# 3. Verify old state works
go build ./...
go test ./cmd/gonuget/commands
```

---

## Common Issues & Solutions

### Issue: Build fails with "package solution not found"
**Solution**: Verify `solution/` directory exists at repository root and contains `.go` files

### Issue: Tests fail after import path update
**Solution**: Ensure all 3 CLI command files have updated import paths (search for old path)

### Issue: Circular dependency error
**Solution**: Check for accidental import of CLI packages in `solution/` package (forbidden)

### Issue: Test coverage decreased
**Solution**: Verify all test files run successfully; coverage may appear lower if tests are in wrong package

---

## Next Steps

After completing this quickstart:

1. ✅ All implementation complete
2. ✅ All tests passing
3. ✅ Ready for commit (see commit message guidance below)

**Suggested Commit Message**:
```
refactor: move solution parsing to root-level library package

Relocate solution file parsing from cmd/gonuget/solution to solution/
at repository root, making it reusable by external Go programs.

Changes:
- Copy 7 source files verbatim to solution/ package
- Update import paths in CLI commands (package_add, package_list, package_remove)
- Delete old cmd/gonuget/solution directory
- Verify 100% test pass rate and build success

No logic changes. Pure package relocation refactor.
```

---

## Estimated Time

- **Total**: ~15-20 minutes
- **Breakdown**:
  - Steps 1-4 (Copy files, verify build): 5 minutes
  - Steps 5-7 (Update imports): 3 minutes
  - Steps 8-12 (Testing): 5 minutes
  - Steps 13-15 (External validation, final checks): 7 minutes

---

## Summary

This quickstart provides a **step-by-step, copy-paste-friendly guide** to implementing the solution library refactor. Each step includes:
- Clear action to perform
- Verification command
- Success criteria
- Expected output

Follow steps sequentially for a safe, validated refactor.

**Status**: ✅ **Quickstart Guide Complete - Ready for Implementation**
