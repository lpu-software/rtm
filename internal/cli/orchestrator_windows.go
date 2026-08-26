//go:build windows
// +build windows

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const DefaultServerURL = "wss://lpushare.onrender.com/ws"

func StartAll() {
	lpuDir, err := getLpuDir()
	if err != nil {
		fmt.Printf("Error: could not find LPU configuration directory: %v\n", err)
		return
	}

	executable, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting executable path: %v\n", err)
		return
	}

	// 1. Check if already running
	hostPID, _ := getPID(lpuDir, "session.pid")
	if hostPID > 0 && checkProcessAlive(hostPID) {
		fmt.Println("LPU is already running. Run 'lpu stop' first to reset.")
		return
	}

	fmt.Println("Starting LPU background service...")

	// Create log file
	hostLog, _ := os.OpenFile(filepath.Join(lpuDir, "lpu.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

	// Clean old session file
	_ = os.Remove(filepath.Join(lpuDir, "session.txt"))

	// 2. Start the host session (lele) pointing directly to the Render server
	hostCmd := exec.Command(executable, "lele", "-server", DefaultServerURL)
	hostCmd.Stdout = hostLog
	hostCmd.Stderr = hostLog
	hostCmd.Env = append(os.Environ(), "LPU_DAEMON_CHILD=1")
	
	// Hide command window and start in a new process group on Windows
	hostCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	if err := hostCmd.Start(); err != nil {
		fmt.Printf("Failed to start host session: %v\n", err)
		return
	}
	_ = os.WriteFile(filepath.Join(lpuDir, "session.pid"), []byte(fmt.Sprintf("%d", hostCmd.Process.Pid)), 0644)
	fmt.Printf("  ✓ Host Session started (PID: %d)\n", hostCmd.Process.Pid)

	// Wait briefly for the host to register and write session code
	fmt.Print("  ✓ Registering session code... ")
	var sessionCode string
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		codeBytes, _ := os.ReadFile(filepath.Join(lpuDir, "session.txt"))
		sessionCode = strings.TrimSpace(string(codeBytes))
		if sessionCode != "" {
			break
		}
	}

	if sessionCode == "" {
		fmt.Println("timeout. (Check logs at lpu.log)")
		return
	}
	fmt.Println("done.")

	// Render receiver URL is https://lpushare.onrender.com
	receiverURL := strings.Replace(DefaultServerURL, "wss://", "https://", 1)
	receiverURL = strings.TrimSuffix(receiverURL, "/ws")

	fmt.Println("\n==============================================")
	fmt.Println(" LPU Public Session Started Successfully!")
	fmt.Println("==============================================")
	fmt.Printf(" Receiver Link:  %s\n", receiverURL)
	fmt.Printf(" Session Code:   %s\n", sessionCode)
	fmt.Println("==============================================")
	fmt.Println("You can now safely close this terminal window.")
	fmt.Println("To stop all services later, run: lpu stop")
}
