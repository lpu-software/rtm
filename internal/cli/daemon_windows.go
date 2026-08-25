//go:build windows
// +build windows

package cli

import (
	"errors"
)

func Daemonize() (bool, error) {
	return false, errors.New("hosting a session is not supported on Windows")
}

func checkProcessAlive(pid int) bool {
	return false
}
