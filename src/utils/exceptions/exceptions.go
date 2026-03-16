package exceptions

import "errors"

// Config domain: required environment variables
var (
	ConfigMissingAPIKeyException   = errors.New("ANTHROPIC_API_KEY environment variable is required")
	ConfigMissingBotTokenException = errors.New("TELEGRAM_BOT_TOKEN environment variable is required")
)

// Capturer domain: window detection and screenshot capture
var (
	CapturerNoForegroundWindowException       = errors.New("No foreground window")
	CapturerAccessibilityDeniedException      = errors.New("Accessibility permission denied: grant access to this app in System Settings > Privacy & Security > Accessibility")
	CapturerAppleScriptFailedException        = errors.New("AppleScript failed")
	CapturerInvalidDisplayDimensionsException = errors.New("Invalid display dimensions")
	CapturerInvalidCaptureRectException       = errors.New("Invalid capture rectangle")
	CapturerCaptureFailedException            = errors.New("Screenshot capture failed")
)

// Listener domain: hotkey event tap
var (
	ListenerEventTapCreateFailedException = errors.New("CGEventTapCreate failed — grant Accessibility permission to this app")
)

// Vision domain: OCR
var (
	VisionEmptyInputException = errors.New("Empty PNG data")
	VisionOCRFailedException  = errors.New("Vision OCR returned null")
)

// Messenger domain: Telegram API
var (
	MessengerRateLimitException   = errors.New("Rate limit")
	MessengerTelegramAPIException = errors.New("Telegram error")
)

// Pipeline domain: timeout errors
var (
	PipelineCaptureTimeoutException = errors.New("Screenshot capture timeout")
	PipelineOCRTimeoutException     = errors.New("OCR timeout")
)
