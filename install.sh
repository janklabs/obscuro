#!/usr/bin/env bash
set -euo pipefail

REPO="https://github.com/janklabs/obscuro.git"
INSTALL_DIR="$HOME/.local/bin"
BINARY="obscuro"

info() { printf "\033[1;34m==>\033[0m %s\n" "$1"; }
err()  { printf "\033[1;31merror:\033[0m %s\n" "$1" >&2; exit 1; }

# Check for go
command -v go >/dev/null 2>&1 || err "Go is not installed. Install it from https://go.dev/dl"

# Clone into temp directory
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

info "Cloning repository..."
git clone --depth 1 "$REPO" "$TMPDIR/obscuro"

info "Building $BINARY..."
(cd "$TMPDIR/obscuro" && go build -o "$BINARY" .)

# Install
mkdir -p "$INSTALL_DIR"
mv "$TMPDIR/obscuro/$BINARY" "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"
info "Installed $BINARY to $INSTALL_DIR/$BINARY"

# Check if INSTALL_DIR is already in PATH
if echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  info "$INSTALL_DIR is already in your PATH. You're all set."
  exit 0
fi

printf "\n\033[1;33m%s is not in your PATH.\033[0m\n" "$INSTALL_DIR"
read -rp "Add it to your shell profile? [y/N] " answer

if [[ "$answer" =~ ^[Yy]$ ]]; then
  # Detect shell config file
  SHELL_NAME=$(basename "$SHELL")
  case "$SHELL_NAME" in
    zsh)  PROFILE="$HOME/.zshrc" ;;
    bash)
      if [[ -f "$HOME/.bash_profile" ]]; then
        PROFILE="$HOME/.bash_profile"
      else
        PROFILE="$HOME/.bashrc"
      fi
      ;;
    fish) PROFILE="$HOME/.config/fish/config.fish" ;;
    *)    PROFILE="$HOME/.profile" ;;
  esac

  LINE="export PATH=\"$INSTALL_DIR:\$PATH\""
  if [[ "$SHELL_NAME" == "fish" ]]; then
    LINE="set -gx PATH $INSTALL_DIR \$PATH"
  fi

  echo "" >> "$PROFILE"
  echo "# Added by obscuro installer" >> "$PROFILE"
  echo "$LINE" >> "$PROFILE"

  info "Added to $PROFILE. Run 'source $PROFILE' or open a new terminal."
else
  info "Skipped. Add this to your shell profile manually:"
  echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
fi
