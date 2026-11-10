//go:build windows && amd64

package app

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"os"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"github.com/kbinani/screenshot"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

type screenshotSession struct {
	previous             windows.Rect
	display              image.Rectangle
	minimized            bool
	original, last, long *image.RGBA
	scrollRect           image.Rectangle
}

func screenshotFrame(img *image.RGBA, added bool) ScreenshotFrame {
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return ScreenshotFrame{Image: "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes()), Width: img.Bounds().Dx(), Height: img.Bounds().Dy(), Added: added}
}
func (s *WindowService) BeginScreenshot() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.capture != nil {
		return nil
	}
	hwnd, err := mainWindowHandle()
	if err != nil {
		return err
	}
	c := &screenshotSession{display: screenshot.GetDisplayBounds(0), minimized: wailsruntime.WindowIsMinimised(s.ctx)}
	if c.minimized {
		wailsruntime.WindowUnminimise(s.ctx)
	}
	windowUser32.NewProc("GetWindowRect").Call(uintptr(hwnd), uintptr(unsafe.Pointer(&c.previous)))
	if err = s.saveFloatingPositionLocked(); err != nil {
		return err
	}
	wailsruntime.WindowHide(s.ctx)
	time.Sleep(100 * time.Millisecond)
	c.original, err = screenshot.CaptureRect(c.display)
	if err != nil {
		wailsruntime.WindowShow(s.ctx)
		return err
	}
	if err = clearWindowRegion(hwnd); err != nil {
		wailsruntime.WindowShow(s.ctx)
		return err
	}
	_ = setWindowTaskbarVisible(hwnd, false)
	wailsruntime.WindowSetMinSize(s.ctx, 1, 1)
	wailsruntime.WindowSetAlwaysOnTop(s.ctx, true)
	if err = placeScreenshotWindow(c.display); err != nil {
		wailsruntime.WindowShow(s.ctx)
		return err
	}
	s.capture = c
	frame := screenshotFrame(c.original, false)
	var work windows.Rect
	if ok, _, _ := windowUser32.NewProc("SystemParametersInfoW").Call(0x0030, 0, uintptr(unsafe.Pointer(&work)), 0); ok != 0 {
		area := image.Rect(int(work.Left), int(work.Top), int(work.Right), int(work.Bottom)).Intersect(c.display).Sub(c.display.Min)
		if !area.Empty() {
			frame.WorkArea = &ScreenshotRect{area.Min.X, area.Min.Y, area.Dx(), area.Dy()}
		}
	}
	wailsruntime.EventsEmit(s.ctx, "screenshot:start", frame)
	return nil
}
func placeScreenshotWindow(r image.Rectangle) error {
	hwnd, err := mainWindowHandle()
	if err != nil {
		return err
	}
	if ok, _, err := windowSetPos.Call(uintptr(hwnd), 0, uintptr(r.Min.X), uintptr(r.Min.Y), uintptr(r.Dx()), uintptr(r.Dy()), 0x0014); ok == 0 {
		return err
	}
	return nil
}
func (s *WindowService) ScreenshotReady() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.capture != nil {
		wailsruntime.WindowShow(s.ctx)
	}
}
func (s *WindowService) CancelScreenshot() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.capture == nil {
		return nil
	}
	c := s.capture
	wailsruntime.WindowHide(s.ctx)
	hwnd, err := mainWindowHandle()
	if err != nil {
		return err
	}
	if err = placeScreenshotWindow(image.Rect(int(c.previous.Left), int(c.previous.Top), int(c.previous.Right), int(c.previous.Bottom))); err != nil {
		return err
	}
	if s.workspace {
		_ = setWindowTaskbarVisible(hwnd, true)
		wailsruntime.WindowSetMinSize(s.ctx, workspaceWidth, workspaceHeight)
	} else {
		_ = applyFloatingRegion(hwnd, s.expanded)
		wailsruntime.WindowSetMinSize(s.ctx, floatingWidth, floatingHeight)
	}
	wailsruntime.WindowSetAlwaysOnTop(s.ctx, !s.workspace)
	s.capture = nil
	wailsruntime.EventsEmit(s.ctx, "screenshot:end")
	wailsruntime.WindowShow(s.ctx)
	if c.minimized {
		wailsruntime.WindowMinimise(s.ctx)
	}
	return nil
}
func (s *WindowService) PinScreenshot(width, height int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.capture == nil {
		return errors.New("没有正在编辑的截图")
	}
	if width < 1 || height < 1 || width > 32768 || height > 32768 {
		return errors.New("截图大小无效")
	}
	// A pinned picture remains movable and always on top, with no taskbar entry.
	d := s.capture.display
	w, h := min(width, d.Dx()-80), min(height, d.Dy()-80)
	scale := min(float64(w)/float64(width), float64(h)/float64(height))
	w, h = int(float64(width)*scale), int(float64(height)*scale)
	return placeScreenshotWindow(image.Rect(d.Min.X+40, d.Min.Y+40, d.Min.X+40+max(120, w), d.Min.Y+40+max(80, h)))
}
func (s *WindowService) SaveScreenshot(data string) (string, error) {
	if len(data) > 120*1024*1024 {
		return "", errors.New("截图过大")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, "data:image/png;base64,"))
	if err != nil {
		return "", err
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 {
		return "", errors.New("无效的 PNG 截图")
	}
	path, err := wailsruntime.SaveFileDialog(s.ctx, wailsruntime.SaveDialogOptions{Title: "保存截图", DefaultFilename: "lychee-" + time.Now().Format("20060102-150405") + ".png", Filters: []wailsruntime.FileFilter{{DisplayName: "PNG 图片", Pattern: "*.png"}}})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, raw, 0600)
}
func (s *WindowService) ScrollScreenshot(x, y, width, height int) (ScreenshotFrame, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.capture
	if c == nil {
		return ScreenshotFrame{}, errors.New("请先截图")
	}
	r := image.Rect(x, y, x+width, y+height)
	if width < 64 || height < 100 || !r.In(image.Rect(0, 0, c.display.Dx(), c.display.Dy())) {
		return ScreenshotFrame{}, errors.New("滚动选区至少为 64 × 100 像素")
	}
	if c.long == nil {
		c.scrollRect = r
		c.last = image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(c.last, c.last.Bounds(), c.original, r.Min, draw.Src)
		c.long = c.last
	}
	if r != c.scrollRect {
		return ScreenshotFrame{}, errors.New("滚动过程中不能修改选区")
	}
	if c.long.Bounds().Dy()+height > 30000 {
		return ScreenshotFrame{}, errors.New("长截图已达到 30000 像素上限")
	}
	wailsruntime.WindowHide(s.ctx)
	defer wailsruntime.WindowShow(s.ctx)
	var old struct{ X, Y int32 }
	windowUser32.NewProc("GetCursorPos").Call(uintptr(unsafe.Pointer(&old)))
	windowUser32.NewProc("SetCursorPos").Call(uintptr(c.display.Min.X+x+width/2), uintptr(c.display.Min.Y+y+height/2))
	// Resolve the actual application beneath the selection after hiding our overlay.
	px, py := int32(c.display.Min.X+x+width/2), int32(c.display.Min.Y+y+height/2)
	under, _, _ := windowUser32.NewProc("WindowFromPoint").Call(uintptr(uint64(uint32(px)) | uint64(uint32(py))<<32))
	root, _, _ := windowUser32.NewProc("GetAncestor").Call(under, 2)
	windowUser32.NewProc("SetForegroundWindow").Call(root)
	time.Sleep(80 * time.Millisecond)
	// Send a normal wheel gesture to the selected application's viewport.
	windowUser32.NewProc("mouse_event").Call(0x0800, 0, 0, uintptr(uint32(0xfffffe98)), 0) // -360
	time.Sleep(450 * time.Millisecond)
	frame, err := screenshot.CaptureRect(r.Add(c.display.Min))
	windowUser32.NewProc("SetCursorPos").Call(uintptr(old.X), uintptr(old.Y))
	if err != nil {
		return ScreenshotFrame{}, err
	}
	merged, added := appendScrollFrame(c.long, c.last, frame)
	if added {
		c.long, c.last = merged, frame
	}
	return screenshotFrame(c.long, added), nil
}
func (s *WindowService) startScreenshotHotkey() func() {
	ready := make(chan uint32, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		tid := windows.GetCurrentThreadId()
		var msg struct {
			Hwnd           uintptr
			Message        uint32
			Wparam, Lparam uintptr
			Time           uint32
			X, Y           int32
			Private        uint32
		}
		windowUser32.NewProc("PeekMessageW").Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)
		ok, _, err := windowUser32.NewProc("RegisterHotKey").Call(0, 0x4c59, 0x4000, 0x70)
		if ok == 0 {
			wailsruntime.LogError(s.ctx, "F1 截图快捷键注册失败: "+err.Error())
			ready <- 0
			return
		}
		ready <- tid
		defer windowUser32.NewProc("UnregisterHotKey").Call(0, 0x4c59)
		for {
			result, _, _ := windowUser32.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(result) <= 0 {
				return
			}
			if msg.Message == 0x0312 {
				go func() {
					if err := s.BeginScreenshot(); err != nil {
						wailsruntime.EventsEmit(s.ctx, "screenshot:error", err.Error())
					}
				}()
			}
		}
	}()
	tid := <-ready
	return func() {
		if tid != 0 {
			windowUser32.NewProc("PostThreadMessageW").Call(uintptr(tid), 0x0012, 0, 0)
		}
	}
}
