//go:build !darwin && !windows

package screenaccess

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"sync"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
)

type FallbackScreenEngine struct {
	mu           sync.RWMutex
	displays     []DisplayInfo
	frameQuality int
}

func NewFallbackScreenEngine() (*FallbackScreenEngine, error) {
	engine := &FallbackScreenEngine{
		frameQuality: 45,
	}
	engine.refreshDisplays()
	return engine, nil
}

func (e *FallbackScreenEngine) refreshDisplays() {
	e.mu.Lock()
	defer e.mu.Unlock()

	numDisplays := screenshot.NumActiveDisplays()
	e.displays = make([]DisplayInfo, 0, numDisplays)

	for i := 0; i < numDisplays; i++ {
		b := screenshot.GetDisplayBounds(i)
		e.displays = append(e.displays, DisplayInfo{
			Index:       i,
			Bounds:      b,
			ScaleFactor: 1.0,
			IsMain:      i == 0,
			Width:       b.Dx(),
			Height:      b.Dy(),
		})
	}
}

func (e *FallbackScreenEngine) GetDisplays() []DisplayInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.displays
}

func (e *FallbackScreenEngine) CaptureDisplay(displayIndex int) (*FrameData, error) {
	start := time.Now()
	bounds := screenshot.GetDisplayBounds(displayIndex)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("capture failed: %w", err)
	}

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	var targetImg image.Image = img
	if w > 1920 {
		targetW := 1920
		targetH := int(float64(h) * (1920.0 / float64(w)))
		targetImg = scaleImageRGBA(img, targetW, targetH)
		w = targetW
		h = targetH
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, targetImg, &jpeg.Options{Quality: e.frameQuality}); err != nil {
		return nil, fmt.Errorf("jpeg encode failed: %w", err)
	}

	return &FrameData{
		JPEGBytes:    buf.Bytes(),
		Width:        w,
		Height:       h,
		Timestamp:    start,
		DisplayIndex: displayIndex,
	}, nil
}

func (e *FallbackScreenEngine) InjectInput(ev RemoteInputEvent) error {
	sw, sh := robotgo.GetScreenSize()
	posX := int(ev.X * float64(sw))
	posY := int(ev.Y * float64(sh))

	switch ev.Type {
	case "mouse_move":
		robotgo.Move(posX, posY)
	case "mouse_click":
		robotgo.Move(posX, posY)
		robotgo.Click("left")
	case "key_press":
		if ev.Key != "" {
			robotgo.KeyTap(ev.Key)
		}
	}
	return nil
}

func (e *FallbackScreenEngine) Close() error {
	return nil
}

func scaleImageRGBA(src *image.RGBA, targetW, targetH int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()
	for y := 0; y < targetH; y++ {
		sy := (y * srcH) / targetH
		for x := 0; x < targetW; x++ {
			sx := (x * srcW) / targetW
			dst.SetRGBA(x, y, src.RGBAAt(sx, sy))
		}
	}
	return dst
}
