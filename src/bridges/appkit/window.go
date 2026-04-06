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
static CGFloat gTextOpacity        = 0.05; // text/pattern alpha, adjusted by hotkey +/-
static CGFloat gDefaultTextOpacity = 0.05; // configured default from TELEPROMPTER_OPACITY
static CGFloat gOverlayInterpolation    = 0.0;   // overlay window alpha: 0.0 = hidden, 1.0 = fully visible
static CGFloat gFontWeight = -0.8; // NSFontWeightUltraLight
static int     gAlignment  = 3;    // 0=left, 1=center, 2=right, 3=dynamic
static CGFloat gMargin       = 20.0;
static CGFloat gFadeDuration = 0.75;
static char    gFontFamily[256] = "";

// Current text for repositioning without a new Display call.
static NSString* gCurrentText = nil;

// Per-pixel adaptive color: when YES, text color is derived from the inverted
// background behind the overlay strip. When NO, text is flat black.
static BOOL gAdaptiveColor = NO;

// Grid animation state.
// gMoveInProgress: YES while a fade-out → reposition → fade-in sequence is active.
// gIntendedVisible: mirrors the Go-side visibility intent; updated by show/hideOverlay.
// Both fields are only written on the main thread so no explicit locking is needed.
static BOOL gMoveInProgress  = NO;
static BOOL gIntendedVisible = NO;

// Current grid position. Updated by commitMoveToGridSpot, read by reposition.
static double gGridCol = 0.5;
static double gGridRow = 0.5;

// Forward declaration — defined after the OverlayDelegate @implementation block.
static NSRect gridSpotFrame(NSWindow* window, NSTextField* label, double col, double row);

// Raw window bounds fallback (Y-down screen coordinates, logical pixels).
// Used by commitMoveToGridSpot when no browser canvas is detected.
static CGFloat gWindowX      = 0.0;
static CGFloat gWindowY      = 0.0;
static CGFloat gWindowWidth  = 0.0;
static CGFloat gWindowHeight = 0.0;

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
// For dynamic mode (3), derives alignment from the current grid column position.
static NSTextAlignment alignmentConstant(void) {
    switch (gAlignment) {
        case 0: return NSTextAlignmentLeft;
        case 1: return NSTextAlignmentCenter;
        case 2: return NSTextAlignmentRight;
        default: // dynamic: adapt to grid column
            if (gGridCol <= 0.0) return NSTextAlignmentLeft;
            if (gGridCol >= 1.0) return NSTextAlignmentRight;
            return NSTextAlignmentCenter;
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
    gTextOpacity        = (CGFloat)opacity;
    gDefaultTextOpacity = (CGFloat)opacity;
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

    // Bake gTextOpacity into the pixel alpha channel. colorWithAlphaComponent:
    // on a pattern NSColor is unreliable; drawing the inverted image at the threshold
    // onto a transparent context ensures each pixel carries the exact target alpha.
    CGContextRef alphaContext = CGBitmapContextCreate(
        NULL, imageWidth, imageHeight, 8, imageWidth * 4, colorSpace, bitmapInfo);
    CFRelease(colorSpace);
    if (alphaContext == NULL) { CFRelease(inverted); return nil; }

    CGRect imageRect2 = CGRectMake(0, 0, imageWidth, imageHeight);
    CGContextClearRect(alphaContext, imageRect2);
    CGContextSetAlpha(alphaContext, gTextOpacity);
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
- (void)orderOutForCapture;
- (void)orderInAfterCapture;
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
    if (gOverlayInterpolation >= 1.0 && [_window isVisible]) return;
    gOverlayInterpolation = 1.0;
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
    gOverlayInterpolation = 0.0;
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
        // Set text temporarily to compute height, then position at grid spot.
        NSDictionary* tempAttributes = @{
            NSFontAttributeName: font,
            NSParagraphStyleAttributeName: paragraph
        };
        label.attributedStringValue =
            [[NSAttributedString alloc] initWithString:text attributes:tempAttributes];
        [label sizeToFit];

        // Position at grid spot; fall back to full-width strip at screen edge.
        NSRect frame = gridSpotFrame(_window, label, gGridCol, gGridRow);
        if (NSEqualRects(frame, NSZeroRect)) {
            NSScreen* screen = [NSScreen mainScreen];
            NSRect screenFrame = [screen frame];
            CGFloat stripWidth  = screenFrame.size.width - 2 * gMargin;
            CGFloat stripHeight = label.frame.size.height;
            CGFloat originX     = screenFrame.origin.x + gMargin;
            CGFloat originY     = screenFrame.origin.y + gMargin;
            frame = NSMakeRect(originX, originY, stripWidth, stripHeight);
        }
        [_window setFrame:frame display:YES];
        [label setFrame:NSMakeRect(0, 0, frame.size.width, frame.size.height)];

        [self renderAdaptiveColor:nil];
        return;
    }

    // Non-adaptive: flat black text, window sized to text.
    NSColor* textColor = [NSColor colorWithWhite:0.0 alpha:gTextOpacity];
    NSDictionary* attributes = @{
        NSFontAttributeName: font,
        NSForegroundColorAttributeName: textColor,
        NSParagraphStyleAttributeName: paragraph
    };
    label.attributedStringValue =
        [[NSAttributedString alloc] initWithString:text attributes:attributes];
    [label sizeToFit];

    // Position at the current grid spot; fall back to screen edge if no bounds set.
    NSRect frame = gridSpotFrame(_window, label, gGridCol, gGridRow);
    if (NSEqualRects(frame, NSZeroRect)) {
        NSSize textSize = label.frame.size;
        NSScreen* screen = [NSScreen mainScreen];
        NSRect screenFrame = [screen frame];
        CGFloat originX;
        switch (gAlignment) {
            case 0:
                originX = screenFrame.origin.x + gMargin;
                break;
            case 2:
                originX = screenFrame.origin.x + screenFrame.size.width - textSize.width - gMargin;
                break;
            default:
                originX = screenFrame.origin.x + (screenFrame.size.width - textSize.width) / 2.0;
                break;
        }
        CGFloat originY = screenFrame.origin.y + gMargin;
        frame = NSMakeRect(originX, originY, textSize.width, textSize.height);
    }
    [_window setFrame:frame display:YES];
    [label setFrame:NSMakeRect(0, 0, frame.size.width, frame.size.height)];
}

- (void)reposition {
    if (gCurrentText == nil || [gCurrentText length] == 0) return;

    // Re-render text attributes (opacity, font, alignment) without repositioning.
    NSView* contentView = [_window contentView];
    NSTextField* label = [contentView viewWithTag:1001];
    if (label == nil) return;

    NSFont* font = resolveFont();
    NSMutableParagraphStyle* paragraph = [[NSMutableParagraphStyle alloc] init];
    [paragraph setAlignment:alignmentConstant()];

    if (gAdaptiveColor) {
        NSDictionary* attributes = @{
            NSFontAttributeName: font,
            NSParagraphStyleAttributeName: paragraph
        };
        label.attributedStringValue =
            [[NSAttributedString alloc] initWithString:gCurrentText attributes:attributes];
        [label sizeToFit];
        [self renderAdaptiveColor:nil];
    } else {
        NSColor* textColor = [NSColor colorWithWhite:0.0 alpha:gTextOpacity];
        NSDictionary* attributes = @{
            NSFontAttributeName: font,
            NSForegroundColorAttributeName: textColor,
            NSParagraphStyleAttributeName: paragraph
        };
        label.attributedStringValue =
            [[NSAttributedString alloc] initWithString:gCurrentText attributes:attributes];
        [label sizeToFit];
    }

    // Reposition at the current grid spot.
    NSRect frame = gridSpotFrame(_window, label, gGridCol, gGridRow);
    if (NSEqualRects(frame, NSZeroRect)) return;
    [_window setFrame:frame display:YES];
    [label setFrame:NSMakeRect(0, 0, frame.size.width, frame.size.height)];
}

// orderOutForCapture hides the overlay so it does not appear in display captures.
// Saves the current alpha to restore later. No animation.
- (void)orderOutForCapture {
    if ([_window isVisible]) {
        [_window orderOut:nil];
    }
}

// orderInAfterCapture restores the overlay after a capture, respecting the
// current gIntendedVisible and gOverlayInterpolation state.
- (void)orderInAfterCapture {
    if (!gIntendedVisible) return;
    if (gMoveInProgress) return;
    [_window setAlphaValue:gOverlayInterpolation];
    [_window makeKeyAndOrderFront:nil];
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
    NSColor* textColor = [NSColor colorWithWhite:0.0 alpha:gTextOpacity];
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

// showOverlay updates gIntendedVisible, then dispatches showWindow unless a grid-move
// animation is in progress (fade-in will occur at the end of the move instead).
void showOverlay(void) {
    gIntendedVisible = YES;
    if (gMoveInProgress) return;
    [gDelegate performSelectorOnMainThread:@selector(showWindow)
                                withObject:nil
                             waitUntilDone:NO];
}

// hideOverlay updates gIntendedVisible, then dispatches hideWindow unless a grid-move
// animation is in progress (the window will stay at alpha=0 and be ordered out when
// the move completes — avoiding a redundant animation conflict).
void hideOverlay(void) {
    gIntendedVisible = NO;
    if (gMoveInProgress) return;
    [gDelegate performSelectorOnMainThread:@selector(hideWindow)
                                withObject:nil
                             waitUntilDone:NO];
}

// hideOverlayForCapture orders the overlay window out synchronously so it does
// not appear in CGDisplayCreateImageForRect captures. Does not alter
// gIntendedVisible or gOverlayInterpolation — purely a capture-time hide.
// Blocks until the main thread completes the order-out.
void hideOverlayForCapture(void) {
    [gDelegate performSelectorOnMainThread:@selector(orderOutForCapture)
                                withObject:nil
                             waitUntilDone:YES];
}

// restoreOverlayAfterCapture brings the overlay back if it was visible before
// the capture. Synchronous to ensure ordering with the capture.
void restoreOverlayAfterCapture(void) {
    [gDelegate performSelectorOnMainThread:@selector(orderInAfterCapture)
                                withObject:nil
                             waitUntilDone:YES];
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

// setTextOpacity adjusts the text opacity by the given delta.
// The value is clamped to [0.0, 1.0]. Re-renders the text with the new
// opacity baked into the text color / adaptive-color pattern alpha.
void setTextOpacity(double delta) {
    gTextOpacity += (CGFloat)delta;
    if (gTextOpacity < 0.0) gTextOpacity = 0.0;
    if (gTextOpacity > 1.0) gTextOpacity = 1.0;
    [gDelegate performSelectorOnMainThread:@selector(reposition)
                                withObject:nil
                             waitUntilDone:NO];
}

// resetTextOpacity restores the text opacity to the configured default
// and re-renders the text.
void resetTextOpacity(void) {
    gTextOpacity = gDefaultTextOpacity;
    [gDelegate performSelectorOnMainThread:@selector(reposition)
                                withObject:nil
                             waitUntilDone:NO];
}

// Canvas bounds for browser content-area grid positioning.
// Updated by setOverlayCanvasBounds; used by commitMoveToGridSpot.
static CGFloat gCanvasX      = 0.0;
static CGFloat gCanvasY      = 0.0;
static CGFloat gCanvasWidth  = 0.0;
static CGFloat gCanvasHeight = 0.0;

// setOverlayCanvasBounds stores the browser canvas rect for grid positioning.
// Pass (0,0,0,0) to clear when a non-browser window is focused.
void setOverlayCanvasBounds(double x, double y, double width, double height) {
    gCanvasX      = (CGFloat)x;
    gCanvasY      = (CGFloat)y;
    gCanvasWidth  = (CGFloat)width;
    gCanvasHeight = (CGFloat)height;
}

// setOverlayWindowBounds stores the raw window rect (Y-down, logical pixels) for
// non-browser fallback. Pass (0,0,0,0) to clear.
void setOverlayWindowBounds(double x, double y, double width, double height) {
    gWindowX      = (CGFloat)x;
    gWindowY      = (CGFloat)y;
    gWindowWidth  = (CGFloat)width;
    gWindowHeight = (CGFloat)height;
}

// gridSpotFrame computes the NSRect (Cocoa Y-up) for the overlay at (col, row).
// col and row are percentages in [0.0, 1.0].
//
// Coordinate systems:
//   gCanvasX/Y and gWindowX/Y are Y-down screen coordinates (from AppleScript).
//   NSRect uses Cocoa Y-up (origin at bottom-left of the main screen).
//   Conversion: cocoaY = screenHeight - yDown - frameHeight.
//
// window and label are passed by the caller (an Obj-C category method) to avoid
// accessing the @protected _window ivar from static C code.
//
// Returns NSZeroRect when no text is set or the window has not been sized yet,
// so callers must guard against it.
static NSRect gridSpotFrame(NSWindow* window, NSTextField* label, double col, double row) {
    if (gCurrentText == nil || [gCurrentText length] == 0) return NSZeroRect;
    if (label == nil) return NSZeroRect;

    // Prefer canvas bounds; fall back to raw window bounds.
    CGFloat areaX = (gCanvasWidth > 0) ? gCanvasX : gWindowX;
    CGFloat areaY = (gCanvasWidth > 0) ? gCanvasY : gWindowY;
    CGFloat areaW = (gCanvasWidth > 0) ? gCanvasWidth  : gWindowWidth;
    CGFloat areaH = (gCanvasWidth > 0) ? gCanvasHeight : gWindowHeight;
    if (areaW <= 0 || areaH <= 0) return NSZeroRect;

    NSScreen* screen = [NSScreen mainScreen];
    CGFloat screenH = [screen frame].size.height;

    // Size the label text to get the frame dimensions.
    [label sizeToFit];
    NSSize textSize = label.frame.size;
    CGFloat frameW = textSize.width;
    CGFloat frameH = textSize.height;

    // Spot position in Y-down space (col/row are 0.0–1.0 percentages).
    CGFloat spotX = areaX + col * areaW;
    CGFloat spotY = areaY + row * areaH;

    // Horizontal origin:
    //   col ≤ 0 → left-aligned: left edge + margin.
    //   col ≥ 1 → right-aligned: right edge − frameW − margin.
    //   otherwise → center the frame on spotX.
    CGFloat originX;
    if (col <= 0.0) {
        originX = areaX + gMargin;
    } else if (col >= 1.0) {
        originX = areaX + areaW - frameW - gMargin;
    } else {
        originX = spotX - frameW / 2.0;
    }

    // Vertical origin (Y-down):
    //   row ≤ 0 → top edge + margin.
    //   row ≥ 1 → bottom edge − frameH − margin.
    //   otherwise → center the frame on spotY.
    CGFloat originYDown;
    if (row <= 0.0) {
        originYDown = areaY + gMargin;
    } else if (row >= 1.0) {
        originYDown = areaY + areaH - frameH - gMargin;
    } else {
        originYDown = spotY - frameH / 2.0;
    }

    // Convert Y-down → Cocoa Y-up.
    CGFloat originY = screenH - originYDown - frameH;
    return NSMakeRect(originX, originY, frameW, frameH);
}

// OverlayDelegate category for grid-move animation steps.
// These run on the main thread and have full access to the delegate's ivars.
@interface OverlayDelegate (GridMove)
- (void)doFadeOutForMove;
- (void)doCommitMoveFromArray:(NSArray*)args;
- (void)doFadeInAfterMove;
@end

@implementation OverlayDelegate (GridMove)

// doFadeOutForMove fades the window to alpha=0 without ordering it out.
// Sets gMoveInProgress=YES so subsequent show/hide calls defer to the move.
- (void)doFadeOutForMove {
    if (gMoveInProgress) return;
    gMoveInProgress = YES;
    gOverlayInterpolation = 0.0;
    if ([_window alphaValue] <= 0.0 || ![_window isVisible]) return;
    _generation++;
    [NSAnimationContext runAnimationGroup:^(NSAnimationContext* context) {
        [context setDuration:gFadeDuration];
        [[_window animator] setAlphaValue:0.0];
    }];
}

// doCommitMoveFromArray: unpacks the [col, row] NSArray and repositions the window
// frame at the computed grid spot. Does NOT call renderAdaptiveColor or
// captureInvertedStrip — safe from the purple screen-capture indicator.
// Only called while the window is at alpha=0 (invisible).
- (void)doCommitMoveFromArray:(NSArray*)args {
    double col = [args[0] doubleValue];
    double row = [args[1] doubleValue];
    gGridCol = col;
    gGridRow = row;
    NSView* contentView = [_window contentView];
    NSTextField* label = [contentView viewWithTag:1001];
    NSRect frame = gridSpotFrame(_window, label, col, row);
    if (NSEqualRects(frame, NSZeroRect)) return;
    [_window setFrame:frame display:YES];
    if (label != nil) {
        [label setFrame:NSMakeRect(0, 0, frame.size.width, frame.size.height)];
    }
}

// doFadeInAfterMove clears gMoveInProgress. If gIntendedVisible, fades window
// alpha to 1.0 (interpolation). Otherwise orders the window out.
- (void)doFadeInAfterMove {
    gMoveInProgress = NO;
    if (!gIntendedVisible) {
        [_window orderOut:nil];
        return;
    }
    gOverlayInterpolation = 1.0;
    if (![_window isVisible]) {
        [_window setAlphaValue:0.0];
        [_window makeKeyAndOrderFront:nil];
    }
    _generation++;
    [NSAnimationContext runAnimationGroup:^(NSAnimationContext* context) {
        [context setDuration:gFadeDuration];
        [[_window animator] setAlphaValue:1.0];
    }];
}

@end

// fadeOutForMove dispatches the move-start fade-out to the main thread.
void fadeOutForMove(void) {
    [gDelegate performSelectorOnMainThread:@selector(doFadeOutForMove)
                                withObject:nil
                             waitUntilDone:NO];
}

// commitMoveToGridSpot repositions the overlay at the given (col, row) grid spot.
// col and row are percentages in [0.0, 1.0].
void commitMoveToGridSpot(double col, double row) {
    NSArray* args = @[@(col), @(row)];
    [gDelegate performSelectorOnMainThread:@selector(doCommitMoveFromArray:)
                                withObject:args
                             waitUntilDone:NO];
}

// fadeInAfterMove clears the move state and fades in (or orders out) on the main thread.
void fadeInAfterMove(void) {
    [gDelegate performSelectorOnMainThread:@selector(doFadeInAfterMove)
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
	Alignment     string  // "left", "center", "right", "dynamic"
	AdaptiveColor bool    // per-pixel background-inverted text color
	FadeDuration  float64 // fade animation duration in seconds
}

// AlignmentFromName maps an alignment name to an integer: 0=left, 1=center, 2=right, 3=dynamic.
func AlignmentFromName(alignment string) int {
	switch alignment {
	case "left":
		return 0
	case "center":
		return 1
	case "right":
		return 2
	case "dynamic":
		return 3
	default:
		return 3
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
		C.int(AlignmentFromName(config.Alignment)),
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

// HideOverlayForCapture orders the overlay out synchronously so it does not
// appear in CGDisplayCreateImageForRect captures. Does not alter visibility
// state — purely a capture-time hide. Blocks until complete.
func HideOverlayForCapture() {
	C.hideOverlayForCapture()
}

// RestoreOverlayAfterCapture brings the overlay back if it was visible before
// the capture. Blocks until complete.
func RestoreOverlayAfterCapture() {
	C.restoreOverlayAfterCapture()
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

// SetTextAlignment changes the text alignment at runtime and repositions.
// alignment: "left", "center", "right", or "dynamic".
// Safe to call from any goroutine; dispatched to the main thread internally.
func SetTextAlignment(alignment string) {
	C.setOverlayPosition(C.int(AlignmentFromName(alignment)))
}

// SetTextOpacity adjusts the text opacity by the given delta (e.g. +0.01 or -0.01).
// The value is clamped to [0.0, 1.0] and controls the text/pattern alpha.
// Overlay visibility (window alpha) is driven independently by the overlay interpolation.
// Safe to call from any goroutine; dispatched to the main thread internally.
func SetTextOpacity(delta float64) {
	C.setTextOpacity(C.double(delta))
}

// ResetTextOpacity restores the text opacity to the configured default
// and re-renders the text.
// Safe to call from any goroutine; dispatched to the main thread internally.
func ResetTextOpacity() {
	C.resetTextOpacity()
}

// SetOverlayCanvasBounds stores the browser content-area rectangle for grid positioning.
// x, y are top-left screen coordinates in logical pixels; w, h are the canvas dimensions.
// Pass (0, 0, 0, 0) to clear when a non-browser window is focused.
// Safe to call from any goroutine; C scalar writes only, no Obj-C dispatch required.
func SetOverlayCanvasBounds(x, y, w, h float64) {
	C.setOverlayCanvasBounds(C.double(x), C.double(y), C.double(w), C.double(h))
}
