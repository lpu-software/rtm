//go:build !darwin

package cli

func CheckScreenRecordingPermission() bool {
	return true
}

func PromptScreenRecordingPermission() bool {
	return true
}
