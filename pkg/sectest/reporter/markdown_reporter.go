package reporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yatishydv/rtm/pkg/sectest/runner"
)

// ExportMarkdown generates the human-readable Markdown security report.
func ExportMarkdown(run *runner.DifferentialTestRun, outPath string) error {
	dir := filepath.Dir(outPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	var b strings.Builder

	b.WriteString("# Screen Capture Security & Anti-Capture Evaluation Report\n\n")
	b.WriteString(fmt.Sprintf("**Session ID:** `%s`  \n", run.SessionID))
	b.WriteString(fmt.Sprintf("**Generated At:** `%s`  \n", run.StartTime.Format(time.RFC1123)))
	b.WriteString(fmt.Sprintf("**Host Platform:** `%s`  \n", run.HostOS))
	b.WriteString(fmt.Sprintf("**Evaluated OS Environments:** `%s`  \n\n", strings.Join(run.EvaluatedOSList, ", ")))

	b.WriteString("---\n\n")
	b.WriteString("## 1. Executive Summary\n\n")
	b.WriteString("This report presents the empirical findings of an authorized screen-capture security evaluation conducted against synthetic target windows on **Windows and macOS**. The objective is to validate how operating systems, window compositors (DWM / Quartz Compositor), and capture APIs behave when interacting with capture-excluded windows, application privacy shields, and sensitive credentials.\n\n")

	b.WriteString("> [!IMPORTANT]\n")
	b.WriteString("> **Authorized Passive Evaluation:** All tests were conducted strictly within authorized security boundaries using documented platform APIs without DLL injection, API hooking, driver installation, or security circumvention.\n\n")

	// Compatibility Matrix Table
	b.WriteString("## 2. Compatibility & Protection Matrix\n\n")
	b.WriteString("| Platform | Capture Method | Scope | Protection Mode | Result / Classification | Enforcement Layer | Status |\n")
	b.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")

	for _, entry := range run.CompatibilityMatrix {
		statusBadge := fmt.Sprintf("**%s**", entry.Status)
		if entry.Status == "Protected" {
			statusBadge = "🔒 `Protected`"
		} else if entry.Status == "Capturable" {
			statusBadge = "👁️ `Capturable`"
		} else if entry.Status == "Partially Protected" {
			statusBadge = "🛡️ `Partially Protected`"
		} else if entry.Status == "Permission Blocked" {
			statusBadge = "⛔ `Permission Blocked`"
		}

		b.WriteString(fmt.Sprintf("| %s | %s | `%s` | %s | `%s` | `%s` | %s |\n",
			strings.ToUpper(entry.Platform),
			entry.CaptureMethod,
			entry.CaptureScope,
			entry.ProtectionMode,
			entry.Classification,
			entry.Enforcement,
			statusBadge,
		))
	}
	b.WriteString("\n---\n\n")

	// Taxonomy & Enforcement Layer Breakdown
	b.WriteString("## 3. Protection Taxonomy & Behavioral Distinctions\n\n")
	b.WriteString("The evaluation categorizes anti-screen-capture behaviors into four distinct security tiers:\n\n")

	b.WriteString("### A. OS-Enforced Protection\n")
	b.WriteString("- **Windows (`SetWindowDisplayAffinity`)**:\n")
	b.WriteString("  - `WDA_EXCLUDEFROMCAPTURE` (`0x00000011`): Instructs the Desktop Window Manager (DWM) and Windows Graphics Capture (WGC) to exclude the target window completely from capture streams. In GDI (`BitBlt`), the window area renders as solid black.\n")
	b.WriteString("  - `WDA_MONITOR` (`0x00000001`): Restricts window rendering exclusively to the physical monitor, returning black pixels to all capture APIs.\n")
	b.WriteString("- **macOS (`NSWindowSharingNone`)**:\n")
	b.WriteString("  - Setting `NSWindow.sharingType = .none` informs the Quartz Compositor and ScreenCaptureKit (`SCContentFilter`) that the window must not be exposed to other processes, resulting in an omitted window stream or empty image buffers.\n\n")

	b.WriteString("### B. Application-Level Visual Masking\n")
	b.WriteString("- Target applications render a privacy veil, hazard scrim, or obfuscation overlay directly over sensitive credential elements.\n")
	b.WriteString("- The window itself remains capturable, but sensitive sub-regions (credit cards, passwords, API tokens) are visually masked before reaching the display buffer.\n\n")

	b.WriteString("### C. Permission-Based Restrictions\n")
	b.WriteString("- **macOS TCC (Transparency, Consent, and Control)**: Requires explicit user grant in *System Settings > Privacy & Security > Screen Recording*.\n")
	b.WriteString("- When permission is denied, APIs like `ScreenCaptureKit` and `CGDisplayCreateImage` return error codes or blank desktop frames.\n\n")

	b.WriteString("### D. Ordinary Capturable Content\n")
	b.WriteString("- Unprotected windows are captured with full structural and color fidelity (SSIM ≥ 0.95, MSE < 50).\n\n")

	b.WriteString("---\n\n")
	b.WriteString("## 4. API-Specific Differential Findings\n\n")
	b.WriteString("### Windows Findings\n")
	b.WriteString("1. **Windows Graphics Capture (WGC)**: On Windows 10 (2004+) and Windows 11, WGC honors `WDA_EXCLUDEFROMCAPTURE` by rendering whatever content is positioned *behind* the excluded window, ensuring clean exclusion without visual artifacts.\n")
	b.WriteString("2. **GDI Screen DC (`BitBlt`)**: When capturing the whole desktop DC, windows with display affinity protection are rendered as solid black rectangles.\n")
	b.WriteString("3. **PrintWindow (`PW_RENDERFULLCONTENT`)**: When invoked on an affinity-protected window, `PrintWindow` returns `FALSE` or generates a zeroed-out bitmap buffer.\n\n")

	b.WriteString("### macOS Findings\n")
	b.WriteString("1. **ScreenCaptureKit (`SCShareableContent`)**: When a window sets `NSWindowSharingNone`, it is either omitted from `shareableContent.windows` or filtered out by `SCScreenshotManager`.\n")
	b.WriteString("2. **CoreGraphics (`CGWindowListCreateImage`)**: Obsoleted in macOS 15.0 in favor of ScreenCaptureKit. On supported versions, setting `NSWindowSharingNone` causes `CGWindowListCreateImage` to return `nil` or transparent frames.\n")
	b.WriteString("3. **Permission Boundaries**: Screen recording permissions are strictly enforced per-application by TCC.\n\n")

	b.WriteString("---\n\n")
	b.WriteString("## 5. Security & Engineering Recommendations\n\n")
	b.WriteString("1. **Dual-Layer Defense**: For maximum sensitive data protection, combine **OS-level capture exclusion** (`SetWindowDisplayAffinity` on Windows / `NSWindowSharingNone` on macOS) with **Application-level visual masking** (hiding credentials on blur or screen share).\n")
	b.WriteString("2. **Continuous Compatibility Testing**: Run this security validation harness in CI/CD before shipping updates to verify that OS compositor updates have not altered affinity handling.\n")
	b.WriteString("3. **Platform Transparency**: Screen-access tools (like LPU) should explicitly inform users when viewing capture-excluded windows to prevent confusion when black or transparent rectangles appear.\n")

	if err := os.WriteFile(outPath, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("failed to write markdown report: %w", err)
	}

	return nil
}
