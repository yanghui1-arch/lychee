//go:build windows && amd64

package app

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var windowUser32 = windows.NewLazySystemDLL("user32.dll")
var windowGetClass = windowUser32.NewProc("GetClassNameW")
var windowGetStyle = windowUser32.NewProc("GetWindowLongPtrW")
var windowSetStyle = windowUser32.NewProc("SetWindowLongPtrW")
var windowSetPos = windowUser32.NewProc("SetWindowPos")
var windowGetDPI = windowUser32.NewProc("GetDpiForWindow")
var windowSetRegion = windowUser32.NewProc("SetWindowRgn")
var windowGDI32 = windows.NewLazySystemDLL("gdi32.dll")
var windowRectRegion = windowGDI32.NewProc("CreateRectRgn")
var windowEllipseRegion = windowGDI32.NewProc("CreateEllipticRgn")
var windowDeleteObject = windowGDI32.NewProc("DeleteObject")

var findMainWindow = syscall.NewCallback(func(hwnd windows.HWND, result *windows.HWND) uintptr {
	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(hwnd, &pid); err != nil || pid != uint32(os.Getpid()) {
		return 1
	}
	var class [128]uint16
	windowGetClass.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
	if windows.UTF16ToString(class[:]) == mainWindowClass {
		*result = hwnd
		return 0
	}
	return 1
})

// Wails hides the native frame visually but retains the overlapped-window styles.
// Windows still enforces their minimum width. A borderless popup removes that
// constraint; the workspace already supplies its own close/minimize controls.
func prepareFloatingWindow() error {
	hwnd, err := mainWindowHandle()
	if err != nil {
		return err
	}
	const styleIndex = ^uintptr(15) // GWL_STYLE = -16.
	style, _, _ := windowGetStyle.Call(uintptr(hwnd), styleIndex)
	const overlappedWindow = 0x00CF0000
	const popup = 0x80000000
	if previous, _, err := windowSetStyle.Call(uintptr(hwnd), styleIndex, style&^overlappedWindow|popup); previous == 0 {
		return fmt.Errorf("设置悬浮球窗口: %w", err)
	}
	// Refresh the non-client frame without changing position, size, or z-order.
	if ok, _, err := windowSetPos.Call(uintptr(hwnd), 0, 0, 0, 0, 0, 0x0037); ok == 0 {
		return fmt.Errorf("更新悬浮球窗口: %w", err)
	}
	return nil
}

func mainWindowHandle() (windows.HWND, error) {
	var hwnd windows.HWND
	_ = windows.EnumWindows(findMainWindow, unsafe.Pointer(&hwnd))
	runtime.KeepAlive(&hwnd)
	if hwnd == 0 {
		return 0, fmt.Errorf("荔枝窗口尚未就绪")
	}
	return hwnd, nil
}

// Wails reads absolute coordinates but sets positions relative to the work area.
// Use absolute coordinates for both saving and restoring to avoid position drift.
func moveFloatingWindow(x, y int) error {
	hwnd, err := mainWindowHandle()
	if err != nil {
		return err
	}
	if ok, _, err := windowSetPos.Call(uintptr(hwnd), 0, uintptr(x), uintptr(y), 0, 0, 0x0015); ok == 0 {
		return fmt.Errorf("设置悬浮球位置: %w", err)
	}
	return nil
}

func setFloatingRegion(expanded bool) error {
	hwnd, err := mainWindowHandle()
	if err != nil {
		return err
	}
	return applyFloatingRegion(hwnd, expanded)
}

func applyFloatingRegion(hwnd windows.HWND, expanded bool) error {
	dpi, _, _ := windowGetDPI.Call(uintptr(hwnd))
	if dpi == 0 {
		dpi = 96
	}
	region, err := floatingRegion(int(dpi), expanded)
	if err != nil {
		return err
	}
	// The compositor already has the complete fixed-size surface. Updating its
	// clip without erasing/repainting the WebView avoids flashes. On success,
	// Windows owns the region and releases the previously assigned one.
	if ok, _, err := windowSetRegion.Call(uintptr(hwnd), region, 0); ok == 0 {
		windowDeleteObject.Call(region)
		return fmt.Errorf("设置悬浮球区域: %w", err)
	}
	return nil
}

func floatingRegion(dpi int, expanded bool) (uintptr, error) {
	scale := func(value int) uintptr { return uintptr((value*dpi + 48) / 96) }
	var region uintptr
	var err error
	if expanded {
		region, _, err = windowRectRegion.Call(0, 0, scale(floatingWidth), scale(floatingHeight))
	} else {
		region, _, err = windowEllipseRegion.Call(scale(floatingOffsetX), scale(floatingOffsetY), scale(floatingWidth), scale(floatingHeight))
	}
	if region == 0 {
		return 0, fmt.Errorf("创建悬浮球区域: %w", err)
	}
	return region, nil
}

func clearFloatingRegion() error {
	hwnd, err := mainWindowHandle()
	if err != nil {
		return err
	}
	return clearWindowRegion(hwnd)
}

func clearWindowRegion(hwnd windows.HWND) error {
	if ok, _, err := windowSetRegion.Call(uintptr(hwnd), 0, 0); ok == 0 {
		return fmt.Errorf("恢复工作区窗口: %w", err)
	}
	return nil
}
