#import <AppKit/AppKit.h>
#import <CoreGraphics/CoreGraphics.h>
#include <stdlib.h>
#include <string.h>

// getMainDisplayWidth returns the pixel width of the primary display.
int getMainDisplayWidth(void) {
	CGRect displayBounds = CGDisplayBounds(CGMainDisplayID());
	return (int)displayBounds.size.width;
}

// getMainDisplayHeight returns the pixel height of the primary display.
int getMainDisplayHeight(void) {
	CGRect displayBounds = CGDisplayBounds(CGMainDisplayID());
	return (int)displayBounds.size.height;
}

// captureScreenRect captures a region of the main display and returns raw RGBA bytes.
// Output format: RGBA, 4 bytes per pixel, pixels in row-major order (top to bottom, left to right).
// outLength is filled with the total byte count (width * height * 4).
// The caller must free() the returned pointer.
unsigned char* captureScreenRect(int x, int y, int width, int height, int* outLength) {
	if (width <= 0 || height <= 0 || !outLength) {
		return NULL;
	}

	// Create CGRect for the capture area
	CGRect captureRect = CGRectMake(x, y, width, height);

	// Capture the screen region
	CGImageRef cgImage = CGDisplayCreateImageForRect(CGMainDisplayID(), captureRect);
	if (!cgImage) {
		return NULL;
	}

	// Get image properties
	size_t imgWidth = CGImageGetWidth(cgImage);
	size_t imgHeight = CGImageGetHeight(cgImage);

	// Create a color space for RGBA
	CGColorSpaceRef colorSpace = CGColorSpaceCreateWithName(kCGColorSpaceGenericRGB);
	if (!colorSpace) {
		CFRelease(cgImage);
		return NULL;
	}

	// Create a bitmap context for RGBA data
	size_t bytesPerPixel = 4;
	size_t bytesPerRow = imgWidth * bytesPerPixel;
	size_t bufferSize = bytesPerRow * imgHeight;

	unsigned char* pixelBuffer = (unsigned char*)malloc(bufferSize);
	if (!pixelBuffer) {
		CFRelease(colorSpace);
		CFRelease(cgImage);
		return NULL;
	}

	CGBitmapInfo bitmapInfo = kCGBitmapByteOrder32Big | kCGImageAlphaPremultipliedLast;
	CGContextRef context = CGBitmapContextCreate(
		pixelBuffer,
		imgWidth,
		imgHeight,
		8,                  // bits per component
		bytesPerRow,
		colorSpace,
		bitmapInfo
	);

	if (!context) {
		free(pixelBuffer);
		CFRelease(colorSpace);
		CFRelease(cgImage);
		return NULL;
	}

	// Draw the image into the context
	CGContextDrawImage(context, CGRectMake(0, 0, imgWidth, imgHeight), cgImage);

	// Clean up
	CFRelease(context);
	CFRelease(colorSpace);
	CFRelease(cgImage);

	*outLength = (int)bufferSize;
	return pixelBuffer;
}
