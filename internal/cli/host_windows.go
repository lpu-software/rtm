//go:build windows
// +build windows

package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	"github.com/yatishydv/rtm/internal/protocol"
	peerpkg "github.com/yatishydv/rtm/internal/webrtc"
	"github.com/yatishydv/rtm/pkg/screenaccess"
)

func RunHost(serverAddr string) {
	fmt.Println("Starting Host Session...")

	engine, err := screenaccess.NewEngine()
	if err != nil {
		log.Printf("Warning: Failed to initialize Windows screen access engine: %v. Using fallback.\n", err)
	}
	if engine != nil {
		defer engine.Close()
	}
	
	var conn *websocket.Conn
	for i := 0; i < 10; i++ {
		conn, _, err = websocket.DefaultDialer.Dial(serverAddr, nil)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatal("Could not connect to signaling server after retries:", err)
	}
	defer conn.Close()

	err = conn.WriteJSON(protocol.Message{Type: "register_host"})
	if err != nil {
		log.Fatal(err)
	}

	var currentSession string
	var peer *peerpkg.Peer
	
	allowMouse := true
	allowKeyboard := true

	acceptConnection := func() {
		fmt.Println("\nSession: ACTIVE")
		fmt.Println("Screen Sharing: ON (Complete Top-Level Composited View + Real Cursor)")
		fmt.Printf("Remote Control: Mouse=%v, Keyboard=%v\n", allowMouse, allowKeyboard)
		
		var err error
		peer, err = peerpkg.NewPeer(true, func(signal string) {
			conn.WriteJSON(protocol.Message{
				Type: "signal",
				Session: currentSession,
				Signal: signal,
			})
		}, func(msg string) {
			var ev screenaccess.RemoteInputEvent
			if err := json.Unmarshal([]byte(msg), &ev); err == nil {
				isMouse := strings.HasPrefix(ev.Type, "mouse") || ev.Type == "double_click"
				isKey := strings.HasPrefix(ev.Type, "key")
				
				if (isMouse && allowMouse) || (isKey && allowKeyboard) {
					if engine != nil {
						_ = engine.InjectInput(ev)
					}
				}
			}
		}, nil)

		if err != nil {
			log.Println("WebRTC init error:", err)
		} else {
			peer.OnDataChannel = func(d *webrtc.DataChannel) {
				go func() {
					for {
						time.Sleep(25 * time.Millisecond) // ~40 FPS adaptive rate

						// Avoid network queue congestion & latency build-up
						if d.BufferedAmount() > 256*1024 {
							continue
						}

						if engine != nil {
							frame, captureErr := engine.CaptureDisplay(0)
							if captureErr != nil || frame == nil || len(frame.JPEGBytes) == 0 {
								continue
							}

							_ = d.Send(frame.JPEGBytes)
						}
					}
				}()
			}
		}

		conn.WriteJSON(protocol.Message{
			Type: "connection_response",
			Session: currentSession,
			Payload: "accepted",
		})
	}

	go func() {
		for {
			var msg protocol.Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Println("\nDisconnected from server.")
				CleanSessionInfo()
				os.Exit(0)
			}

			switch msg.Type {
			case "host_registered":
				currentSession = msg.Session
				fmt.Printf("\nRemote session created\nSession Code: %s\nWaiting for connection...\n\n", msg.Session)
				WriteSessionInfo(msg.Session)
			
			case "incoming_connection":
				fmt.Println("\nRemote connection request received. Auto-accepting...")
				acceptConnection()
			
			case "signal":
				if peer != nil {
					peer.HandleSignal(msg.Signal)
				}
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			select {}
		}
		text = strings.TrimSpace(strings.ToLower(text))
		
		if text == "allow mouse" {
			allowMouse = true
			fmt.Println("Remote Mouse Control: ENABLED")
		} else if text == "allow keyboard" {
			allowKeyboard = true
			fmt.Println("Remote Keyboard Control: ENABLED")
		} else if text == "stop" {
			fmt.Println("Terminating session...")
			CleanSessionInfo()
			os.Exit(0)
		}
	}
}
