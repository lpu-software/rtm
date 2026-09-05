package analyzer

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
	"time"

	"github.com/yatishydv/rtm/pkg/sectest/probes"
	"github.com/yatishydv/rtm/pkg/sectest/testapp"
)

func TestMetricsAndClassification(t *testing.T) {
	cfg := testapp.DefaultConfig("darwin", testapp.ModeNormal)
	refImg := testapp.GeneratePatternImage(cfg)

	// Test 1: Identical Image
	metricsIdentical := ComputeMetrics(refImg, refImg, cfg.SensitiveBounds, cfg.QRBounds, 1)
	if metricsIdentical.SSIM < 0.99 {
		t.Errorf("Expected SSIM > 0.99 for identical image, got %f", metricsIdentical.SSIM)
	}
	if metricsIdentical.MSE > 1.0 {
		t.Errorf("Expected MSE < 1.0 for identical image, got %f", metricsIdentical.MSE)
	}

	capRespIdentical := &probes.CaptureResponse{
		Method:            probes.MethodScreenCaptureKit,
		Scope:             probes.ScopeWindow,
		CapturedImage:     refImg,
		CaptureDuration:   10 * time.Millisecond,
		PermissionGranted: true,
	}
	evalIdentical := ClassifyCapture("darwin", "macOS 15.0", testapp.ModeNormal, capRespIdentical, metricsIdentical)
	if evalIdentical.Classification != ClassificationIdentical {
		t.Errorf("Expected classification IDENTICAL, got %s", evalIdentical.Classification)
	}
	if evalIdentical.Status != StatusCapturable {
		t.Errorf("Expected status Capturable, got %s", evalIdentical.Status)
	}

	// Test 2: Solid Black Image (OS Protection)
	blackImg := image.NewRGBA(refImg.Bounds())
	draw.Draw(blackImg, blackImg.Bounds(), &image.Uniform{C: color.RGBA{0, 0, 0, 255}}, image.Point{}, draw.Src)

	metricsBlack := ComputeMetrics(refImg, blackImg, cfg.SensitiveBounds, cfg.QRBounds, 1)
	if metricsBlack.BlackPixelRatio < 0.99 {
		t.Errorf("Expected BlackPixelRatio > 0.99, got %f", metricsBlack.BlackPixelRatio)
	}

	capRespBlack := &probes.CaptureResponse{
		Method:            probes.MethodGDIScreenDC,
		Scope:             probes.ScopeWindow,
		CapturedImage:     blackImg,
		CaptureDuration:   12 * time.Millisecond,
		PermissionGranted: true,
	}
	evalBlack := ClassifyCapture("windows", "Windows 11 23H2", testapp.ModeOSExclusion, capRespBlack, metricsBlack)
	if evalBlack.Classification != ClassificationCompletelyBlack {
		t.Errorf("Expected classification COMPLETELY_BLACK, got %s", evalBlack.Classification)
	}
	if evalBlack.Status != StatusProtected {
		t.Errorf("Expected status Protected, got %s", evalBlack.Status)
	}

	// Test 3: Application-level Privacy Overlay
	cfgMask := testapp.DefaultConfig("darwin", testapp.ModePrivacyOverlay)
	maskedImg := testapp.GeneratePatternImage(cfgMask)

	metricsMasked := ComputeMetrics(refImg, maskedImg, cfg.SensitiveBounds, cfg.QRBounds, 1)
	capRespMasked := &probes.CaptureResponse{
		Method:            probes.MethodCoreGraphicsWindow,
		Scope:             probes.ScopeWindow,
		CapturedImage:     maskedImg,
		CaptureDuration:   15 * time.Millisecond,
		PermissionGranted: true,
	}
	evalMasked := ClassifyCapture("darwin", "macOS 15.0", testapp.ModePrivacyOverlay, capRespMasked, metricsMasked)
	if evalMasked.Classification != ClassificationVisuallyMasked {
		t.Errorf("Expected classification VISUALLY_MASKED, got %s", evalMasked.Classification)
	}
	if evalMasked.Enforcement != EnforcementAppVisualMasking {
		t.Errorf("Expected enforcement APP_VISUAL_MASKING, got %s", evalMasked.Enforcement)
	}

	// Test 4: Permission Denied Capture
	capRespDenied := &probes.CaptureResponse{
		Method:            probes.MethodScreenCaptureKit,
		Scope:             probes.ScopeFullScreen,
		CapturedImage:     nil,
		PermissionGranted: false,
		ErrorCode:         -1,
		ErrorMessage:      "Screen Recording permission denied",
	}
	evalDenied := ClassifyCapture("darwin", "macOS 15.0", testapp.ModeNormal, capRespDenied, metricsIdentical)
	if evalDenied.Classification != ClassificationUnavailableBlocked {
		t.Errorf("Expected classification UNAVAILABLE_PERMISSION_BLOCKED, got %s", evalDenied.Classification)
	}
	if evalDenied.Status != StatusPermissionBlocked {
		t.Errorf("Expected status Permission Blocked, got %s", evalDenied.Status)
	}
}
