//go:build darwin

package screenaccess

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -Wno-deprecated-declarations -Wno-unguarded-availability
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics -framework CoreMedia -framework CoreVideo -framework ScreenCaptureKit -framework IOSurface -ldl

#import <Cocoa/Cocoa.h>
#import <CoreGraphics/CoreGraphics.h>
#import <CoreMedia/CoreMedia.h>
#import <CoreVideo/CoreVideo.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>
#import <dlfcn.h>
#import <pthread.h>

// Thread-safe double buffer for latest composited GPU frame
typedef struct {
    uint8_t* buffer;
    size_t bufferSize;
    int width;
    int height;
    int bytesPerRow;
    uint64_t frameIndex;
    pthread_mutex_t mutex;
} DarwinFrameStore;

static DarwinFrameStore s_frameStore;
static pthread_once_t s_frameStoreInitOnce = PTHREAD_ONCE_INIT;

static void DarwinInitFrameStoreOnce() {
    pthread_mutex_init(&s_frameStore.mutex, NULL);
    s_frameStore.buffer = NULL;
    s_frameStore.bufferSize = 0;
    s_frameStore.width = 0;
    s_frameStore.height = 0;
    s_frameStore.bytesPerRow = 0;
    s_frameStore.frameIndex = 0;
}

static void DarwinInitAccess() {
    pthread_once(&s_frameStoreInitOnce, DarwinInitFrameStoreOnce);
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        NSApplicationLoad();
    });
}

// Store a newly rendered Metal/GPU frame from ScreenCaptureKit
static void DarwinStoreSCKFrame(const void* src, int width, int height, int bytesPerRow) {
    pthread_mutex_lock(&s_frameStore.mutex);
    size_t requiredSize = (size_t)height * (size_t)bytesPerRow;
    if (s_frameStore.bufferSize < requiredSize) {
        if (s_frameStore.buffer) free(s_frameStore.buffer);
        s_frameStore.buffer = (uint8_t*)malloc(requiredSize);
        s_frameStore.bufferSize = requiredSize;
    }
    if (s_frameStore.buffer && src) {
        // Fast memcpy of complete GPU frame
        memcpy(s_frameStore.buffer, src, requiredSize);
        s_frameStore.width = width;
        s_frameStore.height = height;
        s_frameStore.bytesPerRow = bytesPerRow;
        s_frameStore.frameIndex++;
    }
    pthread_mutex_unlock(&s_frameStore.mutex);
}

// Retrieve a copy of the latest composited frame
static void* DarwinGetLatestSCKFrame(int* outW, int* outH, int* outBytesPerRow, uint64_t* outFrameIdx) {
    pthread_mutex_lock(&s_frameStore.mutex);
    if (!s_frameStore.buffer || s_frameStore.width <= 0 || s_frameStore.height <= 0) {
        pthread_mutex_unlock(&s_frameStore.mutex);
        return NULL;
    }
    *outW = s_frameStore.width;
    *outH = s_frameStore.height;
    *outBytesPerRow = s_frameStore.bytesPerRow;
    *outFrameIdx = s_frameStore.frameIndex;

    size_t size = (size_t)s_frameStore.height * (size_t)s_frameStore.bytesPerRow;
    void* copyBuf = malloc(size);
    if (copyBuf) {
        memcpy(copyBuf, s_frameStore.buffer, size);
    }
    pthread_mutex_unlock(&s_frameStore.mutex);
    return copyBuf;
}

// ScreenCaptureKit Stream Output Delegate
@interface DarwinSCStreamOutput : NSObject <SCStreamOutput, SCStreamDelegate>
@property (nonatomic, assign) int frameCount;
@end

@implementation DarwinSCStreamOutput

- (void)stream:(SCStream *)stream didOutputSampleBuffer:(CMSampleBufferRef)sampleBuffer ofType:(SCStreamOutputType)type {
    if (type != SCStreamOutputTypeScreen) return;
    
    CVImageBufferRef imageBuffer = CMSampleBufferGetImageBuffer(sampleBuffer);
    if (!imageBuffer) return;

    if (CVPixelBufferLockBaseAddress(imageBuffer, kCVPixelBufferLock_ReadOnly) == kCVReturnSuccess) {
        void *baseAddress = CVPixelBufferGetBaseAddress(imageBuffer);
        size_t width = CVPixelBufferGetWidth(imageBuffer);
        size_t height = CVPixelBufferGetHeight(imageBuffer);
        size_t bytesPerRow = CVPixelBufferGetBytesPerRow(imageBuffer);

        DarwinStoreSCKFrame(baseAddress, (int)width, (int)height, (int)bytesPerRow);
        self.frameCount++;

        CVPixelBufferUnlockBaseAddress(imageBuffer, kCVPixelBufferLock_ReadOnly);
    }
}

- (void)stream:(SCStream *)stream didStopWithError:(NSError *)error {
    NSLog(@"[LPU ScreenEngine] SCStream stopped with error: %@", error);
}

@end

static SCStream* s_activeStream = nil;
static DarwinSCStreamOutput* s_streamOutput = nil;

// Start persistent ScreenCaptureKit display stream
static int DarwinStartPersistentSCKStream(uint32_t displayID) {
    DarwinInitAccess();
    if (!@available(macOS 12.3, *)) {
        return -1;
    }

    if (s_activeStream) {
        return 0; // Already running
    }

    dispatch_semaphore_t sema = dispatch_semaphore_create(0);
    __block int result = 0;

    [SCShareableContent getShareableContentWithCompletionHandler:^(SCShareableContent * _Nullable content, NSError * _Nullable error) {
        if (error || !content || content.displays.count == 0) {
            result = -2;
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

        // CRITICAL: Exclude nothing — capture the full composited desktop display with ALL foreground windows
        SCContentFilter* filter = [[SCContentFilter alloc] initWithDisplay:targetDisplay excludingWindows:@[]];
        SCStreamConfiguration* config = [[SCStreamConfiguration alloc] init];
        config.width = (size_t)targetDisplay.width;
        config.height = (size_t)targetDisplay.height;
        config.showsCursor = YES; // Embeds real dynamic system hardware cursor (arrow, I-beam, hand, resize)
        config.pixelFormat = kCVPixelFormatType_32BGRA;
        config.minimumFrameInterval = CMTimeMake(1, 60); // 60 FPS smooth capture
        config.queueDepth = 5;

        s_streamOutput = [[DarwinSCStreamOutput alloc] init];
        s_activeStream = [[SCStream alloc] initWithFilter:filter configuration:config delegate:s_streamOutput];

        dispatch_queue_t queue = dispatch_queue_create("com.lpu.scstream.worker", DISPATCH_QUEUE_SERIAL);
        NSError* addOutputErr = nil;
        [s_activeStream addStreamOutput:s_streamOutput type:SCStreamOutputTypeScreen sampleHandlerQueue:queue error:&addOutputErr];

        if (addOutputErr) {
            result = -3;
            dispatch_semaphore_signal(sema);
            return;
        }

        [s_activeStream startCaptureWithCompletionHandler:^(NSError * _Nullable startErr) {
            if (startErr) {
                result = -4;
            } else {
                result = 0;
            }
            dispatch_semaphore_signal(sema);
        }];
    }];

    dispatch_semaphore_wait(sema, dispatch_time(DISPATCH_TIME_NOW, 4 * NSEC_PER_SEC));
    return result;
}

static void DarwinStopPersistentSCKStream() {
    if (s_activeStream) {
        if (@available(macOS 12.3, *)) {
            dispatch_semaphore_t sema = dispatch_semaphore_create(0);
            [s_activeStream stopCaptureWithCompletionHandler:^(NSError * _Nullable error) {
                dispatch_semaphore_signal(sema);
            }];
            dispatch_semaphore_wait(sema, dispatch_time(DISPATCH_TIME_NOW, 1 * NSEC_PER_SEC));
        }
        s_activeStream = nil;
        s_streamOutput = nil;
    }
}

// Convert CGImage to raw RGBA buffer (Fallback)
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

// Fallback CoreGraphics Display Capture + Real System Cursor Overlay
static void* DarwinCaptureCGDisplayWithCursor(uint32_t displayID, int* outW, int* outH, int* outErr) {
    *outErr = 0;
    DarwinInitAccess();

    typedef CGImageRef (*CGDisplayCreateImage_t)(CGDirectDisplayID displayID);
    typedef CGImageRef (*CGWindowListCreateImage_t)(CGRect screenBounds, uint32_t listOptions, uint32_t windowID, uint32_t imageOptions);
    static CGDisplayCreateImage_t fnCGDisplayCreateImage = NULL;
    static CGWindowListCreateImage_t fnCGWindowListCreateImage = NULL;
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        fnCGDisplayCreateImage = (CGDisplayCreateImage_t)dlsym(RTLD_DEFAULT, "CGDisplayCreateImage");
        fnCGWindowListCreateImage = (CGWindowListCreateImage_t)dlsym(RTLD_DEFAULT, "CGWindowListCreateImage");
    });

    CGDirectDisplayID targetID = displayID ? (CGDirectDisplayID)displayID : CGMainDisplayID();
    CGImageRef dispImage = NULL;
    if (fnCGDisplayCreateImage) {
        dispImage = fnCGDisplayCreateImage(targetID);
    }

    if (!dispImage && fnCGWindowListCreateImage) {
        // Fallback to WindowList with ALL options
        CGRect bounds = CGDisplayBounds(targetID);
        dispImage = fnCGWindowListCreateImage(bounds, 0, 0, 0);
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

// DarwinScreenEngine implements complete display access & input on macOS using persistent ScreenCaptureKit.
type DarwinScreenEngine struct {
	mu           sync.RWMutex
	displays     []DisplayInfo
	frameQuality int
	streamActive bool
}

// NewDarwinScreenEngine initializes the macOS display capture and input engine.
func NewDarwinScreenEngine() (*DarwinScreenEngine, error) {
	engine := &DarwinScreenEngine{
		frameQuality: 45,
	}
	engine.refreshDisplays()

	// Start persistent hardware compositor stream for Display 0
	res := C.DarwinStartPersistentSCKStream(C.uint32_t(0))
	if res == 0 {
		engine.streamActive = true
	} else {
		fmt.Printf("[LPU] Warning: SCStream returned code %d, using fallback capture.\n", res)
	}

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

	// 1. Try retrieving the latest GPU composited frame from persistent SCStream
	if e.streamActive {
		var outW, outH, outStride C.int
		var frameIdx C.uint64_t
		buf := C.DarwinGetLatestSCKFrame(&outW, &outH, &outStride, &frameIdx)
		if buf != nil {
			rawRGBA := bgraBufferToRGBA(buf, int(outW), int(outH), int(outStride))
			if rawRGBA != nil {
				return e.encodeRGBA(rawRGBA, displayIndex, start)
			}
		}
	}

	// 2. Fallback to CoreGraphics Display Capture + Real Cursor overlay
	var outW, outH, outErr C.int
	buf := C.DarwinCaptureCGDisplayWithCursor(C.uint32_t(0), &outW, &outH, &outErr)
	if buf != nil && outErr == 0 {
		rawRGBA := cBufferToRGBA(buf, int(outW), int(outH))
		if rawRGBA != nil {
			return e.encodeRGBA(rawRGBA, displayIndex, start)
		}
	}

	// 3. Fallback to in-memory screenshot
	bounds := screenshot.GetDisplayBounds(displayIndex)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("display capture failed: %w", err)
	}
	return e.encodeRGBA(img, displayIndex, start)
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
	if e.streamActive {
		C.DarwinStopPersistentSCKStream()
		e.streamActive = false
	}
	return nil
}

// Convert BGRA GPU buffer to RGBA Go image
func bgraBufferToRGBA(buf unsafe.Pointer, width, height, bytesPerRow int) *image.RGBA {
	if buf == nil || width <= 0 || height <= 0 {
		return nil
	}
	defer C.free(buf)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	srcSlice := (*[1 << 30]byte)(buf)[: height*bytesPerRow : height*bytesPerRow]

	for y := 0; y < height; y++ {
		srcRow := srcSlice[y*bytesPerRow : (y+1)*bytesPerRow]
		dstRow := img.Pix[y*img.Stride : (y+1)*img.Stride]
		for x := 0; x < width; x++ {
			b := srcRow[x*4+0]
			g := srcRow[x*4+1]
			r := srcRow[x*4+2]
			a := srcRow[x*4+3]
			dstRow[x*4+0] = r
			dstRow[x*4+1] = g
			dstRow[x*4+2] = b
			dstRow[x*4+3] = a
		}
	}
	return img
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
