package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/yatishydv/rtm/internal/signaling"
	"github.com/yatishydv/rtm/web"
)

func main() {
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	port := serveCmd.String("port", "8080", "Port to run the signaling server on")
	// If arguments are passed, parse them; otherwise parse empty slice
	args := os.Args[1:]
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		args = os.Args[2:]
	}
	_ = serveCmd.Parse(args)

	server := signaling.NewServer()
	http.HandleFunc("/ws", server.HandleWS)
	http.Handle("/", http.FileServer(http.FS(web.StaticFS)))

	p := *port
	if envPort := os.Getenv("PORT"); envPort != "" {
		p = envPort
	}
	addr := ":" + p
	log.Printf("Signaling server (with embedded web viewer) listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
