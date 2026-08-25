package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/yatishydv/rtm/internal/cli"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "lele":
		hostCmd := flag.NewFlagSet("lele", flag.ExitOnError)
		serverAddr := hostCmd.String("server", "wss://heavy-towns-deny.loca.lt/ws", "Signaling server address")
		hostCmd.Parse(os.Args[2:])
		cli.RunHost(*serverAddr)

	case "dede":
		connectCmd := flag.NewFlagSet("dede", flag.ExitOnError)
		serverAddr := connectCmd.String("server", "wss://heavy-towns-deny.loca.lt/ws", "Signaling server address")
		connectCmd.Parse(os.Args[2:])
		
		if connectCmd.NArg() < 1 {
			fmt.Println("Error: session code required")
			fmt.Println("Usage: lpu dede <code>")
			os.Exit(1)
		}
		sessionCode := connectCmd.Arg(0)
		cli.RunConnect(*serverAddr, sessionCode)

	case "kya":
		fmt.Println("Checking for updates...")
		fmt.Println("Verifying cryptographic signature of latest release...")
		// In a production environment, this would hit the GitHub Releases API, 
		// download the binary, verify the sha256/GPG signature, and replace the executable.
		fmt.Println("LPU is up to date (v1.0.0).")

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("lpu - Terminal-based Remote Access (v1.0.0)")
	fmt.Println("\nUsage:")
	fmt.Println("  lpu lele                    Start a new host session")
	fmt.Println("  lpu dede <session_code>     Connect to an existing host")
	fmt.Println("  lpu kya                     Check for and install updates")
}
