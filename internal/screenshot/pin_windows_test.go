//go:build windows && amd64

package screenshot

import (
	"image"
	"runtime"
	"sync"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestPinScreenshotPlacesNativeWindowAtCaptureCoordinates(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	class, _ := windows.UTF16PtrFromString("STATIC")
	hwnd, _, err := windowUser32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), 0, 0x80000000, 0, 0, 32, 32, 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatal(err)
	}
	defer windowUser32.NewProc("DestroyWindow").Call(hwnd)
	s := &Service{mu: &sync.Mutex{}, hooks: WindowHooks{Handle: func() (uintptr, error) { return hwnd, nil }}, capture: &screenshotSession{display: image.Rect(0, 0, 1600, 900)}}
	if err := s.PinScreenshot(320, 200, 400, 250); err != nil {
		t.Fatal(err)
	}
	var rect windows.Rect
	if ok, _, err := windowUser32.NewProc("GetWindowRect").Call(hwnd, uintptr(unsafe.Pointer(&rect))); ok == 0 {
		t.Fatal(err)
	}
	if rect.Left != 332 || rect.Top != 212 || rect.Right != 732 || rect.Bottom != 462 {
		t.Fatalf("unexpected native pin rectangle: %+v", rect)
	}
	if !s.capture.pinned {
		t.Fatal("pin state not set")
	}
}
