#!/bin/bash
set -euo pipefail

# Resolve paths dynamically
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
GOBIN="${GOPATH:-$HOME/go}/bin"
PLIST_PATH="$HOME/Library/LaunchAgents/com.vdyalex.lensd.plist"
LOG_DIR="$HOME/Library/Logs/lens"
ENV_FILE="$REPO_DIR/.env"
TEMPLATE_FILE="$SCRIPT_DIR/com.vdyalex.lensd.plist.template"
BINARY_NAME="lensd"

echo "Installing service..."
echo "  Repo dir: $REPO_DIR"
echo "  Bin dir: $GOBIN"
echo "  Plist: $PLIST_PATH"
echo "  Log dir: $LOG_DIR"
echo ""

# Validate .env exists
if [[ ! -f "$ENV_FILE" ]]; then
  echo "Error: .env not found at $ENV_FILE"
  echo "Create it with: cp .env.example .env"
  exit 1
fi

# Source .env and validate required vars
set +e
source "$ENV_FILE"
set -e

for var in ANTHROPIC_API_KEY TELEGRAM_BOT_TOKEN; do
  if [[ -z "${!var:-}" ]]; then
    echo "Error: $var is not set or empty in .env"
    exit 1
  fi
done

echo "✓ .env validated"
echo ""

# Build binary
echo "Building binary: $BINARY_NAME"
go build -o "$GOBIN/$BINARY_NAME" "$REPO_DIR/src"
echo "✓ Binary built at $GOBIN/$BINARY_NAME"
echo ""

# Create log directory
mkdir -p "$LOG_DIR"
chmod 700 "$LOG_DIR"
echo "✓ Log directory created at $LOG_DIR"
echo ""

# XML escape function for SYSTEM_PROMPT
xml_escape() {
  local s="$1"
  s="${s//&/&amp;}"
  s="${s//</&lt;}"
  s="${s//>/&gt;}"
  s="${s//\"/&quot;}"
  s="${s//\'/&apos;}"
  echo "$s"
}

# Set defaults for optional vars
LOG_LEVEL="${LOG_LEVEL:-info}"
CLAUDE_MODEL="${CLAUDE_MODEL:-claude-sonnet-4-6}"
VISION_LANG="${VISION_LANG:-en-US}"
SYSTEM_PROMPT="${SYSTEM_PROMPT:-}"

# XML escape SYSTEM_PROMPT
SYSTEM_PROMPT_ESCAPED=$(xml_escape "$SYSTEM_PROMPT")

# Create temporary plist with substitutions
TEMP_PLIST=$(mktemp)
cat "$TEMPLATE_FILE" > "$TEMP_PLIST"

sed -i '' "s|__BINARY_PATH__|$GOBIN/$BINARY_NAME|g" "$TEMP_PLIST"
sed -i '' "s|__ANTHROPIC_API_KEY__|${ANTHROPIC_API_KEY}|g" "$TEMP_PLIST"
sed -i '' "s|__TELEGRAM_BOT_TOKEN__|${TELEGRAM_BOT_TOKEN}|g" "$TEMP_PLIST"
sed -i '' "s|__TELEGRAM_CHAT_ID__|${TELEGRAM_CHAT_ID}|g" "$TEMP_PLIST"
sed -i '' "s|__LOG_LEVEL__|${LOG_LEVEL}|g" "$TEMP_PLIST"
sed -i '' "s|__CLAUDE_MODEL__|${CLAUDE_MODEL}|g" "$TEMP_PLIST"
sed -i '' "s|__VISION_LANG__|${VISION_LANG}|g" "$TEMP_PLIST"
sed -i '' "s|__SYSTEM_PROMPT__|${SYSTEM_PROMPT_ESCAPED}|g" "$TEMP_PLIST"
sed -i '' "s|__LOG_DIR__|${LOG_DIR}|g" "$TEMP_PLIST"

# Move plist to final location
mv "$TEMP_PLIST" "$PLIST_PATH"
chmod 600 "$PLIST_PATH"
echo "✓ Plist generated at $PLIST_PATH"
echo ""

# Unload existing agent if it exists
if launchctl list com.vdyalex.lensd &>/dev/null; then
  echo "Unloading existing agent..."
  launchctl unload "$PLIST_PATH" || true
fi

# Load the agent
echo "Loading service..."
launchctl load -w "$PLIST_PATH"
echo "✓ Service loaded"
echo ""

# Give launchd a moment to start the process
sleep 1

# Check if running
if launchctl list com.vdyalex.lensd &>/dev/null; then
  PID=$(launchctl list com.vdyalex.lensd 2>/dev/null | grep PID | awk '{print $NF}')
  if [[ -n "$PID" && "$PID" != "PID" ]]; then
    echo "✓ Service running (PID: $PID)"
  else
    echo "✓ Service loaded (no PID yet, may start shortly)"
  fi
else
  echo "⚠ Service may not have started. Check logs:"
  echo "  make service-logs"
fi
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Setup complete!"
echo ""
echo "⚠️  IMPORTANT: Grant permissions in System Settings:"
echo "   1. System Settings → Privacy & Security → Accessibility"
echo "      → Add your app (or Terminal if running via Makefile)"
echo "   2. System Settings → Privacy & Security → Screen Recording"
echo "      → Add your app (or Terminal if running via Makefile)"
echo ""
echo "Then restart the service: make service-stop && make service-start"
echo ""
echo "To view logs: make service-logs"
echo "To stop service: make service-stop"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
