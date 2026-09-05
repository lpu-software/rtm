package testapp

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// ProtectionMode defines the synthetic test window's security state.
type ProtectionMode int

const (
	// ModeNormal - Normal window without any capture protection (Test A).
	ModeNormal ProtectionMode = iota
	// ModeOSExclusion - Window with OS-level capture exclusion (WDA_EXCLUDEFROMCAPTURE / NSWindowSharingNone) (Test B).
	ModeOSExclusion
	// ModePrivacyOverlay - Window with application-level visual masking/overlay (Test C).
	ModePrivacyOverlay
	// ModeCombined - Window with both OS exclusion and privacy overlay enabled (Test D).
	ModeCombined
)

func (m ProtectionMode) String() string {
	switch m {
	case ModeNormal:
		return "Normal (No Protection)"
	case ModeOSExclusion:
		return "OS Capture Exclusion"
	case ModePrivacyOverlay:
		return "App Privacy Overlay"
	case ModeCombined:
		return "Combined (OS Exclusion + App Overlay)"
	default:
		return "Unknown"
	}
}

func (m ProtectionMode) Code() string {
	switch m {
	case ModeNormal:
		return "TEST_A_NORMAL"
	case ModeOSExclusion:
		return "TEST_B_OS_EXCLUSION"
	case ModePrivacyOverlay:
		return "TEST_C_PRIVACY_OVERLAY"
	case ModeCombined:
		return "TEST_D_COMBINED"
	default:
		return "UNKNOWN"
	}
}

// WindowConfig contains configuration parameters for rendering synthetic test windows.
type WindowConfig struct {
	Width            int
	Height           int
	SessionID        string
	RandomNonce      string
	Platform         string
	Mode             ProtectionMode
	ShowSensitive    bool
	FrameNumber      int64
	Timestamp        time.Time
	SyntheticSecret  string
	SensitiveBounds  image.Rectangle
	QRBounds         image.Rectangle
	ColorGridBounds  image.Rectangle
}

// DefaultConfig returns standard dimensions and dummy secrets for testing.
func DefaultConfig(platform string, mode ProtectionMode) *WindowConfig {
	now := time.Now().UTC()
	return &WindowConfig{
		Width:           800,
		Height:          600,
		SessionID:       "SEC-VAL-84920",
		RandomNonce:     fmt.Sprintf("%08x", now.UnixNano()&0xFFFFFFFF),
		Platform:        platform,
		Mode:            mode,
		ShowSensitive:   true,
		FrameNumber:     1,
		Timestamp:       now,
		SyntheticSecret: "MOCK-AUTH-KEY-8942-B7F0-9931-ALPHA-BETA",
		SensitiveBounds: image.Rect(40, 160, 480, 270),
		QRBounds:        image.Rect(520, 160, 750, 390),
		ColorGridBounds: image.Rect(40, 300, 480, 520),
	}
}

// GeneratePatternImage renders the synthetic test pattern bitmap.
func GeneratePatternImage(cfg *WindowConfig) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))

	// 1. Background (Dark Theme: Slate Navy #0F172A)
	bgColor := color.RGBA{R: 15, G: 23, B: 42, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	// 2. Window Title Bar & Header
	headerColor := color.RGBA{R: 30, G: 41, B: 59, A: 255}
	draw.Draw(img, image.Rect(0, 0, cfg.Width, 50), &image.Uniform{C: headerColor}, image.Point{}, draw.Src)

	// Header accent stripe
	modeAccent := color.RGBA{R: 56, G: 189, B: 248, A: 255} // Blue
	if cfg.Mode == ModeOSExclusion || cfg.Mode == ModeCombined {
		modeAccent = color.RGBA{R: 239, G: 68, B: 68, A: 255} // Red
	} else if cfg.Mode == ModePrivacyOverlay {
		modeAccent = color.RGBA{R: 245, G: 158, B: 11, A: 255} // Amber
	}
	draw.Draw(img, image.Rect(0, 48, cfg.Width, 50), &image.Uniform{C: modeAccent}, image.Point{}, draw.Src)

	// Window Controls (Mock macOS/Windows dots)
	drawCircle(img, 20, 25, 6, color.RGBA{R: 239, G: 68, B: 68, A: 255})
	drawCircle(img, 40, 25, 6, color.RGBA{R: 245, G: 158, B: 11, A: 255})
	drawCircle(img, 60, 25, 6, color.RGBA{R: 34, G: 197, B: 94, A: 255})

	drawString(img, 85, 30, fmt.Sprintf("Screen Capture Security Validation Harness — %s [%s]", cfg.Platform, cfg.Mode.Code()), color.RGBA{R: 241, G: 245, B: 249, A: 255})

	// 3. Normal Metadata Area
	drawString(img, 40, 80, fmt.Sprintf("SESSION: %s | NONCE: %s | PLATFORM: %s", cfg.SessionID, cfg.RandomNonce, cfg.Platform), color.RGBA{R: 148, G: 163, B: 184, A: 255})
	drawString(img, 40, 100, fmt.Sprintf("TEST MODE: %s", cfg.Mode.String()), modeAccent)
	drawString(img, 40, 120, fmt.Sprintf("TIMESTAMP: %s | FRAME: #%06d", cfg.Timestamp.Format("2006-01-02 15:04:05.000 MST"), cfg.FrameNumber), color.RGBA{R: 203, G: 213, B: 225, A: 255})

	// Live tick progress bar (0 to 100% based on sub-second milliseconds)
	ms := cfg.Timestamp.Nanosecond() / 1_000_000
	progressWidth := int(float64(cfg.Width-80) * (float64(ms) / 1000.0))
	drawRect(img, image.Rect(40, 135, cfg.Width-40, 140), color.RGBA{R: 51, G: 65, B: 85, A: 255})
	if progressWidth > 0 {
		drawRect(img, image.Rect(40, 135, 40+progressWidth, 140), modeAccent)
	}

	// 4. Sensitive Content Box (Left-Middle)
	drawRect(img, cfg.SensitiveBounds, color.RGBA{R: 30, G: 41, B: 59, A: 255})
	drawRectOutline(img, cfg.SensitiveBounds, color.RGBA{R: 71, G: 85, B: 105, A: 255}, 1)
	drawString(img, cfg.SensitiveBounds.Min.X+15, cfg.SensitiveBounds.Min.Y+25, "[SENSITIVE TEST REGION - AUTHORIZED EVALUATION ONLY]", color.RGBA{R: 248, G: 113, B: 113, A: 255})

	if cfg.ShowSensitive {
		drawString(img, cfg.SensitiveBounds.Min.X+15, cfg.SensitiveBounds.Min.Y+50, "MOCK CREDENTIAL / TOKEN PATTERN:", color.RGBA{R: 148, G: 163, B: 184, A: 255})
		drawString(img, cfg.SensitiveBounds.Min.X+15, cfg.SensitiveBounds.Min.Y+70, cfg.SyntheticSecret, color.RGBA{R: 250, G: 204, B: 21, A: 255})
		drawString(img, cfg.SensitiveBounds.Min.X+15, cfg.SensitiveBounds.Min.Y+95, "SYNTHETIC CARD NUMBER: 4000-1234-5678-9010 (EXP 12/30)", color.RGBA{R: 226, G: 232, B: 240, A: 255})
	} else {
		drawString(img, cfg.SensitiveBounds.Min.X+15, cfg.SensitiveBounds.Min.Y+60, "[CONTENT HIDDEN BY APP CONTROL]", color.RGBA{R: 100, G: 116, B: 139, A: 255})
	}

	// 5. Color Calibration Grid (Left-Bottom)
	drawColorCalibrationGrid(img, cfg.ColorGridBounds)

	// 6. QR-Like High-Frequency Matrix Pattern (Right-Middle)
	drawQRMatrix(img, cfg.QRBounds, fmt.Sprintf("%s:%s:%d", cfg.SessionID, cfg.SyntheticSecret, cfg.FrameNumber))

	// 7. Footer Status Bar
	footerColor := color.RGBA{R: 15, G: 23, B: 42, A: 255}
	drawRect(img, image.Rect(0, cfg.Height-40, cfg.Width, cfg.Height), footerColor)
	drawRect(img, image.Rect(0, cfg.Height-40, cfg.Width, cfg.Height-38), color.RGBA{R: 51, G: 65, B: 85, A: 255})
	drawString(img, 40, cfg.Height-18, "Passive Security Validation Suite | No Circumvention | Legitimate Screen-Access Compatibility Test", color.RGBA{R: 100, G: 116, B: 139, A: 255})

	// 8. Application-Level Privacy Overlay (If ModePrivacyOverlay or ModeCombined)
	if cfg.Mode == ModePrivacyOverlay || cfg.Mode == ModeCombined {
		applyVisualPrivacyOverlay(img, cfg.SensitiveBounds, cfg.QRBounds)
	}

	return img
}

// drawColorCalibrationGrid renders a precise 4x4 matrix of pure RGB and secondary colors.
func drawColorCalibrationGrid(img *image.RGBA, bounds image.Rectangle) {
	colors := []color.RGBA{
		{R: 255, G: 0, B: 0, A: 255},     // Red
		{R: 0, G: 255, B: 0, A: 255},     // Green
		{R: 0, G: 0, B: 255, A: 255},     // Blue
		{R: 255, G: 255, B: 0, A: 255},   // Yellow
		{R: 0, G: 255, B: 255, A: 255},   // Cyan
		{R: 255, G: 0, B: 255, A: 255},   // Magenta
		{R: 255, G: 255, B: 255, A: 255}, // White
		{R: 0, G: 0, B: 0, A: 255},       // Black
		{R: 255, G: 128, B: 0, A: 255},   // Orange
		{R: 128, G: 0, B: 255, A: 255},   // Violet
		{R: 0, G: 128, B: 128, A: 255},   // Teal
		{R: 128, G: 128, B: 128, A: 255}, // Mid Gray
		{R: 64, G: 64, B: 64, A: 255},     // Dark Gray
		{R: 192, G: 192, B: 192, A: 255}, // Light Gray
		{R: 16, G: 185, B: 129, A: 255},  // Emerald
		{R: 236, G: 72, B: 153, A: 255},  // Pink
	}

	drawRectOutline(img, bounds, color.RGBA{R: 71, G: 85, B: 105, A: 255}, 1)
	drawString(img, bounds.Min.X+10, bounds.Min.Y+20, "COLOR CALIBRATION & GEOMETRIC TEST GRID (4x4)", color.RGBA{R: 148, G: 163, B: 184, A: 255})

	gridArea := image.Rect(bounds.Min.X+10, bounds.Min.Y+30, bounds.Max.X-10, bounds.Max.Y-10)
	cols, rows := 4, 4
	cellW := (gridArea.Dx() - (cols-1)*4) / cols
	cellH := (gridArea.Dy() - (rows-1)*4) / rows

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			idx := r*cols + c
			x0 := gridArea.Min.X + c*(cellW+4)
			y0 := gridArea.Min.Y + r*(cellH+4)
			cellRect := image.Rect(x0, y0, x0+cellW, y0+cellH)
			drawRect(img, cellRect, colors[idx])
			drawRectOutline(img, cellRect, color.RGBA{R: 15, G: 23, B: 42, A: 255}, 1)
		}
	}
}

// drawQRMatrix renders a deterministic 2D synthetic matrix code based on sha256 hash.
func drawQRMatrix(img *image.RGBA, bounds image.Rectangle, payload string) {
	drawRect(img, bounds, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	drawRectOutline(img, bounds, color.RGBA{R: 71, G: 85, B: 105, A: 255}, 2)

	// Finder patterns in top-left, top-right, bottom-left
	pad := 15
	matrixArea := image.Rect(bounds.Min.X+pad, bounds.Min.Y+pad, bounds.Max.X-pad, bounds.Max.Y-pad)
	gridSize := 21 // 21x21 modules like QR Version 1
	cellSize := matrixArea.Dx() / gridSize

	h := sha256.Sum256([]byte(payload))

	for r := 0; r < gridSize; r++ {
		for c := 0; c < gridSize; c++ {
			isDark := false

			// Finder pattern 1 (Top-Left 7x7)
			if r < 7 && c < 7 {
				if r == 0 || r == 6 || c == 0 || c == 6 || (r >= 2 && r <= 4 && c >= 2 && c <= 4) {
					isDark = true
				}
			} else if r < 7 && c >= gridSize-7 { // Finder pattern 2 (Top-Right 7x7)
				cRel := c - (gridSize - 7)
				if r == 0 || r == 6 || cRel == 0 || cRel == 6 || (r >= 2 && r <= 4 && cRel >= 2 && cRel <= 4) {
					isDark = true
				}
			} else if r >= gridSize-7 && c < 7 { // Finder pattern 3 (Bottom-Left 7x7)
				rRel := r - (gridSize - 7)
				if rRel == 0 || rRel == 6 || c == 0 || c == 6 || (rRel >= 2 && rRel <= 4 && c >= 2 && c <= 4) {
					isDark = true
				}
			} else if r == 6 || c == 6 { // Timing patterns
				isDark = (r+c)%2 == 0
			} else {
				// Deterministic payload bit from hash
				byteIdx := (r*gridSize + c) / 8 % len(h)
				bitIdx := uint((r*gridSize + c) % 8)
				isDark = ((h[byteIdx] >> bitIdx) & 1) == 1
			}

			if isDark {
				x0 := matrixArea.Min.X + c*cellSize
				y0 := matrixArea.Min.Y + r*cellSize
				drawRect(img, image.Rect(x0, y0, x0+cellSize, y0+cellSize), color.RGBA{R: 0, G: 0, B: 0, A: 255})
			}
		}
	}
}

// applyVisualPrivacyOverlay simulates application-level privacy masking (e.g. privacy shield / blur scrim).
func applyVisualPrivacyOverlay(img *image.RGBA, sensitiveBox, qrBox image.Rectangle) {
	// Overlay on Sensitive Region
	overlayBox := sensitiveBox
	veilColor := color.RGBA{R: 15, G: 23, B: 42, A: 245}
	drawRect(img, overlayBox, veilColor)
	drawRectOutline(img, overlayBox, color.RGBA{R: 245, G: 158, B: 11, A: 255}, 2)

	// Diagonal hazard stripes across the overlay
	stripeColor := color.RGBA{R: 245, G: 158, B: 11, A: 80}
	for x := overlayBox.Min.X; x < overlayBox.Max.X+overlayBox.Dy(); x += 20 {
		for y := 0; y < overlayBox.Dy(); y++ {
			px := x - y
			py := overlayBox.Min.Y + y
			if px >= overlayBox.Min.X && px < overlayBox.Max.X {
				img.SetRGBA(px, py, stripeColor)
			}
		}
	}

	// Bold Shield text banner
	bannerRect := image.Rect(overlayBox.Min.X+20, overlayBox.Min.Y+30, overlayBox.Max.X-20, overlayBox.Min.Y+80)
	drawRect(img, bannerRect, color.RGBA{R: 245, G: 158, B: 11, A: 255})
	drawString(img, bannerRect.Min.X+15, bannerRect.Min.Y+30, "PROTECTED CONTENT SHIELD ACTIVE", color.RGBA{R: 15, G: 23, B: 42, A: 255})
	drawString(img, bannerRect.Min.X+15, bannerRect.Min.Y+45, "[APPLICATION VISUAL PRIVACY MASKING ENGAGED]", color.RGBA{R: 30, G: 41, B: 59, A: 255})
}

// Utility drawing helpers
func drawRect(img *image.RGBA, rect image.Rectangle, c color.RGBA) {
	draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func drawRectOutline(img *image.RGBA, rect image.Rectangle, c color.RGBA, thickness int) {
	// Top
	drawRect(img, image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+thickness), c)
	// Bottom
	drawRect(img, image.Rect(rect.Min.X, rect.Max.Y-thickness, rect.Max.X, rect.Max.Y), c)
	// Left
	drawRect(img, image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+thickness, rect.Max.Y), c)
	// Right
	drawRect(img, image.Rect(rect.Max.X-thickness, rect.Min.Y, rect.Max.X, rect.Max.Y), c)
}

func drawCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				img.SetRGBA(cx+x, cy+y, c)
			}
		}
	}
}

func drawString(img *image.RGBA, x, y int, label string, c color.RGBA) {
	point := fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}

// ComputeSubImageDifference evaluates two RGBA images of the same dimensions.
func ComputePixelMSE(imgA, imgB *image.RGBA) float64 {
	bA := imgA.Bounds()
	bB := imgB.Bounds()
	if bA.Dx() != bB.Dx() || bA.Dy() != bB.Dy() {
		return math.MaxFloat64
	}

	var sumSq float64
	totalPixels := float64(bA.Dx() * bA.Dy() * 3)

	for y := bA.Min.Y; y < bA.Max.Y; y++ {
		for x := bA.Min.X; x < bA.Max.X; x++ {
			cA := imgA.RGBAAt(x, y)
			cB := imgB.RGBAAt(x, y)

			dr := float64(cA.R) - float64(cB.R)
			dg := float64(cA.G) - float64(cB.G)
			db := float64(cA.B) - float64(cB.B)

			sumSq += dr*dr + dg*dg + db*db
		}
	}

	return sumSq / totalPixels
}
