//go:build darwin

package screenaccess

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -Wno-deprecated-declarations -Wno-unguarded-availability
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics -framework ScreenCaptureKit -ldl

#import <Cocoa/Cocoa.h>
#import <CoreGraphics/CoreGraphics.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>
#import <dlfcn.h>

// Ensure Cocoa application initialization
static void DarwinInitAccess() {
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        NSApplicationLoad();
    });
}

// Convert CGImage to raw RGBA buffer
static void* DarwinCGImageToRGBA(CGImageRef imageRef, int* outW, int* outH) {
    if (!imageRef) return NULL;
    size_t width = CGImageGetWidth(imageRef);
    size_t height = CGImageGetHeight(imageRef);
    *outW = (int)width;
    *outH = (int)height;

    size_t bytesPerRow = width * 4;
    void* buffer = malloc(height * bytesPerRow);
    if (!buffer) return NULL;

    CGColorSpaceRef cs = CGColorSpaceCreateDeviceRGB();
    CGContextRef ctx = CGBitmapContextCreate(
        buffer, width, height, 8, bytesPerRow, cs,
        kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big
    );
    CGColorSpaceRelease(cs);

    if (!ctx) {
        free(buffer);
        return NULL;
    }

    CGRect rect = CGRectMake(0, 0, width, height);
    CGContextClearRect(ctx, rect);
    CGContextDrawImage(ctx, rect, imageRef);
    CGContextRelease(ctx);

    return buffer;
}

// ScreenCaptureKit Complete Display Capture (includes real cursor composited by Quartz)
static void* DarwinCaptureSCKDisplay(uint32_t displayID, int* outW, int* outH, int* outErr) {
    *outErr = 0;
    DarwinInitAccess();

    if (@available(macOS 12.3, *)) {
        __block void* resultBuf = NULL;
        __block int resW = 0;
        __block int resH = 0;
        __block int localErr = 0;

        dispatch_semaphore_t sema = dispatch_semaphore_create(0);

        [SCShareableContent getShareableContentWithCompletionHandler:^(SCShareableContent * _Nullable content, NSError * _Nullable error) {
            if (error || !content || content.displays.count == 0) {
                localErr = (int)(error ? [error code] : -1);
                dispatch_semaphore_signal(sema);
                return;
            }

            SCDisplay* targetDisplay = content.displays.firstObject;
            for (SCDisplay* disp in content.displays) {
                if (disp.displayID == displayID) {
                    targetDisplay = disp;
                    break;
                }
            }

            // Exclude nothing — capture the full composited desktop display
            SCContentFilter* filter = [[SCContentFilter alloc] initWithDisplay:targetDisplay excludingWindows:@[]];
            SCStreamConfiguration* config = [[SCStreamConfiguration alloc] init];
            config.width = (size_t)targetDisplay.width;
            config.height = (size_t)targetDisplay.height;
            config.showsCursor = YES; // Embeds real dynamic system hardware cursor (arrow, I-beam, hand, resize)

            if (@available(macOS 13.0, *)) {
                [SCScreenshotManager captureImageWithFilter:filter configuration:config completionHandler:^(CGImageRef  _Nullable img, NSError * _Nullable capErr) {
                    if (capErr || !img) {
                        localErr = (int)(capErr ? [capErr code] : -2);
                    } else {
                        resultBuf = DarwinCGImageToRGBA(img, &resW, &resH);
                    }
                    dispatch_semaphore_signal(sema);
                }];
            } else {
                localErr = -3;
                dispatch_semaphore_signal(sema);
            }
        }];

        dispatch_time_t timeout = dispatch_time(DISPATCH_TIME_NOW, (int64_t)(1.5 * NSEC_PER_SEC));
        long waitRes = dispatch_semaphore_wait(sema, timeout);
        if (waitRes != 0) {
            *outErr = -4; // Timeout
            return NULL;
        }

        *outW = resW;
        *outH = resH;
        *outErr = localErr;
        return resultBuf;
    }

    *outErr = -5;
    return NULL;
}

// Fallback CoreGraphics Display Capture + Real System Cursor Overlay
static void* DarwinCaptureCGDisplayWithCursor(uint32_t displayID, int* outW, int* outH, int* outErr) {
    *outErr = 0;
    DarwinInitAccess();

    typedef CGImageRef (*CGDisplayCreateImage_t)(CGDirectDisplayID displayID);
    static CGDisplayCreateImage_t fnCGDisplayCreateImage = NULL;
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        fnCGDisplayCreateImage = (CGDisplayCreateImage_t)dlsym(RTLD_DEFAULT, "CGDisplayCreateImage");
    });

    CGDirectDisplayID targetID = displayID ? (CGDirectDisplayID)displayID : CGMainDisplayID();
    CGImageRef dispImage = NULL;
    if (fnCGDisplayCreateImage) {
        dispImage = fnCGDisplayCreateImage(targetID);
    }

    if (!dispImage) {
        *outErr = -1;
        return NULL;
    }

    size_t width = CGImageGetWidth(dispImage);
    size_t height = CGImageGetHeight(dispImage);
    *outW = (int)width;
    *outH = (int)height;

    size_t bytesPerRow = width * 4;
    void* buffer = malloc(height * bytesPerRow);
    if (!buffer) {
        CGImageRelease(dispImage);
        return NULL;
    }

    CGColorSpaceRef cs = CGColorSpaceCreateDeviceRGB();
    CGContextRef ctx = CGBitmapContextCreate(
        buffer, width, height, 8, bytesPerRow, cs,
        kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big
    );
    CGColorSpaceRelease(cs);

    if (!ctx) {
        free(buffer);
        CGImageRelease(dispImage);
        return NULL;
    }

    // Draw display
    CGRect rect = CGRectMake(0, 0, width, height);
    CGContextClearRect(ctx, rect);
    CGContextDrawImage(ctx, rect, dispImage);
    CGImageRelease(dispImage);

    // Composite Real System Cursor (arrow, I-beam, pointing hand, resize cursor, etc.)
    NSPoint mouseLoc = [NSEvent mouseLocation];
    NSScreen* mainScreen = [NSScreen mainScreen];
    if (mainScreen) {
        NSRect screenFrame = [mainScreen frame];
        NSCursor* currentCursor = [NSCursor currentSystemCursor];
        if (!currentCursor) {
            currentCursor = [NSCursor currentCursor];
        }

        if (currentCursor) {
            NSImage* cursorImg = [currentCursor image];
            NSPoint hotspot = [currentCursor hotSpot];
            if (cursorImg) {
                // Convert Cocoa bottom-left coordinates to top-down bitmap coordinates
                CGFloat scaleX = (CGFloat)width / screenFrame.size.width;
                CGFloat scaleY = (CGFloat)height / screenFrame.size.height;

                CGFloat curX = (mouseLoc.x - hotspot.x) * scaleX;
                CGFloat curY = ((screenFrame.size.height - mouseLoc.y) - (cursorImg.size.height - hotspot.y)) * scaleY;

                CGImageRef cursorCG = [cursorImg CGImageForProposedRect:NULL context:NULL hints:NULL];
                if (cursorCG) {
                    CGRect cursorRect = CGRectMake(
                        curX,
                        (CGFloat)height - curY - (cursorImg.size.height * scaleY),
                        cursorImg.size.width * scaleX,
                        cursorImg.size.height * scaleY
                    );
                    CGContextDrawImage(ctx, cursorRect, cursorCG);
                }
            }
        }
    }

    CGContextRelease(ctx);
    return buffer;
}

// Mouse and Keyboard Input Injection using CoreGraphics Events
static void DarwinInjectMouseMove(double normX, double normY) {
    DarwinInitAccess();
    NSScreen* mainScreen = [NSScreen mainScreen];
    if (!mainScreen) return;
    NSRect frame = [mainScreen frame];

    CGPoint pt = CGPointMake(normX * frame.size.width, normY * frame.size.height);
    CGEventRef moveEv = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, pt, kCGMouseButtonLeft);
    if (moveEv) {
        CGEventPost(kCGHIDEventTap, moveEv);
        CFRelease(moveEv);
    }
}

static void DarwinInjectMouseButton(const char* button, const char* state, double normX, double normY) {
    DarwinInitAccess();
    NSScreen* mainScreen = [NSScreen mainScreen];
    if (!mainScreen) return;
    NSRect frame = [mainScreen frame];

    CGPoint pt = CGPointMake(normX * frame.size.width, normY * frame.size.height);
    CGMouseButton btn = kCGMouseButtonLeft;
    CGEventType downType = kCGEventLeftMouseDown;
    CGEventType upType = kCGEventLeftMouseUp;

    if (strcmp(button, "right") == 0) {
        btn = kCGMouseButtonRight;
        downType = kCGEventRightMouseDown;
        upType = kCGEventRightMouseUp;
    } else if (strcmp(button, "middle") == 0) {
        btn = kCGMouseButtonCenter;
        downType = kCGEventOtherMouseDown;
        upType = kCGEventOtherMouseUp;
    }

    if (strcmp(state, "down") == 0) {
        CGEventRef ev = CGEventCreateMouseEvent(NULL, downType, pt, btn);
        if (ev) { CGEventPost(kCGHIDEventTap, ev); CFRelease(ev); }
    } else if (strcmp(state, "up") == 0) {
        CGEventRef ev = CGEventCreateMouseEvent(NULL, upType, pt, btn);
        if (ev) { CGEventPost(kCGHIDEventTap, ev); CFRelease(ev); }
    } else if (strcmp(state, "click") == 0) {
        CGEventRef down = CGEventCreateMouseEvent(NULL, downType, pt, btn);
        CGEventRef up = CGEventCreateMouseEvent(NULL, upType, pt, btn);
        if (down) { CGEventPost(kCGHIDEventTap, down); CFRelease(down); }
        if (up) { CGEventPost(kCGHIDEventTap, up); CFRelease(up); }
    } else if (strcmp(state, "double_click") == 0) {
        CGEventRef down1 = CGEventCreateMouseEvent(NULL, downType, pt, btn);
        CGEventRef up1 = CGEventCreateMouseEvent(NULL, upType, pt, btn);
        CGEventRef down2 = CGEventCreateMouseEvent(NULL, downType, pt, btn);
        CGEventRef up2 = CGEventCreateMouseEvent(NULL, upType, pt, btn);
        if (down1) { CGEventSetIntegerValueField(down1, kCGMouseEventClickState, 1); CGEventPost(kCGHIDEventTap, down1); CFRelease(down1); }
        if (up1) { CGEventSetIntegerValueField(up1, kCGMouseEventClickState, 1); CGEventPost(kCGHIDEventTap, up1); CFRelease(up1); }
        if (down2) { CGEventSetIntegerValueField(down2, kCGMouseEventClickState, 2); CGEventPost(kCGHIDEventTap, down2); CFRelease(down2); }
        if (up2) { CGEventSetIntegerValueField(up2, kCGMouseEventClickState, 2); CGEventPost(kCGHIDEventTap, up2); CFRelease(up2); }
    }
}

static void DarwinInjectScroll(int deltaX, int deltaY) {
    DarwinInitAccess();
    CGEventRef scroll = CGEventCreateScrollWheelEvent2(NULL, kCGScrollEventUnitPixel, 2, deltaY, deltaX, 0);
    if (scroll) {
        CGEventPost(kCGHIDEventTap, scroll);
        CFRelease(scroll);
    }
}

*/
import "C"
import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"sync"
	"time"
	"unsafe"

	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
)

// DarwinScreenEngine implements complete display access & input on macOS.
type DarwinScreenEngine struct {
	mu           sync.RWMutex
	displays     []DisplayInfo
	lastFrame    *FrameData
	frameQuality int
}

// NewDarwinScreenEngine initializes the macOS display capture and input engine.
func NewDarwinScreenEngine() (*DarwinScreenEngine, error) {
	engine := &DarwinScreenEngine{
		frameQuality: 45,
	}
	engine.refreshDisplays()
	return engine, nil
}

func (e *DarwinScreenEngine) refreshDisplays() {
	e.mu.Lock()
	defer e.mu.Unlock()

	numDisplays := screenshot.NumActiveDisplays()
	e.displays = make([]DisplayInfo, 0, numDisplays)

	for i := 0; i < numDisplays; i++ {
		b := screenshot.GetDisplayBounds(i)
		e.displays = append(e.displays, DisplayInfo{
			Index:       i,
			Bounds:      b,
			ScaleFactor: 2.0, // Retina default
			IsMain:      i == 0,
			Width:       b.Dx(),
			Height:      b.Dy(),
		})
	}
}

func (e *DarwinScreenEngine) GetDisplays() []DisplayInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.displays
}

func (e *DarwinScreenEngine) CaptureDisplay(displayIndex int) (*FrameData, error) {
	start := time.Now()

	// 1. Try ScreenCaptureKit display capture (with hardware cursor composited)
	var outW, outH, outErr C.int
	buf := C.DarwinCaptureSCKDisplay(C.uint32_t(0), &outW, &outH, &outErr)

	// 2. Fallback to CoreGraphics Display Capture + Real Cursor overlay
	if buf == nil || outErr != 0 {
		buf = C.DarwinCaptureCGDisplayWithCursor(C.uint32_t(0), &outW, &outH, &outErr)
	}

	// 3. Fallback to in-memory screenshot
	if buf == nil || outErr != 0 {
		bounds := screenshot.GetDisplayBounds(displayIndex)
		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			return nil, fmt.Errorf("display capture failed: %w", err)
		}
		return e.encodeRGBA(img, displayIndex, start)
	}

	rawRGBA := cBufferToRGBA(buf, int(outW), int(outH))
	if rawRGBA == nil {
		return nil, fmt.Errorf("failed to convert captured buffer to RGBA")
	}

	return e.encodeRGBA(rawRGBA, displayIndex, start)
}

func (e *DarwinScreenEngine) encodeRGBA(img *image.RGBA, displayIndex int, start time.Time) (*FrameData, error) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	// High-DPI / Retina downscaling to 1080p for silky smooth 30-60 FPS over WebRTC
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

func (e *DarwinScreenEngine) InjectInput(ev RemoteInputEvent) error {
	switch ev.Type {
	case "mouse_move":
		C.DarwinInjectMouseMove(C.double(ev.X), C.double(ev.Y))
	case "mouse_down":
		btn := C.CString(ev.Button)
		defer C.free(unsafe.Pointer(btn))
		state := C.CString("down")
		defer C.free(unsafe.Pointer(state))
		C.DarwinInjectMouseButton(btn, state, C.double(ev.X), C.double(ev.Y))
	case "mouse_up":
		btn := C.CString(ev.Button)
		defer C.free(unsafe.Pointer(btn))
		state := C.CString("up")
		defer C.free(unsafe.Pointer(state))
		C.DarwinInjectMouseButton(btn, state, C.double(ev.X), C.double(ev.Y))
	case "mouse_click":
		btn := C.CString(ev.Button)
		defer C.free(unsafe.Pointer(btn))
		state := C.CString("click")
		defer C.free(unsafe.Pointer(state))
		C.DarwinInjectMouseButton(btn, state, C.double(ev.X), C.double(ev.Y))
	case "double_click":
		btn := C.CString(ev.Button)
		defer C.free(unsafe.Pointer(btn))
		state := C.CString("double_click")
		defer C.free(unsafe.Pointer(state))
		C.DarwinInjectMouseButton(btn, state, C.double(ev.X), C.double(ev.Y))
	case "mouse_scroll":
		C.DarwinInjectScroll(C.int(ev.DeltaX), C.int(ev.DeltaY))
	case "key_press":
		if ev.Key != "" {
			robotgo.KeyTap(ev.Key)
		}
	}
	return nil
}

func (e *DarwinScreenEngine) Close() error {
	return nil
}

func cBufferToRGBA(buf unsafe.Pointer, width, height int) *image.RGBA {
	if buf == nil || width <= 0 || height <= 0 {
		return nil
	}
	defer C.free(buf)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	srcSlice := (*[1 << 30]byte)(buf)[: width*height*4 : width*height*4]
	copy(img.Pix, srcSlice)
	return img
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
