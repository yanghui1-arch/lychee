//go:build windows && amd64

package screenshot

import (
	"image"
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestScrollOverlayLeavesOriginalApplicationInteractive(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	className, _ := windows.UTF16PtrFromString("STATIC")
	hwnd, _, err := windowUser32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(className)), 0, 0x80000000, 0, 0, 800, 600, 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatal(err)
	}
	defer windowUser32.NewProc("DestroyWindow").Call(hwnd)
	s := &Service{hooks: WindowHooks{Handle: func() (uintptr, error) { return hwnd, nil }}, capture: &screenshotSession{display: image.Rect(0, 0, 800, 600), scrollRect: image.Rect(100, 100, 500, 500)}}
	gdi := windows.NewLazySystemDLL("gdi32.dll")
	region, _, _ := gdi.NewProc("CreateRectRgn").Call(0, 0, 0, 0)
	if region == 0 {
		t.Fatal("create test region")
	}
	defer gdi.NewProc("DeleteObject").Call(region)
	for i := 0; i < 3; i++ {
		if err := s.setScrollOverlay(image.Rect(520, 260, 760, 500), image.Rect(200, 520, 660, 568), 4); err != nil {
			t.Fatal(err)
		}
		if kind, _, _ := windowUser32.NewProc("GetWindowRgn").Call(hwnd, region); kind == 0 {
			t.Fatal("missing native region")
		}
		for _, point := range []struct {
			x, y   int
			inside bool
		}{{99, 100, true}, {100, 100, false}, {499, 499, false}, {500, 500, true}, {600, 200, false}, {600, 300, true}, {50, 300, false}, {300, 50, false}, {300, 550, true}, {100, 550, false}, {799, 599, false}} {
			inside, _, _ := gdi.NewProc("PtInRegion").Call(region, uintptr(point.x), uintptr(point.y))
			if (inside != 0) != point.inside {
				t.Fatalf("incorrect hit region at %d,%d", point.x, point.y)
			}
		}
		if err := s.clearScrollOverlay(); err != nil {
			t.Fatal(err)
		}
		if kind, _, _ := windowUser32.NewProc("GetWindowRgn").Call(hwnd, region); kind != 0 {
			t.Fatal("overlay remained after cancellation")
		}
	}
}
