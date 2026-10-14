//go:build windows && amd64

package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var extractAppIcon = windows.NewLazySystemDLL("shell32.dll").NewProc("ExtractIconExW")
var iconUser32 = windows.NewLazySystemDLL("user32.dll")
var iconGDI32 = windows.NewLazySystemDLL("gdi32.dll")
var drawAppIcon = iconUser32.NewProc("DrawIconEx")
var destroyAppIcon = iconUser32.NewProc("DestroyIcon")
var createIconDC = iconGDI32.NewProc("CreateCompatibleDC")
var deleteIconDC = iconGDI32.NewProc("DeleteDC")
var createIconBitmap = iconGDI32.NewProc("CreateDIBSection")
var selectIconBitmap = iconGDI32.NewProc("SelectObject")
var deleteIconBitmap = iconGDI32.NewProc("DeleteObject")
var flushIconDrawing = iconGDI32.NewProc("GdiFlush")

type cachedApplicationIcon struct {
	modified int64
	size     int64
	data     string
}

var applicationIcons = struct {
	sync.Mutex
	items map[string]cachedApplicationIcon
}{items: make(map[string]cachedApplicationIcon)}

func applicationIcon(process windows.Handle) string {
	path := make([]uint16, 32768)
	size := uint32(len(path))
	if windows.QueryFullProcessImageName(process, 0, &path[0], &size) != nil {
		return ""
	}
	return applicationIconFromPath(windows.UTF16ToString(path[:size]))
}

func applicationIconFromPath(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	applicationIcons.Lock()
	defer applicationIcons.Unlock()
	if cached, ok := applicationIcons.items[path]; ok && cached.modified == info.ModTime().UnixNano() && cached.size == info.Size() {
		return cached.data
	}
	icon := readApplicationIcon(path)
	if icon == "" {
		return ""
	}
	// Bound memory usage when many different applications appear over a long session.
	if len(applicationIcons.items) >= 128 {
		for key := range applicationIcons.items {
			delete(applicationIcons.items, key)
			break
		}
	}
	applicationIcons.items[path] = cachedApplicationIcon{info.ModTime().UnixNano(), info.Size(), icon}
	return icon
}

func readApplicationIcon(path string) string {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}
	var icon uintptr
	count, _, _ := extractAppIcon.Call(uintptr(unsafe.Pointer(name)), 0, uintptr(unsafe.Pointer(&icon)), 0, 1)
	if icon != 0 {
		defer destroyAppIcon.Call(icon)
	}
	if count != 1 || icon == 0 {
		return ""
	}
	black := drawIconPixels(icon, 0)
	white := drawIconPixels(icon, 255)
	if black == nil || white == nil {
		return ""
	}
	output := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for i := 0; i < len(black); i += 4 {
		// Two backgrounds recover alpha for both modern and legacy mask-based icons.
		alpha := 255 - max(0, int(white[i])-int(black[i]), int(white[i+1])-int(black[i+1]), int(white[i+2])-int(black[i+2]))
		if alpha == 0 {
			continue
		}
		output.Pix[i] = byte(min(255, int(black[i+2])*255/alpha))
		output.Pix[i+1] = byte(min(255, int(black[i+1])*255/alpha))
		output.Pix[i+2] = byte(min(255, int(black[i])*255/alpha))
		output.Pix[i+3] = byte(alpha)
	}
	var encoded bytes.Buffer
	if png.Encode(&encoded, output) != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes())
}

type iconBitmapInfo struct {
	Size                        uint32
	Width, Height               int32
	Planes, Bits                uint16
	Compression, ImageSize      uint32
	XPixels, YPixels            int32
	ColorsUsed, ColorsImportant uint32
	Color                       uint32
}

func drawIconPixels(icon uintptr, background byte) []byte {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	dc, _, _ := createIconDC.Call(0)
	if dc == 0 {
		return nil
	}
	defer deleteIconDC.Call(dc)
	info := iconBitmapInfo{Size: 40, Width: 32, Height: -32, Planes: 1, Bits: 32, ImageSize: 32 * 32 * 4}
	var pixels unsafe.Pointer
	bitmap, _, _ := createIconBitmap.Call(dc, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&pixels)), 0, 0)
	if bitmap == 0 {
		return nil
	}
	defer deleteIconBitmap.Call(bitmap)
	if pixels == nil {
		return nil
	}
	previous, _, _ := selectIconBitmap.Call(dc, bitmap)
	if previous == 0 || previous == ^uintptr(0) {
		return nil
	}
	defer selectIconBitmap.Call(dc, previous)
	data := unsafe.Slice((*byte)(pixels), 32*32*4)
	for i := 0; i < len(data); i += 4 {
		data[i], data[i+1], data[i+2], data[i+3] = background, background, background, 255
	}
	success, _, _ := drawAppIcon.Call(dc, 0, 0, icon, 32, 32, 0, 0, 3)
	if success == 0 {
		return nil
	}
	flushed, _, _ := flushIconDrawing.Call()
	if flushed == 0 {
		return nil
	}
	return append([]byte(nil), data...)
}
