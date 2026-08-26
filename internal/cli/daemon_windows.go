//go:build windows
// +build windows

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func Daemonize() (bool, error) {
	// If this env var is set, we are already the background child process.
	if os.Getenv("LPU_DAEMON_CHILD") == "1" {
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

	// Hide the command prompt window and run in a detached state on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NO_WINDOW,
	}

	if err := cmd.Start(); err != nil {
		return false, err
	}

	// Save PID to session.pid
	pidPath := filepath.Join(lpuDir, "session.pid")
	_ = os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644)

	fmt.Printf("LPU Host is starting in the background (PID: %d).\n", cmd.Process.Pid)
	fmt.Printf("Logs will be written to: %s\\lpu.log\n", lpuDir)
	fmt.Println("To stop this session, run: lpu stop")

	// Exit the parent process
	os.Exit(0)
	return false, nil
}

func checkProcessAlive(pid int) bool {
	const STILL_ACTIVE = 259
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle)
	
	var exitCode uint32
	err = syscall.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return false
	}
	return exitCode == STILL_ACTIVE
}
