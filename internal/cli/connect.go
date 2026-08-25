package cli

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"os/exec"
	"runtime"
	"io"

	"github.com/gorilla/websocket"
	"github.com/yatishydv/rtm/internal/protocol"
	peerpkg "github.com/yatishydv/rtm/internal/webrtc"
)

var (
	frameMu sync.Mutex
	latestFrame []byte
)

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

func startViewerServer(peer *peerpkg.Peer) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html>
			<head>
				<title>RTM Viewer</title>
				<style>
					body { background: #000; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; }
					img { max-width: 100%; max-height: 100%; object-fit: contain; }
				</style>
			</head>
			<body>
				<img id="screen" src="/frame" />
				<script>
					setInterval(() => {
						document.getElementById('screen').src = '/frame?' + new Date().getTime();
					}, 33); // ~30 fps refresh

					function sendMouse(type, e) {
						const img = document.getElementById('screen');
						if (!img.naturalWidth) return;
						const rect = img.getBoundingClientRect();
						const imgRatio = img.naturalWidth / img.naturalHeight;
						const rectRatio = rect.width / rect.height;
						
						let renderWidth = rect.width;
						let renderHeight = rect.height;
						let offsetX = 0;
						let offsetY = 0;

						if (rectRatio > imgRatio) {
							renderWidth = rect.height * imgRatio;
							offsetX = (rect.width - renderWidth) / 2;
						} else {
							renderHeight = rect.width / imgRatio;
							offsetY = (rect.height - renderHeight) / 2;
						}

						let x = e.clientX - rect.left - offsetX;
						let y = e.clientY - rect.top - offsetY;

						// Bound it so it doesn't click outside the actual image
						if (x < 0 || x > renderWidth || y < 0 || y > renderHeight) return;

						// Calculate normalized coordinates (0.0 to 1.0)
						let normX = x / renderWidth;
						let normY = y / renderHeight;

						fetch('/input', {
							method: 'POST',
							body: JSON.stringify({ type: type, x: normX, y: normY })
						});
					}

					let lastMove = 0;
					document.addEventListener('mousemove', (e) => {
						const now = Date.now();
						if (now - lastMove > 16) { // ~60fps
							lastMove = now;
							sendMouse('mouse_move', e);
						}
					});

					document.addEventListener('click', (e) => {
						sendMouse('mouse_click', e);
					});

					document.addEventListener('keydown', (e) => {
						fetch('/input', {
							method: 'POST',
							body: JSON.stringify({ type: 'key_press', key: e.key })
						});
					});
				</script>
			</body>
			</html>
		`))
	})

	http.HandleFunc("/frame", func(w http.ResponseWriter, r *http.Request) {
		frameMu.Lock()
		defer frameMu.Unlock()
		if len(latestFrame) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(latestFrame)
	})

	http.HandleFunc("/input", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if peer != nil && peer.DataChannel != nil {
			peer.DataChannel.SendText(string(body))
		}
	})

	go func() {
		log.Println("Starting local viewer at http://localhost:9090")
		http.ListenAndServe(":9090", nil)
	}()
}

func RunConnect(serverAddr, sessionCode string) {
	fmt.Printf("Connecting to session %s...\n", sessionCode)
	
	conn, _, err := websocket.DefaultDialer.Dial(serverAddr, nil)
	if err != nil {
		log.Fatal("Could not connect to signaling server:", err)
	}
	defer conn.Close()

	err = conn.WriteJSON(protocol.Message{
		Type: "connection_request",
		Session: sessionCode,
	})
	if err != nil {
		log.Fatal(err)
	}

	var peer *peerpkg.Peer

	for {
		var msg protocol.Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Disconnected from server.")
			os.Exit(0)
		}

		switch msg.Type {
		case "error":
			fmt.Println("Error:", msg.Payload)
			os.Exit(1)
		
		case "connection_response":
			if msg.Payload == "accepted" {
				fmt.Println("Connection accepted by host!")
				fmt.Println("Session: ACTIVE")
				
				var err error
				peer, err = peerpkg.NewPeer(false, func(signal string) {
					conn.WriteJSON(protocol.Message{
						Type: "signal",
						Session: sessionCode,
						Signal: signal,
					})
				}, func(msg string) {
					fmt.Println("\n[Host]:", msg)
				}, func(data []byte) {
					frameMu.Lock()
					latestFrame = data
					frameMu.Unlock()
				})
				
				if err != nil {
					log.Println("WebRTC error:", err)
				}

				startViewerServer(peer)
				openBrowser("http://localhost:9090")
				
				peer.CreateOffer()

			} else {
				fmt.Println("Connection rejected by host.")
				os.Exit(0)
			}
			
		case "signal":
			if peer != nil {
				peer.HandleSignal(msg.Signal)
			}
		}
	}
}
