package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/yatishydv/rtm/pkg/sectest/probes"
	"github.com/yatishydv/rtm/pkg/sectest/reporter"
	"github.com/yatishydv/rtm/pkg/sectest/runner"
	"github.com/yatishydv/rtm/pkg/sectest/testapp"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd := flag.NewFlagSet("run", flag.ExitOnError)
		outDir := runCmd.String("output-dir", "./sectest_output", "Directory to export reports (JSON, HTML, MD)")
		trials := runCmd.Int("trials", 3, "Number of trials per test case for determinism testing")
		simCross := runCmd.Bool("cross-platform", true, "Include cross-platform specification simulations")
		runCmd.Parse(os.Args[2:])

		executeFullRun(*outDir, *trials, *simCross)

	case "app":
		appCmd := flag.NewFlagSet("app", flag.ExitOnError)
		port := appCmd.Int("port", 18492, "Port for test app HTTP control server")
		mode := appCmd.Int("mode", 0, "Initial mode (0=Normal, 1=OS Exclusion, 2=Privacy Overlay, 3=Combined)")
		appCmd.Parse(os.Args[2:])

		launchApp(*port, testapp.ProtectionMode(*mode))

	case "matrix":
		printCompatibilityMatrix()

	case "probe":
		probeLocalAPIs()

	default:
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("================================================================================")
	fmt.Println("  Screen Capture Security & Anti-Capture Evaluation Module (LPU Sectest)")
	fmt.Println("================================================================================")
	fmt.Println("Usage:")
	fmt.Println("  sectest run [--output-dir <dir>] [--trials <n>]  Run full differential test suite")
	fmt.Println("  sectest app [--port <port>] [--mode <0-3>]       Launch interactive synthetic test app")
	fmt.Println("  sectest probe                                    Probe available OS capture APIs")
	fmt.Println("  sectest matrix                                   Print OS compatibility & protection matrix")
	fmt.Println("")
	fmt.Println("Test Modes:")
	fmt.Println("  0: Test A - Normal (No Protection)")
	fmt.Println("  1: Test B - OS Capture Exclusion (WDA_EXCLUDEFROMCAPTURE / NSWindowSharingNone)")
	fmt.Println("  2: Test C - App Privacy Overlay (Visual Masking)")
	fmt.Println("  3: Test D - Combined (OS Exclusion + App Privacy Overlay)")
	fmt.Println("================================================================================")
}

func executeFullRun(outDir string, trials int, simCross bool) {
	fmt.Println("\n================================================================================")
	fmt.Printf("🚀 Starting Screen Capture Security Differential Evaluation Suite\n")
	fmt.Printf("   Platform: %s (%s) | Trials: %d | Cross-Platform Matrix: %v\n", runtime.GOOS, runtime.GOARCH, trials, simCross)
	fmt.Println("================================================================================")

	harness := runner.NewTestHarness(runner.HarnessConfig{
		TrialsCount:        trials,
		IncludeSimulations: simCross,
		OutputDir:          outDir,
		Headless:           true,
	})

	testRun, err := harness.RunSuite()
	if err != nil {
		fmt.Printf("❌ Test execution failed: %v\n", err)
		os.Exit(1)
	}

	// Export JSON
	jsonPath := filepath.Join(outDir, "results.json")
	if err := reporter.ExportJSON(testRun, jsonPath); err != nil {
		fmt.Printf("⚠️  Failed to export JSON: %v\n", err)
	} else {
		fmt.Printf("✅ JSON Telemetry exported: %s\n", jsonPath)
	}

	// Export Markdown
	mdPath := filepath.Join(outDir, "SECURITY_REPORT.md")
	if err := reporter.ExportMarkdown(testRun, mdPath); err != nil {
		fmt.Printf("⚠️  Failed to export Markdown: %v\n", err)
	} else {
		fmt.Printf("✅ Markdown Security Report exported: %s\n", mdPath)
	}

	// Export HTML
	htmlPath := filepath.Join(outDir, "security_eval_report.html")
	if err := reporter.ExportHTML(testRun, htmlPath); err != nil {
		fmt.Printf("⚠️  Failed to export HTML: %v\n", err)
	} else {
		fmt.Printf("✅ Interactive HTML Report exported: %s\n", htmlPath)
	}

	// Print Console Matrix
	fmt.Println("\n--------------------------------------------------------------------------------")
	fmt.Println("📊 COMPATIBILITY & PROTECTION MATRIX SUMMARY")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("%-10s | %-32s | %-24s | %-20s | %-12s\n", "PLATFORM", "CAPTURE METHOD", "PROTECTION MODE", "RESULT", "STATUS")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, e := range testRun.CompatibilityMatrix {
		fmt.Printf("%-10s | %-32s | %-24s | %-20s | %-12s\n",
			strings.ToUpper(e.Platform),
			truncate(e.CaptureMethod, 32),
			truncate(e.ProtectionMode, 24),
			truncate(e.Classification, 20),
			e.Status,
		)
	}
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("\n✨ Evaluation complete! Open %s in your browser for the full interactive visual report.\n\n", htmlPath)
}

func launchApp(port int, mode testapp.ProtectionMode) {
	fmt.Printf("🛡️  Launching Synthetic Security Test Application on http://127.0.0.1:%d\n", port)
	fmt.Printf("   Initial Mode: %s\n", mode.String())

	ctrl := testapp.NewAppController(port, mode)
	if err := ctrl.StartController(); err != nil {
		fmt.Printf("Failed to start controller: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("   Press Ctrl+C to terminate test application.")
	select {}
}

func probeLocalAPIs() {
	fmt.Printf("🔍 Probing screen capture APIs on host: %s (%s)...\n\n", runtime.GOOS, runtime.GOARCH)
	reg := probes.NewProbeRegistry()
	probesList := reg.GetAllProbes()

	for _, p := range probesList {
		avail := p.IsAvailable()
		granted, desc := p.CheckPermission()
		status := "✅ Available"
		if !avail {
			status = "❌ Unavailable"
		}
		permStatus := "Authorized"
		if !granted {
			permStatus = "Permission Denied / Required"
		}

		fmt.Printf("• Probe: %s\n", p.Name())
		fmt.Printf("  Method: %s | Platform: %s\n", p.Method(), p.Platform())
		fmt.Printf("  Status: %s | Permission: %s (%s)\n\n", status, permStatus, desc)
	}
}

func printCompatibilityMatrix() {
	fmt.Println("================================================================================")
	fmt.Println("  Platform Screen Capture Protection Compatibility Matrix (Documented APIs)")
	fmt.Println("================================================================================")
	fmt.Printf("%-10s | %-32s | %-24s | %-20s | %-12s\n", "PLATFORM", "CAPTURE METHOD", "PROTECTION MODE", "OBSERVED RESULT", "STATUS")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("%-10s | %-32s | %-24s | %-20s | %-12s\n", "WINDOWS", "Windows Graphics Capture (WGC)", "WDA_EXCLUDEFROMCAPTURE", "Transparent/Omitted", "Protected")
	fmt.Printf("%-10s | %-32s | %-24s | %-20s | %-12s\n", "WINDOWS", "GDI Screen DC (BitBlt)", "WDA_EXCLUDEFROMCAPTURE", "Completely Black", "Protected")
	fmt.Printf("%-10s | %-32s | %-24s | %-20s | %-12s\n", "WINDOWS", "PrintWindow (PW_RENDERFULL)", "WDA_EXCLUDEFROMCAPTURE", "Error / Black DC", "Protected")
	fmt.Printf("%-10s | %-32s | %-24s | %-20s | %-12s\n", "WINDOWS", "Window Capture", "Normal (No Exclusion)", "Normal Content", "Capturable")
	fmt.Printf("%-10s | %-32s | %-24s | %-20s | %-12s\n", "MACOS", "ScreenCaptureKit", "NSWindowSharingNone", "Omitted from Stream", "Protected")
	fmt.Printf("%-10s | %-32s | %-24s | %-20s | %-12s\n", "MACOS", "CoreGraphics Window", "NSWindowSharingNone", "Empty / Nil Frame", "Protected")
	fmt.Printf("%-10s | %-32s | %-24s | %-20s | %-12s\n", "MACOS", "ScreenCaptureKit", "App Privacy Overlay", "Visually Masked", "Protected")
	fmt.Printf("%-10s | %-32s | %-24s | %-20s | %-12s\n", "MACOS", "Display Capture", "Normal", "Normal Content", "Capturable")
	fmt.Println("--------------------------------------------------------------------------------")
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
