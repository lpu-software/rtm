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

// Composite real dynamic system mouse cursor (with exact shape, hotspot, and position)
static void DarwinCompositeRealCursor(uint8_t* rgbaPixels, int width, int height) {
    if (!rgbaPixels || width <= 0 || height <= 0) return;

    @autoreleasepool {
        NSPoint mouseLoc = [NSEvent mouseLocation];
        NSScreen* mainScreen = [NSScreen mainScreen];
        if (!mainScreen) return;
        NSRect screenFrame = [mainScreen frame];
        if (screenFrame.size.width <= 0 || screenFrame.size.height <= 0) return;

        NSCursor* cur = [NSCursor currentSystemCursor];
        if (!cur) cur = [NSCursor currentCursor];
        if (!cur) return;

        NSImage* img = [cur image];
        NSPoint hotSpot = [cur hotSpot];
        if (!img) return;

        NSSize imgSize = [img size];
        if (imgSize.width <= 0 || imgSize.height <= 0) return;

        CGFloat scaleX = (CGFloat)width / screenFrame.size.width;
        CGFloat scaleY = (CGFloat)height / screenFrame.size.height;

        int curPixelX = (int)((mouseLoc.x - hotSpot.x) * scaleX);
        int curPixelY = (int)(((screenFrame.size.height - mouseLoc.y) - (imgSize.height - hotSpot.y)) * scaleY);
        int curW = (int)(imgSize.width * scaleX);
        int curH = (int)(imgSize.height * scaleY);

        if (curW <= 0 || curH <= 0) return;

        CGImageRef cgImg = [img CGImageForProposedRect:NULL context:NULL hints:NULL];
        if (!cgImg) return;

        size_t cursorBytesPerRow = curW * 4;
        void* cursorBuf = malloc(curH * cursorBytesPerRow);
        if (!cursorBuf) return;

        CGColorSpaceRef cs = CGColorSpaceCreateDeviceRGB();
        CGContextRef ctx = CGBitmapContextCreate(
            cursorBuf, curW, curH, 8, cursorBytesPerRow, cs,
            kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big
        );
        CGColorSpaceRelease(cs);

        if (ctx) {
            CGContextClearRect(ctx, CGRectMake(0, 0, curW, curH));
            CGContextDrawImage(ctx, CGRectMake(0, 0, curW, curH), cgImg);
            CGContextRelease(ctx);

            uint8_t* cPtr = (uint8_t*)cursorBuf;
            for (int cy = 0; cy < curH; cy++) {
                int py = curPixelY + cy;
                if (py < 0 || py >= height) continue;

                uint8_t* dstRow = rgbaPixels + (py * width * 4);
                uint8_t* srcRow = cPtr + (cy * curW * 4);

                for (int cx = 0; cx < curW; cx++) {
                    int px = curPixelX + cx;
                    if (px < 0 || px >= width) continue;

                    uint8_t srcR = srcRow[cx * 4 + 0];
                    uint8_t srcG = srcRow[cx * 4 + 1];
                    uint8_t srcB = srcRow[cx * 4 + 2];
                    uint8_t srcA = srcRow[cx * 4 + 3];

                    if (srcA == 0) continue;

                    if (srcA == 255) {
                        dstRow[px * 4 + 0] = srcR;
                        dstRow[px * 4 + 1] = srcG;
                        dstRow[px * 4 + 2] = srcB;
                        dstRow[px * 4 + 3] = 255;
                    } else {
                        uint8_t dstR = dstRow[px * 4 + 0];
                        uint8_t dstG = dstRow[px * 4 + 1];
                        uint8_t dstB = dstRow[px * 4 + 2];

                        float a = (float)srcA / 255.0f;
                        float invA = 1.0f - a;

                        dstRow[px * 4 + 0] = (uint8_t)(srcR * a + dstR * invA);
                        dstRow[px * 4 + 1] = (uint8_t)(srcG * a + dstG * invA);
                        dstRow[px * 4 + 2] = (uint8_t)(srcB * a + dstB * invA);
                        dstRow[px * 4 + 3] = 255;
                    }
                }
            }
        }
        free(cursorBuf);
    }
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
        return 0;
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

        SCContentFilter* filter = [[SCContentFilter alloc] initWithDisplay:targetDisplay excludingWindows:@[]];
        SCStreamConfiguration* config = [[SCStreamConfiguration alloc] init];
        config.width = (size_t)targetDisplay.width;
        config.height = (size_t)targetDisplay.height;
        config.showsCursor = YES;
        config.pixelFormat = kCVPixelFormatType_32BGRA;
        config.minimumFrameInterval = CMTimeMake(1, 60);
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

// Fallback CoreGraphics Display Capture
static void* DarwinCaptureCGDisplay(uint32_t displayID, int* outW, int* outH, int* outErr) {
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

    CGRect rect = CGRectMake(0, 0, width, height);
    CGContextClearRect(ctx, rect);
    CGContextDrawImage(ctx, rect, dispImage);
    CGImageRelease(dispImage);
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
		frameQuality: 50,
	}
	engine.refreshDisplays()

	// Start persistent hardware compositor stream for Display 0
	res := C.DarwinStartPersistentSCKStream(C.uint32_t(0))
	if res == 0 {
		engine.streamActive = true
	} else {
		fmt.Printf("[LPU] Notice: SCStream returned code %d, using fallback CoreGraphics engine.\n", res)
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
				// Composite the real system cursor onto the frame
				C.DarwinCompositeRealCursor((*C.uint8_t)(unsafe.Pointer(&rawRGBA.Pix[0])), C.int(rawRGBA.Rect.Dx()), C.int(rawRGBA.Rect.Dy()))
				return e.encodeRGBA(rawRGBA, displayIndex, start)
			}
		}
	}

	// 2. Fallback to CoreGraphics Display Capture + Real Cursor overlay
	var outW, outH, outErr C.int
	buf := C.DarwinCaptureCGDisplay(C.uint32_t(0), &outW, &outH, &outErr)
	if buf != nil && outErr == 0 {
		rawRGBA := cBufferToRGBA(buf, int(outW), int(outH))
		if rawRGBA != nil {
			// Composite the real system cursor onto the frame
			C.DarwinCompositeRealCursor((*C.uint8_t)(unsafe.Pointer(&rawRGBA.Pix[0])), C.int(rawRGBA.Rect.Dx()), C.int(rawRGBA.Rect.Dy()))
			return e.encodeRGBA(rawRGBA, displayIndex, start)
		}
	}

	// 3. Fallback to in-memory screenshot
	bounds := screenshot.GetDisplayBounds(displayIndex)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("display capture failed: %w", err)
	}
	if len(img.Pix) > 0 {
		C.DarwinCompositeRealCursor((*C.uint8_t)(unsafe.Pointer(&img.Pix[0])), C.int(img.Rect.Dx()), C.int(img.Rect.Dy()))
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
