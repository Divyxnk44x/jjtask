#!/usr/bin/env bash
# Development setup: symlink jjtask files to ~/.config/claude/ for live editing
# Usage: ./dev-setup.sh
#
# Creates symlinks so changes in jjtask repo are immediately usable in Claude Code.
# Run ./dev-teardown.sh to restore original setup.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLAUDE_DIR="${HOME}/.config/claude"
BACKUP_DIR="${SCRIPT_DIR}/.dev-backup"

echo "Setting up jjtask development environment..."
echo "Source: $SCRIPT_DIR"
echo "Target: $CLAUDE_DIR"
echo ""

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Helper to backup and symlink
link_item() {
  local src="$1"
  local dst="$2"
  local backup_path="$BACKUP_DIR/$(basename "$dst")"

  if [[ -L "$dst" ]]; then
    # Already a symlink, remove it
    rm "$dst"
  elif [[ -e "$dst" ]]; then
    # Exists and not a symlink, backup it
    if [[ ! -e "$backup_path" ]]; then
      echo "  Backing up: $dst -> $backup_path"
      mv "$dst" "$backup_path"
    else
      echo "  Backup exists, removing: $dst"
      rm -rf "$dst"
    fi
  fi

  ln -s "$src" "$dst"
  echo "  Linked: $dst -> $src"
}

# 1. Link bin scripts to agent-space profile
AGENT_BIN="${CLAUDE_DIR}/.agent-space/profile/bin"
mkdir -p "$AGENT_BIN"
echo "Linking bin/* to $AGENT_BIN/"
for script in "$SCRIPT_DIR/bin"/*; do
  [[ -f "$script" ]] || continue
  name=$(basename "$script")
  link_item "$script" "$AGENT_BIN/$name"
done

# 2. Link config to jj-config
JJ_CONFIG_DIR="${CLAUDE_DIR}/.agent-space/jj-config"
mkdir -p "$JJ_CONFIG_DIR"
echo ""
echo "Linking config/conf.d/ to $JJ_CONFIG_DIR/"
for cfg in "$SCRIPT_DIR/config/conf.d"/*.toml; do
  [[ -f "$cfg" ]] || continue
  name=$(basename "$cfg")
  link_item "$cfg" "$JJ_CONFIG_DIR/$name"
done

# 3. Link commands
COMMANDS_DIR="${CLAUDE_DIR}/commands/jjtask"
mkdir -p "$COMMANDS_DIR"
echo ""
echo "Linking claude-plugin/commands/* to $COMMANDS_DIR/"
for cmd in "$SCRIPT_DIR/claude-plugin/commands"/*.md; do
  [[ -f "$cmd" ]] || continue
  name=$(basename "$cmd")
  link_item "$cmd" "$COMMANDS_DIR/$name"
done

# 4. Link skills
SKILLS_DIR="${CLAUDE_DIR}/skills"
mkdir -p "$SKILLS_DIR"
echo ""
echo "Linking claude-plugin/skills/ to $SKILLS_DIR/"
for skill_dir in "$SCRIPT_DIR/claude-plugin/skills"/*; do
  [[ -d "$skill_dir" ]] || continue
  name=$(basename "$skill_dir")
  link_item "$skill_dir" "$SKILLS_DIR/$name"
done

echo ""
echo "Development setup complete!"
echo ""
echo "Changes to files in $SCRIPT_DIR will now be immediately"
echo "available in Claude Code sessions."
echo ""
echo "To restore original setup: ./dev-teardown.sh"
