package diff

import (
	"image"
	"math"
)

// RingBuffer stores the last N screenshot images in memory for comparison.
type RingBuffer struct {
	images []*image.RGBA
	size   int
	idx    int
	count  int
}

// NewRingBuffer creates a ring buffer with the given capacity.
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		images: make([]*image.RGBA, size),
		size:   size,
	}
}

// Push adds a new image to the ring buffer.
func (rb *RingBuffer) Push(img *image.RGBA) {
	rb.images[rb.idx] = img
	rb.idx = (rb.idx + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

// Last returns the most recently pushed image, or nil if empty.
func (rb *RingBuffer) Last() *image.RGBA {
	if rb.count == 0 {
		return nil
	}
	lastIdx := (rb.idx - 1 + rb.size) % rb.size
	return rb.images[lastIdx]
}

// Count returns how many images are currently stored.
func (rb *RingBuffer) Count() int {
	return rb.count
}

// HasChanged compares two images and returns true if they differ
// beyond the given threshold. Threshold is 0.0 to 1.0, representing
// the fraction of pixels that must differ.
func HasChanged(a, b *image.RGBA, threshold float64) bool {
	if a == nil || b == nil {
		return true
	}

	boundsA := a.Bounds()
	boundsB := b.Bounds()

	// Different dimensions = definitely changed
	if boundsA.Dx() != boundsB.Dx() || boundsA.Dy() != boundsB.Dy() {
		return true
	}

	totalPixels := boundsA.Dx() * boundsA.Dy()
	if totalPixels == 0 {
		return false
	}

	diffCount := 0
	requiredDiffs := int(math.Ceil(float64(totalPixels) * threshold))

	for y := boundsA.Min.Y; y < boundsA.Max.Y; y++ {
		for x := boundsA.Min.X; x < boundsA.Max.X; x++ {
			idxA := a.PixOffset(x, y)
			idxB := b.PixOffset(x-boundsA.Min.X+boundsB.Min.X, y-boundsA.Min.Y+boundsB.Min.Y)

			// Compare RGB channels (ignore alpha), allow small per-channel tolerance
			// to handle minor anti-aliasing or sub-pixel rendering differences
			const channelTolerance = 10
			dr := absDiff(a.Pix[idxA], b.Pix[idxB])
			dg := absDiff(a.Pix[idxA+1], b.Pix[idxB+1])
			db := absDiff(a.Pix[idxA+2], b.Pix[idxB+2])

			if dr > channelTolerance || dg > channelTolerance || db > channelTolerance {
				diffCount++
				if diffCount >= requiredDiffs {
					return true
				}
			}
		}
	}

	return false
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
