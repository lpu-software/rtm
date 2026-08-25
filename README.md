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

## How to Use (Global Internet Setup)

Because LPU is a peer-to-peer WebRTC application, it requires a "Signaling Server" to introduce the Host and the Receiver. We packed this server (and the Receiver's web viewer) directly into the binary!

Here is how to set it up for free so anyone in the world can connect to you.

### Step 1: Start the Signaling Server (Free)
You (or anyone) need to run the signaling server and expose it to the internet. 

1. Start the built-in server on your computer:
```bash
lpu serve
```
2. Open a **new terminal window** and use a free tunneling service (like LocalTunnel) to expose port 8080 to the internet:
```bash
npx localtunnel --port 8080
```
*It will give you a public URL like: `https://my-lpu.loca.lt`*

### Step 2: The Host Shares Their Screen
The person who wants to share their screen (the Host) runs this command, pointing to the secure `wss://` WebSocket version of your tunneling URL:

```bash
lpu lele -server wss://my-lpu.loca.lt/ws
```
*The terminal will display a temporary session code (e.g. `a1b2c3d4`) and wait.*

#### Optional: Running in the Background (Daemon Mode)
If you want the host session to keep running even if you close the terminal window, run it with the `-d` (daemon) flag:
```bash
lpu lele -d -server wss://my-lpu.loca.lt/ws
```

In background mode:
- The session is started invisibly, and connections are **automatically accepted**.
- To check the active session code, run: **`lpu status`**
- To terminate the background session, run: **`lpu stop`**
- Logs are written transparently to `~/.lpu/lpu.log`.

### Step 3: The Receiver Connects (Zero Installation!)
The person who wants to view and control the screen (the Receiver) **does not need to install anything!**

1. The Receiver simply opens your LocalTunnel URL in any modern web browser (e.g. `https://my-lpu.loca.lt`).
2. They enter the session code (`a1b2c3d4`) on the webpage.
3. The Host types `y` in their terminal to accept the connection.

### Step 4: Enable Remote Control
By default, the Receiver can only *view* the screen. To grant them actual control, the Host must type these commands in their active terminal session:
- `allow mouse`
- `allow keyboard`
- `stop` (to immediately sever the connection and kick the Receiver out)

---

## Security

LPU is designed as a legitimate remote support tool. It strictly enforces:
1. **Explicit Authorization:** Connecting alone grants no control. The Host must explicitly type `allow mouse` or `allow keyboard` in the active session.
2. **Brute-Force Protection:** Sessions drop permanently after 5 failed connection attempts.
3. **Session Expiration:** Temporary session codes expire unconditionally after 2 hours.
4. **OS Enforcement:** LPU respects operating system boundaries (e.g. macOS Gatekeeper, Screen Recording, and Accessibility permission dialogs).
