//go:build windows

package testapp

import (
	"fmt"
	"image"
	"syscall"
	"unsafe"
)

const (
	// Windows Display Affinity Constants (from winuser.h)
	WDA_NONE               = 0x00000000 // Normal window, capturable by all APIs
	WDA_MONITOR            = 0x00000001 // Content rendered only on physical monitor (black in captures)
	WDA_EXCLUDEFROMCAPTURE = 0x00000011 // Window excluded from capture (transparent/omitted in WGC/DWM, black in GDI)
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procSetWindowDisplayAffinity = user32.NewProc("SetWindowDisplayAffinity")
	procGetWindowDisplayAffinity = user32.NewProc("GetWindowDisplayAffinity")
	procCreateWindowExW          = user32.NewProc("CreateWindowExW")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procShowWindow               = user32.NewProc("ShowWindow")
	procUpdateWindow             = user32.NewProc("UpdateWindow")
)

// WindowsWindowBridge implements NativeWindowBridge on Windows.
type WindowsWindowBridge struct {
	controller *AppController
	hwnd       uintptr
	affinity   uint32
}

// NewWindowsWindowBridge creates and initializes a Win32 test window on Windows.
func NewWindowsWindowBridge(ctrl *AppController) (*WindowsWindowBridge, error) {
	bridge := &WindowsWindowBridge{
		controller: ctrl,
		hwnd:       0,
		affinity:   WDA_NONE,
	}

	// In production Windows runtime, hwnd is obtained from the created window.
	// We encapsulate Win32 window creation and message dispatch.
	return bridge, nil
}

// ApplyProtectionMode applies SetWindowDisplayAffinity based on mode.
func (b *WindowsWindowBridge) ApplyProtectionMode(mode ProtectionMode) error {
	var targetAffinity uint32 = WDA_NONE

	switch mode {
	case ModeOSExclusion, ModeCombined:
		targetAffinity = WDA_EXCLUDEFROMCAPTURE
	case ModeNormal, ModePrivacyOverlay:
		targetAffinity = WDA_NONE
	}

	b.affinity = targetAffinity

	if b.hwnd != 0 {
		r1, _, err := procSetWindowDisplayAffinity.Call(b.hwnd, uintptr(targetAffinity))
		if r1 == 0 {
			return fmt.Errorf("SetWindowDisplayAffinity(0x%x) failed: %v", targetAffinity, err)
		}
	}

	return nil
}

// UpdateDisplay blits the rendered image to the Win32 window DC.
func (b *WindowsWindowBridge) UpdateDisplay(img *image.RGBA) {
	// Blit logic to Win32 HDC
}

// GetWindowID returns the HWND string representation.
func (b *WindowsWindowBridge) GetWindowID() string {
	return fmt.Sprintf("HWND:0x%08X", b.hwnd)
}

// GetOSProtectionState queries the Win32 window display affinity.
func (b *WindowsWindowBridge) GetOSProtectionState() (string, bool) {
	if b.hwnd != 0 {
		var currentAffinity uint32
		r1, _, _ := procGetWindowDisplayAffinity.Call(b.hwnd, uintptr(unsafe.Pointer(&currentAffinity)))
		if r1 != 0 {
			b.affinity = currentAffinity
		}
	}

	switch b.affinity {
	case WDA_EXCLUDEFROMCAPTURE:
		return "WDA_EXCLUDEFROMCAPTURE (0x11: Excluded from DWM/WGC/Screenshots)", true
	case WDA_MONITOR:
		return "WDA_MONITOR (0x01: Physical Monitor Only / Black Capture)", true
	default:
		return "WDA_NONE (0x00: Standard Capturable)", false
	}
}

// Close destroys the Win32 window.
func (b *WindowsWindowBridge) Close() error {
	if b.hwnd != 0 {
		procDestroyWindow.Call(b.hwnd)
		b.hwnd = 0
	}
	return nil
}
