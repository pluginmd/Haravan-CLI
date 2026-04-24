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
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
BINARY="${SCRIPT_DIR}/../haravan-cli"

if [[ ! -x "$BINARY" ]]; then
  echo "haravan-cli binary not found at $BINARY" >&2
  echo "Build it first: (cd '$(dirname "$BINARY")' && make build)" >&2
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
