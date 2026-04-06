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

// capturedWindowPID returns the process identifier of the application whose
// window is being captured (the foreground app at capture time).
//
// Uses NSWorkspace — a pure metadata query that does not capture screen pixels
// and does not trigger the screen-recording indicator.
//
// Returns the PID on success, or -1 if the application cannot be determined.
int capturedWindowPID(void) {
    NSRunningApplication* app = [[NSWorkspace sharedWorkspace] frontmostApplication];
    if (app == nil) return -1;
    return (int)[app processIdentifier];
}

// capturedWindowRect returns the bounding rect of the front window belonging to the
// given PID. Searches the on-screen window list for the first layer-0 window owned
// by the PID.
//
// Returns 1 on success and populates *outX, *outY, *outW, *outH.
// Returns 0 if no matching window is found.
int capturedWindowRect(int targetPID, int* outX, int* outY, int* outW, int* outH) {
    CFArrayRef windowList = CGWindowListCopyWindowInfo(
        kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
        kCGNullWindowID);
    if (windowList == NULL) return 0;

    int found = 0;
    CFIndex count = CFArrayGetCount(windowList);
    for (CFIndex i = 0; i < count; i++) {
        CFDictionaryRef info = (CFDictionaryRef)CFArrayGetValueAtIndex(windowList, i);

        CFNumberRef pidRef = (CFNumberRef)CFDictionaryGetValue(info, kCGWindowOwnerPID);
        if (pidRef == NULL) continue;
        int windowPID = 0;
        CFNumberGetValue(pidRef, kCFNumberIntType, &windowPID);
        if (windowPID != targetPID) continue;

        CFNumberRef layerRef = (CFNumberRef)CFDictionaryGetValue(info, kCGWindowLayer);
        if (layerRef != NULL) {
            int layer = 0;
            CFNumberGetValue(layerRef, kCFNumberIntType, &layer);
            if (layer != 0) continue;
        }

        CFDictionaryRef boundsRef = (CFDictionaryRef)CFDictionaryGetValue(info, kCGWindowBounds);
        if (boundsRef == NULL) continue;
        CGRect rect;
        if (!CGRectMakeWithDictionaryRepresentation(boundsRef, &rect)) continue;

        *outX = (int)rect.origin.x;
        *outY = (int)rect.origin.y;
        *outW = (int)rect.size.width;
        *outH = (int)rect.size.height;
        found = 1;
        break;
    }

    CFRelease(windowList);
    return found;
}

// captureScreenRect captures a region of the main display and returns raw RGBA bytes.
// Output format: RGBA, 4 bytes per pixel, pixels in row-major order (top to bottom, left to right).
// outLength is filled with the total byte count (width * height * 4).
// outWidth and outHeight are filled with the actual physical image dimensions (accounting for display scaling on HiDPI displays).
// The caller must free() the returned pointer.
unsigned char* captureScreenRect(int x, int y, int width, int height, int* outLength, int* outWidth, int* outHeight) {
	if (width <= 0 || height <= 0 || !outLength || !outWidth || !outHeight) {
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
	*outWidth = (int)imgWidth;
	*outHeight = (int)imgHeight;
	return pixelBuffer;
}
