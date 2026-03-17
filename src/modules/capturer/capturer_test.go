//go:build darwin

package capturer

import (
	"image"
	"testing"
)

func TestParseWindowInfo_valid(t *testing.T) {
	info, err := parseWindowInfo("Chrome,10,20,800,600")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if info.Title != "Chrome" {
		t.Errorf("expected title 'Chrome', got %q", info.Title)
	}
	if info.X != 10 || info.Y != 20 || info.Width != 800 || info.Height != 600 {
		t.Errorf("expected (10,20,800,600), got (%d,%d,%d,%d)", info.X, info.Y, info.Width, info.Height)
	}
}

func TestParseWindowInfo_titleWithSpaces(t *testing.T) {
	info, err := parseWindowInfo("  App Name  ,  10  ,  20  ,  100  ,  200  ")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if info.Title != "App Name" {
		t.Errorf("expected title 'App Name' (trimmed), got %q", info.Title)
	}
	if info.X != 10 {
		t.Errorf("expected X=10 (trimmed), got %d", info.X)
	}
}

func TestParseWindowInfo_wrongPartCount(t *testing.T) {
	_, err := parseWindowInfo("a,b,c")

	if err == nil {
		t.Errorf("expected error for wrong part count, got nil")
	}
}

func TestParseWindowInfo_nonNumericX(t *testing.T) {
	_, err := parseWindowInfo("App,notnum,0,100,100")

	if err == nil {
		t.Errorf("expected parse error for non-numeric X, got nil")
	}
}

func TestParseWindowInfo_nonNumericY(t *testing.T) {
	_, err := parseWindowInfo("App,0,notnum,100,100")

	if err == nil {
		t.Errorf("expected parse error for non-numeric Y, got nil")
	}
}

func TestParseWindowInfo_nonNumericWidth(t *testing.T) {
	_, err := parseWindowInfo("App,0,0,notnum,100")

	if err == nil {
		t.Errorf("expected parse error for non-numeric Width, got nil")
	}
}

func TestParseWindowInfo_nonNumericHeight(t *testing.T) {
	_, err := parseWindowInfo("App,0,0,100,notnum")

	if err == nil {
		t.Errorf("expected parse error for non-numeric Height, got nil")
	}
}

func TestComputeCaptureRect_noBounds(t *testing.T) {
	window := &WindowInfo{X: 100, Y: 50, Width: 800, Height: 600}

	rect, err := computeCaptureRect(window, nil, 1920, 1080)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rect.Min.X != 100 || rect.Min.Y != 50 || rect.Max.X != 900 || rect.Max.Y != 650 {
		t.Errorf("expected (100,50)-(900,650), got (%d,%d)-(%d,%d)", rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y)
	}
}

func TestComputeCaptureRect_fullscreenWindow(t *testing.T) {
	window := &WindowInfo{X: 0, Y: 0, Width: 1920, Height: 1080}

	rect, err := computeCaptureRect(window, nil, 1920, 1080)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rect.Min.X != 0 || rect.Min.Y != 0 || rect.Max.X != 1920 || rect.Max.Y != 1080 {
		t.Errorf("expected (0,0)-(1920,1080), got (%d,%d)-(%d,%d)", rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y)
	}
}

func TestComputeCaptureRect_customBounds(t *testing.T) {
	window := &WindowInfo{X: 100, Y: 50, Width: 800, Height: 600}
	bounds := image.Rect(200, 150, 400, 300)

	rect, err := computeCaptureRect(window, &bounds, 1920, 1080)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rect != bounds {
		t.Errorf("expected %v, got %v", bounds, rect)
	}
}

func TestComputeCaptureRect_clampNegativeOrigin(t *testing.T) {
	window := &WindowInfo{X: -10, Y: -10, Width: 800, Height: 600}

	rect, err := computeCaptureRect(window, nil, 1920, 1080)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rect.Min.X != 0 || rect.Min.Y != 0 {
		t.Errorf("expected clamped to (0,0), got (%d,%d)", rect.Min.X, rect.Min.Y)
	}
}

func TestComputeCaptureRect_clampExceedScreen(t *testing.T) {
	window := &WindowInfo{X: 1500, Y: 800, Width: 800, Height: 600}

	rect, err := computeCaptureRect(window, nil, 1920, 1080)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rect.Max.X > 1920 || rect.Max.Y > 1080 {
		t.Errorf("expected clamped to screen (1920,1080), got (%d,%d)", rect.Max.X, rect.Max.Y)
	}
}

func TestComputeCaptureRect_zeroAreaAfterClamp(t *testing.T) {
	window := &WindowInfo{X: 2000, Y: 2000, Width: 100, Height: 100}

	_, err := computeCaptureRect(window, nil, 1920, 1080)

	if err == nil {
		t.Errorf("expected error for zero-area rect after clamp, got nil")
	}
}

func TestComputeCaptureRect_invalidRect(t *testing.T) {
	bounds := image.Rect(100, 100, 100, 100) // Min.X == Max.X
	window := &WindowInfo{X: 0, Y: 0, Width: 800, Height: 600}

	_, err := computeCaptureRect(window, &bounds, 1920, 1080)

	if err == nil {
		t.Errorf("expected error for invalid rect, got nil")
	}
}
