#!/bin/bash

# This script precisely replaces occurrences of an old Go module major version
# (e.g., "v2") with a new major version (e.g., "v3").
# It automatically detects the module path from go.mod and specifically targets
# the 'module' directive in go.mod files and import paths within .go files.
# This version properly handles Go module versioning conventions:
# - v0 and v1 use no version suffix in module paths
# - v2+ use /vX suffix in module paths

# --- Usage ---
# Save this script as, for example, `update-go-version.sh`.
# Make it executable: `chmod +x update-go-version.sh`
# Run it: `./update-go-version.sh <OLD_VERSION_NUMBER> <NEW_VERSION_NUMBER> [--dry-run]`
#
# Examples:
# `./update-go-version.sh 1 2`     # v1 to v2 (no suffix to /v2)
# `./update-go-version.sh 2 1`     # v2 to v1 (/v2 to no suffix)
# `./update-go-version.sh 2 3`     # v2 to v3 (/v2 to /v3)
# `./update-go-version.sh 0 2`     # v0 to v2 (no suffix to /v2)
#
# To dry-run and see what files would be affected (RECOMMENDED FIRST!):
# `./update-go-version.sh 2 3 --dry-run`

# --- Safety Precautions ---
# ALWAYS run with --dry-run first to see affected files.
# ALWAYS make a backup of your project before running this script, e.g.:
# `cp -R your_project_dir your_project_dir_backup`
# The script can create automatic backups with --backup flag.

# Exit immediately if a command exits with a non-zero status.
set -e

OLD_VERSION_NUM="$1"
NEW_VERSION_NUM="$2"
DRY_RUN=false
CREATE_BACKUP=false

# Process optional flags
shift 2 2>/dev/null || true  # Remove first two args, ignore errors if less than 2 args
while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    --backup)
      CREATE_BACKUP=true
      shift
      ;;
    *)
      echo "Unknown option: $1"
      echo "Valid options: --dry-run, --backup"
      exit 1
      ;;
  esac
done

# Function to perform sed replacement with proper escaping
perform_replacement() {
  local file="$1"
  local search_pattern="$2"
  local replace_pattern="$3"

  if sed --version >/dev/null 2>&1; then
    # GNU sed (Linux)
    sed -i "s|${search_pattern}|${replace_pattern}|g" "$file"
  else
    # BSD sed (macOS)
    sed -i '' "s|${search_pattern}|${replace_pattern}|g" "$file"
  fi
}

# Function to display usage
show_usage() {
  echo "Usage: $0 <OLD_VERSION_NUMBER> <NEW_VERSION_NUMBER> [--dry-run] [--backup]"
  echo ""
  echo "Examples:"
  echo "  $0 1 2                    # Change v1 to v2 (no suffix to /v2)"
  echo "  $0 2 1                    # Change v2 to v1 (/v2 to no suffix)"
  echo "  $0 2 3                    # Change v2 to v3 (/v2 to /v3)"
  echo "  $0 0 2                    # Change v0 to v2 (no suffix to /v2)"
  echo "  $0 2 3 --dry-run         # Preview changes without applying"
  echo "  $0 1 2 --backup          # Create backup before applying changes"
  echo ""
  echo "Note: Go module versioning conventions:"
  echo "  - v0 and v1 use no version suffix in module paths"
  echo "  - v2+ use /vX suffix in module paths"
  echo ""
  echo "This script automatically detects your module path from go.mod"
}

# Validate arguments
if [ -z "$OLD_VERSION_NUM" ] || [ -z "$NEW_VERSION_NUM" ]; then
  show_usage
  exit 1
fi

# Validate version numbers are integers
if ! [[ "$OLD_VERSION_NUM" =~ ^[0-9]+$ ]] || ! [[ "$NEW_VERSION_NUM" =~ ^[0-9]+$ ]]; then
  echo "Error: Version numbers must be integers"
  show_usage
  exit 1
fi

# Validate that we're in a Go module directory
if [ ! -f "go.mod" ]; then
  echo "Error: go.mod file not found in current directory"
  echo "Please run this script from the root of your Go module"
  exit 1
fi

# Function to extract module path from go.mod
get_module_path_from_gomod() {
  # Extract the module path from the first line starting with "module "
  # Remove any existing version suffix (e.g., /v2, /v3, etc.)
  awk '/^module / {
    gsub(/^module[[:space:]]+/, "", $0)
    gsub(/\/v[0-9]+$/, "", $0)
    print $0
    exit
  }' go.mod
}

# Function to get current module path using Go tooling
get_module_path_go_list() {
  if command -v go >/dev/null 2>&1; then
    # Use go list to get the current module path, remove version suffix
    go list -m 2>/dev/null | sed -E 's/\/v[0-9]+$//' || echo ""
  else
    echo ""
  fi
}

# Function to determine version suffix based on Go module conventions
get_version_suffix() {
  local version_num="$1"
  if [ "$version_num" = "0" ] || [ "$version_num" = "1" ]; then
    echo ""  # No suffix for v0 and v1
  else
    echo "/v${version_num}"  # /vX suffix for v2+
  fi
}

# Detect module base path
echo "Detecting module path..."

# Try go list first (more reliable), fallback to parsing go.mod
MODULE_BASE_PATH=$(get_module_path_go_list)
if [ -z "$MODULE_BASE_PATH" ]; then
  echo "Go tooling not available or failed, parsing go.mod directly..."
  MODULE_BASE_PATH=$(get_module_path_from_gomod)
fi

if [ -z "$MODULE_BASE_PATH" ]; then
  echo "Error: Could not detect module path from go.mod"
  echo "Please ensure go.mod contains a valid 'module' directive"
  exit 1
fi

echo "Detected module path: $MODULE_BASE_PATH"

# Determine version suffixes based on Go module conventions
OLD_VERSION_SUFFIX=$(get_version_suffix "$OLD_VERSION_NUM")
NEW_VERSION_SUFFIX=$(get_version_suffix "$NEW_VERSION_NUM")

OLD_MODULE_PATH="${MODULE_BASE_PATH}${OLD_VERSION_SUFFIX}"
NEW_MODULE_PATH="${MODULE_BASE_PATH}${NEW_VERSION_SUFFIX}"

echo "Version transition: v${OLD_VERSION_NUM} → v${NEW_VERSION_NUM}"
echo "Module path change: ${OLD_MODULE_PATH} → ${NEW_MODULE_PATH}"

# Validate that old version exists in current module
CURRENT_MODULE_LINE=$(grep "^module " go.mod || echo "")
if [ -n "$CURRENT_MODULE_LINE" ]; then
  CURRENT_MODULE_PATH=$(echo "$CURRENT_MODULE_LINE" | sed 's/^module[[:space:]]*//')
  if [ "$CURRENT_MODULE_PATH" != "$OLD_MODULE_PATH" ]; then
    echo "Warning: Current go.mod uses a different module path than expected"
    echo "Current module line: $CURRENT_MODULE_LINE"
    echo "Expected module path: $OLD_MODULE_PATH"
    echo "Detected module path: $CURRENT_MODULE_PATH"
    echo ""
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      echo "Aborted by user"
      exit 1
    fi
  fi
fi

echo "Attempting to replace '${OLD_MODULE_PATH}' with '${NEW_MODULE_PATH}' in .go, .md, and go.mod files."

# Create backup if requested
if [ "$CREATE_BACKUP" = true ]; then
  BACKUP_DIR="backup_$(date +%Y%m%d_%H%M%S)"
  echo "Creating backup in: $BACKUP_DIR"
  if [ "$DRY_RUN" = false ]; then
    mkdir -p "$BACKUP_DIR"
    find . -type f \( -name "*.go" -o -name "*.md" -o -name "*.txt" -o -name "README*" -o -name "go.mod" \) -exec cp --parents {} "$BACKUP_DIR/" \; 2>/dev/null || \
    find . -type f \( -name "*.go" -o -name "*.md" -o -name "*.txt" -o -name "README*" -o -name "go.mod" \) | while read -r file; do
      mkdir -p "$BACKUP_DIR/$(dirname "$file")"
      cp "$file" "$BACKUP_DIR/$file"
    done
    echo "Backup created successfully"
  else
    echo "Backup would be created in: $BACKUP_DIR (dry-run mode)"
  fi
fi

# Find files that would be affected (fixed file pattern with proper OR logic)
AFFECTED_FILES=$(find . -type f \( -name "*.go" -o -name "*.md" -o -name "*.txt" -o -name "README*" -o -name "go.mod" \) -exec grep -l "${OLD_MODULE_PATH}" {} \; 2>/dev/null || true)

if [ -z "$AFFECTED_FILES" ]; then
  echo "No files found containing '${OLD_MODULE_PATH}'"
  echo "This might mean:"
  echo "  - The version v${OLD_VERSION_NUM} is not currently used"
  echo "  - The module path detection was incorrect"
  echo "  - There are no import statements using the versioned path"

  # For v0/v1 transitions, also check if we need to add version suffixes
  if [ "$OLD_VERSION_NUM" = "0" ] || [ "$OLD_VERSION_NUM" = "1" ]; then
    echo "  - For v${OLD_VERSION_NUM}, checking if any files reference the base path without version suffix"
    # This is expected for v0/v1 as they don't use suffixes
  fi

  exit 0
fi

if [ "$DRY_RUN" = true ]; then
  echo "--- DRY RUN MODE ---"
  echo "Files that would be affected:"
  echo "$AFFECTED_FILES"
  echo ""
  echo "Preview of changes:"
  echo "$AFFECTED_FILES" | while read -r file; do
    if [ -n "$file" ]; then
      echo "=== $file ==="
      # Show the lines that would change
      grep -n "${OLD_MODULE_PATH}" "$file" 2>/dev/null | head -5 | while read -r line; do
        echo "  BEFORE: $line"
        echo "  AFTER:  $(echo "$line" | sed "s|${OLD_MODULE_PATH}|${NEW_MODULE_PATH}|g")"
      done
      echo ""
    fi
  done
  echo "--------------------"
  echo "No changes were made. To apply changes, run without '--dry-run'."
else
  echo "--- APPLYING CHANGES ---"
  echo "Modifying files in place..."

  PROCESSED_COUNT=0
  echo "$AFFECTED_FILES" | while read -r file; do
    if [ -n "$file" ]; then
      echo "Processing: $file"

      # Handle all file types uniformly - replace old module path with new module path
      perform_replacement "$file" "${OLD_MODULE_PATH}" "${NEW_MODULE_PATH}"

      PROCESSED_COUNT=$((PROCESSED_COUNT + 1))
    fi
  done

  echo "------------------------"
  echo "Replacement complete! Processed files with changes."
  echo ""
  echo "Version transition summary:"
  echo "  From: v${OLD_VERSION_NUM} (${OLD_MODULE_PATH})"
  echo "  To:   v${NEW_VERSION_NUM} (${NEW_MODULE_PATH})"
  echo ""
  echo "Next steps:"
  echo "1. Review the changes with: git diff"
  echo "2. Run: go mod tidy"
  echo "3. Test your code: go test ./..."
  echo "4. Update any documentation that references the old version"
  echo "5. Create and push a new git tag: git tag v${NEW_VERSION_NUM}.0.0"
fi
