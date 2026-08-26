//go:build !windows
// +build !windows

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
)

func Daemonize() (bool, error) {
	// If this env var is set, we are already the background child process.
	if os.Getenv("LPU_DAEMON_CHILD") == "1" {
		signal.Ignore(syscall.SIGHUP)
		return true, nil
	}

	executable, err := os.Executable()
	if err != nil {
		return false, err
	}

	// Filter out the "-d" and "--background" flags so the child doesn't daemonize again.
	var childArgs []string
	for _, arg := range os.Args[1:] {
		if arg != "-d" && arg != "--background" {
			childArgs = append(childArgs, arg)
		}
	}

	lpuDir, err := getLpuDir()
	if err != nil {
		return false, err
	}

	logFile, err := os.OpenFile(filepath.Join(lpuDir, "lpu.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return false, err
	}

	cmd := exec.Command(executable, childArgs...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil // Detach stdin

	// Set environment variable so the child knows it is the daemon
	cmd.Env = append(os.Environ(), "LPU_DAEMON_CHILD=1")

	// We keep Setsid: false on macOS so the process retains its connection
	// to the active WindowServer GUI context. This allows it to capture all
	// user application windows. We handle terminal close by ignoring SIGHUP.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: false,
	}

	if err := cmd.Start(); err != nil {
		return false, err
	}

	// Save PID to session.pid
	pidPath := filepath.Join(lpuDir, "session.pid")
	_ = os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644)

	fmt.Printf("LPU Host is starting in the background (PID: %d).\n", cmd.Process.Pid)
	fmt.Printf("Logs will be written to: %s/lpu.log\n", lpuDir)
	fmt.Println("To stop this session, run: lpu stop")

	// Exit the parent process
	os.Exit(0)
	return false, nil
}

func checkProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
