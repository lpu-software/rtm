//go:build darwin

package testapp

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics

#import <Cocoa/Cocoa.h>
#import <CoreGraphics/CoreGraphics.h>

static NSWindow* gTestWindow = nil;
static NSImageView* gImageView = nil;
static NSUInteger gCurrentSharingType = NSWindowSharingReadOnly;

// Initialize native Cocoa window
static void CreateCocoaTestWindow(int width, int height, const char* title) {
    dispatch_sync(dispatch_get_main_queue(), ^{
        if (gTestWindow != nil) return;

        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];

        NSRect frame = NSMakeRect(200, 200, width, height);
        NSUInteger styleMask = NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskMiniaturizable | NSWindowStyleMaskResizable;
        gTestWindow = [[NSWindow alloc] initWithContentRect:frame
                                                  styleMask:styleMask
                                                    backing:NSBackingStoreBuffered
                                                      defer:NO];
        [gTestWindow setTitle:[NSString stringWithUTF8String:title]];
        [gTestWindow makeKeyAndOrderFront:nil];
        [gTestWindow setSharingType:NSWindowSharingReadOnly];

        NSView* contentView = [gTestWindow contentView];
        gImageView = [[NSImageView alloc] initWithFrame:[contentView bounds]];
        [gImageView setImageScaling:NSImageScaleAxesIndependently];
        [gImageView setAutoresizingMask:NSViewWidthSizable | NSViewHeightSizable];
        [contentView addSubview:gImageView];

        [NSApp activateIgnoringOtherApps:YES];
    });
}

// Update Window sharing protection state
static void SetCocoaWindowProtection(int enableOSExclusion) {
    dispatch_sync(dispatch_get_main_queue(), ^{
        if (gTestWindow == nil) return;
        if (enableOSExclusion) {
            // NSWindowSharingNone = 0: Excluded from screen capture
            [gTestWindow setSharingType:NSWindowSharingNone];
            gCurrentSharingType = NSWindowSharingNone;
        } else {
            // NSWindowSharingReadOnly = 1: Standard capturable window
            [gTestWindow setSharingType:NSWindowSharingReadOnly];
            gCurrentSharingType = NSWindowSharingReadOnly;
        }
    });
}

// Update displayed image frame in native window
static void UpdateCocoaWindowBitmap(const void* rgbaBytes, int width, int height) {
    if (gTestWindow == nil || gImageView == nil) return;

    // Create NSBitmapImageRep from raw RGBA bytes
    NSBitmapImageRep* rep = [[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL
                                                                    pixelsWide:width
                                                                    pixelsHigh:height
                                                                 bitsPerSample:8
                                                               samplesPerPixel:4
                                                                      hasAlpha:YES
                                                                      isPlanar:NO
                                                                colorSpaceName:NSDeviceRGBColorSpace
                                                                   bytesPerRow:width * 4
                                                                  bitsPerPixel:32];
    unsigned char* bitmapData = [rep bitmapData];
    if (bitmapData) {
        memcpy(bitmapData, rgbaBytes, width * height * 4);
        NSImage* img = [[NSImage alloc] initWithSize:NSMakeSize(width, height)];
        [img addRepresentation:rep];

        dispatch_async(dispatch_get_main_queue(), ^{
            [gImageView setImage:img];
        });
    }
}

static uint32_t GetCocoaWindowNumber() {
    if (gTestWindow == nil) return 0;
    return (uint32_t)[gTestWindow windowNumber];
}

static int GetCocoaSharingType() {
    return (int)gCurrentSharingType;
}

static void CloseCocoaWindow() {
    dispatch_sync(dispatch_get_main_queue(), ^{
        if (gTestWindow != nil) {
            [gTestWindow close];
            gTestWindow = nil;
            gImageView = nil;
        }
    });
}
*/
import "C"
import (
	"fmt"
	"image"
	"unsafe"
)

// DarwinWindowBridge implements NativeWindowBridge on macOS.
type DarwinWindowBridge struct {
	controller *AppController
	windowID   uint32
}

// NewDarwinWindowBridge creates and initializes a native Cocoa test window on macOS.
func NewDarwinWindowBridge(ctrl *AppController) (*DarwinWindowBridge, error) {
	cfg := ctrl.GetConfig()
	cTitle := C.CString(fmt.Sprintf("LPU Security Test Target — %s", cfg.SessionID))
	defer C.free(unsafe.Pointer(cTitle))

	C.CreateCocoaTestWindow(C.int(cfg.Width), C.int(cfg.Height), cTitle)
	winNum := uint32(C.GetCocoaWindowNumber())

	bridge := &DarwinWindowBridge{
		controller: ctrl,
		windowID:   winNum,
	}

	return bridge, nil
}

// ApplyProtectionMode applies NSWindowSharingNone / NSWindowSharingReadOnly based on mode.
func (b *DarwinWindowBridge) ApplyProtectionMode(mode ProtectionMode) error {
	switch mode {
	case ModeOSExclusion, ModeCombined:
		C.SetCocoaWindowProtection(1) // NSWindowSharingNone
	case ModeNormal, ModePrivacyOverlay:
		C.SetCocoaWindowProtection(0) // NSWindowSharingReadOnly
	}
	return nil
}

// UpdateDisplay updates the Cocoa window's content with the rendered frame.
func (b *DarwinWindowBridge) UpdateDisplay(img *image.RGBA) {
	if img == nil {
		return
	}
	bnds := img.Bounds()
	C.UpdateCocoaWindowBitmap(
		unsafe.Pointer(&img.Pix[0]),
		C.int(bnds.Dx()),
		C.int(bnds.Dy()),
	)
}

// GetWindowID returns the CGWindowID string representation.
func (b *DarwinWindowBridge) GetWindowID() string {
	if b.windowID == 0 {
		b.windowID = uint32(C.GetCocoaWindowNumber())
	}
	return fmt.Sprintf("CGWindowID:%d", b.windowID)
}

// GetOSProtectionState queries the Cocoa window sharing type.
func (b *DarwinWindowBridge) GetOSProtectionState() (string, bool) {
	st := int(C.GetCocoaSharingType())
	if st == 0 {
		return "NSWindowSharingNone (Excluded from Capture)", true
	}
	return "NSWindowSharingReadOnly (Standard Capturable)", false
}

// Close destroys the Cocoa window.
func (b *DarwinWindowBridge) Close() error {
	C.CloseCocoaWindow()
	return nil
}
