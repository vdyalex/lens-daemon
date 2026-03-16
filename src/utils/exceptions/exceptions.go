package exceptions

import "errors"

// Config domain: required environment variables
var (
	// ConfigMissingAPIKeyException is returned when ANTHROPIC_API_KEY is not set.
	ConfigMissingAPIKeyException = errors.New("ANTHROPIC_API_KEY environment variable is required")
	// ConfigMissingBotTokenException is returned when TELEGRAM_BOT_TOKEN is not set.
	ConfigMissingBotTokenException = errors.New("TELEGRAM_BOT_TOKEN environment variable is required")
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

// Vision domain: OCR
var (
	// VisionEmptyInputException is returned when the PNG data passed to the Vision framework is empty.
	VisionEmptyInputException = errors.New("Empty PNG data")
	// VisionOCRFailedException is returned when the Vision OCR engine fails to process the image (returns nil result).
	VisionOCRFailedException = errors.New("Vision OCR returned null")
)

// Messenger domain: Telegram API
var (
	// MessengerRateLimitException is returned when the Telegram API responds with HTTP 429 (Too Many Requests).
	MessengerRateLimitException = errors.New("Rate limit")
	// MessengerTelegramAPIException is returned when the Telegram API returns a non-200 error response.
	MessengerTelegramAPIException = errors.New("Telegram error")
)

// Pipeline domain: timeout errors
var (
	// PipelineCaptureTimeoutException is returned when the screenshot capture step exceeds its timeout threshold.
	PipelineCaptureTimeoutException = errors.New("Screenshot capture timeout")
	// PipelineOCRTimeoutException is returned when the OCR extraction step exceeds its timeout threshold.
	PipelineOCRTimeoutException = errors.New("OCR timeout")
)
