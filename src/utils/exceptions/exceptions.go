package exceptions

import "errors"

// Config domain: required environment variables
var (
	// ErrConfigMissingAPIKey is returned when ANTHROPIC_API_KEY is not set.
	ErrConfigMissingAPIKey = errors.New("anthropic_api_key environment variable is required")
	// ErrConfigMissingBotToken is returned when TELEGRAM_BOT_TOKEN is not set.
	ErrConfigMissingBotToken = errors.New("telegram_bot_token environment variable is required")
	// ErrConfigInvalidHotkey is returned when HOTKEY_TRIGGER_KEYNAME or HOTKEY_BOUNDS_KEYNAME is not a recognized key name.
	ErrConfigInvalidHotkey = errors.New("invalid hotkey name: supported values are LeftShift, RightShift, LeftControl, RightControl, LeftCommand, RightCommand, LeftOption, RightOption, Fn")
)

// Capturer domain: window detection and screenshot capture
var (
	// ErrCapturerNoForegroundWindow is returned when no foreground window is detected (e.g., showing desktop).
	ErrCapturerNoForegroundWindow = errors.New("no foreground window")
	// ErrCapturerAccessibilityDenied is returned when Accessibility permission is not granted to the application.
	ErrCapturerAccessibilityDenied = errors.New("accessibility permission denied: grant access to this app in System Settings > Privacy & Security > Accessibility")
	// ErrCapturerAppleScriptFailed is returned when an AppleScript command fails for reasons other than missing window or accessibility denial.
	ErrCapturerAppleScriptFailed = errors.New("applescript failed")
	// ErrCapturerInvalidDisplayDimensions is returned when the display width or height is zero or negative.
	ErrCapturerInvalidDisplayDimensions = errors.New("invalid display dimensions")
	// ErrCapturerInvalidCaptureRect is returned when the capture rectangle is invalid (empty or outside screen bounds after clamping).
	ErrCapturerInvalidCaptureRect = errors.New("invalid capture rectangle")
	// ErrCapturerCaptureFailed is returned when the CoreGraphics capture call fails to return pixel data.
	ErrCapturerCaptureFailed = errors.New("screenshot capture failed")
)

// Listener domain: hotkey event tap
var (
	// ErrListenerEventTapCreateFailed is returned when CGEventTapCreate fails, typically due to missing Accessibility permission.
	ErrListenerEventTapCreateFailed = errors.New("cgeventtapcreate failed — grant Accessibility permission to this app")
)

// OCR domain: text recognition
var (
	// ErrOCREmptyInput is returned when the PNG data passed to the OCR engine is empty.
	ErrOCREmptyInput = errors.New("empty png data")
	// ErrOCRFailed is returned when the OCR engine fails to process the image (returns nil result).
	ErrOCRFailed = errors.New("ocr returned null")
)

// IM domain: Telegram API
var (
	// ErrIMRateLimit is returned when the Telegram API responds with HTTP 429 (Too Many Requests).
	ErrIMRateLimit = errors.New("rate limit")
	// ErrIMTelegramAPI is returned when the Telegram API returns a non-200 error response.
	ErrIMTelegramAPI = errors.New("telegram error")
)

// Pipeline domain: timeout errors
var (
	// ErrPipelineCaptureTimeout is returned when the screenshot capture step exceeds its timeout threshold.
	ErrPipelineCaptureTimeout = errors.New("screenshot capture timeout")
	// ErrPipelineOCRTimeout is returned when the OCR extraction step exceeds its timeout threshold.
	ErrPipelineOCRTimeout = errors.New("ocr timeout")
)
