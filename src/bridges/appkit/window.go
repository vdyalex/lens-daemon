//go:build darwin

package appkit

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>
#include <string.h>

// Configurable appearance globals (set before createOverlayWindow).
static CGFloat gFontSize   = 16.0;
static CGFloat gOpacity    = 0.075;
static CGFloat gFontWeight = -0.8; // NSFontWeightUltraLight
static int     gAlignment  = 1;    // 0=left, 1=center, 2=right
static CGFloat gMargin     = 20.0;
static char    gFontFamily[256] = "";

// Current text for repositioning without a new Display call.
static NSString* gCurrentText = nil;

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
                      double fontSize, double opacity, int alignment) {
    gFontSize   = (CGFloat)fontSize;
    gOpacity    = (CGFloat)opacity;
    gFontWeight = fontWeightFromName(fontWeight);
    gAlignment  = alignment;
    if (fontFamily != NULL && strlen(fontFamily) > 0) {
        strlcpy(gFontFamily, fontFamily, sizeof(gFontFamily));
    }
}

// OverlayDelegate receives performSelector calls on the main thread
// and forwards them to the overlay window.
@interface OverlayDelegate : NSObject {
    NSWindow* _window;
}
- (instancetype)initWithWindow:(NSWindow*)window;
- (void)showWindow;
- (void)hideWindow;
- (void)updateText:(NSString*)text;
- (void)reposition;
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
    [_window makeKeyAndOrderFront:nil];
}

- (void)hideWindow {
    [_window orderOut:nil];
}

- (void)updateText:(NSString*)text {
    gCurrentText = [text copy];

    NSView* contentView = [_window contentView];
    NSTextField* label = [contentView viewWithTag:1001];
    if (label == nil) return;

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
        [[NSAttributedString alloc] initWithString:text attributes:attributes];

    // Resize label and window to fit text on a single line.
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
        [self updateText:gCurrentText];
    }
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
*/
import "C"

import "unsafe"

// OverlayConfig holds the configurable appearance settings for the overlay window.
type OverlayConfig struct {
	FontFamily string
	FontWeight string
	FontSize   float64
	Opacity    float64
	Position   string // "left", "center", "right"
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
	C.configureOverlay(cFamily, cWeight, C.double(config.FontSize), C.double(config.Opacity), C.int(AlignmentFromPosition(config.Position)))
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
