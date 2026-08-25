package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func getLpuDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	lpuDir := filepath.Join(home, ".lpu")
	if err := os.MkdirAll(lpuDir, 0755); err != nil {
		return "", err
	}
	return lpuDir, nil
}

func WriteSessionInfo(sessionCode string) {
	lpuDir, err := getLpuDir()
	if err == nil {
		_ = os.WriteFile(filepath.Join(lpuDir, "session.txt"), []byte(sessionCode), 0644)
	}
}

func CleanSessionInfo() {
	lpuDir, err := getLpuDir()
	if err == nil {
		_ = os.Remove(filepath.Join(lpuDir, "session.txt"))
		_ = os.Remove(filepath.Join(lpuDir, "session.pid"))
	}
}

func StatusHost() {
	lpuDir, err := getLpuDir()
	if err != nil {
		fmt.Printf("Error: could not find LPU configuration directory: %v\n", err)
		return
	}

	pidBytes, err := os.ReadFile(filepath.Join(lpuDir, "session.pid"))
	if err != nil {
		fmt.Println("No active LPU background session found.")
		return
	}

	pidStr := strings.TrimSpace(string(pidBytes))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		fmt.Println("Error reading session PID.")
		return
	}

	// Check if process is running using platform-specific check
	if !checkProcessAlive(pid) {
		fmt.Println("No active LPU background session found.")
		CleanSessionInfo()
		return
	}

	sessionBytes, _ := os.ReadFile(filepath.Join(lpuDir, "session.txt"))
	sessionCode := strings.TrimSpace(string(sessionBytes))
	if sessionCode == "" {
		sessionCode = "Unknown"
	}

	fmt.Println("LPU Host is running in the background.")
	fmt.Printf("  PID:          %d\n", pid)
	fmt.Printf("  Session Code: %s\n", sessionCode)
	fmt.Printf("  Log File:     %s/lpu.log\n", lpuDir)
	fmt.Println("\nTo stop this session, run: lpu stop")
}

func StopHost() {
	lpuDir, err := getLpuDir()
	if err != nil {
		fmt.Printf("Error: could not find LPU configuration directory: %v\n", err)
		return
	}

	pidBytes, err := os.ReadFile(filepath.Join(lpuDir, "session.pid"))
	if err != nil {
		fmt.Println("No active LPU background session to stop.")
		return
	}

	pidStr := strings.TrimSpace(string(pidBytes))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		fmt.Println("Error reading session PID.")
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("No active LPU background session found.")
		CleanSessionInfo()
		return
	}

	fmt.Printf("Stopping LPU background session (PID: %d)... ", pid)
	err = process.Kill()
	if err != nil {
		// If process is already dead
		fmt.Println("already stopped.")
	} else {
		fmt.Println("stopped.")
	}
	CleanSessionInfo()
}
