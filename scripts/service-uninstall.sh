#!/bin/bash
set -euo pipefail

# Resolve paths dynamically (same logic as install)
REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOBIN="${GOPATH:-$HOME/go}/bin"
PLIST_PATH="$HOME/Library/LaunchAgents/com.vdyalex.assistant.plist"

echo "Uninstalling service..."
echo ""

# Get binary name from go.mod
BINARY_NAME=$(grep '^module ' "$REPO_DIR/go.mod" | awk '{print $NF}' | xargs basename)

# Unload the service
if launchctl list com.vdyalex.assistant &>/dev/null; then
  echo "Unloading service..."
  launchctl unload "$PLIST_PATH" || true
  echo "✓ Service unloaded"
else
  echo "ℹ Service not currently loaded"
fi
echo ""

# Remove plist
if [[ -f "$PLIST_PATH" ]]; then
  rm -f "$PLIST_PATH"
  echo "✓ Plist removed: $PLIST_PATH"
fi

# Remove binary
if [[ -f "$GOBIN/$BINARY_NAME" ]]; then
  rm -f "$GOBIN/$BINARY_NAME"
  echo "✓ Binary removed: $GOBIN/$BINARY_NAME"
fi

echo ""
echo "✓ Service uninstalled successfully"
