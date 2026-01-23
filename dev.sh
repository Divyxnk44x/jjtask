#!/usr/bin/env bash
# Development setup - symlinks jjtask and jjtask-env.fish for direct usage
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="${HOME}/.local/bin"
FISH_FUNCTIONS_DIR="${__fish_config_dir:-${XDG_CONFIG_HOME:-$HOME/.config}/fish}/functions"

symlink() {
  local src="$1" dst="$2"
  if [[ -L "$dst" ]]; then
    local current=$(readlink "$dst")
    if [[ "$current" == "$src" ]]; then
      return 0
    fi
    rm "$dst"
  elif [[ -e "$dst" ]]; then
    echo "  Warning: $dst exists and is not a symlink, skipping" >&2
    return 1
  fi
  ln -s "$src" "$dst"
  echo "  $(basename "$dst")"
}

ensure_bin_scripts() {
  mkdir -p "$BIN_DIR"
  echo "Symlinking jjtask to $BIN_DIR:"
  symlink "$SCRIPT_DIR/bin/jjtask" "$BIN_DIR/jjtask" || true
}

ensure_fish_function() {
  local src="$SCRIPT_DIR/shell/fish/functions/jjtask-env.fish"
  local dst="$FISH_FUNCTIONS_DIR/jjtask-env.fish"
  if [[ ! -f "$src" ]]; then
    echo "Error: $src not found" >&2
    return 1
  fi
  mkdir -p "$FISH_FUNCTIONS_DIR"
  echo "Symlinking jjtask-env.fish:"
  symlink "$src" "$dst"
}

ensure_fish_completions() {
  local binary="$SCRIPT_DIR/bin/jjtask-go"
  if [[ ! -x "$binary" ]]; then
    echo "Skipping fish completions (run 'mise run build' first)"
    return
  fi
  local comp_dir="${__fish_config_dir:-${XDG_CONFIG_HOME:-$HOME/.config}/fish}/completions"
  mkdir -p "$comp_dir"
  echo "Generating fish completions:"
  "$binary" completion fish > "$comp_dir/jjtask.fish"
  echo "  jjtask.fish"
  "$binary" jj-completion fish > "$comp_dir/jj_task.fish"
  echo "  jj_task.fish (jj task completions)"
}

ensure_jj_alias() {
  local current
  current=$(jj config get aliases.task 2>/dev/null || echo "")
  if [[ -z "$current" ]]; then
    echo "Setting jj alias.task:"
    jj config set --user 'aliases.task' '["util", "exec", "--", "jjtask"]'
    echo "  jj task -> jjtask"
  else
    echo "jj alias.task already set"
  fi
}

ensure_bin_scripts
ensure_fish_function
ensure_fish_completions
ensure_jj_alias

echo ""
echo "Done. Ensure ~/.local/bin is in PATH."
