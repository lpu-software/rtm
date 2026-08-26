//go:build windows
// +build windows

package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
	"github.com/pion/webrtc/v4"
	"github.com/yatishydv/rtm/internal/protocol"
	peerpkg "github.com/yatishydv/rtm/internal/webrtc"
)

type InputEvent struct {
	Type string  `json:"type"`
	X    float64 `json:"x,omitempty"`
	Y    float64 `json:"y,omitempty"`
	Key  string  `json:"key,omitempty"`
}

func RunHost(serverAddr string) {
	fmt.Println("Starting Host Session...")
	
	var conn *websocket.Conn
	var err error
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
		fmt.Println("Screen Sharing: ON")
		fmt.Printf("Remote Control: Mouse=%v, Keyboard=%v\n", allowMouse, allowKeyboard)
		
		var err error
		peer, err = peerpkg.NewPeer(true, func(signal string) {
			conn.WriteJSON(protocol.Message{
				Type: "signal",
				Session: currentSession,
				Signal: signal,
			})
		}, func(msg string) {
			var ev InputEvent
			if err := json.Unmarshal([]byte(msg), &ev); err == nil {
				if ev.Type == "mouse_click" && allowMouse {
					sw, sh := robotgo.GetScreenSize()
					robotgo.Move(int(ev.X * float64(sw)), int(ev.Y * float64(sh)))
					robotgo.Click("left")
				} else if ev.Type == "mouse_move" && allowMouse {
					sw, sh := robotgo.GetScreenSize()
					robotgo.Move(int(ev.X * float64(sw)), int(ev.Y * float64(sh)))
				} else if ev.Type == "key_press" && allowKeyboard {
					robotgo.KeyTap(ev.Key)
				}
			}
		}, nil)

		if err != nil {
			log.Println("WebRTC init error:", err)
		} else {
			peer.OnDataChannel = func(d *webrtc.DataChannel) {
				go func() {
					for {
						time.Sleep(33 * time.Millisecond)
						bounds := screenshot.GetDisplayBounds(0)
						img, err := screenshot.CaptureRect(bounds)
						if err != nil {
							continue
						}
						var buf bytes.Buffer
						jpeg.Encode(&buf, img, &jpeg.Options{Quality: 30})
						d.Send(buf.Bytes())
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
