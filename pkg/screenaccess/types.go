package screenaccess

import (
	"image"
	"time"
)

// DisplayInfo contains resolution, DPI scaling, and bounds of a physical monitor.
type DisplayInfo struct {
	Index        int             `json:"index"`
	Bounds       image.Rectangle `json:"bounds"`
	ScaleFactor  float64         `json:"scale_factor"`
	IsMain       bool            `json:"is_main"`
	Width        int             `json:"width"`
	Height       int             `json:"height"`
}

// CursorInfo describes the active real system cursor on the host.
type CursorInfo struct {
	X       int       `json:"x"`
	Y       int       `json:"y"`
	Visible bool      `json:"visible"`
	Type    string    `json:"type"` // "arrow", "ibeam", "pointing_hand", "resize_ns", "resize_ew", "custom", etc.
	Hotspot image.Point
	Image   *image.RGBA
}

// RemoteInputEvent defines mouse, keyboard, and scroll interactions from Computer B.
type RemoteInputEvent struct {
	Type     string  `json:"type"`     // "mouse_move", "mouse_down", "mouse_up", "mouse_click", "double_click", "mouse_scroll", "key_down", "key_up", "key_press"
	X        float64 `json:"x"`        // Normalized coordinate 0.0 to 1.0
	Y        float64 `json:"y"`        // Normalized coordinate 0.0 to 1.0
	Button   string  `json:"button"`   // "left", "right", "middle"
	DeltaX   int     `json:"delta_x"`  // Scroll wheel horizontal delta
	DeltaY   int     `json:"delta_y"`  // Scroll wheel vertical delta
	Key      string  `json:"key"`      // Key name or character
	Code     string  `json:"code"`     // Key code (e.g. "KeyA", "Enter")
	AltKey   bool    `json:"alt_key"`
	CtrlKey  bool    `json:"ctrl_key"`
	ShiftKey bool    `json:"shift_key"`
	MetaKey  bool    `json:"meta_key"`
	Display  int     `json:"display"`  // Target monitor index
}

// FrameData contains compressed display pixels and real cursor metadata.
type FrameData struct {
	JPEGBytes    []byte
	Width        int
	Height       int
	Timestamp    time.Time
	Cursor       CursorInfo
	DisplayIndex int
}

// ScreenEngine defines the contract for OS-specific complete display capture and input injection.
type ScreenEngine interface {
	GetDisplays() []DisplayInfo
	CaptureDisplay(displayIndex int) (*FrameData, error)
	InjectInput(ev RemoteInputEvent) error
	Close() error
}
