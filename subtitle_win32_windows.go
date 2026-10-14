//go:build windows && amd64

package main

import "golang.org/x/sys/windows"

var subtitleUser32 = windows.NewLazySystemDLL("user32.dll")
var subtitleGDI32 = windows.NewLazySystemDLL("gdi32.dll")
var subtitleRegisterClass = subtitleUser32.NewProc("RegisterClassExW")
var subtitleCreateWindow = subtitleUser32.NewProc("CreateWindowExW")
var subtitleDefWindow = subtitleUser32.NewProc("DefWindowProcW")
var subtitleGetMessage = subtitleUser32.NewProc("GetMessageW")
var subtitleDispatchMessage = subtitleUser32.NewProc("DispatchMessageW")
var subtitlePostMessage = subtitleUser32.NewProc("PostMessageW")
var subtitleDestroyWindow = subtitleUser32.NewProc("DestroyWindow")
var subtitleQuit = subtitleUser32.NewProc("PostQuitMessage")
var subtitleShowWindow = subtitleUser32.NewProc("ShowWindow")
var subtitleSetPosition = subtitleUser32.NewProc("SetWindowPos")
var subtitleUpdateLayer = subtitleUser32.NewProc("UpdateLayeredWindow")
var subtitleForeground = subtitleUser32.NewProc("GetForegroundWindow")
var subtitleMonitorFromWindow = subtitleUser32.NewProc("MonitorFromWindow")
var subtitleGetMonitorInfo = subtitleUser32.NewProc("GetMonitorInfoW")
var subtitleGetDPI = subtitleUser32.NewProc("GetDpiForWindow")
var subtitleMonitorDPI = windows.NewLazySystemDLL("shcore.dll").NewProc("GetDpiForMonitor")
var subtitleThreadDPI = subtitleUser32.NewProc("SetThreadDpiAwarenessContext")
var subtitleCreateDC = subtitleGDI32.NewProc("CreateCompatibleDC")
var subtitleDeleteDC = subtitleGDI32.NewProc("DeleteDC")
var subtitleCreateBitmap = subtitleGDI32.NewProc("CreateDIBSection")
var subtitleSelectObject = subtitleGDI32.NewProc("SelectObject")
var subtitleDeleteObject = subtitleGDI32.NewProc("DeleteObject")
var subtitleCreateFont = subtitleGDI32.NewProc("CreateFontW")
var subtitleTextColor = subtitleGDI32.NewProc("SetTextColor")
var subtitleBackgroundMode = subtitleGDI32.NewProc("SetBkMode")
var subtitleDrawText = subtitleUser32.NewProc("DrawTextW")
var subtitleFlush = subtitleGDI32.NewProc("GdiFlush")

const (
	subtitleUpdateMessage = 0x8001
	subtitleCloseMessage  = 0x8002
	subtitleWindowStyles  = 0x080800A8 // NOACTIVATE | LAYERED | TOOLWINDOW | TOPMOST | TRANSPARENT.
)

type subtitlePoint struct{ X, Y int32 }
type subtitleRect struct{ Left, Top, Right, Bottom int32 }
type subtitleMonitorInfo struct {
	Size    uint32
	Monitor subtitleRect
	Work    subtitleRect
	Flags   uint32
}
type subtitleWindowClass struct {
	Size, Style             uint32
	Procedure               uintptr
	ClassExtra, WindowExtra int32
	Instance, Icon, Cursor  uintptr
	Background              uintptr
	MenuName, ClassName     *uint16
	SmallIcon               uintptr
}
type subtitleMessage struct {
	Window         uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Point          subtitlePoint
	Private        uint32
}
type subtitleBitmapInfo struct {
	Size                        uint32
	Width, Height               int32
	Planes, Bits                uint16
	Compression, ImageSize      uint32
	XPixels, YPixels            int32
	ColorsUsed, ColorsImportant uint32
}
