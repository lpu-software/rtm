# LPU - Remote Terminal Management

LPU is a command-first, zero-manual-installation remote support platform built in Go. It enables an authorized Host computer to securely share its screen and grant remote control capabilities to another person (the Receiver) over the internet—all directly from the terminal.

## Architecture

* **Language:** Go (Golang) compiled to static, self-contained binaries.
* **Global Connectivity:** Pion WebRTC for peer-to-peer encrypted data channels and NAT traversal.
* **Signaling:** A lightweight WebSocket server for initial SDP and ICE candidate exchange.
* **Screen Sharing:** MJPEG streaming natively captured via `kbinani/screenshot`.
* **Remote Control:** Cross-platform native input injection via `go-vgo/robotgo`.

## Installation

LPU is designed to be instantly available on any machine.

### Mac
Install via Homebrew:
```bash
brew tap lpu-software/lpu
brew install lpu
```

### Windows
Install via PowerShell (downloads and installs instantly):
```powershell
iwr https://tinyurl.com/lpu-windows | iex
```

---

## How to Use

LPU connects to a hosted signaling server on Render (`https://lpushare.onrender.com`). This means no local tunnels (Cloudflare/Localtunnel) or local server ports are required on your machine!

### Start Sharing

1. Open your terminal and run:
```bash
lpu start
```

This will start the screen sharing daemon in the background and instantly output the sharing details:
```
Starting LPU background service...
  ✓ Host Session started (PID: 91366)
  ✓ Registering session code... done.

==============================================
 LPU Public Session Started Successfully!
==============================================
 Receiver Link:  https://lpushare.onrender.com
 Session Code:   1f02dc1c
==============================================
You can now safely close this terminal window.
```

2. **The Receiver Connects (Zero Installation):** 
   - The Receiver opens the printed link (`https://lpushare.onrender.com`) in any web browser.
   - They enter the Session Code (`1f02dc1c`) and instantly see your screen.
   - Connections are **automatically accepted** by default.

3. **Manage the Session:**
   - Check status at any time: `lpu status`
   - Terminate screen sharing: `lpu stop`
   - View background logs: `~/.lpu/lpu.log`

---

### Advanced / Manual Setup

If you want to run the server on a custom port or use a custom signaling server:

#### Step 1: Start your own Signaling Server
```bash
lpu serve -port 8080
```

#### Step 2: The Host Shares Their Screen
Point to your custom signaling WebSocket URL:
```bash
lpu lele -server wss://my-custom-server.com/ws
```

#### Step 3: Enable Remote Control (Host-Side)
In the interactive terminal session, the Host can explicitly type:
- `allow mouse`
- `allow keyboard`
- `stop` (to disconnect immediately)

---

## Security

LPU is designed as a legitimate remote support tool. It strictly enforces:
1. **Explicit Authorization:** Connecting alone grants no control. The Host must explicitly type `allow mouse` or `allow keyboard` in the active session.
2. **Brute-Force Protection:** Sessions drop permanently after 5 failed connection attempts.
3. **Session Expiration:** Temporary session codes expire unconditionally after 2 hours.
4. **OS Enforcement:** LPU respects operating system boundaries (e.g. macOS Gatekeeper, Screen Recording, and Accessibility permission dialogs).
