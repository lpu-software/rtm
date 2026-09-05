package probes

import (
	"fmt"
	"image"
	"image/draw"
	"time"

	"github.com/kbinani/screenshot"
)

type GenericScreenshotProbe struct{}

func (p *GenericScreenshotProbe) Name() string {
	return "Generic Screen API (kbinani/screenshot)"
}

func (p *GenericScreenshotProbe) Method() CaptureMethod {
	return MethodGenericScreenshot
}

func (p *GenericScreenshotProbe) Platform() string {
	return "cross-platform"
}

func (p *GenericScreenshotProbe) IsAvailable() bool {
	return screenshot.NumActiveDisplays() > 0
}

func (p *GenericScreenshotProbe) CheckPermission() (bool, string) {
	return true, "Desktop screen capture access"
}

func (p *GenericScreenshotProbe) Capture(req CaptureRequest) (*CaptureResponse, error) {
	start := time.Now()
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return nil, fmt.Errorf("no active displays found")
	}

	bounds := screenshot.GetDisplayBounds(0)
	if req.Scope == ScopeRegion && req.TargetBounds.Dx() > 0 && req.TargetBounds.Dy() > 0 {
		bounds = req.TargetBounds
	}

	img, err := screenshot.CaptureRect(bounds)
	duration := time.Since(start)

	resp := &CaptureResponse{
		Method:            MethodGenericScreenshot,
		Scope:             req.Scope,
		TargetWindowID:    req.TargetWindowID,
		CaptureDuration:   duration,
		Timestamp:         start,
		PermissionGranted: true,
		IsDeterministic:   true,
	}

	if err != nil {
		resp.ErrorCode = -1
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)

	resp.CapturedImage = rgba
	resp.Width = rgba.Bounds().Dx()
	resp.Height = rgba.Bounds().Dy()
	return resp, nil
}

func (r *ProbeRegistry) registerFallbackProbes() {
	r.probes[MethodGenericScreenshot] = &GenericScreenshotProbe{}
}
