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
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>LPU Remote Viewer</title>
	<style>
		body { background: #090d16; color: #fff; margin: 0; padding: 0; overflow: hidden; display: flex; justify-content: center; align-items: center; height: 100vh; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
		#container { width: 100vw; height: 100vh; display: flex; justify-content: center; align-items: center; }
		img { max-width: 100vw; max-height: 100vh; object-fit: contain; box-shadow: 0 10px 40px rgba(0,0,0,0.8); }
	</style>
</head>
<body>
	<div id="container">
		<img id="screen" src="/frame" alt="Remote Screen" />
	</div>
	<script>
		const screenImg = document.getElementById('screen');
		const container = document.getElementById('container');

		setInterval(() => {
			screenImg.src = '/frame?' + Date.now();
		}, 25); // ~40 fps refresh

		function getNormalizedCoords(e) {
			if (!screenImg.naturalWidth) return null;
			const rect = screenImg.getBoundingClientRect();
			const imgRatio = screenImg.naturalWidth / screenImg.naturalHeight;
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

			if (x < 0 || x > renderWidth || y < 0 || y > renderHeight) return null;

			return {
				x: Math.max(0, Math.min(1, x / renderWidth)),
				y: Math.max(0, Math.min(1, y / renderHeight))
			};
		}

		function getButtonName(code) {
			if (code === 2) return 'right';
			if (code === 1) return 'middle';
			return 'left';
		}

		function sendInput(payload) {
			fetch('/input', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			}).catch(() => {});
		}

		let lastMove = 0;
		container.addEventListener('mousemove', (e) => {
			const coords = getNormalizedCoords(e);
			if (!coords) return;
			const now = Date.now();
			if (now - lastMove > 16) {
				lastMove = now;
				sendInput({ type: 'mouse_move', x: coords.x, y: coords.y });
			}
		});

		container.addEventListener('mousedown', (e) => {
			const coords = getNormalizedCoords(e);
			if (!coords) return;
			sendInput({ type: 'mouse_down', button: getButtonName(e.button), x: coords.x, y: coords.y });
		});

		container.addEventListener('mouseup', (e) => {
			const coords = getNormalizedCoords(e);
			if (!coords) return;
			sendInput({ type: 'mouse_up', button: getButtonName(e.button), x: coords.x, y: coords.y });
		});

		container.addEventListener('dblclick', (e) => {
			const coords = getNormalizedCoords(e);
			if (!coords) return;
			sendInput({ type: 'double_click', button: 'left', x: coords.x, y: coords.y });
		});

		container.addEventListener('contextmenu', (e) => {
			e.preventDefault();
			const coords = getNormalizedCoords(e);
			if (!coords) return;
			sendInput({ type: 'mouse_click', button: 'right', x: coords.x, y: coords.y });
		});

		container.addEventListener('wheel', (e) => {
			e.preventDefault();
			const coords = getNormalizedCoords(e);
			sendInput({
				type: 'mouse_scroll',
				delta_x: Math.round(e.deltaX),
				delta_y: Math.round(e.deltaY),
				x: coords ? coords.x : 0,
				y: coords ? coords.y : 0
			});
		}, { passive: false });

		document.addEventListener('keydown', (e) => {
			if (["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Space", "Tab", "Backspace"].includes(e.code)) {
				e.preventDefault();
			}
			sendInput({
				type: 'key_press',
				key: e.key,
				code: e.code,
				alt_key: e.altKey,
				ctrl_key: e.ctrlKey,
				shift_key: e.shiftKey,
				meta_key: e.metaKey
			});
		});
	</script>
</body>
</html>`))
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
