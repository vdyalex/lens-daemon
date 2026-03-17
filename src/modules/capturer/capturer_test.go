//go:build darwin

package capturer

import (
	"image"
	"testing"
)

func TestParseWindowInfo_valid(test *testing.T) {
	info, err := parseWindowInfo("Chrome,10,20,800,600")

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if info.Title != "Chrome" {
		test.Errorf("expected title 'Chrome', got %q", info.Title)
	}
	if info.X != 10 || info.Y != 20 || info.Width != 800 || info.Height != 600 {
		test.Errorf("expected (10,20,800,600), got (%d,%d,%d,%d)", info.X, info.Y, info.Width, info.Height)
	}
}

func TestParseWindowInfo_titleWithSpaces(test *testing.T) {
	info, err := parseWindowInfo("  App Name  ,  10  ,  20  ,  100  ,  200  ")

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if info.Title != "App Name" {
		test.Errorf("expected title 'App Name' (trimmed), got %q", info.Title)
	}
	if info.X != 10 {
		test.Errorf("expected X=10 (trimmed), got %d", info.X)
	}
}

func TestParseWindowInfo_wrongPartCount(test *testing.T) {
	_, err := parseWindowInfo("a,b,c")

	if err == nil {
		test.Errorf("expected error for wrong part count, got nil")
	}
}

func TestParseWindowInfo_nonNumericX(test *testing.T) {
	_, err := parseWindowInfo("App,notnum,0,100,100")

	if err == nil {
		test.Errorf("expected parse error for non-numeric X, got nil")
	}
}

func TestParseWindowInfo_nonNumericY(test *testing.T) {
	_, err := parseWindowInfo("App,0,notnum,100,100")

	if err == nil {
		test.Errorf("expected parse error for non-numeric Y, got nil")
	}
}

func TestParseWindowInfo_nonNumericWidth(test *testing.T) {
	_, err := parseWindowInfo("App,0,0,notnum,100")

	if err == nil {
		test.Errorf("expected parse error for non-numeric Width, got nil")
	}
}

func TestParseWindowInfo_nonNumericHeight(test *testing.T) {
	_, err := parseWindowInfo("App,0,0,100,notnum")

	if err == nil {
		test.Errorf("expected parse error for non-numeric Height, got nil")
	}
}

func TestComputeCaptureRect_noBounds(test *testing.T) {
	window := &WindowInfo{X: 100, Y: 50, Width: 800, Height: 600}

	rect, err := computeCaptureRect(window, nil, 1920, 1080)

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if rect.Min.X != 100 || rect.Min.Y != 50 || rect.Max.X != 900 || rect.Max.Y != 650 {
		test.Errorf("expected (100,50)-(900,650), got (%d,%d)-(%d,%d)", rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y)
	}
}

func TestComputeCaptureRect_fullscreenWindow(test *testing.T) {
	window := &WindowInfo{X: 0, Y: 0, Width: 1920, Height: 1080}

	rect, err := computeCaptureRect(window, nil, 1920, 1080)

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if rect.Min.X != 0 || rect.Min.Y != 0 || rect.Max.X != 1920 || rect.Max.Y != 1080 {
		test.Errorf("expected (0,0)-(1920,1080), got (%d,%d)-(%d,%d)", rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y)
	}
}

func TestComputeCaptureRect_customBounds(test *testing.T) {
	window := &WindowInfo{X: 100, Y: 50, Width: 800, Height: 600}
	bounds := image.Rect(200, 150, 400, 300)

	rect, err := computeCaptureRect(window, &bounds, 1920, 1080)

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if rect != bounds {
		test.Errorf("expected %v, got %v", bounds, rect)
	}
}

func TestComputeCaptureRect_clampNegativeOrigin(test *testing.T) {
	window := &WindowInfo{X: -10, Y: -10, Width: 800, Height: 600}

	rect, err := computeCaptureRect(window, nil, 1920, 1080)

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if rect.Min.X != 0 || rect.Min.Y != 0 {
		test.Errorf("expected clamped to (0,0), got (%d,%d)", rect.Min.X, rect.Min.Y)
	}
}

func TestComputeCaptureRect_clampExceedScreen(test *testing.T) {
	window := &WindowInfo{X: 1500, Y: 800, Width: 800, Height: 600}

	rect, err := computeCaptureRect(window, nil, 1920, 1080)

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if rect.Max.X > 1920 || rect.Max.Y > 1080 {
		test.Errorf("expected clamped to screen (1920,1080), got (%d,%d)", rect.Max.X, rect.Max.Y)
	}
}

func TestComputeCaptureRect_zeroAreaAfterClamp(test *testing.T) {
	window := &WindowInfo{X: 2000, Y: 2000, Width: 100, Height: 100}

	_, err := computeCaptureRect(window, nil, 1920, 1080)

	if err == nil {
		test.Errorf("expected error for zero-area rect after clamp, got nil")
	}
}

func TestComputeCaptureRect_invalidRect(test *testing.T) {
	bounds := image.Rect(100, 100, 100, 100) // Min.X == Max.X
	window := &WindowInfo{X: 0, Y: 0, Width: 800, Height: 600}

	_, err := computeCaptureRect(window, &bounds, 1920, 1080)

	if err == nil {
		test.Errorf("expected error for invalid rect, got nil")
	}
}
