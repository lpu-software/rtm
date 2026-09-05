package probes

import (
	"image"
	"time"
)

// CaptureScope defines the spatial extent of the capture operation.
type CaptureScope string

const (
	ScopeFullScreen CaptureScope = "FULL_SCREEN"
	ScopeWindow     CaptureScope = "WINDOW"
	ScopeRegion     CaptureScope = "REGION"
)

// CaptureMethod defines the API used to capture the screen/window.
type CaptureMethod string

const (
	// Windows Capture APIs
	MethodWindowsGraphicsCapture CaptureMethod = "Windows Graphics Capture (WGC)"
	MethodGDIScreenDC            CaptureMethod = "GDI Screen DC (BitBlt)"
	MethodGDIWindowDC            CaptureMethod = "GDI Window DC (GetWindowDC)"
	MethodPrintWindowFull        CaptureMethod = "PrintWindow (PW_RENDERFULLCONTENT)"
	MethodPrintWindowDefault     CaptureMethod = "PrintWindow (Default GDI)"
	MethodDXGIDuplication        CaptureMethod = "DXGI Desktop Duplication"

	// macOS Capture APIs
	MethodScreenCaptureKit    CaptureMethod = "ScreenCaptureKit (SCShareableContent)"
	MethodCoreGraphicsWindow  CaptureMethod = "CoreGraphics Window (CGWindowListCreateImage)"
	MethodCoreGraphicsDisplay CaptureMethod = "CoreGraphics Display (CGDisplayCreateImage)"
	MethodCoreGraphicsRegion  CaptureMethod = "CoreGraphics Region Crop"

	// Fallback Cross-Platform API
	MethodGenericScreenshot CaptureMethod = "Generic Screen API (kbinani/screenshot)"
)

// CaptureRequest specifies the parameters for a capture attempt.
type CaptureRequest struct {
	Scope           CaptureScope
	Method          CaptureMethod
	TargetWindowID  string
	TargetBounds    image.Rectangle
	CaptureTimeout  time.Duration
	AllowPermission bool
}

// CaptureResponse contains the captured output bitmap and telemetry.
type CaptureResponse struct {
	Method             CaptureMethod
	Scope              CaptureScope
	TargetWindowID     string
	CapturedImage      *image.RGBA
	Width              int
	Height             int
	CaptureDuration    time.Duration
	Timestamp          time.Time
	ErrorCode          int
	ErrorMessage       string
	PermissionGranted  bool
	OSReportedExcluded bool
	ObservedAffinity   string
	IsDeterministic    bool
}

// CaptureProbe defines the contract for OS-specific capture API implementations.
type CaptureProbe interface {
	Name() string
	Method() CaptureMethod
	Platform() string
	IsAvailable() bool
	CheckPermission() (bool, string)
	Capture(req CaptureRequest) (*CaptureResponse, error)
}
