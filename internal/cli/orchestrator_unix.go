//go:build !windows
// +build !windows

package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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

	fmt.Println("Starting LPU background services...")

	// Create log files
	serverLog, _ := os.OpenFile(filepath.Join(lpuDir, "server.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	tunnelLog, _ := os.OpenFile(filepath.Join(lpuDir, "tunnel.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	hostLog, _ := os.OpenFile(filepath.Join(lpuDir, "lpu.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

	// 2. Start the local signaling server
	serverCmd := exec.Command(executable, "serve")
	serverCmd.Stdout = serverLog
	serverCmd.Stderr = serverLog
	if err := serverCmd.Start(); err != nil {
		fmt.Printf("Failed to start local signaling server: %v\n", err)
		return
	}
	_ = os.WriteFile(filepath.Join(lpuDir, "server.pid"), []byte(fmt.Sprintf("%d", serverCmd.Process.Pid)), 0644)
	fmt.Printf("  ✓ Local Server started (PID: %d)\n", serverCmd.Process.Pid)

	// Wait briefly for the server to bind to port 8080
	time.Sleep(1 * time.Second)

	// 3. Start localtunnel
	fmt.Print("  ✓ Exposing server via public tunnel (npx localtunnel)... ")
	tunnelCmd := exec.Command("npx", "localtunnel", "--port", "8080")
	stdout, err := tunnelCmd.StdoutPipe()
	if err != nil {
		fmt.Printf("failed to pipe localtunnel: %v\n", err)
		return
	}
	tunnelCmd.Stderr = tunnelLog // Send stderr to log
	
	if err := tunnelCmd.Start(); err != nil {
		fmt.Printf("failed (is Node/npx installed?): %v\n", err)
		return
	}
	_ = os.WriteFile(filepath.Join(lpuDir, "tunnel.pid"), []byte(fmt.Sprintf("%d", tunnelCmd.Process.Pid)), 0644)

	// Read localtunnel URL from stdout
	var publicURL string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "your url is:") {
			parts := strings.Split(line, "your url is:")
			if len(parts) > 1 {
				publicURL = strings.TrimSpace(parts[1])
				break
			}
		}
	}
	
	if publicURL == "" {
		fmt.Println("failed to get public URL.")
		return
	}
	fmt.Println("done.")
	_ = os.WriteFile(filepath.Join(lpuDir, "tunnel.txt"), []byte(publicURL), 0644)

	// Expose secure WebSocket URL for the host
	// Localtunnel uses https. We map it to wss.
	wsURL := strings.Replace(publicURL, "https://", "wss://", 1) + "/ws"

	// 4. Start the host session (lele)
	hostCmd := exec.Command(executable, "lele", "-server", wsURL)
	hostCmd.Stdout = hostLog
	hostCmd.Stderr = hostLog
	hostCmd.Env = append(os.Environ(), "LPU_DAEMON_CHILD=1")
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
		fmt.Println("timeout. (Check logs at ~/.lpu/lpu.log)")
		return
	}
	fmt.Println("done.")

	fmt.Println("\n==============================================")
	fmt.Println(" LPU Public Session Started Successfully!")
	fmt.Println("==============================================")
	fmt.Printf(" Receiver Link:  %s\n", publicURL)
	fmt.Printf(" Session Code:   %s\n", sessionCode)
	fmt.Println("==============================================")
	fmt.Println("You can now safely close this terminal window.")
	fmt.Println("To stop all services later, run: lpu stop")
}
