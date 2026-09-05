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
import "os/exec"

// CheckScreenRecordingPermission verifies and requests macOS Screen Recording (TCC) permission.
func CheckScreenRecordingPermission() bool {
	return C.CheckAndRequestScreenCapturePermission() == 1
}

// PromptScreenRecordingPermission prompts macOS for permission and opens System Settings if missing.
func PromptScreenRecordingPermission() bool {
	granted := CheckScreenRecordingPermission()
	if !granted {
		_ = exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture").Start()
	}
	return granted
}
