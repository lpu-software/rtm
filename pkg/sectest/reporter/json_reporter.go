package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yatishydv/rtm/pkg/sectest/runner"
)

// ExportJSON writes the complete test run telemetry to a JSON file.
func ExportJSON(run *runner.DifferentialTestRun, outPath string) error {
	dir := filepath.Dir(outPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON results: %w", err)
	}

	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write JSON results file: %w", err)
	}

	return nil
}
