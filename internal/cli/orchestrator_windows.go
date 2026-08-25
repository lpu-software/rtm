//go:build windows
// +build windows

package cli

import "fmt"

func StartAll() {
	fmt.Println("Error: Hosting a session is currently only supported on macOS and Linux.")
}
