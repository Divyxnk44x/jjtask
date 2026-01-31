#!/usr/bin/env bash
# Development setup for jjtask
# Usage: ./dev-setup.sh
#
# Sets up:
# - CLI: symlinks jjtask to ~/.local/bin, fish completions, jj alias
# - Plugin: symlinks binary, points installed_plugins.json to local source
#
# Run ./dev-teardown.sh to restore release plugin version.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PLUGIN_SOURCE="$SCRIPT_DIR/claude-plugin"
BIN_DIR="${HOME}/.local/bin"
FISH_FUNCTIONS_DIR="${__fish_config_dir:-${XDG_CONFIG_HOME:-$HOME/.config}/fish}/functions"
PLUGINS_JSON="${HOME}/.claude/plugins/installed_plugins.json"
BACKUP_FILE="${SCRIPT_DIR}/.dev-backup/installed_plugins.json"

echo "Setting up jjtask development environment..."
echo "Source: $SCRIPT_DIR"
echo ""

symlink() {
  local src="$1" dst="$2"
  if [[ -L "$dst" ]]; then
    local current=$(readlink "$dst")
    if [[ "$current" == "$src" ]]; then
      echo "  Already linked: $(basename "$dst")"
      return 0
    fi
    rm "$dst"
  elif [[ -e "$dst" ]]; then
    echo "  Warning: $dst exists and is not a symlink, skipping" >&2
    return 1
  fi
  ln -s "$src" "$dst"
  echo "  Linked: $(basename "$dst")"
}

# 1. Symlink jjtask CLI to ~/.local/bin
echo "CLI setup:"
mkdir -p "$BIN_DIR"
symlink "$SCRIPT_DIR/bin/jjtask" "$BIN_DIR/jjtask" || true

# 2. Symlink jjtask-go in plugin dir and cache to local build
if [[ -x "$SCRIPT_DIR/bin/jjtask-go" ]]; then
  # Plugin source dir
  dst="$PLUGIN_SOURCE/bin/jjtask-go"
  if [[ -e "$dst" ]] || [[ -L "$dst" ]]; then
    rm "$dst"
  fi
  ln -s "$SCRIPT_DIR/bin/jjtask-go" "$dst"
  echo "  Linked: plugin jjtask-go -> bin/jjtask-go"

  # Plugin cache (used by agent sessions)
  PLUGIN_CACHE_DIR="${HOME}/.claude/plugins/cache/jjtask-marketplace/jjtask"
  if [[ -d "$PLUGIN_CACHE_DIR" ]]; then
    for version_dir in "$PLUGIN_CACHE_DIR"/*/bin; do
      [[ -d "$version_dir" ]] || continue
      cache_dst="$version_dir/jjtask-go"
      if [[ -e "$cache_dst" ]] || [[ -L "$cache_dst" ]]; then
        rm "$cache_dst"
      fi
      ln -s "$SCRIPT_DIR/bin/jjtask-go" "$cache_dst"
      echo "  Linked: cache $(basename "$(dirname "$version_dir")")/jjtask-go -> bin/jjtask-go"
    done
  fi
else
  echo "  Skipping plugin binary (run 'mise run build' first)"
fi

# 3. Fish shell setup
echo ""
echo "Fish setup:"
if [[ -f "$SCRIPT_DIR/shell/fish/functions/jjtask-env.fish" ]]; then
  mkdir -p "$FISH_FUNCTIONS_DIR"
  symlink "$SCRIPT_DIR/shell/fish/functions/jjtask-env.fish" "$FISH_FUNCTIONS_DIR/jjtask-env.fish" || true
fi

if [[ -x "$SCRIPT_DIR/bin/jjtask-go" ]]; then
  comp_dir="${__fish_config_dir:-${XDG_CONFIG_HOME:-$HOME/.config}/fish}/completions"
  mkdir -p "$comp_dir"
  "$SCRIPT_DIR/bin/jjtask-go" completion fish > "$comp_dir/jjtask.fish"
  "$SCRIPT_DIR/bin/jjtask-go" jj-completion fish > "$comp_dir/jj_task.fish"
  echo "  Generated: jjtask.fish, jj_task.fish"
else
  echo "  Skipping completions (run 'mise run build' first)"
fi

# 4. JJ alias
echo ""
echo "JJ setup:"
current_alias=$(jj config get aliases.task 2>/dev/null || echo "")
if [[ -z "$current_alias" ]]; then
  jj config set --user 'aliases.task' '["util", "exec", "--", "jjtask"]'
  echo "  Set: jj task -> jjtask"
else
  echo "  Already set: jj alias.task"
fi

# 5. Point Claude Code plugin to local source
echo ""
echo "Claude Code plugin setup:"
if [[ -f "$PLUGINS_JSON" ]] && grep -q '"jjtask@jjtask-marketplace"' "$PLUGINS_JSON"; then
  CURRENT_PATH=$(grep -A5 '"jjtask@jjtask-marketplace"' "$PLUGINS_JSON" | grep installPath | head -1 | sed 's/.*: "//;s/".*//')

  if [[ "$CURRENT_PATH" == "$PLUGIN_SOURCE" ]]; then
    echo "  Already pointing to dev source"
  else
    mkdir -p "$(dirname "$BACKUP_FILE")"
    if [[ ! -f "$BACKUP_FILE" ]]; then
      cp "$PLUGINS_JSON" "$BACKUP_FILE"
      echo "  Backed up: installed_plugins.json"
    fi
    sed -i '' "s|$CURRENT_PATH|$PLUGIN_SOURCE|" "$PLUGINS_JSON"
    echo "  Updated: installPath -> $PLUGIN_SOURCE"
  fi
else
  echo "  Skipping (plugin not installed via marketplace)"
fi

# 6. Agent-space JJ config
AGENT_JJ_CONFIG="${HOME}/.config/claude/.agent-space/jj-config"
if [[ -d "${HOME}/.config/claude/.agent-space" ]]; then
  mkdir -p "$AGENT_JJ_CONFIG"
  for cfg in "$SCRIPT_DIR/config/conf.d"/*.toml; do
    [[ -f "$cfg" ]] || continue
    name=$(basename "$cfg")
    symlink "$cfg" "$AGENT_JJ_CONFIG/$name" || true
  done
fi

echo ""
echo "Done. Ensure ~/.local/bin is in PATH."
echo "Run ./dev-teardown.sh to restore release plugin version."
