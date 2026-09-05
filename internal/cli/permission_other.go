//go:build !darwin

package cli

// CheckScreenRecordingPermission returns true on non-macOS systems.
func CheckScreenRecordingPermission() bool {
	return true
}
