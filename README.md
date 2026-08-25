# LPU - Remote Terminal Management

LPU is a command-first, zero-manual-installation remote support platform built in Go. It enables an authorized Host computer to securely share its screen and grant remote control capabilities to another computer over the internet directly from the terminal.

## Architecture

* **Language:** Go (Golang) compiled to static, self-contained binaries.
* **Global Connectivity:** Pion WebRTC for peer-to-peer encrypted data channels and NAT traversal.
* **Signaling:** A lightweight WebSocket server for initial SDP and ICE candidate exchange.
* **Screen Sharing:** MJPEG streaming natively captured via `kbinani/screenshot`.
* **Remote Control:** Cross-platform native input injection via `go-vgo/robotgo`.

## Security

LPU is designed as a legitimate remote support tool. It strictly enforces:
1. **Explicit Authorization:** Connecting alone grants no control. The Host must explicitly type `allow mouse` or `allow keyboard` in the active session.
2. **Brute-Force Protection:** Sessions drop permanently after 5 failed connection attempts.
3. **Session Expiration:** Temporary session codes expire unconditionally after 2 hours.
4. **OS Enforcement:** LPU respects operating system boundaries (e.g. macOS Gatekeeper, Screen Recording, and Accessibility permission dialogs).

## Deployment (Zero-Installation)

The core requirement is that Hosts do not need to download GUI installers or manage configurations.

**Install/Run:**
```bash
curl -sL https://raw.githubusercontent.com/yatishydv/lpu/main/install.sh | bash
lpu lele
```

## Usage

**Start a Host Session:**
```bash
lpu lele
```
The terminal will display a temporary session code (e.g. `a1b2c3d4`) and wait.

**Connect to a Host:**
```bash
lpu dede <SESSION_CODE>
```
Once the Host types `y` to accept, the connection establishes a secure WebRTC peer-to-peer data channel.

**Browser Viewing:**
The `connect` command securely streams the Host's screen to a local HTTP server (`http://localhost:9090`) and automatically opens the user's default web browser.

**Enable Remote Control (Host-Side):**
In the active terminal session, the Host types:
- `allow mouse`
- `allow keyboard`
- `stop` (to immediately sever the connection)
