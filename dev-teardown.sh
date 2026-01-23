#!/usr/bin/env bash
# Development teardown: restore original ~/.config/claude/ setup
# Usage: ./dev-teardown.sh
#
# Removes symlinks created by dev-setup.sh and restores backups.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLAUDE_DIR="${HOME}/.config/claude"
BACKUP_DIR="${SCRIPT_DIR}/.dev-backup"

echo "Tearing down jjtask development environment..."
echo ""

# Helper to remove symlink and restore backup
unlink_item() {
  local dst="$1"
  local backup_path="$BACKUP_DIR/$(basename "$dst")"

  if [[ -L "$dst" ]]; then
    rm "$dst"
    echo "  Removed symlink: $dst"

    if [[ -e "$backup_path" ]]; then
      mv "$backup_path" "$dst"
      echo "  Restored backup: $backup_path -> $dst"
    fi
  fi
}

# 1. Unlink bin scripts
AGENT_BIN="${CLAUDE_DIR}/.agent-space/profile/bin"
if [[ -d "$AGENT_BIN" ]]; then
  echo "Unlinking bin scripts from $AGENT_BIN/"
  for script in "$SCRIPT_DIR/bin"/*; do
    [[ -f "$script" ]] || continue
    name=$(basename "$script")
    unlink_item "$AGENT_BIN/$name"
  done
fi

# 2. Unlink config
JJ_CONFIG_DIR="${CLAUDE_DIR}/.agent-space/jj-config"
echo ""
echo "Unlinking config from $JJ_CONFIG_DIR/"
unlink_item "$JJ_CONFIG_DIR/10-jjtask.toml"

# 3. Unlink commands
COMMANDS_DIR="${CLAUDE_DIR}/commands/jjtask"
if [[ -d "$COMMANDS_DIR" ]]; then
  echo ""
  echo "Unlinking commands from $COMMANDS_DIR/"
  for cmd in "$SCRIPT_DIR/commands"/*.md; do
    [[ -f "$cmd" ]] || continue
    name=$(basename "$cmd")
    unlink_item "$COMMANDS_DIR/$name"
  done
  # Remove directory if empty
  rmdir "$COMMANDS_DIR" 2>/dev/null || true
fi

# 4. Unlink skills
SKILLS_DIR="${CLAUDE_DIR}/skills"
echo ""
echo "Unlinking skills from $SKILLS_DIR/"
unlink_item "$SKILLS_DIR/jjtask"
unlink_item "$SKILLS_DIR/jj-dev"

# Clean up backup directory if empty
if [[ -d "$BACKUP_DIR" ]] && [[ -z "$(ls -A "$BACKUP_DIR" 2>/dev/null)" ]]; then
  rmdir "$BACKUP_DIR"
fi

echo ""
echo "Development teardown complete!"
echo "Original ~/.config/claude/ setup restored."
