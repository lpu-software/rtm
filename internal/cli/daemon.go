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

func getPID(lpuDir, filename string) (int, error) {
	pidBytes, err := os.ReadFile(filepath.Join(lpuDir, filename))
	if err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(pidBytes))
	return strconv.Atoi(pidStr)
}

func StatusHost() {
	lpuDir, err := getLpuDir()
	if err != nil {
		fmt.Printf("Error: could not find LPU configuration directory: %v\n", err)
		return
	}

	hostPID, errHost := getPID(lpuDir, "session.pid")
	hostActive := errHost == nil && checkProcessAlive(hostPID)

	if !hostActive {
		fmt.Println("No active LPU background session found.")
		CleanSessionInfo()
		return
	}

	fmt.Println("LPU Background Session Status:")
	sessionBytes, _ := os.ReadFile(filepath.Join(lpuDir, "session.txt"))
	sessionCode := strings.TrimSpace(string(sessionBytes))
	if sessionCode == "" {
		sessionCode = "Generating..."
	}
	fmt.Printf("  ● Host Session:  Running (PID: %d, Code: %s)\n", hostPID, sessionCode)

	fmt.Printf("\nLogs directory: %s\n", lpuDir)
	fmt.Println("To stop the background service, run: lpu stop")
}

func killProcess(lpuDir, filename string) {
	pid, err := getPID(lpuDir, filename)
	if err != nil {
		return
	}
	process, err := os.FindProcess(pid)
	if err == nil {
		_ = process.Kill()
	}
	_ = os.Remove(filepath.Join(lpuDir, filename))
}

func StopHost() {
	lpuDir, err := getLpuDir()
	if err != nil {
		fmt.Printf("Error: could not find LPU configuration directory: %v\n", err)
		return
	}

	fmt.Println("Stopping LPU background service...")
	
	killProcess(lpuDir, "session.pid")
	
	CleanSessionInfo()
	fmt.Println("All services stopped.")
}
