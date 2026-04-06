//go:build darwin

package listener

import (
	"image"
	"testing"
)

// TestGoCallbacksDoNotCrossContaminate verifies that calling one Go callback
// function does not send events to channels belonging to other callbacks.
// This is the Go-side regression guard for the gControlUsed fix: even if the
// C event callback were to call both goPositionChangeXY and goRecordBounds in
// a single session, the Go channels remain independent.
func TestGoCallbacksDoNotCrossContaminate(t *testing.T) {
	listener := New()
	current = listener

	t.Run("position change does not send to bounds", func(t *testing.T) {
		goPositionChangeXY(1, 0)

		// Position channel should have the event.
		select {
		case dir := <-listener.teleprompterGridPositionCh:
			if dir != [2]int{1, 0} {
				t.Fatalf("expected [1,0], got %v", dir)
			}
		default:
			t.Fatal("expected position event, channel was empty")
		}

		// Bounds channel must remain empty.
		select {
		case rect := <-listener.boundsCh:
			t.Fatalf("bounds channel should be empty, got %v", rect)
		default:
			// expected
		}
	})

	t.Run("bounds recording does not send to position", func(t *testing.T) {
		goRecordBounds(10, 20, 100, 200)

		// Bounds channel should have the event.
		select {
		case rect := <-listener.boundsCh:
			expected := image.Rect(10, 20, 100, 200)
			if rect != expected {
				t.Fatalf("expected %v, got %v", expected, rect)
			}
		default:
			t.Fatal("expected bounds event, channel was empty")
		}

		// Position channel must remain empty.
		select {
		case dir := <-listener.teleprompterGridPositionCh:
			t.Fatalf("position channel should be empty, got %v", dir)
		default:
			// expected
		}
	})

	t.Run("opacity change does not send to bounds or position", func(t *testing.T) {
		goOpacityChange(1)

		select {
		case dir := <-listener.teleprompterOverlayOpacityCh:
			if dir != 1 {
				t.Fatalf("expected 1, got %d", dir)
			}
		default:
			t.Fatal("expected opacity event, channel was empty")
		}

		select {
		case rect := <-listener.boundsCh:
			t.Fatalf("bounds channel should be empty, got %v", rect)
		default:
		}
		select {
		case dir := <-listener.teleprompterGridPositionCh:
			t.Fatalf("position channel should be empty, got %v", dir)
		default:
		}
	})

	t.Run("toggle does not send to bounds or position", func(t *testing.T) {
		goTeleprompterToggle()

		select {
		case <-listener.togglesCh:
			// expected
		default:
			t.Fatal("expected toggle event, channel was empty")
		}

		select {
		case rect := <-listener.boundsCh:
			t.Fatalf("bounds channel should be empty, got %v", rect)
		default:
		}
		select {
		case dir := <-listener.teleprompterGridPositionCh:
			t.Fatalf("position channel should be empty, got %v", dir)
		default:
		}
	})

	t.Run("font size change does not send to bounds or position", func(t *testing.T) {
		goFontSizeChange(1)

		select {
		case dir := <-listener.teleprompterTextFontSizeCh:
			if dir != 1 {
				t.Fatalf("expected 1, got %d", dir)
			}
		default:
			t.Fatal("expected font size event, channel was empty")
		}

		select {
		case rect := <-listener.boundsCh:
			t.Fatalf("bounds channel should be empty, got %v", rect)
		default:
		}
		select {
		case dir := <-listener.teleprompterGridPositionCh:
			t.Fatalf("position channel should be empty, got %v", dir)
		default:
		}
		select {
		case dir := <-listener.teleprompterOverlayOpacityCh:
			t.Fatalf("opacity channel should be empty, got %d", dir)
		default:
		}
	})

	t.Run("position drain and replace keeps latest", func(t *testing.T) {
		goPositionChangeXY(1, 0)
		goPositionChangeXY(0, -1)

		select {
		case dir := <-listener.teleprompterGridPositionCh:
			if dir != [2]int{0, -1} {
				t.Fatalf("expected latest direction [0,-1], got %v", dir)
			}
		default:
			t.Fatal("expected position event, channel was empty")
		}
	})

	t.Run("bounds drain and replace keeps latest", func(t *testing.T) {
		goRecordBounds(0, 0, 50, 50)
		goRecordBounds(10, 10, 200, 200)

		select {
		case rect := <-listener.boundsCh:
			expected := image.Rect(10, 10, 200, 200)
			if rect != expected {
				t.Fatalf("expected latest bounds %v, got %v", expected, rect)
			}
		default:
			t.Fatal("expected bounds event, channel was empty")
		}
	})

	current = nil
}
