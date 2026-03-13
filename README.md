# ccat-assistant

A macOS daemon that continuously monitors your screen, extracts text via OCR, processes it through Claude AI, and sends the results to Telegram.

## How It Works

```
Screen Capture → Change Detection → OCR → Claude AI → Telegram
```

1. **Capture** — grabs the center 60% of the active window using AppleScript and the macOS screenshot API
2. **Diff** — compares the new screenshot against the previous one pixel-by-pixel; skips processing if the screen hasn't changed enough
3. **OCR** — extracts text from the screenshot using Tesseract (in-memory, no temp files)
4. **AI** — sends the extracted text to Claude with a configurable system prompt
5. **Notify** — delivers Claude's response to a Telegram chat (auto-chunks messages over 4096 chars)

Everything runs in-memory. No screenshots or intermediate files are written to disk.

## Prerequisites

- **macOS** (uses AppleScript for window detection)
- **Go 1.24+**
- **Tesseract OCR** — install via Homebrew:
  ```bash
  brew install tesseract
  ```
- **Anthropic API key** — get one at [console.anthropic.com](https://console.anthropic.com)
- **Telegram bot** — create one via [@BotFather](https://t.me/BotFather) and get the chat ID

## Installation

```bash
git clone https://github.com/vdyalex/ccat-assistant.git
cd ccat-assistant
make build
```

## Configuration

All configuration is done through environment variables. Create a `.env` file or export them directly.

### Required

| Variable | Description |
|---|---|
| `ANTHROPIC_API_KEY` | Claude API key |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token from @BotFather |
| `TELEGRAM_CHAT_ID` | Telegram chat ID to send messages to |

### Optional

| Variable | Default | Description |
|---|---|---|
| `CCAT_POLL_INTERVAL` | `2s` | How often to check for screen changes (Go duration) |
| `CCAT_DIFF_THRESHOLD` | `0.01` | Fraction of pixels that must differ to trigger processing (0.0–1.0) |
| `CCAT_MAX_HISTORY` | `50` | Number of screenshots kept in the ring buffer |
| `CCAT_SCREENSHOT_QUALITY` | `80` | JPEG quality for in-memory encoding (1–100) |
| `CCAT_TESSERACT_LANG` | `eng` | Tesseract language pack code |
| `CCAT_CLAUDE_MODEL` | `claude-sonnet-4-6` | Claude model ID |
| `CCAT_SYSTEM_PROMPT` | *(built-in)* | System prompt sent to Claude with each request |

## Usage

```bash
# Export required variables
export ANTHROPIC_API_KEY="sk-ant-..."
export TELEGRAM_BOT_TOKEN="123456:ABC..."
export TELEGRAM_CHAT_ID="987654321"

# Run in foreground
./ccat

# Or run in background
make run
```

The daemon polls the screen every `CCAT_POLL_INTERVAL` (default 2s). When the screen content changes beyond the diff threshold, it extracts text, sends it to Claude, and forwards the response to Telegram.

Press `Ctrl+C` to stop — the daemon handles SIGINT/SIGTERM gracefully.

## Project Structure

```
cmd/ccat/           Entry point, signal handling, graceful shutdown
internal/
  pipeline/         Orchestrates the capture → OCR → AI → Telegram loop
  capture/          Screen capture and window detection (macOS / AppleScript)
  diff/             Pixel-level image comparison and ring buffer
  ocr/              Tesseract OCR wrapper (in-memory)
  agent/            Claude API client
  telegram/         Telegram Bot API sender with message chunking
  config/           Environment variable loader with defaults
```

## Permissions

macOS will prompt for **Screen Recording** permission the first time ccat captures the screen. Grant access in **System Settings → Privacy & Security → Screen Recording**.

## License

See [LICENSE](LICENSE) for details.
