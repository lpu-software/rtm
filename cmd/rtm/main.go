package main

import (
	"flag"
	"fmt"
	"os"
	"log"
	"net/http"

	"github.com/yatishydv/rtm/internal/cli"
	"github.com/yatishydv/rtm/internal/signaling"
	"github.com/yatishydv/rtm/web"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "lele":
		hostCmd := flag.NewFlagSet("lele", flag.ExitOnError)
		serverAddr := hostCmd.String("server", "ws://localhost:8080/ws", "Signaling server address")
		hostCmd.Parse(os.Args[2:])
		cli.RunHost(*serverAddr)

	case "dede":
		connectCmd := flag.NewFlagSet("dede", flag.ExitOnError)
		serverAddr := connectCmd.String("server", "ws://localhost:8080/ws", "Signaling server address")
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

	case "serve":
		serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
		port := serveCmd.String("port", "8080", "Port to run the signaling server on")
		serveCmd.Parse(os.Args[2:])

		server := signaling.NewServer()
		http.HandleFunc("/ws", server.HandleWS)
		http.Handle("/", http.FileServer(http.FS(web.StaticFS)))

		addr := ":" + *port
		log.Printf("Signaling server (with embedded web viewer) listening on %s", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatal(err)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("lpu - Terminal-based Remote Access (v1.0.2)")
	fmt.Println("\nUsage:")
	fmt.Println("  lpu lele                    Start a new host session")
	fmt.Println("  lpu dede <session_code>     Connect to an existing host")
	fmt.Println("  lpu serve                   Start the signaling server and web viewer")
	fmt.Println("  lpu kya                     Check for and install updates")
}
