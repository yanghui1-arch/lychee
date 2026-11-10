//go:build windows && amd64

package app

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/png"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// CF_DIB provides compatibility with Windows chat apps and editors; the PNG
// format retains the original lossless image for applications that support it.
func (s *WindowService) CopyScreenshot(data string) error {
	if len(data) > 120*1024*1024 {
		return errors.New("截图过大")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, "data:image/png;base64,"))
	if err != nil {
		return err
	}
	config, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	if config.Width < 1 || config.Height < 1 || int64(config.Width)*int64(config.Height) > 80_000_000 {
		return errors.New("截图尺寸过大")
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	if err = copyScreenshotClipboard(img, raw); err != nil {
		return err
	}
	return s.CancelScreenshot()
}

func screenshotDIB(img image.Image) []byte {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	data := make([]byte, 40+w*h*4)
	binary.LittleEndian.PutUint32(data[0:], 40)
	binary.LittleEndian.PutUint32(data[4:], uint32(w))
	binary.LittleEndian.PutUint32(data[8:], uint32(-int32(h)))
	binary.LittleEndian.PutUint16(data[12:], 1)
	binary.LittleEndian.PutUint16(data[14:], 32)
	binary.LittleEndian.PutUint32(data[20:], uint32(w*h*4))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			i := 40 + (y*w+x)*4
			// Composite transparent pixels onto white for classic DIB consumers.
			data[i], data[i+1], data[i+2], data[i+3] = byte((b+65535-a)>>8), byte((g+65535-a)>>8), byte((r+65535-a)>>8), 255
		}
	}
	return data
}

func copyScreenshotClipboard(img image.Image, pngData []byte) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	allocate := func(data []byte) (uintptr, error) {
		handle, _, err := kernel.NewProc("GlobalAlloc").Call(0x0002, uintptr(len(data)))
		if handle == 0 {
			return 0, err
		}
		ptr, _, err := kernel.NewProc("GlobalLock").Call(handle)
		if ptr == 0 {
			kernel.NewProc("GlobalFree").Call(handle)
			return 0, err
		}
		copy(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(data)), data)
		kernel.NewProc("GlobalUnlock").Call(handle)
		return handle, nil
	}
	dib, err := allocate(screenshotDIB(img))
	if err != nil {
		return err
	}
	defer func() {
		if dib != 0 {
			kernel.NewProc("GlobalFree").Call(dib)
		}
	}()
	pngHandle, err := allocate(pngData)
	if err != nil {
		return err
	}
	defer func() {
		if pngHandle != 0 {
			kernel.NewProc("GlobalFree").Call(pngHandle)
		}
	}()
	hwnd, err := mainWindowHandle()
	if err != nil {
		return err
	}
	opened := false
	for attempt := 0; attempt < 10; attempt++ {
		if ok, _, _ := windowUser32.NewProc("OpenClipboard").Call(uintptr(hwnd)); ok != 0 {
			opened = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !opened {
		return errors.New("剪贴板正被其它程序使用，请重试")
	}
	defer windowUser32.NewProc("CloseClipboard").Call()
	if ok, _, err := windowUser32.NewProc("EmptyClipboard").Call(); ok == 0 {
		return err
	}
	if ok, _, err := windowUser32.NewProc("SetClipboardData").Call(8, dib); ok == 0 {
		return fmt.Errorf("复制截图失败: %w", err)
	}
	dib = 0 // Ownership has transferred to Windows.
	name, _ := windows.UTF16PtrFromString("PNG")
	format, _, _ := windowUser32.NewProc("RegisterClipboardFormatW").Call(uintptr(unsafe.Pointer(name)))
	if format != 0 {
		if ok, _, _ := windowUser32.NewProc("SetClipboardData").Call(format, pngHandle); ok != 0 {
			pngHandle = 0
		}
	}
	return nil
}
