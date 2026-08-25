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
		_ = os.Remove(filepath.Join(lpuDir, "server.pid"))
		_ = os.Remove(filepath.Join(lpuDir, "tunnel.pid"))
		_ = os.Remove(filepath.Join(lpuDir, "tunnel.txt"))
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

	serverPID, errServer := getPID(lpuDir, "server.pid")
	serverActive := errServer == nil && checkProcessAlive(serverPID)

	tunnelPID, errTunnel := getPID(lpuDir, "tunnel.pid")
	tunnelActive := errTunnel == nil && checkProcessAlive(tunnelPID)

	if !hostActive && !serverActive && !tunnelActive {
		fmt.Println("No active LPU background session found.")
		CleanSessionInfo()
		return
	}

	fmt.Println("LPU Background Services Status:")
	
	if hostActive {
		sessionBytes, _ := os.ReadFile(filepath.Join(lpuDir, "session.txt"))
		sessionCode := strings.TrimSpace(string(sessionBytes))
		if sessionCode == "" {
			sessionCode = "Generating..."
		}
		fmt.Printf("  ● Host Session:  Running (PID: %d, Code: %s)\n", hostPID, sessionCode)
	} else {
		fmt.Println("  ○ Host Session:  Stopped")
	}

	if serverActive {
		fmt.Printf("  ● Local Server:  Running (PID: %d)\n", serverPID)
	} else {
		fmt.Println("  ○ Local Server:  Stopped")
	}

	if tunnelActive {
		tunnelURLBytes, _ := os.ReadFile(filepath.Join(lpuDir, "tunnel.txt"))
		tunnelURL := strings.TrimSpace(string(tunnelURLBytes))
		if tunnelURL == "" {
			tunnelURL = "Exposing..."
		}
		fmt.Printf("  ● Public Tunnel: Running (PID: %d, URL: %s)\n", tunnelPID, tunnelURL)
	} else {
		fmt.Println("  ○ Public Tunnel: Stopped")
	}

	fmt.Printf("\nLogs directory: %s\n", lpuDir)
	fmt.Println("To stop all background services, run: lpu stop")
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

	fmt.Println("Stopping LPU background services...")
	
	killProcess(lpuDir, "session.pid")
	killProcess(lpuDir, "server.pid")
	killProcess(lpuDir, "tunnel.pid")
	
	CleanSessionInfo()
	fmt.Println("All services stopped.")
}
