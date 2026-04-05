//go:build darwin

package appkit

/*
#cgo CFLAGS: -x objective-c -mmacosx-version-min=13.0
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics

#import <Cocoa/Cocoa.h>
#import <CoreGraphics/CoreGraphics.h>
#include <string.h>

// Configurable appearance globals (set before createOverlayWindow).
static CGFloat gFontSize   = 14.0;
static CGFloat gOpacity        = 0.075;
static CGFloat gDefaultOpacity = 0.075;
static CGFloat gFontWeight = -0.8; // NSFontWeightUltraLight
static int     gAlignment  = 1;    // 0=left, 1=center, 2=right
static CGFloat gMargin       = 20.0;
static CGFloat gFadeDuration = 0.75;
static char    gFontFamily[256] = "";

// Current text for repositioning without a new Display call.
static NSString* gCurrentText = nil;

// Per-pixel adaptive color: when YES, text color is derived from the inverted
// background behind the overlay strip. When NO, text is flat black.
static BOOL gAdaptiveColor = NO;

// fontWeightFromName maps a weight name to an NSFontWeight value.
static CGFloat fontWeightFromName(const char* name) {
    if (strcmp(name, "ultralight") == 0) return NSFontWeightUltraLight;
    if (strcmp(name, "thin") == 0)       return NSFontWeightThin;
    if (strcmp(name, "light") == 0)      return NSFontWeightLight;
    if (strcmp(name, "regular") == 0)    return NSFontWeightRegular;
    if (strcmp(name, "medium") == 0)     return NSFontWeightMedium;
    if (strcmp(name, "semibold") == 0)   return NSFontWeightSemibold;
    if (strcmp(name, "bold") == 0)       return NSFontWeightBold;
    if (strcmp(name, "heavy") == 0)      return NSFontWeightHeavy;
    if (strcmp(name, "black") == 0)      return NSFontWeightBlack;
    return NSFontWeightUltraLight;
}

// alignmentConstant maps gAlignment to an NSTextAlignment.
static NSTextAlignment alignmentConstant(void) {
    switch (gAlignment) {
        case 0: return NSTextAlignmentLeft;
        case 1: return NSTextAlignmentCenter;
        default: return NSTextAlignmentRight;
    }
}

// resolveFont creates an NSFont from the global config.
static NSFont* resolveFont(void) {
    if (strlen(gFontFamily) > 0) {
        NSString* family = [NSString stringWithUTF8String:gFontFamily];
        NSFont* font = [NSFont fontWithName:family size:gFontSize];
        if (font != nil) return font;
    }
    return [NSFont systemFontOfSize:gFontSize weight:gFontWeight];
}

// configureOverlay sets the global appearance variables from C parameters.
// Must be called before createOverlayWindow.
void configureOverlay(const char* fontFamily, const char* fontWeight,
                      double fontSize, double opacity, int alignment,
                      int adaptiveColor, double fadeDuration) {
    gFontSize       = (CGFloat)fontSize;
    gOpacity        = (CGFloat)opacity;
    gDefaultOpacity = (CGFloat)opacity;
    gFontWeight     = fontWeightFromName(fontWeight);
    gAlignment      = alignment;
    gAdaptiveColor  = (adaptiveColor != 0);
    gFadeDuration   = (CGFloat)fadeDuration;
    if (fontFamily != NULL && strlen(fontFamily) > 0) {
        strlcpy(gFontFamily, fontFamily, sizeof(gFontFamily));
    }
}

// captureInvertedStrip captures the screen strip behind the overlay and returns
// an NSColor pattern whose pixels are the per-pixel color inversion of the background.
// The pattern is sized to stripWidth x stripHeight logical points.
// Returns nil if capture or context creation fails.
static NSColor* captureInvertedStrip(CGFloat stripX, CGFloat stripY,
                                     CGFloat stripWidth, CGFloat stripHeight,
                                     CGFloat screenHeight) {
    // CoreGraphics coordinates (top-left origin).
    CGFloat cgY = screenHeight - stripY - stripHeight;
    CGRect captureRect = CGRectMake(stripX, cgY, stripWidth, stripHeight);
    CGImageRef background = CGDisplayCreateImageForRect(CGMainDisplayID(), captureRect);
    if (background == NULL) return nil;

    size_t imageWidth  = CGImageGetWidth(background);
    size_t imageHeight = CGImageGetHeight(background);

    CGColorSpaceRef colorSpace = CGColorSpaceCreateWithName(kCGColorSpaceGenericRGB);
    if (colorSpace == NULL) { CFRelease(background); return nil; }

    CGBitmapInfo bitmapInfo = kCGBitmapByteOrder32Big | kCGImageAlphaPremultipliedLast;
    CGContextRef context = CGBitmapContextCreate(
        NULL, imageWidth, imageHeight, 8, imageWidth * 4, colorSpace, bitmapInfo);
    if (context == NULL) {
        CFRelease(colorSpace);
        CFRelease(background);
        return nil;
    }

    // Draw background then invert via difference blend with solid white.
    CGRect imageRect = CGRectMake(0, 0, imageWidth, imageHeight);
    CGContextDrawImage(context, imageRect, background);
    CGContextSetBlendMode(context, kCGBlendModeDifference);
    CGContextSetRGBFillColor(context, 1.0, 1.0, 1.0, 1.0);
    CGContextFillRect(context, imageRect);

    CGImageRef inverted = CGBitmapContextCreateImage(context);
    CFRelease(context);
    CFRelease(background);
    if (inverted == NULL) { CFRelease(colorSpace); return nil; }

    // Bake gOpacity into the pixel alpha channel. colorWithAlphaComponent: on a
    // pattern NSColor is unreliable; drawing the inverted image at gOpacity onto
    // a transparent context ensures each pixel carries the exact target alpha.
    CGContextRef alphaContext = CGBitmapContextCreate(
        NULL, imageWidth, imageHeight, 8, imageWidth * 4, colorSpace, bitmapInfo);
    CFRelease(colorSpace);
    if (alphaContext == NULL) { CFRelease(inverted); return nil; }

    CGRect imageRect2 = CGRectMake(0, 0, imageWidth, imageHeight);
    CGContextClearRect(alphaContext, imageRect2);
    CGContextSetAlpha(alphaContext, gOpacity);
    CGContextDrawImage(alphaContext, imageRect2, inverted);
    CFRelease(inverted);

    CGImageRef finalImage = CGBitmapContextCreateImage(alphaContext);
    CFRelease(alphaContext);
    if (finalImage == NULL) return nil;

    // Create NSImage sized to logical strip dimensions for correct pattern alignment.
    NSImage* patternImage = [[NSImage alloc] initWithCGImage:finalImage
                                                        size:NSMakeSize(stripWidth, stripHeight)];
    CFRelease(finalImage);

    return [NSColor colorWithPatternImage:patternImage];
}

// OverlayDelegate receives performSelector calls on the main thread
// and forwards them to the overlay window.
@interface OverlayDelegate : NSObject {
    NSWindow* _window;
    NSInteger _generation;
}
- (instancetype)initWithWindow:(NSWindow*)window;
- (void)showWindow;
- (void)hideWindow;
- (void)updateText:(NSString*)text;
- (void)reposition;
- (void)renderAdaptiveColor:(NSTimer*)timer;
@end

@implementation OverlayDelegate

- (instancetype)initWithWindow:(NSWindow*)window {
    self = [super init];
    if (self) {
        _window = window;
    }
    return self;
}

- (void)showWindow {
    if ([_window alphaValue] >= 1.0 && [_window isVisible]) return;
    _generation++;
    [_window setAlphaValue:0.0];
    [_window makeKeyAndOrderFront:nil];
    [NSAnimationContext runAnimationGroup:^(NSAnimationContext* context) {
        [context setDuration:gFadeDuration];
        [[_window animator] setAlphaValue:1.0];
    }];
}

- (void)hideWindow {
    if (![_window isVisible]) return;
    _generation++;
    NSInteger generation = _generation;
    [NSAnimationContext runAnimationGroup:^(NSAnimationContext* context) {
        [context setDuration:gFadeDuration];
        [[_window animator] setAlphaValue:0.0];
    } completionHandler:^{
        if (self->_generation == generation) {
            [self->_window orderOut:nil];
        }
    }];
}

- (void)updateText:(NSString*)text {
    if (gFadeDuration <= 0.0 || ![_window isVisible]) {
        [self _applyText:text];
        return;
    }
    _generation++;
    NSInteger generation = _generation;
    [NSAnimationContext runAnimationGroup:^(NSAnimationContext* context) {
        [context setDuration:gFadeDuration];
        [[_window animator] setAlphaValue:0.0];
    } completionHandler:^{
        if (self->_generation != generation) return;
        [self _applyText:text];
        [NSAnimationContext runAnimationGroup:^(NSAnimationContext* context) {
            [context setDuration:gFadeDuration];
            [[self->_window animator] setAlphaValue:1.0];
        }];
    }];
}

- (void)_applyText:(NSString*)text {
    gCurrentText = [text copy];

    NSView* contentView = [_window contentView];
    NSTextField* label = [contentView viewWithTag:1001];
    if (label == nil) return;

    NSFont* font = resolveFont();
    NSMutableParagraphStyle* paragraph = [[NSMutableParagraphStyle alloc] init];
    [paragraph setAlignment:alignmentConstant()];

    if (gAdaptiveColor) {
        // Set text temporarily to compute height, then layout full strip.
        NSDictionary* tempAttributes = @{
            NSFontAttributeName: font,
            NSParagraphStyleAttributeName: paragraph
        };
        label.attributedStringValue =
            [[NSAttributedString alloc] initWithString:text attributes:tempAttributes];
        [label sizeToFit];

        NSScreen* screen = [NSScreen mainScreen];
        NSRect screenFrame = [screen frame];
        CGFloat stripWidth  = screenFrame.size.width - 2 * gMargin;
        CGFloat stripHeight = label.frame.size.height;
        CGFloat originX     = screenFrame.origin.x + gMargin;
        CGFloat originY     = screenFrame.origin.y + gMargin;

        [_window setFrame:NSMakeRect(originX, originY, stripWidth, stripHeight) display:YES];
        [label setFrame:NSMakeRect(0, 0, stripWidth, stripHeight)];

        [self renderAdaptiveColor:nil];
        return;
    }

    // Non-adaptive: flat black text, window sized to text.
    NSColor* textColor = [NSColor colorWithWhite:0.0 alpha:gOpacity];
    NSDictionary* attributes = @{
        NSFontAttributeName: font,
        NSForegroundColorAttributeName: textColor,
        NSParagraphStyleAttributeName: paragraph
    };
    label.attributedStringValue =
        [[NSAttributedString alloc] initWithString:text attributes:attributes];

    [label sizeToFit];
    NSSize textSize = label.frame.size;
    NSScreen* screen = [NSScreen mainScreen];
    NSRect screenFrame = [screen frame];

    CGFloat originX;
    switch (gAlignment) {
        case 0: // left
            originX = screenFrame.origin.x + gMargin;
            break;
        case 2: // right
            originX = screenFrame.origin.x + screenFrame.size.width - textSize.width - gMargin;
            break;
        default: // center
            originX = screenFrame.origin.x + (screenFrame.size.width - textSize.width) / 2.0;
            break;
    }
    CGFloat originY = screenFrame.origin.y + gMargin;
    NSRect windowFrame = NSMakeRect(originX, originY, textSize.width, textSize.height);
    [_window setFrame:windowFrame display:YES];
    [label setFrame:NSMakeRect(0, 0, textSize.width, textSize.height)];
}

- (void)reposition {
    if (gCurrentText != nil) {
        [self _applyText:gCurrentText];
    }
}

- (void)renderAdaptiveColor:(NSTimer*)timer {
    if (gCurrentText == nil || [gCurrentText length] == 0) return;

    NSView* contentView = [_window contentView];
    NSTextField* label = [contentView viewWithTag:1001];
    if (label == nil) return;

    NSRect windowFrame = [_window frame];
    if (windowFrame.size.width <= 0 || windowFrame.size.height <= 0) return;

    NSScreen* screen = [NSScreen mainScreen];
    CGFloat screenHeight = [screen frame].size.height;

    NSColor* patternColor = captureInvertedStrip(
        windowFrame.origin.x, windowFrame.origin.y,
        windowFrame.size.width, windowFrame.size.height,
        screenHeight);
    if (patternColor == nil) return;

    NSFont* font = resolveFont();
    NSMutableParagraphStyle* paragraph = [[NSMutableParagraphStyle alloc] init];
    [paragraph setAlignment:alignmentConstant()];
    NSDictionary* attributes = @{
        NSFontAttributeName: font,
        NSForegroundColorAttributeName: patternColor,
        NSParagraphStyleAttributeName: paragraph
    };
    label.attributedStringValue =
        [[NSAttributedString alloc] initWithString:gCurrentText attributes:attributes];
}

@end

// Global delegate for cross-thread calls.
static OverlayDelegate* gDelegate = nil;

// createOverlayWindow creates a borderless, click-through, capture-excluded overlay.
// Position is determined by gAlignment. MUST be called on the main thread.
//
// Returns a pointer to the NSWindow.
void* createOverlayWindow(void) {
    NSRect windowFrame = NSMakeRect(0, 0, 1, 1);

    NSWindow* window = [[NSWindow alloc]
        initWithContentRect:windowFrame
        styleMask:NSWindowStyleMaskBorderless
        backing:NSBackingStoreBuffered
        defer:NO];

    [window setOpaque:NO];
    [window setBackgroundColor:[NSColor clearColor]];
    [window setIgnoresMouseEvents:YES];
    [window setLevel:NSFloatingWindowLevel];
    [window setHasShadow:NO];

    // Exclude from all screen capture pipelines.
    [window setSharingType:NSWindowSharingNone];

    // Exclude from Mission Control, space thumbnails, and Cmd+Tab.
    [window setCollectionBehavior:
        NSWindowCollectionBehaviorCanJoinAllSpaces |
        NSWindowCollectionBehaviorStationary |
        NSWindowCollectionBehaviorIgnoresCycle];

    // Single-line text label with no background, no padding, no radius.
    NSTextField* label = [[NSTextField alloc] initWithFrame:NSMakeRect(0, 0, 1, 1)];
    [label setBezeled:NO];
    [label setDrawsBackground:NO];
    [label setEditable:NO];
    [label setSelectable:NO];
    [label setLineBreakMode:NSLineBreakByTruncatingTail];
    [label setMaximumNumberOfLines:1];
    label.alignment = alignmentConstant();

    NSFont* font = resolveFont();
    NSColor* textColor = [NSColor colorWithWhite:0.0 alpha:gOpacity];
    NSMutableParagraphStyle* paragraph = [[NSMutableParagraphStyle alloc] init];
    [paragraph setAlignment:alignmentConstant()];
    NSDictionary* attributes = @{
        NSFontAttributeName: font,
        NSForegroundColorAttributeName: textColor,
        NSParagraphStyleAttributeName: paragraph
    };
    label.attributedStringValue =
        [[NSAttributedString alloc] initWithString:@"" attributes:attributes];

    label.tag = 1001;
    [[window contentView] addSubview:label];

    [window setAlphaValue:0.0];
    [window orderOut:nil]; // Start hidden.

    gDelegate = [[OverlayDelegate alloc] initWithWindow:window];

    return (void*)window;
}

// showOverlay dispatches showWindow to the main thread.
void showOverlay(void) {
    [gDelegate performSelectorOnMainThread:@selector(showWindow)
                                withObject:nil
                             waitUntilDone:NO];
}

// hideOverlay dispatches hideWindow to the main thread.
void hideOverlay(void) {
    [gDelegate performSelectorOnMainThread:@selector(hideWindow)
                                withObject:nil
                             waitUntilDone:NO];
}

// updateOverlayText dispatches text update to the main thread.
void updateOverlayText(const char* text) {
    NSString* string = [NSString stringWithUTF8String:text];
    [gDelegate performSelectorOnMainThread:@selector(updateText:)
                                withObject:string
                             waitUntilDone:NO];
}

// setOverlayPosition updates gAlignment and repositions the window.
void setOverlayPosition(int alignment) {
    gAlignment = alignment;
    [gDelegate performSelectorOnMainThread:@selector(reposition)
                                withObject:nil
                             waitUntilDone:NO];
}

// setOverlayOpacity adjusts gOpacity by the given delta and re-renders the current text.
// The value is clamped to [0.0, 1.0].
void setOverlayOpacity(double delta) {
    gOpacity += (CGFloat)delta;
    if (gOpacity < 0.0) gOpacity = 0.0;
    if (gOpacity > 1.0) gOpacity = 1.0;
    [gDelegate performSelectorOnMainThread:@selector(reposition)
                                withObject:nil
                             waitUntilDone:NO];
}

// resetOverlayOpacity restores gOpacity to the configured default and re-renders.
void resetOverlayOpacity(void) {
    gOpacity = gDefaultOpacity;
    [gDelegate performSelectorOnMainThread:@selector(reposition)
                                withObject:nil
                             waitUntilDone:NO];
}

*/
import "C"

import "unsafe"

// OverlayConfig holds the configurable appearance settings for the overlay window.
type OverlayConfig struct {
	FontFamily    string
	FontWeight    string
	FontSize      float64
	Opacity       float64
	Position      string  // "left", "center", "right"
	AdaptiveColor bool    // per-pixel background-inverted text color
	FadeDuration  float64 // fade animation duration in seconds
}

// AlignmentFromPosition maps a position name to an integer: 0=left, 1=center, 2=right.
func AlignmentFromPosition(position string) int {
	switch position {
	case "left":
		return 0
	case "center":
		return 1
	default:
		return 2
	}
}

// ConfigureOverlay sets the overlay appearance before window creation.
// Must be called before CreateOverlayWindow.
func ConfigureOverlay(config OverlayConfig) {
	cFamily := C.CString(config.FontFamily)
	defer C.free(unsafe.Pointer(cFamily))
	cWeight := C.CString(config.FontWeight)
	defer C.free(unsafe.Pointer(cWeight))

	adaptiveFlag := 0
	if config.AdaptiveColor {
		adaptiveFlag = 1
	}

	C.configureOverlay(cFamily, cWeight,
		C.double(config.FontSize), C.double(config.Opacity),
		C.int(AlignmentFromPosition(config.Position)),
		C.int(adaptiveFlag),
		C.double(config.FadeDuration))
}

// CreateOverlayWindow creates a borderless, click-through, capture-excluded overlay.
// MUST be called on the main thread via RunOnMainThread.
// Call ConfigureOverlay first to set font, opacity, and position.
//
// Returns an opaque pointer to the NSWindow.
func CreateOverlayWindow() unsafe.Pointer {
	return unsafe.Pointer(C.createOverlayWindow())
}

// ShowOverlay makes the overlay window visible.
// Safe to call from any goroutine; dispatched to the main thread internally.
func ShowOverlay() {
	C.showOverlay()
}

// HideOverlay hides the overlay window without destroying it.
// Safe to call from any goroutine; dispatched to the main thread internally.
func HideOverlay() {
	C.hideOverlay()
}

// UpdateOverlayText updates the text displayed on the overlay window.
// Safe to call from any goroutine; dispatched to the main thread internally.
//
// text: the string to display. Empty string clears the label.
func UpdateOverlayText(text string) {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	C.updateOverlayText(cText)
}

// SetOverlayPosition changes the window alignment at runtime and repositions.
// position: "left", "center", or "right".
// Safe to call from any goroutine; dispatched to the main thread internally.
func SetOverlayPosition(position string) {
	C.setOverlayPosition(C.int(AlignmentFromPosition(position)))
}

// SetOverlayOpacity adjusts gOpacity by the given delta (e.g. +0.025 or -0.025)
// and re-renders the current text. The value is clamped to [0.0, 1.0].
// Safe to call from any goroutine; dispatched to the main thread internally.
func SetOverlayOpacity(delta float64) {
	C.setOverlayOpacity(C.double(delta))
}

// ResetOverlayOpacity restores gOpacity to the configured default value
// and re-renders the current text.
// Safe to call from any goroutine; dispatched to the main thread internally.
func ResetOverlayOpacity() {
	C.resetOverlayOpacity()
}
