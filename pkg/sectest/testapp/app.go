package testapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// NativeWindowBridge interface abstracts OS-specific window control.
type NativeWindowBridge interface {
	ApplyProtectionMode(mode ProtectionMode) error
	GetWindowID() string
	GetOSProtectionState() (string, bool)
	Close() error
}

// AppController manages the synthetic test application state, timers, and HTTP/IPC controls.
type AppController struct {
	mu           sync.RWMutex
	cfg          *WindowConfig
	bridge       NativeWindowBridge
	server       *http.Server
	port         int
	isRunning    bool
	stopChan     chan struct{}
	ticker       *time.Ticker
	frameCounter int64
	testResults  []map[string]interface{}
}

// NewAppController initializes a new test application instance.
func NewAppController(port int, initialMode ProtectionMode) *AppController {
	cfg := DefaultConfig(runtime.GOOS, initialMode)
	return &AppController{
		cfg:         cfg,
		port:        port,
		stopChan:    make(chan struct{}),
		testResults: make([]map[string]interface{}, 0),
	}
}

// SetBridge registers the OS-specific native window bridge.
func (a *AppController) SetBridge(bridge NativeWindowBridge) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.bridge = bridge
}

// GetConfig returns a copy of the current configuration.
func (a *AppController) GetConfig() WindowConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	cfg := *a.cfg
	cfg.FrameNumber = atomic.LoadInt64(&a.frameCounter)
	cfg.Timestamp = time.Now().UTC()
	return cfg
}

// SetMode updates the protection mode and informs the native window bridge.
func (a *AppController) SetMode(mode ProtectionMode) error {
	a.mu.Lock()
	a.cfg.Mode = mode
	bridge := a.bridge
	a.mu.Unlock()

	if bridge != nil {
		if err := bridge.ApplyProtectionMode(mode); err != nil {
			return fmt.Errorf("native bridge failed to apply mode %v: %w", mode, err)
		}
	}
	return nil
}

// ToggleSensitive hides or reveals the synthetic sensitive credentials.
func (a *AppController) ToggleSensitive(show bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.ShowSensitive = show
}

// RenderCurrentFrame generates the current bitmap frame.
func (a *AppController) RenderCurrentFrame() *image.RGBA {
	cfg := a.GetConfig()
	return GeneratePatternImage(&cfg)
}

// StartController starts the background clock ticker and HTTP control server.
func (a *AppController) StartController() error {
	a.mu.Lock()
	if a.isRunning {
		a.mu.Unlock()
		return nil
	}
	a.isRunning = true
	a.ticker = time.NewTicker(33 * time.Millisecond) // ~30-60 Hz frame tick
	a.mu.Unlock()

	// Clock ticker loop
	go func() {
		for {
			select {
			case <-a.stopChan:
				return
			case <-a.ticker.C:
				atomic.AddInt64(&a.frameCounter, 1)
			}
		}
	}()

	// HTTP control router
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", a.handleStatus)
	mux.HandleFunc("/api/mode", a.handleSetMode)
	mux.HandleFunc("/api/sensitive", a.handleToggleSensitive)
	mux.HandleFunc("/api/reference.png", a.handleReferencePNG)
	mux.HandleFunc("/api/export", a.handleExport)
	mux.HandleFunc("/api/record-result", a.handleRecordResult)
	mux.HandleFunc("/", a.handleWebUI)

	a.server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", a.port),
		Handler: mux,
	}

	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[TestApp] HTTP server error: %v\n", err)
		}
	}()

	return nil
}

// Stop shuts down the app controller.
func (a *AppController) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.isRunning {
		return nil
	}
	a.isRunning = false
	close(a.stopChan)
	if a.ticker != nil {
		a.ticker.Stop()
	}
	if a.server != nil {
		a.server.Close()
	}
	if a.bridge != nil {
		a.bridge.Close()
	}
	return nil
}

// HTTP Handlers
func (a *AppController) handleStatus(w http.ResponseWriter, r *http.Request) {
	cfg := a.GetConfig()
	var winID, osState string
	var osActive bool
	if a.bridge != nil {
		winID = a.bridge.GetWindowID()
		osState, osActive = a.bridge.GetOSProtectionState()
	}

	resp := map[string]interface{}{
		"session_id":        cfg.SessionID,
		"platform":          cfg.Platform,
		"mode":              cfg.Mode.String(),
		"mode_code":         cfg.Mode.Code(),
		"mode_id":           int(cfg.Mode),
		"show_sensitive":    cfg.ShowSensitive,
		"frame_number":      cfg.FrameNumber,
		"timestamp":         cfg.Timestamp.Format(time.RFC3339Nano),
		"window_id":         winID,
		"os_protection":     osState,
		"os_protect_active": osActive,
		"app_visual_mask":   cfg.Mode == ModePrivacyOverlay || cfg.Mode == ModeCombined,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *AppController) handleSetMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Mode int `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Mode < 0 || req.Mode > 3 {
		http.Error(w, "Invalid mode (must be 0-3)", http.StatusBadRequest)
		return
	}

	if err := a.SetMode(ProtectionMode(req.Mode)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	a.handleStatus(w, r)
}

func (a *AppController) handleToggleSensitive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Show bool `json:"show"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	a.ToggleSensitive(req.Show)
	a.handleStatus(w, r)
}

func (a *AppController) handleReferencePNG(w http.ResponseWriter, r *http.Request) {
	img := a.RenderCurrentFrame()
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		http.Error(w, "Failed to encode PNG", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(buf.Bytes())
}

func (a *AppController) handleRecordResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var result map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	a.testResults = append(a.testResults, result)
	a.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "recorded", "count": len(a.testResults)})
}

func (a *AppController) handleExport(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	results := make([]map[string]interface{}, len(a.testResults))
	copy(results, a.testResults)
	cfg := *a.cfg
	a.mu.RUnlock()

	exportData := map[string]interface{}{
		"export_time": time.Now().UTC().Format(time.RFC3339),
		"session_id":  cfg.SessionID,
		"platform":    cfg.Platform,
		"results":     results,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=sectest_results.json")
	json.NewEncoder(w).Encode(exportData)
}

func (a *AppController) handleWebUI(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Security Validation Test Application</title>
  <style>
    :root { --bg: #0f172a; --card: #1e293b; --text: #f8fafc; --accent: #38bdf8; --amber: #f59e0b; --red: #ef4444; --green: #22c55e; }
    body { margin: 0; padding: 20px; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: var(--bg); color: var(--text); }
    .container { max-width: 900px; margin: 0 auto; }
    .card { background: var(--card); border-radius: 12px; padding: 20px; margin-bottom: 20px; border: 1px solid #334155; }
    .btn { background: #334155; color: white; border: none; padding: 10px 18px; border-radius: 6px; cursor: pointer; font-weight: 600; margin-right: 8px; margin-bottom: 8px; }
    .btn:hover { background: #475569; }
    .btn-active { background: var(--accent); color: #0f172a; }
    .btn-red { background: var(--red); }
    .btn-amber { background: var(--amber); color: #0f172a; }
    .btn-green { background: var(--green); color: #0f172a; }
    .preview-box { text-align: center; margin-top: 15px; }
    .preview-box img { max-width: 100%; border-radius: 8px; border: 2px solid #334155; }
    .badge { display: inline-block; padding: 4px 10px; border-radius: 9999px; font-size: 12px; font-weight: 700; }
  </style>
</head>
<body>
  <div class="container">
    <div class="card">
      <h2>🛡️ Screen Capture Security Test Target Window</h2>
      <p style="color: #94a3b8;">Synthetic Test Pattern & Anti-Capture Protection Validation Platform</p>
      <div style="margin-top: 15px;">
        <button class="btn" id="btn-mode-0" onclick="setMode(0)">Test A: Normal (No Protection)</button>
        <button class="btn" id="btn-mode-1" onclick="setMode(1)">Test B: OS Capture Exclusion</button>
        <button class="btn" id="btn-mode-2" onclick="setMode(2)">Test C: App Privacy Overlay</button>
        <button class="btn" id="btn-mode-3" onclick="setMode(3)">Test D: Combined (OS + App)</button>
      </div>
      <div style="margin-top: 15px;">
        <button class="btn" onclick="toggleSensitive(true)">Show Sensitive Region</button>
        <button class="btn" onclick="toggleSensitive(false)">Hide Sensitive Region</button>
        <button class="btn btn-green" onclick="exportResults()">📥 Export Test Telemetry</button>
      </div>
    </div>
    <div class="card preview-box">
      <h3>Active Reference Synthetic Pattern (Live)</h3>
      <img id="ref-img" src="/api/reference.png" alt="Reference Image" />
    </div>
  </div>
  <script>
    async function setMode(mode) {
      await fetch('/api/mode', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ mode }) });
      refresh();
    }
    async function toggleSensitive(show) {
      await fetch('/api/sensitive', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ show }) });
      refresh();
    }
    function refresh() {
      document.getElementById('ref-img').src = '/api/reference.png?t=' + Date.now();
    }
    function exportResults() {
      window.location.href = '/api/export';
    }
    setInterval(refresh, 1000);
  </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}
