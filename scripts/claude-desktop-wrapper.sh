#!/bin/bash
# Wrapper used by Claude Desktop (and other MCP clients) so the raw access
# token never ends up in claude_desktop_config.json. The token lives in the
# macOS Keychain — rotate it via:
#
#   security add-generic-password -U -a "$USER" -s "haravan-cli-token" -w 'NEW_TOKEN'
#
# Delete with:
#
#   security delete-generic-password -a "$USER" -s "haravan-cli-token"
#
# IMPORTANT — installation location matters on macOS:
#
# macOS TCC (Transparency, Consent, Control) silently refuses to execute
# shell scripts from ~/Downloads (and a few other protected folders) even
# for apps that can happily execute compiled binaries there. Claude Desktop
# is sandboxed and hits this.
#
# Install this wrapper to ~/.local/bin (or /usr/local/bin) instead. The
# repo copy exists only as the canonical source; the Makefile install-
# target / manual `cp` puts it somewhere TCC permits.
set -euo pipefail

# Override this path with HARAVAN_CLI_BINARY env if you install the Go
# binary to /usr/local/bin or similar. Default assumes the repo layout.
BINARY="${HARAVAN_CLI_BINARY:-}"
if [[ -z "$BINARY" ]]; then
  SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
  # When installed to ~/.local/bin, SCRIPT_DIR is unrelated to the repo,
  # so fall back to a well-known path set by the installer.
  if [[ -x "${SCRIPT_DIR}/../haravan-cli" ]]; then
    BINARY="${SCRIPT_DIR}/../haravan-cli"
  elif [[ -x "$HOME/.local/bin/haravan-cli" ]]; then
    BINARY="$HOME/.local/bin/haravan-cli"
  elif [[ -x "/usr/local/bin/haravan-cli" ]]; then
    BINARY="/usr/local/bin/haravan-cli"
  elif command -v haravan-cli >/dev/null 2>&1; then
    BINARY="$(command -v haravan-cli)"
  fi
fi

if [[ -z "$BINARY" ]] || [[ ! -x "$BINARY" ]]; then
  echo "haravan-cli binary not found. Set HARAVAN_CLI_BINARY env or install to ~/.local/bin or /usr/local/bin." >&2
  exit 1
fi

if ! TOKEN=$(security find-generic-password -a "$USER" -s "haravan-cli-token" -w 2>/dev/null); then
  cat >&2 <<EOF
Keychain entry 'haravan-cli-token' not found for user $USER.
Store your token with:

  security add-generic-password -U -a "\$USER" -s "haravan-cli-token" -w 'YOUR_TOKEN'
EOF
  exit 1
fi

export HARAVAN_ACCESS_TOKEN="$TOKEN"
exec "$BINARY" "$@"
