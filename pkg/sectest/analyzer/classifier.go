package analyzer

import (
	"fmt"
	"time"

	"github.com/yatishydv/rtm/pkg/sectest/probes"
	"github.com/yatishydv/rtm/pkg/sectest/testapp"
)

// ClassificationResult represents the categorical security evaluation of a capture operation.
type ClassificationResult string

const (
	ClassificationIdentical             ClassificationResult = "IDENTICAL"
	ClassificationCompletelyBlack       ClassificationResult = "COMPLETELY_BLACK"
	ClassificationTransparentOmitted     ClassificationResult = "TRANSPARENT_OMITTED"
	ClassificationVisuallyMasked        ClassificationResult = "VISUALLY_MASKED"
	ClassificationBlurred               ClassificationResult = "BLURRED"
	ClassificationPartiallyObscured     ClassificationResult = "PARTIALLY_OBSCURED"
	ClassificationUnavailableBlocked    ClassificationResult = "UNAVAILABLE_PERMISSION_BLOCKED"
	ClassificationFrozen                ClassificationResult = "FROZEN"
	ClassificationOtherwiseModified     ClassificationResult = "OTHERWISE_MODIFIED"
)

// ProtectionEnforcement classifies the layer responsible for the observed protection.
type ProtectionEnforcement string

const (
	EnforcementOSEnforced            ProtectionEnforcement = "OS_ENFORCED_PROTECTION"
	EnforcementAppVisualMasking      ProtectionEnforcement = "APP_VISUAL_MASKING"
	EnforcementPermissionRestricted  ProtectionEnforcement = "PERMISSION_RESTRICTED"
	EnforcementNoneCapturable        ProtectionEnforcement = "NONE_CAPTURABLE"
	EnforcementCombinedProtection    ProtectionEnforcement = "COMBINED_OS_AND_APP_PROTECTION"
)

// ProtectionStatus represents the high-level matrix status.
type ProtectionStatus string

const (
	StatusProtected          ProtectionStatus = "Protected"
	StatusCapturable         ProtectionStatus = "Capturable"
	StatusPartiallyProtected ProtectionStatus = "Partially Protected"
	StatusPermissionBlocked  ProtectionStatus = "Permission Blocked"
	StatusUnavailable        ProtectionStatus = "Unavailable"
)

// SecurityEvaluation combines all test telemetry, metrics, and classification.
type SecurityEvaluation struct {
	Platform              string                `json:"platform"`
	OSVersion             string                `json:"os_version"`
	CaptureMethod         probes.CaptureMethod  `json:"capture_method"`
	CaptureScope          probes.CaptureScope   `json:"capture_scope"`
	TestMode              testapp.ProtectionMode `json:"test_mode"`
	TestModeCode          string                `json:"test_mode_code"`
	TargetWindowID        string                `json:"target_window_id"`
	ObservedOSProtection  string                `json:"observed_os_protection"`
	Metrics               *ImageMetrics         `json:"metrics"`
	Classification        ClassificationResult  `json:"classification"`
	Enforcement           ProtectionEnforcement `json:"enforcement"`
	Status                ProtectionStatus      `json:"status"`
	Summary               string                `json:"summary"`
	Timestamp             time.Time             `json:"timestamp"`
	DurationMs            int64                 `json:"duration_ms"`
	ErrorCode             int                   `json:"error_code"`
	ErrorMessage          string                `json:"error_message"`
	IsDeterministic       bool                  `json:"is_deterministic"`
}

// ClassifyCapture evaluates a capture response against reference metrics and determines the security verdict.
func ClassifyCapture(
	platform string,
	osVersion string,
	mode testapp.ProtectionMode,
	capResp *probes.CaptureResponse,
	metrics *ImageMetrics,
) *SecurityEvaluation {
	now := time.Now().UTC()
	eval := &SecurityEvaluation{
		Platform:             platform,
		OSVersion:            osVersion,
		CaptureMethod:        capResp.Method,
		CaptureScope:         capResp.Scope,
		TestMode:             mode,
		TestModeCode:         mode.Code(),
		TargetWindowID:       capResp.TargetWindowID,
		ObservedOSProtection: capResp.ObservedAffinity,
		Metrics:              metrics,
		Timestamp:            now,
		DurationMs:           capResp.CaptureDuration.Milliseconds(),
		ErrorCode:            capResp.ErrorCode,
		ErrorMessage:         capResp.ErrorMessage,
		IsDeterministic:      capResp.IsDeterministic,
	}

	// 1. Check for Permission Denial / API Error
	if !capResp.PermissionGranted || capResp.ErrorCode != 0 || capResp.CapturedImage == nil {
		eval.Classification = ClassificationUnavailableBlocked
		eval.Enforcement = EnforcementPermissionRestricted
		eval.Status = StatusPermissionBlocked
		eval.Summary = fmt.Sprintf("Capture was blocked or unavailable by operating system permissions (%s)", capResp.ErrorMessage)
		return eval
	}

	// 2. Check for OS-Level Exclusion (Transparent / Omitted)
	if capResp.OSReportedExcluded || metrics.AlphaTransparentRatio >= 0.90 || (metrics.SSIM < 0.10 && metrics.BlackPixelRatio < 0.50 && metrics.MSE > 10000) {
		if mode == testapp.ModeOSExclusion || mode == testapp.ModeCombined {
			eval.Classification = ClassificationTransparentOmitted
			eval.Enforcement = EnforcementOSEnforced
			eval.Status = StatusProtected
			eval.Summary = "Window content was omitted from capture by OS-level display affinity / window sharing exclusion"
			return eval
		}
	}

	// 3. Check for Completely Black Window (Standard WDA_MONITOR or GDI WDA_EXCLUDEFROMCAPTURE behavior)
	if metrics.BlackPixelRatio >= 0.95 {
		eval.Classification = ClassificationCompletelyBlack
		eval.Enforcement = EnforcementOSEnforced
		eval.Status = StatusProtected
		eval.Summary = "Window rendered as a solid black rectangle, completely concealing protected content"
		return eval
	}

	// 4. Check for Visual Privacy Overlay (Application-level visual masking)
	if mode == testapp.ModePrivacyOverlay || mode == testapp.ModeCombined {
		if metrics.SensitiveRegionMaskDetected || (metrics.SensitiveRegionSimilarity < 0.45 && metrics.SSIM < 0.70) {
			if mode == testapp.ModeCombined {
				eval.Enforcement = EnforcementCombinedProtection
			} else {
				eval.Enforcement = EnforcementAppVisualMasking
			}
			eval.Classification = ClassificationVisuallyMasked
			eval.Status = StatusProtected
			eval.Summary = "Sensitive test content was obfuscated by the application's active visual privacy shield/overlay"
			return eval
		}
	}

	// 5. Check for Blurred Content
	if metrics.BlurLaplacianVariance < 0.20*metrics.ReferenceSharpness && metrics.SSIM > 0.35 && metrics.SSIM < 0.85 {
		eval.Classification = ClassificationBlurred
		eval.Enforcement = EnforcementAppVisualMasking
		eval.Status = StatusPartiallyProtected
		eval.Summary = "Window content was captured in a blurred or low-pass filtered state"
		return eval
	}

	// 6. Check for Partial Obscuration
	if metrics.SensitiveRegionSimilarity < 0.50 && metrics.SSIM >= 0.70 {
		eval.Classification = ClassificationPartiallyObscured
		eval.Enforcement = EnforcementAppVisualMasking
		eval.Status = StatusPartiallyProtected
		eval.Summary = "Sensitive region was concealed while surrounding application UI remained visible"
		return eval
	}

	// 7. Check for Live Tick Frame Freezing
	if metrics.IsFrozen {
		eval.Classification = ClassificationFrozen
		eval.Enforcement = EnforcementOSEnforced
		eval.Status = StatusPartiallyProtected
		eval.Summary = "Capture returned a stale or frozen frame while live timer was progressing"
		return eval
	}

	// 8. Normal Identical / Capturable Window
	if metrics.SSIM >= 0.90 || metrics.MSE < 100 {
		eval.Classification = ClassificationIdentical
		eval.Enforcement = EnforcementNoneCapturable
		eval.Status = StatusCapturable
		eval.Summary = "Window was captured normally with full visual fidelity (No capture exclusion observed)"
		return eval
	}

	// 9. Fallback Otherwise Modified
	eval.Classification = ClassificationOtherwiseModified
	eval.Enforcement = EnforcementNoneCapturable
	eval.Status = StatusCapturable
	eval.Summary = "Captured output showed structural variance from reference pattern"
	return eval
}
