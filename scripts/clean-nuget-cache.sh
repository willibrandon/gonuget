#!/bin/bash
# clean-nuget-cache.sh
# Cleans all NuGet caches and test project artifacts for gonuget testing
#
# Usage:
#   ./scripts/clean-nuget-cache.sh           # Clean everything
#   ./scripts/clean-nuget-cache.sh --dry-run # Preview what would be deleted
#   ./scripts/clean-nuget-cache.sh --packages-only  # Only clear global packages
#   ./scripts/clean-nuget-cache.sh --test-only      # Only clear test artifacts

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
DRY_RUN=false
PACKAGES_ONLY=false
TEST_ONLY=false
SILENT=false

for arg in "$@"; do
  case $arg in
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    --packages-only)
      PACKAGES_ONLY=true
      shift
      ;;
    --test-only)
      TEST_ONLY=true
      shift
      ;;
    --silent|-q|--quiet)
      SILENT=true
      shift
      ;;
    --help|-h)
      echo "Usage: $0 [OPTIONS]"
      echo ""
      echo "Options:"
      echo "  --dry-run         Preview what would be deleted without deleting"
      echo "  --packages-only   Only clear global NuGet package cache"
      echo "  --test-only       Only clear test project artifacts"
      echo "  --silent, -q      Silent mode - no output"
      echo "  --help, -h        Show this help message"
      echo ""
      echo "What gets cleaned:"
      echo "  1. Global NuGet package cache (~/.nuget/packages)"
      echo "  2. NuGet HTTP cache (~/.local/share/NuGet/v3-cache or ~/Library/Caches/NuGet)"
      echo "  3. NuGet temp files (/tmp/NuGetScratch*)"
      echo "  4. NuGet plugins cache (~/.nuget/plugins-cache)"
      echo "  5. Test scenario obj/bin folders (tests/test-scenarios/*/obj, tests/test-scenarios/*/bin)"
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown option: $arg${NC}"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

# Function to get directory size
get_size() {
  local path="$1"
  if [ -d "$path" ]; then
    du -sh "$path" 2>/dev/null | cut -f1
  else
    echo "0B"
  fi
}

# Function to remove directory with confirmation
remove_dir() {
  local path="$1"
  local description="$2"

  if [ ! -d "$path" ]; then
    if [ "$SILENT" = false ]; then
      echo -e "${YELLOW}  ⊘ $description not found: $path${NC}"
    fi
    return
  fi

  local size=$(get_size "$path")

  if [ "$DRY_RUN" = true ]; then
    if [ "$SILENT" = false ]; then
      echo -e "${BLUE}  ⊙ Would delete $description: $path ($size)${NC}"
    fi
  else
    if [ "$SILENT" = false ]; then
      echo -e "${GREEN}  ✓ Deleting $description: $path ($size)${NC}"
    fi
    rm -rf "$path"
  fi
}

# Function to remove glob pattern
remove_glob() {
  local pattern="$1"
  local description="$2"

  # Use find to count and size
  local count=$(find $pattern -maxdepth 0 2>/dev/null | wc -l | tr -d ' ')

  if [ "$count" -eq 0 ]; then
    if [ "$SILENT" = false ]; then
      echo -e "${YELLOW}  ⊘ No $description found: $pattern${NC}"
    fi
    return
  fi

  if [ "$DRY_RUN" = true ]; then
    if [ "$SILENT" = false ]; then
      echo -e "${BLUE}  ⊙ Would delete $count $description: $pattern${NC}"
    fi
    find $pattern -maxdepth 0 2>/dev/null | while read -r dir; do
      local size=$(get_size "$dir")
      if [ "$SILENT" = false ]; then
        echo -e "${BLUE}      - $dir ($size)${NC}"
      fi
    done
  else
    if [ "$SILENT" = false ]; then
      echo -e "${GREEN}  ✓ Deleting $count $description: $pattern${NC}"
    fi
    find $pattern -maxdepth 0 2>/dev/null | while read -r dir; do
      local size=$(get_size "$dir")
      if [ "$SILENT" = false ]; then
        echo -e "${GREEN}      - $dir ($size)${NC}"
      fi
      rm -rf "$dir"
    done
  fi
}

if [ "$SILENT" = false ]; then
  echo ""
  if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}=== DRY RUN MODE (no files will be deleted) ===${NC}"
  else
    echo -e "${GREEN}=== Cleaning NuGet Caches and Test Artifacts ===${NC}"
  fi
  echo ""
fi

# Calculate total size before cleaning
TOTAL_SIZE_BEFORE=0

# NuGet Caches (unless test-only)
if [ "$TEST_ONLY" = false ]; then
  [ "$SILENT" = false ] && echo -e "${BLUE}Cleaning NuGet Caches:${NC}"

  # Global package cache
  remove_dir "$HOME/.nuget/packages" "Global NuGet package cache"

  # HTTP cache (platform-specific)
  if [ "$(uname)" = "Darwin" ]; then
    # macOS
    remove_dir "$HOME/Library/Caches/NuGet/v3-cache" "NuGet HTTP cache (macOS)"
    remove_dir "$HOME/Library/Caches/NuGet/http-cache" "NuGet HTTP cache (macOS)"
  else
    # Linux
    remove_dir "$HOME/.local/share/NuGet/v3-cache" "NuGet HTTP cache (Linux)"
    remove_dir "$HOME/.local/share/NuGet/http-cache" "NuGet HTTP cache (Linux)"
  fi

  # Plugins cache
  remove_dir "$HOME/.nuget/plugins-cache" "NuGet plugins cache"

  # Temp files
  remove_glob "/tmp/NuGetScratch*" "NuGet temp directories"
  remove_glob "/tmp/.NETCore*" ".NET Core temp directories"

  [ "$SILENT" = false ] && echo ""
fi

# Test project artifacts (unless packages-only)
if [ "$PACKAGES_ONLY" = false ]; then
  [ "$SILENT" = false ] && echo -e "${BLUE}Cleaning Test Project Artifacts:${NC}"

  # Determine script directory to find test scenarios relative to repo root
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
  TEST_SCENARIOS_DIR="$REPO_ROOT/tests/test-scenarios"

  # Test scenarios (simple, multitarget, complex, nu1101, nu1102, nu1103)
  if [ -d "$TEST_SCENARIOS_DIR" ]; then
    for scenario_dir in "$TEST_SCENARIOS_DIR"/*; do
      if [ -d "$scenario_dir" ]; then
        scenario_name=$(basename "$scenario_dir")
        remove_dir "$scenario_dir/obj" "$scenario_name obj folder"
        remove_dir "$scenario_dir/bin" "$scenario_name bin folder"
      fi
    done
  fi

  [ "$SILENT" = false ] && echo ""
fi

# Summary
if [ "$SILENT" = false ]; then
  echo -e "${GREEN}=== Summary ===${NC}"
  if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}Dry run completed. No files were deleted.${NC}"
    echo -e "${YELLOW}Run without --dry-run to actually delete these files.${NC}"
  else
    echo -e "${GREEN}Cache cleaning completed successfully!${NC}"
    echo ""
    echo "Cleaned:"
    if [ "$TEST_ONLY" = false ]; then
      echo "  - NuGet global package cache"
      echo "  - NuGet HTTP cache"
      echo "  - NuGet plugins cache"
      echo "  - NuGet temp files"
    fi
    if [ "$PACKAGES_ONLY" = false ]; then
      echo "  - Test project obj/bin folders"
    fi
    echo ""
    echo -e "${BLUE}Tip: Run 'gonuget restore' or 'dotnet restore' to rebuild the cache${NC}"
  fi
  echo ""
fi
