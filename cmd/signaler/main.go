package main

import (
	"log"
	"net/http"

	"github.com/yatishydv/rtm/internal/signaling"
)

func main() {
	server := signaling.NewServer()
	http.HandleFunc("/ws", server.HandleWS)

	// Serve the static frontend application
	http.Handle("/", http.FileServer(http.Dir("web")))

	log.Println("Signaling server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
