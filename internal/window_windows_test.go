//go:build windows && amd64

package app

import (
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestFloatingRegionLeavesHiddenToolbarClickThrough(t *testing.T) {
	contains := windowGDI32.NewProc("PtInRegion")
	for _, dpi := range []int{96, 106, 120, 144, 168, 192} {
		scale := func(value int) uintptr { return uintptr((value*dpi + 48) / 96) }
		for _, expanded := range []bool{false, true} {
			region, err := floatingRegion(dpi, expanded)
			if err != nil {
				t.Fatal(err)
			}
			for _, point := range []struct {
				x, y int
				want bool
			}{
				{90, 38, true},     // The ball always accepts input.
				{28, 38, expanded}, // Hidden toolbar must not block the desktop.
				{58, 38, expanded}, // Bridge keeps the toolbar open while crossing the gap.
				{109, 9, expanded}, // Exit button only exists while expanded.
			} {
				inside, _, _ := contains.Call(region, scale(point.x), scale(point.y))
				if (inside != 0) != point.want {
					t.Errorf("dpi=%d expanded=%t point=(%d,%d): incorrect hit region", dpi, expanded, point.x, point.y)
				}
			}
			windowDeleteObject.Call(region)
		}
	}
}

func TestFloatingHoverKeepsNativeWindowBounds(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// Exercise the real Win32 APIs on an invisible test-owned window. A hover
	// cycle must not change any of its four bounds, even temporarily.
	className, _ := windows.UTF16PtrFromString("LycheeFloatingWindowTest")
	class := struct {
		Size, Style             uint32
		Procedure               uintptr
		ClassExtra, WindowExtra int32
		Instance, Icon, Cursor  uintptr
		Background              uintptr
		MenuName, ClassName     *uint16
		SmallIcon               uintptr
	}{Procedure: windowUser32.NewProc("DefWindowProcW").Addr(), ClassName: className}
	class.Size = uint32(unsafe.Sizeof(class))
	if ok, _, err := windowUser32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&class))); ok == 0 {
		t.Fatal(err)
	}
	defer windowUser32.NewProc("UnregisterClassW").Call(uintptr(unsafe.Pointer(className)), 0)
	hwnd, _, err := windowUser32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(className)), 0, 0x80000000, 200, 300, floatingWidth, floatingHeight, 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatal(err)
	}
	defer windowUser32.NewProc("DestroyWindow").Call(hwnd)
	type rect struct{ Left, Top, Right, Bottom int32 }
	getRect := windowUser32.NewProc("GetWindowRect")
	var before rect
	if ok, _, err := getRect.Call(hwnd, uintptr(unsafe.Pointer(&before))); ok == 0 {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 20; cycle++ {
		for _, expanded := range []bool{true, false} {
			if err := applyFloatingRegion(windows.HWND(hwnd), expanded); err != nil {
				t.Fatal(err)
			}
			var after rect
			if ok, _, err := getRect.Call(hwnd, uintptr(unsafe.Pointer(&after))); ok == 0 {
				t.Fatal(err)
			}
			if before != after {
				t.Fatalf("hover changed native bounds: %v -> %v", before, after)
			}
		}
	}
	if err := clearWindowRegion(windows.HWND(hwnd)); err != nil {
		t.Fatal(err)
	}
}
