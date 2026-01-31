#!/usr/bin/env bash
# Development teardown: restore original plugin setup
# Usage: ./dev-teardown.sh
#
# Restores installed_plugins.json from backup.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PLUGINS_JSON="${HOME}/.claude/plugins/installed_plugins.json"
BACKUP_FILE="${SCRIPT_DIR}/.dev-backup/installed_plugins.json"

echo "Tearing down jjtask development environment..."
echo ""

if [[ ! -f "$BACKUP_FILE" ]]; then
  echo "No backup found at $BACKUP_FILE"
  echo "Nothing to restore."
  exit 0
fi

echo "Restoring installed_plugins.json from backup..."
cp "$BACKUP_FILE" "$PLUGINS_JSON"
rm "$BACKUP_FILE"

# Restore plugin cache binaries (download fresh copies)
PLUGIN_CACHE_DIR="${HOME}/.claude/plugins/cache/jjtask-marketplace/jjtask"
if [[ -d "$PLUGIN_CACHE_DIR" ]]; then
  echo ""
  echo "Restoring plugin cache binaries..."
  for version_dir in "$PLUGIN_CACHE_DIR"/*/bin; do
    [[ -d "$version_dir" ]] || continue
    cache_bin="$version_dir/jjtask-go"
    if [[ -L "$cache_bin" ]]; then
      rm "$cache_bin"
      echo "  Removed symlink: $(basename "$(dirname "$version_dir")")/jjtask-go"
      echo "  Note: Re-install plugin or run 'jjtask' to download binary"
    fi
  done
fi

# Clean up agent-space symlinks
AGENT_JJ_CONFIG="${HOME}/.config/claude/.agent-space/jj-config"
if [[ -d "$AGENT_JJ_CONFIG" ]]; then
  echo ""
  echo "Removing JJ config symlinks from agent-space..."
  for cfg in "$SCRIPT_DIR/config/conf.d"/*.toml; do
    [[ -f "$cfg" ]] || continue
    name=$(basename "$cfg")
    cfg_link="$AGENT_JJ_CONFIG/$name"
    if [[ -L "$cfg_link" ]]; then
      rm "$cfg_link"
      echo "  Removed: $name"
    fi
  done
fi

# Clean up backup directory if empty
BACKUP_DIR="${SCRIPT_DIR}/.dev-backup"
if [[ -d "$BACKUP_DIR" ]] && [[ -z "$(ls -A "$BACKUP_DIR" 2>/dev/null)" ]]; then
  rmdir "$BACKUP_DIR"
fi

echo ""
echo "Development teardown complete!"
echo "Plugin now points to cached marketplace version."
