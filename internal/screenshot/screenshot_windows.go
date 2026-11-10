//go:build windows && amd64

package screenshot

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

var windowUser32 = windows.NewLazySystemDLL("user32.dll")
var windowSetPos = windowUser32.NewProc("SetWindowPos")

func (s *Service) windowHandle() (windows.HWND, error) {
	handle, err := s.hooks.Handle()
	return windows.HWND(handle), err
}

type screenshotSession struct {
	previous            windows.Rect
	display             image.Rectangle
	minimized           bool
	original            *image.RGBA
	stitch              *scrollCapture
	scrollRect          image.Rectangle
	contextRect         image.Rectangle
	excluded            bool
	overlaySet          bool
	pinned              bool
	sampling            chan struct{}
	revision, delivered uint64
	sampleErr           error
}

func screenshotFrame(img *image.RGBA, added bool) ScreenshotFrame {
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return ScreenshotFrame{Image: "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes()), Width: img.Bounds().Dx(), Height: img.Bounds().Dy(), Added: added}
}
func (s *Service) BeginScreenshot() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.capture != nil {
		return nil
	}
	hwnd, err := s.windowHandle()
	if err != nil {
		return err
	}
	c := &screenshotSession{display: screenshot.GetDisplayBounds(0), minimized: wailsruntime.WindowIsMinimised(s.ctx)}
	if c.minimized {
		wailsruntime.WindowUnminimise(s.ctx)
	}
	windowUser32.NewProc("GetWindowRect").Call(uintptr(hwnd), uintptr(unsafe.Pointer(&c.previous)))
	if err = s.hooks.SavePosition(); err != nil {
		return err
	}
	wailsruntime.WindowHide(s.ctx)
	time.Sleep(100 * time.Millisecond)
	c.original, err = screenshot.CaptureRect(c.display)
	if err != nil {
		wailsruntime.WindowShow(s.ctx)
		return err
	}
	if err = s.hooks.Prepare(); err != nil {
		wailsruntime.WindowShow(s.ctx)
		return err
	}
	wailsruntime.WindowSetMinSize(s.ctx, 1, 1)
	wailsruntime.WindowSetAlwaysOnTop(s.ctx, true)
	if err = s.placeWindow(c.display); err != nil {
		wailsruntime.WindowShow(s.ctx)
		return err
	}
	s.capture = c
	s.active.Store(true)
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
func (s *Service) placeWindow(r image.Rectangle) error {
	hwnd, err := s.windowHandle()
	if err != nil {
		return err
	}
	if ok, _, err := windowSetPos.Call(uintptr(hwnd), 0, uintptr(r.Min.X), uintptr(r.Min.Y), uintptr(r.Dx()), uintptr(r.Dy()), 0x0014); ok == 0 {
		return err
	}
	return nil
}
func (s *Service) ScreenshotReady() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.capture != nil {
		wailsruntime.WindowShow(s.ctx)
	}
}
func (s *Service) CancelScreenshot() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.capture == nil {
		return nil
	}
	c := s.capture
	c.stopSampling()
	if c.excluded {
		if hwnd, err := s.windowHandle(); err == nil {
			windowUser32.NewProc("SetWindowDisplayAffinity").Call(uintptr(hwnd), 0)
		}
	}
	wailsruntime.WindowHide(s.ctx)
	if err := s.clearScrollOverlay(); err != nil {
		return err
	}
	if err := s.placeWindow(image.Rect(int(c.previous.Left), int(c.previous.Top), int(c.previous.Right), int(c.previous.Bottom))); err != nil {
		return err
	}
	if err := s.hooks.Restore(); err != nil {
		return err
	}
	s.capture = nil
	s.active.Store(false)
	wailsruntime.EventsEmit(s.ctx, "screenshot:end")
	wailsruntime.WindowShow(s.ctx)
	if c.minimized {
		wailsruntime.WindowMinimise(s.ctx)
	}
	return nil
}
func (s *Service) PinScreenshot(width, height int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.capture == nil {
		return errors.New("没有正在编辑的截图")
	}
	if width < 1 || height < 1 || width > 32768 || height > 32768 {
		return errors.New("截图大小无效")
	}
	if s.capture.overlaySet {
		if err := s.clearScrollOverlay(); err != nil {
			return err
		}
		if s.capture.excluded {
			if hwnd, err := s.windowHandle(); err == nil {
				windowUser32.NewProc("SetWindowDisplayAffinity").Call(uintptr(hwnd), 0)
			}
			s.capture.excluded = false
		}
		s.capture.pinned = true
	}
	// A pinned picture remains movable and always on top, with no taskbar entry.
	d := s.capture.display
	w, h := min(width, d.Dx()-80), min(height, d.Dy()-80)
	scale := min(float64(w)/float64(width), float64(h)/float64(height))
	w, h = int(float64(width)*scale), int(float64(height)*scale)
	return s.placeWindow(image.Rect(d.Min.X+40, d.Min.Y+40, d.Min.X+40+max(120, w), d.Min.Y+40+max(80, h)))
}
func (s *Service) SaveScreenshot(data string) (string, error) {
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
func (s *Service) ScrollScreenshot(x, y, width, height int) (ScreenshotFrame, error) {
	s.mu.Lock()
	c := s.capture
	if c == nil {
		s.mu.Unlock()
		return ScreenshotFrame{}, errors.New("请先截图")
	}
	r := image.Rect(x, y, x+width, y+height)
	if width < 40 || height < 40 || !r.In(image.Rect(0, 0, c.display.Dx(), c.display.Dy())) {
		s.mu.Unlock()
		return ScreenshotFrame{}, errors.New("滚动选区至少为 40 × 40 像素")
	}
	if c.stitch == nil {
		c.scrollRect = r
		first := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(first, first.Bounds(), c.original, r.Min, draw.Src)
		c.contextRect = r
		c.stitch = newScrollCapture(first, first.Bounds())
		s.mu.Unlock()
		return screenshotFrame(first, false), nil
	}
	if r != c.scrollRect {
		s.mu.Unlock()
		return ScreenshotFrame{}, errors.New("滚动过程中不能修改选区")
	}
	img, err := c.stitch.long, c.sampleErr
	changed := c.revision != c.delivered
	if err == nil {
		c.delivered = c.revision
	}
	s.mu.Unlock()
	if err != nil {
		return ScreenshotFrame{}, err
	}
	if !changed {
		return ScreenshotFrame{Width: img.Bounds().Dx(), Height: img.Bounds().Dy()}, nil
	}
	// Long images are immutable. Encoding outside the lock lets acquisition keep
	// running while the preview is encoded and transferred to the frontend.
	return screenshotFrame(img, true), nil
}

func (c *screenshotSession) stopSampling() {
	if c.sampling != nil {
		close(c.sampling)
		c.sampling = nil
	}
}

func (s *Service) sampleScroll(c *screenshotSession) error {
	frame, err := screenshot.CaptureRect(c.contextRect.Add(c.display.Min))
	if err != nil {
		return err
	}
	added, err := c.stitch.append(frame)
	if added {
		c.revision++
	}
	return err
}

func (s *Service) sampleScrollLoop(c *screenshotSession, stop <-chan struct{}) {
	// Frequent samples keep consecutive frames overlapping, even for short
	// selections and fast wheel scrolling. Identical frames are skipped cheaply.
	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s.mu.Lock()
			if s.capture != c || c.sampling != stop {
				s.mu.Unlock()
				return
			}
			rect := c.contextRect.Add(c.display.Min)
			s.mu.Unlock()
			// Capture outside the lock so preview polling is not blocked by it.
			frame, err := screenshot.CaptureRect(rect)
			s.mu.Lock()
			if s.capture != c || c.sampling != stop {
				s.mu.Unlock()
				return
			}
			if err == nil {
				var added bool
				added, err = c.stitch.append(frame)
				if added {
					c.revision++
				}
			}
			c.sampleErr = err
			s.mu.Unlock()
		}
	}
}

func (s *Service) StopScrollScreenshot() (ScreenshotFrame, error) {
	s.mu.Lock()
	c := s.capture
	if c == nil || c.stitch == nil {
		s.mu.Unlock()
		return ScreenshotFrame{}, errors.New("没有正在进行的滚动截图")
	}
	c.stopSampling()
	sampleErr := s.sampleScroll(c)
	img := c.stitch.long
	s.mu.Unlock()
	wailsruntime.WindowShow(s.ctx)
	if sampleErr != nil {
		wailsruntime.EventsEmit(s.ctx, "screenshot:error", sampleErr.Error())
	}
	return screenshotFrame(img, false), nil
}

// SetScrollScreenshotOverlay leaves the real desktop visible and interactive
// everywhere except the preview, shared screenshot toolbar and selection outline.
func (s *Service) SetScrollScreenshotOverlay(x, y, width, height, border, toolbarX, toolbarY, toolbarWidth, toolbarHeight int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.capture
	if c == nil || c.stitch == nil {
		return errors.New("没有正在进行的滚动截图")
	}
	if c.pinned {
		return nil
	}
	preview := image.Rect(x, y, x+width, y+height)
	toolbar := image.Rect(toolbarX, toolbarY, toolbarX+toolbarWidth, toolbarY+toolbarHeight)
	screen := image.Rect(0, 0, c.display.Dx(), c.display.Dy())
	if toolbarWidth <= 0 || toolbarHeight <= 0 || !toolbar.In(screen) || width <= 0 || height <= 0 || !preview.In(screen) || !preview.Intersect(c.scrollRect).Empty() || border < 1 || border > 32 {
		return errors.New("滚动截图预览位置无效")
	}
	if err := s.setScrollOverlay(preview, toolbar, border); err != nil {
		return err
	}
	if !c.overlaySet {
		// Give normal wheel and keyboard scrolling back to the selected application,
		// including systems with scrolling of inactive windows disabled.
		px := int32(c.display.Min.X + c.scrollRect.Min.X + c.scrollRect.Dx()/2)
		py := int32(c.display.Min.Y + c.scrollRect.Min.Y + c.scrollRect.Dy()/2)
		under, _, _ := windowUser32.NewProc("WindowFromPoint").Call(uintptr(uint64(uint32(px)) | uint64(uint32(py))<<32))
		root, _, _ := windowUser32.NewProc("GetAncestor").Call(under, 2)
		// Exclude our outline/preview from desktop captures before widening the
		// tracking area. The output remains the exact selected horizontal strip.
		if hwnd, err := s.windowHandle(); err == nil {
			if ok, _, _ := windowUser32.NewProc("SetWindowDisplayAffinity").Call(uintptr(hwnd), 0x11); ok != 0 {
				c.excluded = true
				c.contextRect = scrollTrackingRect(under, c.scrollRect, c.display)
				context := image.NewRGBA(image.Rect(0, 0, c.contextRect.Dx(), c.contextRect.Dy()))
				draw.Draw(context, context.Bounds(), c.original, c.contextRect.Min, draw.Src)
				c.stitch = newScrollCapture(context, c.scrollRect.Sub(c.contextRect.Min))
			}
		}
		if root != 0 {
			windowUser32.NewProc("SetForegroundWindow").Call(root)
		}
		c.overlaySet = true
		c.sampling = make(chan struct{})
		go s.sampleScrollLoop(c, c.sampling)
	}
	return nil
}

// Use the native viewport under the selection, not the whole display: adjacent
// panes and browser chrome must not become matching context for the scroll area.
func scrollTrackingRect(under uintptr, selected, display image.Rectangle) image.Rectangle {
	var client windows.Rect
	var origin struct{ X, Y int32 }
	if under == 0 {
		return selected
	}
	ok, _, _ := windowUser32.NewProc("GetClientRect").Call(under, uintptr(unsafe.Pointer(&client)))
	if ok == 0 {
		return selected
	}
	ok, _, _ = windowUser32.NewProc("ClientToScreen").Call(under, uintptr(unsafe.Pointer(&origin)))
	if ok == 0 {
		return selected
	}
	viewport := image.Rect(int(origin.X), int(origin.Y), int(origin.X)+int(client.Right), int(origin.Y)+int(client.Bottom)).Intersect(display).Sub(display.Min)
	if !selected.In(viewport) {
		return selected
	}
	// Limit tracking work on very tall displays while providing at least several
	// short-selection heights of headroom in both scroll directions.
	margin := max(400, selected.Dy()*2)
	return image.Rect(selected.Min.X, max(viewport.Min.Y, selected.Min.Y-margin), selected.Max.X, min(viewport.Max.Y, selected.Max.Y+margin))
}

func (s *Service) clearScrollOverlay() error {
	hwnd, err := s.windowHandle()
	if err != nil {
		return err
	}
	if ok, _, err := windowUser32.NewProc("SetWindowRgn").Call(uintptr(hwnd), 0, 1); ok == 0 {
		return err
	}
	return nil
}

func (s *Service) setScrollOverlay(preview, toolbar image.Rectangle, border int) error {
	hwnd, err := s.windowHandle()
	if err != nil {
		return err
	}
	gdi := windows.NewLazySystemDLL("gdi32.dll")
	create := gdi.NewProc("CreateRectRgn")
	remove := gdi.NewProc("DeleteObject")
	rectRegion := func(r image.Rectangle) uintptr {
		region, _, _ := create.Call(uintptr(r.Min.X), uintptr(r.Min.Y), uintptr(r.Max.X), uintptr(r.Max.Y))
		return region
	}
	selection := s.capture.scrollRect
	outline := rectRegion(selection.Inset(-border))
	cut := rectRegion(selection)
	panel := rectRegion(preview)
	controls := rectRegion(toolbar)
	if cut != 0 {
		defer remove.Call(cut)
	}
	if panel != 0 {
		defer remove.Call(panel)
	}
	if controls != 0 {
		defer remove.Call(controls)
	}
	if outline == 0 || cut == 0 || panel == 0 || controls == 0 {
		if outline != 0 {
			remove.Call(outline)
		}
		return errors.New("无法创建滚动截图悬浮区域")
	}
	combine := gdi.NewProc("CombineRgn")
	// Subtract the capture rectangle, then union preview and shared toolbar. Everything else is
	// absent from our window, so wheel input works across the original application.
	difference, _, _ := combine.Call(outline, outline, cut, 4)
	union, _, _ := combine.Call(outline, outline, panel, 2)
	toolbarUnion, _, _ := combine.Call(outline, outline, controls, 2)
	if difference == 0 || union == 0 || toolbarUnion == 0 {
		remove.Call(outline)
		return errors.New("无法设置滚动截图悬浮区域")
	}
	if ok, _, err := windowUser32.NewProc("SetWindowRgn").Call(uintptr(hwnd), outline, 1); ok == 0 {
		remove.Call(outline)
		return err
	}
	// The window owns outline after a successful SetWindowRgn.
	return nil
}

func (s *Service) StartHotkey() func() {
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
