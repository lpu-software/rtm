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

LPU is packed with a built-in orchestrator that runs the signaling server, localtunnel, and screen sharing together in a **single command**.

### The Easy Way (One-Command Setup)

1. Open your terminal and run:
```bash
lpu start
```

This will run everything in the background, automatically create a public tunnel, and print the receiver link and code:
```
==============================================
 LPU Public Session Started Successfully!
==============================================
 Receiver Link:  https://stale-laws-enter.loca.lt
 Session Code:   1f02dc1c
==============================================
You can now safely close this terminal window.
```

2. **The Receiver Connects (Zero Installation):** 
   - The Receiver opens the printed link (e.g. `https://stale-laws-enter.loca.lt`) in their web browser.
   - They enter the Session Code (`1f02dc1c`) and instantly see your screen.
   - Connections in this mode are **automatically accepted**.

3. **Manage the Session:**
   - Check status at any time: `lpu status`
   - Terminate the sharing and stop the server/tunnel completely: `lpu stop`
   - View background logs: `~/.lpu/lpu.log`

---

### Advanced / Manual Setup (Three-Terminal Mode)

If you want to run the server on a custom port, manually verify connections, or manage your own tunnel:

#### Step 1: Start the Signaling Server
```bash
lpu serve
```
Expose it to the internet (in a new terminal):
```bash
npx localtunnel --port 8080
```
*Gives you a URL like: `https://my-lpu.loca.lt`*

#### Step 2: The Host Shares Their Screen
Point to your tunneling URL:
```bash
lpu lele -server wss://my-lpu.loca.lt/ws
```

#### Step 3: Enable Remote Control (Host-Side)
In the interactive terminal session, the Host must explicitly type:
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
