//go:build windows
// +build windows

package cli

import (
	"fmt"
	"os"
)

func RunHost(serverAddr string) {
	fmt.Println("Error: Hosting a session is currently only supported on macOS and Linux.")
	fmt.Println("You can still use this Windows machine to connect to a host using: lpu dede <session_code>")
	os.Exit(1)
}
