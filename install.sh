#!/usr/bin/env bash
# jjtask installer
# Usage: ./install.sh [--agent] [--wrapper] [--uninstall]
#   --agent: Print JJ_CONFIG setup instructions for Claude Code
#   --wrapper: Install shell wrapper for claude command
#   --uninstall: Remove symlinks and config

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="${HOME}/.local/bin"
JJ_CONFIG_DIR="${HOME}/.config/jj"
FISH_COMPLETIONS_DIR="${HOME}/.config/fish/completions"
FISH_FUNCTIONS_DIR="${HOME}/.config/fish/functions"

# Parse arguments
AGENT_MODE=false
UNINSTALL=false
INSTALL_WRAPPER=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --agent) AGENT_MODE=true; shift ;;
    --wrapper) INSTALL_WRAPPER=true; shift ;;
    --uninstall) UNINSTALL=true; shift ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

uninstall() {
  echo "Uninstalling jjtask..."

  # Remove symlinks
  for name in jjtask jjtask-go; do
    link="$BIN_DIR/$name"
    if [[ -L "$link" ]]; then
      rm "$link"
      echo "  Removed: $link"
    fi
  done

  # Remove fish completions
  if [[ -f "$FISH_COMPLETIONS_DIR/jjtask.fish" ]]; then
    rm "$FISH_COMPLETIONS_DIR/jjtask.fish"
    echo "  Removed: $FISH_COMPLETIONS_DIR/jjtask.fish"
  fi
  if [[ -f "$FISH_COMPLETIONS_DIR/jj_task.fish" ]]; then
    rm "$FISH_COMPLETIONS_DIR/jj_task.fish"
    echo "  Removed: $FISH_COMPLETIONS_DIR/jj_task.fish"
  fi

  # Remove bash completions
  local bash_comp="${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion/completions/jjtask"
  if [[ -f "$bash_comp" ]]; then
    rm "$bash_comp"
    echo "  Removed: $bash_comp"
  fi

  # Remove zsh completions
  local zsh_comp="${XDG_DATA_HOME:-$HOME/.local/share}/zsh/site-functions/_jjtask"
  if [[ -f "$zsh_comp" ]]; then
    rm "$zsh_comp"
    echo "  Removed: $zsh_comp"
  fi

  # Remove fish wrapper
  local fish_wrapper="$FISH_FUNCTIONS_DIR/claude.fish"
  if [[ -f "$fish_wrapper" ]] && grep -q "jjtask claude wrapper" "$fish_wrapper" 2>/dev/null; then
    rm "$fish_wrapper"
    echo "  Removed: $fish_wrapper"
  fi

  # Remove bash wrapper
  local bash_wrapper="$BIN_DIR/claude-jjtask-wrapper"
  if [[ -f "$bash_wrapper" ]]; then
    rm "$bash_wrapper"
    echo "  Removed: $bash_wrapper"
  fi

  # Remove jj config symlink
  local jj_config="$JJ_CONFIG_DIR/conf.d/10-jjtask.toml"
  if [[ -L "$jj_config" ]]; then
    rm "$jj_config"
    echo "  Removed: $jj_config"
  fi

  echo "Done."
}

GITHUB_REPO="coobaha/jjtask"

download_binary() {
  local binary="$SCRIPT_DIR/bin/jjtask-go"
  local os arch

  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$arch" in
    x86_64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) echo "Unsupported architecture: $arch" >&2; return 1 ;;
  esac

  local release_url="https://github.com/$GITHUB_REPO/releases/latest/download/jjtask-${os}-${arch}.tar.gz"
  echo "  Downloading from $release_url..."

  local tmp_dir
  tmp_dir="$(mktemp -d)"
  trap "rm -rf '$tmp_dir'" EXIT

  if curl -fsSL "$release_url" -o "$tmp_dir/jjtask.tar.gz"; then
    tar -xzf "$tmp_dir/jjtask.tar.gz" -C "$tmp_dir"
    mkdir -p "$SCRIPT_DIR/bin"
    mv "$tmp_dir/jjtask-go" "$binary"
    chmod +x "$binary"
    echo "  Downloaded: $binary"
    return 0
  else
    echo "  Download failed" >&2
    return 1
  fi
}

build_go_binary() {
  local binary="$SCRIPT_DIR/bin/jjtask-go"

  if [[ -f "$binary" ]]; then
    echo "  Go binary already exists: $binary"
    return 0
  fi

  # Try downloading pre-built binary first
  if download_binary; then
    return 0
  fi

  # Fall back to building from source
  if ! command -v go &>/dev/null; then
    echo "Error: No pre-built binary available and Go is not installed." >&2
    echo "Install Go 1.22+ or download binary manually from:" >&2
    echo "  https://github.com/$GITHUB_REPO/releases" >&2
    exit 1
  fi

  echo "  Building Go binary..."
  (cd "$SCRIPT_DIR" && go build -o bin/jjtask-go ./cmd/jjtask)
  echo "  Built: $binary"
}

install_binary() {
  mkdir -p "$BIN_DIR"

  echo "Installing jjtask to $BIN_DIR..."

  # Symlink dispatcher
  local src="$SCRIPT_DIR/bin/jjtask"
  local dst="$BIN_DIR/jjtask"
  if [[ -L "$dst" ]]; then
    rm "$dst"
  elif [[ -e "$dst" ]]; then
    echo "  Warning: $dst exists and is not a symlink, skipping"
    return
  fi
  ln -s "$src" "$dst"
  echo "  jjtask -> $dst"

  # Symlink Go binary
  src="$SCRIPT_DIR/bin/jjtask-go"
  dst="$BIN_DIR/jjtask-go"
  if [[ -L "$dst" ]]; then
    rm "$dst"
  elif [[ -e "$dst" ]]; then
    echo "  Warning: $dst exists and is not a symlink, skipping"
    return
  fi
  ln -s "$src" "$dst"
  echo "  jjtask-go -> $dst"
}

install_fish_completions() {
  local binary="$SCRIPT_DIR/bin/jjtask-go"
  if [[ ! -x "$binary" ]]; then
    echo "  Skipping fish completions (binary not built)"
    return
  fi

  if command -v fish &>/dev/null; then
    mkdir -p "$FISH_COMPLETIONS_DIR"

    # jjtask completions
    "$binary" completion fish > "$FISH_COMPLETIONS_DIR/jjtask.fish"
    echo "  Fish completions -> $FISH_COMPLETIONS_DIR/jjtask.fish"

    # jj task completions (extends jj's completions)
    "$binary" jj-completion fish > "$FISH_COMPLETIONS_DIR/jj_task.fish"
    echo "  Fish jj task completions -> $FISH_COMPLETIONS_DIR/jj_task.fish"
  fi
}

install_bash_completions() {
  local binary="$SCRIPT_DIR/bin/jjtask-go"
  if [[ ! -x "$binary" ]]; then
    return
  fi

  local bash_comp_dir="${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion/completions"
  mkdir -p "$bash_comp_dir"
  "$binary" completion bash > "$bash_comp_dir/jjtask"
  echo "  Bash completions -> $bash_comp_dir/jjtask"
}

install_zsh_completions() {
  local binary="$SCRIPT_DIR/bin/jjtask-go"
  if [[ ! -x "$binary" ]]; then
    return
  fi

  local zsh_comp_dir="${XDG_DATA_HOME:-$HOME/.local/share}/zsh/site-functions"
  mkdir -p "$zsh_comp_dir"
  "$binary" completion zsh > "$zsh_comp_dir/_jjtask"
  echo "  Zsh completions -> $zsh_comp_dir/_jjtask"
}

install_jj_config() {
  local conf_d="$JJ_CONFIG_DIR/conf.d"
  mkdir -p "$conf_d"

  local src="$SCRIPT_DIR/config/conf.d/10-jjtask.toml"
  local dst="$conf_d/10-jjtask.toml"

  if [[ ! -f "$src" ]]; then
    echo "Warning: $src not found, skipping jj config"
    return
  fi

  if [[ -L "$dst" ]]; then
    rm "$dst"
  elif [[ -e "$dst" ]]; then
    echo "  Warning: $dst exists and is not a symlink, skipping"
    return
  fi

  ln -s "$src" "$dst"
  echo "  JJ config -> $dst"
}

setup_agent_mode() {
  echo ""
  echo "Agent mode setup:"
  echo "  Set the following environment variable in your shell profile:"
  echo ""
  echo "    export JJ_CONFIG=\"$SCRIPT_DIR/config/conf.d\""
  echo ""
  echo "  Or use the fish function: source $SCRIPT_DIR/shell/fish/functions/jjtask-env.fish"
  echo "  Then run: jjtask-env"
}

install_fish_wrapper() {
  mkdir -p "$FISH_FUNCTIONS_DIR"
  local wrapper="$FISH_FUNCTIONS_DIR/claude.fish"

  if [[ -f "$wrapper" ]]; then
    if grep -q "jjtask claude wrapper" "$wrapper" 2>/dev/null; then
      echo "  Fish wrapper already installed, updating..."
      rm "$wrapper"
    else
      echo "  Warning: $wrapper exists but is not a jjtask wrapper, skipping"
      return 1
    fi
  fi

  cat > "$wrapper" <<FISH
# jjtask claude wrapper - auto-generated by install.sh
# Customize the environment variables below as needed
function claude
    set -lx JJTASK_PROFILE "$SCRIPT_DIR"
    # Layer agent.toml on top of user config (: separator, later wins)
    set -l user_config (set -q XDG_CONFIG_HOME; and echo "\$XDG_CONFIG_HOME/jj/config.toml"; or echo "\$HOME/.config/jj/config.toml")
    if test -f "\$user_config"
        set -lx JJ_CONFIG "\$user_config:\$JJTASK_PROFILE/config/conf.d"
    else
        set -lx JJ_CONFIG "\$JJTASK_PROFILE/config/conf.d"
    end
    set -lx PATH \$JJTASK_PROFILE/bin \$PATH

    # Add your customizations below (uncomment as needed)
    # set -lx MISE_ENV agent
    # set -lx ANTHROPIC_API_KEY your-key

    command claude \$argv
end
FISH

  echo "  Fish wrapper -> $wrapper"
  return 0
}

install_bash_wrapper() {
  mkdir -p "$BIN_DIR"
  local wrapper="$BIN_DIR/claude-jjtask-wrapper"

  if [[ -f "$wrapper" ]]; then
    echo "  Bash wrapper already installed, updating..."
    rm "$wrapper"
  fi

  cat > "$wrapper" <<BASH
#!/usr/bin/env bash
# jjtask claude wrapper - auto-generated by install.sh
# Customize the environment variables below as needed

export JJTASK_PROFILE="$SCRIPT_DIR"
# Layer agent.toml on top of user config (: separator, later wins)
USER_CONFIG="\${XDG_CONFIG_HOME:-\$HOME/.config}/jj/config.toml"
if [[ -f "\$USER_CONFIG" ]]; then
    export JJ_CONFIG="\$USER_CONFIG:\$JJTASK_PROFILE/config/conf.d"
else
    export JJ_CONFIG="\$JJTASK_PROFILE/config/conf.d"
fi
export PATH="\$JJTASK_PROFILE/bin:\$PATH"

# Add your customizations below (uncomment as needed)
# export MISE_ENV=agent
# export ANTHROPIC_API_KEY=your-key

exec claude "\$@"
BASH

  chmod +x "$wrapper"
  echo "  Bash wrapper -> $wrapper"
  echo ""
  echo "  To use: alias claude='$wrapper'"
  echo "  Add the alias to your ~/.bashrc or ~/.zshrc"
  return 0
}

install_wrapper() {
  echo ""
  echo "Installing claude wrapper..."

  local installed=false

  if command -v fish &>/dev/null; then
    if install_fish_wrapper; then
      installed=true
    fi
  fi

  if ! $installed || ! command -v fish &>/dev/null; then
    install_bash_wrapper
    installed=true
  fi

  if $installed; then
    echo ""
    echo "Wrapper installed. Customize by editing the file to add:"
    echo "  - MISE_ENV, ANTHROPIC_API_KEY, or other environment variables"
    echo "  - Pre/post hooks for your workflow"
  fi
}

# Main
if $UNINSTALL; then
  uninstall
  exit 0
fi

echo "Installing jjtask from $SCRIPT_DIR"
echo ""

build_go_binary
install_binary
install_fish_completions
install_bash_completions
install_zsh_completions
install_jj_config

if $AGENT_MODE; then
  setup_agent_mode
fi

if $INSTALL_WRAPPER; then
  install_wrapper
fi

echo ""
echo "Installation complete!"
echo ""

# Check PATH
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
  echo "Note: $BIN_DIR may not be in your PATH."
  echo "Add to your shell profile:"
  echo "  export PATH=\"$BIN_DIR:\$PATH\""
fi
