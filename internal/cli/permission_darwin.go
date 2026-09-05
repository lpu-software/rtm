//go:build darwin

package cli

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework CoreGraphics -framework Cocoa
#import <CoreGraphics/CoreGraphics.h>
#import <Cocoa/Cocoa.h>

static int CheckAndRequestScreenCapturePermission() {
    if (@available(macOS 10.15, *)) {
        if (!CGPreflightScreenCaptureAccess()) {
            CGRequestScreenCaptureAccess();
            return 0;
        }
        return 1;
    }
    return 1;
}
*/
import "C"

// CheckScreenRecordingPermission verifies and requests macOS Screen Recording (TCC) permission.
func CheckScreenRecordingPermission() bool {
	return C.CheckAndRequestScreenCapturePermission() == 1
}
