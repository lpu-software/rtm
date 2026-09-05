//go:build darwin

package probes

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -Wno-deprecated-declarations -Wno-unguarded-availability
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics -framework ScreenCaptureKit -ldl

#import <Cocoa/Cocoa.h>
#import <CoreGraphics/CoreGraphics.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>
#import <dlfcn.h>

typedef CGImageRef (*CGWindowListCreateImage_t)(CGRect screenBounds, CGWindowListOption listOptions, CGWindowID windowID, CGWindowImageOption imageOptions);
typedef CGImageRef (*CGDisplayCreateImage_t)(CGDirectDisplayID displayID);

// Initialize Cocoa and CoreGraphics runtime safely in CLI/daemon context
static void DarwinInitCoreGraphics() {
    static dispatch_once_t initToken;
    dispatch_once(&initToken, ^{
        NSApplicationLoad();
    });
}

// Preflight screen capture permissions
static int DarwinCheckScreenCapturePermission() {
    DarwinInitCoreGraphics();
    if (@available(macOS 10.15, *)) {
        return CGPreflightScreenCaptureAccess() ? 1 : 0;
    }
    return 1;
}

// Request permission prompt
static void DarwinRequestScreenCapturePermission() {
    DarwinInitCoreGraphics();
    if (@available(macOS 10.15, *)) {
        CGRequestScreenCaptureAccess();
    }
}

// Helper to convert CGImageRef to RGBA byte buffer
static void* CopyCGImageToRGBA(CGImageRef imageRef, int* outWidth, int* outHeight) {
    if (!imageRef) return NULL;

    size_t width = CGImageGetWidth(imageRef);
    size_t height = CGImageGetHeight(imageRef);
    *outWidth = (int)width;
    *outHeight = (int)height;

    size_t bytesPerRow = width * 4;
    void* buffer = malloc(height * bytesPerRow);
    if (!buffer) return NULL;

    CGColorSpaceRef colorSpace = CGColorSpaceCreateDeviceRGB();
    CGContextRef context = CGBitmapContextCreate(
        buffer,
        width,
        height,
        8,
        bytesPerRow,
        colorSpace,
        kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big
    );
    CGColorSpaceRelease(colorSpace);

    if (!context) {
        free(buffer);
        return NULL;
    }

    CGRect rect = CGRectMake(0, 0, width, height);
    CGContextClearRect(context, rect);
    CGContextDrawImage(context, rect, imageRef);
    CGContextRelease(context);

    return buffer;
}

// CoreGraphics Window capture: dynamic lookup for legacy compatibility
static void* DarwinCaptureWindowCG(uint32_t windowID, int* outWidth, int* outHeight, int* outErr) {
    *outErr = 0;
    DarwinInitCoreGraphics();
    static CGWindowListCreateImage_t fnCGWindowListCreateImage = NULL;
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        fnCGWindowListCreateImage = (CGWindowListCreateImage_t)dlsym(RTLD_DEFAULT, "CGWindowListCreateImage");
    });

    if (!fnCGWindowListCreateImage) {
        *outErr = -10; // Symbol not available in OS
        return NULL;
    }

    CGWindowImageOption imageOptions = kCGWindowImageBoundsIgnoreFraming | kCGWindowImageNominalResolution;
    CGImageRef imageRef = fnCGWindowListCreateImage(
        CGRectNull,
        kCGWindowListOptionIncludingWindow,
        (CGWindowID)windowID,
        imageOptions
    );

    if (!imageRef) {
        *outErr = -1; // Window capture unavailable or window ID invalid
        return NULL;
    }

    void* rgba = CopyCGImageToRGBA(imageRef, outWidth, outHeight);
    CGImageRelease(imageRef);
    return rgba;
}

// CoreGraphics Display capture: dynamic lookup for legacy compatibility
static void* DarwinCaptureDisplayCG(int* outWidth, int* outHeight, int* outErr) {
    *outErr = 0;
    DarwinInitCoreGraphics();
    static CGDisplayCreateImage_t fnCGDisplayCreateImage = NULL;
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        fnCGDisplayCreateImage = (CGDisplayCreateImage_t)dlsym(RTLD_DEFAULT, "CGDisplayCreateImage");
    });

    if (!fnCGDisplayCreateImage) {
        *outErr = -11; // Symbol not available in OS
        return NULL;
    }

    CGDirectDisplayID mainDisplay = CGMainDisplayID();
    CGImageRef imageRef = fnCGDisplayCreateImage(mainDisplay);

    if (!imageRef) {
        *outErr = -2; // Display capture failed / Permission denied
        return NULL;
    }

    void* rgba = CopyCGImageToRGBA(imageRef, outWidth, outHeight);
    CGImageRelease(imageRef);
    return rgba;
}

// CoreGraphics Region capture
static void* DarwinCaptureRegionCG(int x, int y, int w, int h, int* outWidth, int* outHeight, int* outErr) {
    *outErr = 0;
    DarwinInitCoreGraphics();
    static CGWindowListCreateImage_t fnCGWindowListCreateImage = NULL;
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        fnCGWindowListCreateImage = (CGWindowListCreateImage_t)dlsym(RTLD_DEFAULT, "CGWindowListCreateImage");
    });

    if (!fnCGWindowListCreateImage) {
        *outErr = -10;
        return NULL;
    }

    CGRect captureRect = CGRectMake(x, y, w, h);
    CGImageRef imageRef = fnCGWindowListCreateImage(
        captureRect,
        kCGWindowListOptionOnScreenOnly,
        kCGNullWindowID,
        kCGWindowImageDefault
    );

    if (!imageRef) {
        *outErr = -3;
        return NULL;
    }

    void* rgba = CopyCGImageToRGBA(imageRef, outWidth, outHeight);
    CGImageRelease(imageRef);
    return rgba;
}

// ScreenCaptureKit Window Capture
static void* DarwinCaptureSCKWindow(uint32_t windowID, int* outWidth, int* outHeight, int* outErr) {
    *outErr = 0;
    if (@available(macOS 12.3, *)) {
        __block void* resultBuffer = NULL;
        __block int resWidth = 0;
        __block int resHeight = 0;
        __block int localErr = 0;

        dispatch_semaphore_t sema = dispatch_semaphore_create(0);

        [SCShareableContent getShareableContentWithCompletionHandler:^(SCShareableContent * _Nullable shareableContent, NSError * _Nullable error) {
            if (error || !shareableContent) {
                localErr = (int)[error code];
                dispatch_semaphore_signal(sema);
                return;
            }

            SCWindow* targetWin = nil;
            for (SCWindow* win in shareableContent.windows) {
                if (win.windowID == windowID) {
                    targetWin = win;
                    break;
                }
            }

            if (!targetWin && shareableContent.windows.count > 0) {
                targetWin = shareableContent.windows.firstObject;
            }

            if (!targetWin) {
                localErr = -4; // Target window not in shareable list
                dispatch_semaphore_signal(sema);
                return;
            }

            SCContentFilter* filter = [[SCContentFilter alloc] initWithDesktopIndependentWindow:targetWin];
            SCStreamConfiguration* config = [[SCStreamConfiguration alloc] init];
            config.width = (size_t)targetWin.frame.size.width;
            config.height = (size_t)targetWin.frame.size.height;
            config.showsCursor = NO;

            if (@available(macOS 13.0, *)) {
                [SCScreenshotManager captureImageWithFilter:filter configuration:config completionHandler:^(CGImageRef  _Nullable userImage, NSError * _Nullable captureError) {
                    if (captureError || !userImage) {
                        localErr = (int)[captureError code];
                    } else {
                        resultBuffer = CopyCGImageToRGBA(userImage, &resWidth, &resHeight);
                    }
                    dispatch_semaphore_signal(sema);
                }];
            } else {
                localErr = -5; // SCScreenshotManager requires macOS 13.0+
                dispatch_semaphore_signal(sema);
            }
        }];

        dispatch_time_t timeout = dispatch_time(DISPATCH_TIME_NOW, (int64_t)(3.0 * NSEC_PER_SEC));
        long waitRes = dispatch_semaphore_wait(sema, timeout);
        if (waitRes != 0) {
            *outErr = -6; // SCK capture timeout
            return NULL;
        }

        *outWidth = resWidth;
        *outHeight = resHeight;
        *outErr = localErr;
        return resultBuffer;
    } else {
        *outErr = -7; // SCK not supported on macOS < 12.3
        return NULL;
    }
}
*/
import "C"
import (
	"fmt"
	"image"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

func init() {
	C.DarwinInitCoreGraphics()
}

// Helper to convert C RGBA buffer to Go *image.RGBA
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

// ================= ScreenCaptureKit Probe =================

type DarwinSCKProbe struct{}

func NewDarwinSCKProbe() *DarwinSCKProbe {
	return &DarwinSCKProbe{}
}

func (p *DarwinSCKProbe) Name() string {
	return "macOS ScreenCaptureKit"
}

func (p *DarwinSCKProbe) Method() CaptureMethod {
	return MethodScreenCaptureKit
}

func (p *DarwinSCKProbe) Platform() string {
	return "darwin"
}

func (p *DarwinSCKProbe) IsAvailable() bool {
	return true
}

func (p *DarwinSCKProbe) CheckPermission() (bool, string) {
	perm := int(C.DarwinCheckScreenCapturePermission())
	if perm == 1 {
		return true, "Screen Recording Permission Authorized"
	}
	return false, "Screen Recording Permission Denied / Not Determined"
}

func (p *DarwinSCKProbe) Capture(req CaptureRequest) (*CaptureResponse, error) {
	start := time.Now()
	permGranted, permDesc := p.CheckPermission()

	var winID uint32
	if strings.HasPrefix(req.TargetWindowID, "CGWindowID:") {
		idStr := strings.TrimPrefix(req.TargetWindowID, "CGWindowID:")
		if val, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			winID = uint32(val)
		}
	}

	var outW, outH, outErr C.int
	buf := C.DarwinCaptureSCKWindow(C.uint32_t(winID), &outW, &outH, &outErr)
	duration := time.Since(start)

	resp := &CaptureResponse{
		Method:            MethodScreenCaptureKit,
		Scope:             req.Scope,
		TargetWindowID:    req.TargetWindowID,
		CaptureDuration:   duration,
		Timestamp:         start,
		PermissionGranted: permGranted,
		IsDeterministic:   true,
	}

	if outErr != 0 || buf == nil {
		resp.ErrorCode = int(outErr)
		resp.ErrorMessage = fmt.Sprintf("ScreenCaptureKit error code: %d (%s)", outErr, permDesc)
		if !permGranted {
			resp.ErrorMessage = "Screen Recording permission denied by macOS TCC"
		}
		return resp, nil
	}

	resp.CapturedImage = cBufferToRGBA(buf, int(outW), int(outH))
	resp.Width = int(outW)
	resp.Height = int(outH)
	return resp, nil
}

// ================= CoreGraphics Window Capture Probe =================

type DarwinCGWindowProbe struct{}

func NewDarwinCGWindowProbe() *DarwinCGWindowProbe {
	return &DarwinCGWindowProbe{}
}

func (p *DarwinCGWindowProbe) Name() string {
	return "macOS CoreGraphics Window"
}

func (p *DarwinCGWindowProbe) Method() CaptureMethod {
	return MethodCoreGraphicsWindow
}

func (p *DarwinCGWindowProbe) Platform() string {
	return "darwin"
}

func (p *DarwinCGWindowProbe) IsAvailable() bool {
	return true
}

func (p *DarwinCGWindowProbe) CheckPermission() (bool, string) {
	perm := int(C.DarwinCheckScreenCapturePermission())
	if perm == 1 {
		return true, "Screen Recording Permission Authorized"
	}
	return false, "Screen Recording Permission Denied"
}

func (p *DarwinCGWindowProbe) Capture(req CaptureRequest) (*CaptureResponse, error) {
	start := time.Now()
	permGranted, _ := p.CheckPermission()

	var winID uint32
	if strings.HasPrefix(req.TargetWindowID, "CGWindowID:") {
		idStr := strings.TrimPrefix(req.TargetWindowID, "CGWindowID:")
		if val, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			winID = uint32(val)
		}
	}

	var outW, outH, outErr C.int
	buf := C.DarwinCaptureWindowCG(C.uint32_t(winID), &outW, &outH, &outErr)
	duration := time.Since(start)

	resp := &CaptureResponse{
		Method:            MethodCoreGraphicsWindow,
		Scope:             ScopeWindow,
		TargetWindowID:    req.TargetWindowID,
		CaptureDuration:   duration,
		Timestamp:         start,
		PermissionGranted: permGranted,
		IsDeterministic:   true,
	}

	if outErr != 0 || buf == nil {
		resp.ErrorCode = int(outErr)
		resp.ErrorMessage = fmt.Sprintf("CGWindowListCreateImage failed (error %d)", outErr)
		return resp, nil
	}

	resp.CapturedImage = cBufferToRGBA(buf, int(outW), int(outH))
	resp.Width = int(outW)
	resp.Height = int(outH)
	return resp, nil
}

// ================= CoreGraphics Display Capture Probe =================

type DarwinCGDisplayProbe struct{}

func NewDarwinCGDisplayProbe() *DarwinCGDisplayProbe {
	return &DarwinCGDisplayProbe{}
}

func (p *DarwinCGDisplayProbe) Name() string {
	return "macOS CoreGraphics Display"
}

func (p *DarwinCGDisplayProbe) Method() CaptureMethod {
	return MethodCoreGraphicsDisplay
}

func (p *DarwinCGDisplayProbe) Platform() string {
	return "darwin"
}

func (p *DarwinCGDisplayProbe) IsAvailable() bool {
	return true
}

func (p *DarwinCGDisplayProbe) CheckPermission() (bool, string) {
	perm := int(C.DarwinCheckScreenCapturePermission())
	if perm == 1 {
		return true, "Screen Recording Permission Authorized"
	}
	return false, "Screen Recording Permission Denied"
}

func (p *DarwinCGDisplayProbe) Capture(req CaptureRequest) (*CaptureResponse, error) {
	start := time.Now()
	permGranted, _ := p.CheckPermission()

	var outW, outH, outErr C.int
	buf := C.DarwinCaptureDisplayCG(&outW, &outH, &outErr)
	duration := time.Since(start)

	resp := &CaptureResponse{
		Method:            MethodCoreGraphicsDisplay,
		Scope:             ScopeFullScreen,
		TargetWindowID:    req.TargetWindowID,
		CaptureDuration:   duration,
		Timestamp:         start,
		PermissionGranted: permGranted,
		IsDeterministic:   true,
	}

	if outErr != 0 || buf == nil {
		resp.ErrorCode = int(outErr)
		resp.ErrorMessage = "CGDisplayCreateImage failed"
		return resp, nil
	}

	resp.CapturedImage = cBufferToRGBA(buf, int(outW), int(outH))
	resp.Width = int(outW)
	resp.Height = int(outH)
	return resp, nil
}

// ================= CoreGraphics Region Capture Probe =================

type DarwinCGRegionProbe struct{}

func NewDarwinCGRegionProbe() *DarwinCGRegionProbe {
	return &DarwinCGRegionProbe{}
}

func (p *DarwinCGRegionProbe) Name() string {
	return "macOS CoreGraphics Region"
}

func (p *DarwinCGRegionProbe) Method() CaptureMethod {
	return MethodCoreGraphicsRegion
}

func (p *DarwinCGRegionProbe) Platform() string {
	return "darwin"
}

func (p *DarwinCGRegionProbe) IsAvailable() bool {
	return true
}

func (p *DarwinCGRegionProbe) CheckPermission() (bool, string) {
	perm := int(C.DarwinCheckScreenCapturePermission())
	if perm == 1 {
		return true, "Screen Recording Permission Authorized"
	}
	return false, "Screen Recording Permission Denied"
}

func (p *DarwinCGRegionProbe) Capture(req CaptureRequest) (*CaptureResponse, error) {
	start := time.Now()
	permGranted, _ := p.CheckPermission()

	x := req.TargetBounds.Min.X
	y := req.TargetBounds.Min.Y
	w := req.TargetBounds.Dx()
	h := req.TargetBounds.Dy()

	if w <= 0 || h <= 0 {
		w = 800
		h = 600
	}

	var outW, outH, outErr C.int
	buf := C.DarwinCaptureRegionCG(C.int(x), C.int(y), C.int(w), C.int(h), &outW, &outH, &outErr)
	duration := time.Since(start)

	resp := &CaptureResponse{
		Method:            MethodCoreGraphicsRegion,
		Scope:             ScopeRegion,
		TargetWindowID:    req.TargetWindowID,
		CaptureDuration:   duration,
		Timestamp:         start,
		PermissionGranted: permGranted,
		IsDeterministic:   true,
	}

	if outErr != 0 || buf == nil {
		resp.ErrorCode = int(outErr)
		resp.ErrorMessage = "Region capture failed"
		return resp, nil
	}

	resp.CapturedImage = cBufferToRGBA(buf, int(outW), int(outH))
	resp.Width = int(outW)
	resp.Height = int(outH)
	return resp, nil
}
