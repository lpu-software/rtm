//go:build windows

package screenaccess

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
)

const (
	SRCCOPY        = 0x00CC0020
	BI_RGB         = 0
	DIB_RGB_COLORS = 0
	CURSOR_SHOWING = 0x00000001
	DI_NORMAL      = 0x0003
	DI_COMPAT      = 0x0004
)

type POINT struct {
	X, Y int32
}

type CURSORINFO struct {
	CbSize      uint32
	Flags       uint32
	HCursor     uintptr
	PtScreenPos POINT
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]uint32
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

var (
	modUser32Win              = syscall.NewLazyDLL("user32.dll")
	modGdi32Win               = syscall.NewLazyDLL("gdi32.dll")
	procGetDCWin              = modUser32Win.NewProc("GetDC")
	procReleaseDCWin          = modUser32Win.NewProc("ReleaseDC")
	procGetCursorInfoWin      = modUser32Win.NewProc("GetCursorInfo")
	procDrawIconExWin         = modUser32Win.NewProc("DrawIconEx")
	procCreateCompatibleDCWin = modGdi32Win.NewProc("CreateCompatibleDC")
	procCreateDIBSectionWin   = modGdi32Win.NewProc("CreateDIBSection")
	procSelectObjectWin       = modGdi32Win.NewProc("SelectObject")
	procBitBltWin             = modGdi32Win.NewProc("BitBlt")
	procDeleteDCWin           = modGdi32Win.NewProc("DeleteDC")
	procDeleteObjectWin       = modGdi32Win.NewProc("DeleteObject")
)

type WindowsScreenEngine struct {
	mu           sync.RWMutex
	displays     []DisplayInfo
	frameQuality int
}

func NewWindowsScreenEngine() (*WindowsScreenEngine, error) {
	engine := &WindowsScreenEngine{
		frameQuality: 45,
	}
	engine.refreshDisplays()
	return engine, nil
}

func (e *WindowsScreenEngine) refreshDisplays() {
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

func (e *WindowsScreenEngine) GetDisplays() []DisplayInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.displays
}

func (e *WindowsScreenEngine) CaptureDisplay(displayIndex int) (*FrameData, error) {
	start := time.Now()

	bounds := screenshot.GetDisplayBounds(displayIndex)
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		w, h = 1920, 1080
	}

	hdcScreen, _, _ := procGetDCWin.Call(0)
	if hdcScreen == 0 {
		return nil, fmt.Errorf("GetDC(0) failed")
	}
	defer procReleaseDCWin.Call(0, hdcScreen)

	hdcMem, _, _ := procCreateCompatibleDCWin.Call(hdcScreen)
	if hdcMem == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer procDeleteDCWin.Call(hdcMem)

	var bi BITMAPINFO
	bi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bi.BmiHeader))
	bi.BmiHeader.BiWidth = int32(w)
	bi.BmiHeader.BiHeight = -int32(h) // top-down
	bi.BmiHeader.BiPlanes = 1
	bi.BmiHeader.BiBitCount = 32
	bi.BmiHeader.BiCompression = BI_RGB

	var pBits unsafe.Pointer
	hBitmap, _, _ := procCreateDIBSectionWin.Call(
		hdcMem,
		uintptr(unsafe.Pointer(&bi)),
		DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(&pBits)),
		0,
		0,
	)
	if hBitmap == 0 || pBits == nil {
		return nil, fmt.Errorf("CreateDIBSection failed")
	}
	defer procDeleteObjectWin.Call(hBitmap)

	procSelectObjectWin.Call(hdcMem, hBitmap)
	procBitBltWin.Call(hdcMem, 0, 0, uintptr(w), uintptr(h), hdcScreen, uintptr(bounds.Min.X), uintptr(bounds.Min.Y), SRCCOPY)

	// Draw Real System Cursor directly onto desktop memory DC
	var ci CURSORINFO
	ci.CbSize = uint32(unsafe.Sizeof(ci))
	rCur, _, _ := procGetCursorInfoWin.Call(uintptr(unsafe.Pointer(&ci)))
	if rCur != 0 && (ci.Flags&CURSOR_SHOWING) != 0 && ci.HCursor != 0 {
		curX := ci.PtScreenPos.X - int32(bounds.Min.X)
		curY := ci.PtScreenPos.Y - int32(bounds.Min.Y)
		procDrawIconExWin.Call(
			hdcMem,
			uintptr(curX),
			uintptr(curY),
			ci.HCursor,
			0, 0, 0, 0,
			DI_NORMAL|DI_COMPAT,
		)
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	srcSlice := (*[1 << 30]byte)(pBits)[: w*h*4 : w*h*4]
	for i := 0; i < len(srcSlice); i += 4 {
		img.Pix[i+0] = srcSlice[i+2] // R
		img.Pix[i+1] = srcSlice[i+1] // G
		img.Pix[i+2] = srcSlice[i+0] // B
		img.Pix[i+3] = 255          // A
	}

	var targetImg image.Image = img
	if w > 1920 {
		targetW := 1920
		targetH := int(float64(h) * (1920.0 / float64(w)))
		targetImg = scaleImageRGBAWin(img, targetW, targetH)
		w = targetW
		h = targetH
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, targetImg, &jpeg.Options{Quality: e.frameQuality}); err != nil {
		return nil, fmt.Errorf("jpeg compression failed: %w", err)
	}

	return &FrameData{
		JPEGBytes:    buf.Bytes(),
		Width:        w,
		Height:       h,
		Timestamp:    start,
		DisplayIndex: displayIndex,
	}, nil
}

func (e *WindowsScreenEngine) InjectInput(ev RemoteInputEvent) error {
	sw, sh := robotgo.GetScreenSize()
	posX := int(ev.X * float64(sw))
	posY := int(ev.Y * float64(sh))

	switch ev.Type {
	case "mouse_move":
		robotgo.Move(posX, posY)
	case "mouse_down":
		robotgo.Move(posX, posY)
		btn := "left"
		if ev.Button != "" {
			btn = ev.Button
		}
		robotgo.MouseDown(btn)
	case "mouse_up":
		robotgo.Move(posX, posY)
		btn := "left"
		if ev.Button != "" {
			btn = ev.Button
		}
		robotgo.MouseUp(btn)
	case "mouse_click":
		robotgo.Move(posX, posY)
		btn := "left"
		if ev.Button != "" {
			btn = ev.Button
		}
		robotgo.Click(btn)
	case "double_click":
		robotgo.Move(posX, posY)
		robotgo.Click("left", true)
	case "mouse_scroll":
		if ev.DeltaY != 0 {
			robotgo.Scroll(0, ev.DeltaY)
		}
	case "key_press":
		if ev.Key != "" {
			robotgo.KeyTap(ev.Key)
		}
	}
	return nil
}

func (e *WindowsScreenEngine) Close() error {
	return nil
}

func scaleImageRGBAWin(src *image.RGBA, targetW, targetH int) *image.RGBA {
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
