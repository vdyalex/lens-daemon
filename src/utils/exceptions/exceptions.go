package exceptions

import "errors"

// Config domain: required environment variables
var (
	// ConfigMissingAPIKeyException is returned when ANTHROPIC_API_KEY is not set.
	ConfigMissingAPIKeyException = errors.New("ANTHROPIC_API_KEY environment variable is required")
	// ConfigMissingBotTokenException is returned when TELEGRAM_BOT_TOKEN is not set.
	ConfigMissingBotTokenException = errors.New("TELEGRAM_BOT_TOKEN environment variable is required")
	// ConfigInvalidHotkeyException is returned when HOTKEY_TRIGGER_KEYNAME or HOTKEY_BOUNDS_KEYNAME is not a recognized key name.
	ConfigInvalidHotkeyException = errors.New("Invalid hotkey name: supported values are LeftShift, RightShift, LeftControl, RightControl, LeftCommand, RightCommand, LeftOption, RightOption, Fn")
)

// Capturer domain: window detection and screenshot capture
var (
	// CapturerNoForegroundWindowException is returned when no foreground window is detected (e.g., showing desktop).
	CapturerNoForegroundWindowException = errors.New("No foreground window")
	// CapturerAccessibilityDeniedException is returned when Accessibility permission is not granted to the application.
	CapturerAccessibilityDeniedException = errors.New("Accessibility permission denied: grant access to this app in System Settings > Privacy & Security > Accessibility")
	// CapturerAppleScriptFailedException is returned when an AppleScript command fails for reasons other than missing window or accessibility denial.
	CapturerAppleScriptFailedException = errors.New("AppleScript failed")
	// CapturerInvalidDisplayDimensionsException is returned when the display width or height is zero or negative.
	CapturerInvalidDisplayDimensionsException = errors.New("Invalid display dimensions")
	// CapturerInvalidCaptureRectException is returned when the capture rectangle is invalid (empty or outside screen bounds after clamping).
	CapturerInvalidCaptureRectException = errors.New("Invalid capture rectangle")
	// CapturerCaptureFailedException is returned when the CoreGraphics capture call fails to return pixel data.
	CapturerCaptureFailedException = errors.New("Screenshot capture failed")
)

// Listener domain: hotkey event tap
var (
	// ListenerEventTapCreateFailedException is returned when CGEventTapCreate fails, typically due to missing Accessibility permission.
	ListenerEventTapCreateFailedException = errors.New("CGEventTapCreate failed — grant Accessibility permission to this app")
)

// OCR domain: text recognition
var (
	// OCREmptyInputException is returned when the PNG data passed to the OCR engine is empty.
	OCREmptyInputException = errors.New("Empty PNG data")
	// OCRFailedException is returned when the OCR engine fails to process the image (returns nil result).
	OCRFailedException = errors.New("OCR returned null")
)

// IM domain: Telegram API
var (
	// IMRateLimitException is returned when the Telegram API responds with HTTP 429 (Too Many Requests).
	IMRateLimitException = errors.New("Rate limit")
	// IMTelegramAPIException is returned when the Telegram API returns a non-200 error response.
	IMTelegramAPIException = errors.New("Telegram error")
)

// Pipeline domain: timeout errors
var (
	// PipelineCaptureTimeoutException is returned when the screenshot capture step exceeds its timeout threshold.
	PipelineCaptureTimeoutException = errors.New("Screenshot capture timeout")
	// PipelineOCRTimeoutException is returned when the OCR extraction step exceeds its timeout threshold.
	PipelineOCRTimeoutException = errors.New("OCR timeout")
)
