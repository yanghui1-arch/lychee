//go:build windows && amd64

package subtitle

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type subtitleUpdate struct {
	settings   SubtitleSettings
	visible    bool
	frame      SubtitleFrame
	result     chan error
	pageResult chan subtitlePageResult
}

type subtitleWindowResult struct {
	window uintptr
	err    error
}
type windowsDesktopSubtitle struct {
	mu       sync.Mutex
	window   uintptr
	commands chan subtitleUpdate
	done     chan struct{}
	closed   bool
}
type nativeSubtitleState struct {
	commands  chan subtitleUpdate
	settings  SubtitleSettings
	visible   bool
	active    bool
	monitor   uintptr
	scale     float64
	frame     SubtitleFrame
	surface   *subtitleSurface
	point     subtitlePoint
	fadeStart time.Time
}

var subtitleWindows sync.Map
var subtitleClassOnce sync.Once
var subtitleClassError error
var subtitleClassName, _ = windows.UTF16PtrFromString("LycheeDesktopSubtitle")
var subtitleWindowProcedure = syscall.NewCallback(func(window uintptr, message uint32, wparam, lparam uintptr) uintptr {
	value, ok := subtitleWindows.Load(window)
	if ok {
		state := value.(*nativeSubtitleState)
		switch message {
		case subtitleUpdateMessage:
			update := <-state.commands
			if update.pageResult != nil {
				page, err := state.page(window, update.settings, update.frame.Source, update.frame.Translation)
				update.pageResult <- subtitlePageResult{page: page, err: err}
				return 0
			}
			update.result <- state.update(window, update.settings, update.frame, update.visible)
			return 0
		case 0x0113: // WM_TIMER: fade the cached bitmap, never re-rasterize glyphs.
			if state.surface != nil && state.visible {
				_ = state.paint(window)
				if time.Since(state.fadeStart) >= 180*time.Millisecond {
					subtitleKillTimer.Call(window, 1)
				}
			}
			return 0
		case subtitleCloseMessage:
			subtitleDestroyWindow.Call(window)
			subtitleQuit.Call(0)
			return 0
		case 0x0010: // WM_CLOSE: only the owning service may close this passive window.
			return 0
		case 0x007E, 0x001A, 0x02E0: // Display, work-area, or DPI changes.
			if state.active {
				_ = state.update(window, state.settings, state.frame, true)
			}
			return 0
		}
	}
	result, _, _ := subtitleDefWindow.Call(window, uintptr(message), wparam, lparam)
	return result
})

func newDesktopSubtitle() DesktopSubtitle { return &windowsDesktopSubtitle{} }

// Page uses the same monitor, DPI, fonts and fixed rows as native rendering.
// Like Update, all GDI work stays on the owning Windows message thread.
func (s *windowsDesktopSubtitle) Page(settings SubtitleSettings, source, translation string) (SubtitlePage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return SubtitlePage{}, errors.New("桌面字幕窗口已关闭")
	}
	if s.window == 0 {
		if err := s.start(); err != nil {
			return SubtitlePage{}, err
		}
	}
	command := subtitleUpdate{settings: settings, frame: SubtitleFrame{Source: source, Translation: translation}, pageResult: make(chan subtitlePageResult, 1)}
	select {
	case s.commands <- command:
	case <-s.done:
		return SubtitlePage{}, errors.New("桌面字幕窗口线程已退出")
	}
	if result, _, err := subtitlePostMessage.Call(s.window, subtitleUpdateMessage, 0, 0); result == 0 {
		<-s.commands
		return SubtitlePage{}, err
	}
	select {
	case result := <-command.pageResult:
		return result.page, result.err
	case <-s.done:
		return SubtitlePage{}, errors.New("桌面字幕窗口线程已退出")
	}
}

// Update serializes native operations onto a dedicated Windows message thread.
func (s *windowsDesktopSubtitle) Update(settings SubtitleSettings, frame SubtitleFrame, visible bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("桌面字幕窗口已关闭")
	}
	if s.window == 0 && !visible {
		return nil
	}
	if s.window == 0 {
		if err := s.start(); err != nil {
			return err
		}
	}
	update := subtitleUpdate{settings: settings, frame: frame, visible: visible, result: make(chan error, 1)}
	select {
	case s.commands <- update:
	case <-s.done:
		return errors.New("桌面字幕窗口线程已退出")
	}
	if result, _, err := subtitlePostMessage.Call(s.window, subtitleUpdateMessage, 0, 0); result == 0 {
		<-s.commands
		return fmt.Errorf("更新桌面字幕失败：%w", err)
	}
	select {
	case err := <-update.result:
		return err
	case <-s.done:
		return errors.New("桌面字幕窗口线程已退出")
	}
}

// start is called with s.mu held; only runSubtitleWindow owns the message loop.
func (s *windowsDesktopSubtitle) start() error {
	s.commands = make(chan subtitleUpdate, 1)
	s.done = make(chan struct{})
	ready := make(chan subtitleWindowResult, 1)
	go runSubtitleWindow(s.commands, s.done, ready)
	result := <-ready
	if result.err != nil {
		<-s.done
		return result.err
	}
	s.window = result.window
	return nil
}

func runSubtitleWindow(commands chan subtitleUpdate, done chan struct{}, ready chan<- subtitleWindowResult) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(done)
	subtitleThreadDPI.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2.
	window, err := createNativeSubtitleWindow()
	if err == nil {
		state := &nativeSubtitleState{commands: commands}
		subtitleWindows.Store(window, state)
		defer func() {
			if state.surface != nil {
				state.surface.close()
			}
		}()
	}
	ready <- subtitleWindowResult{window: window, err: err}
	if err != nil {
		return
	}
	defer subtitleWindows.Delete(window)
	var message subtitleMessage
	for {
		result, _, _ := subtitleGetMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) <= 0 {
			return
		}
		subtitleDispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
	}
}

// Close destroys the window and joins its message thread before shutdown.
func (s *windowsDesktopSubtitle) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	if s.window != 0 {
		subtitlePostMessage.Call(s.window, subtitleCloseMessage, 0, 0)
		<-s.done
		s.window = 0
	}
}

func createNativeSubtitleWindow() (uintptr, error) {
	var instance windows.Handle
	if err := windows.GetModuleHandleEx(2, nil, &instance); err != nil {
		return 0, err
	}
	subtitleClassOnce.Do(func() {
		class := subtitleWindowClass{Procedure: subtitleWindowProcedure, Instance: uintptr(instance), ClassName: subtitleClassName}
		class.Size = uint32(unsafe.Sizeof(class))
		if result, _, err := subtitleRegisterClass.Call(uintptr(unsafe.Pointer(&class))); result == 0 {
			subtitleClassError = fmt.Errorf("注册桌面字幕窗口失败：%w", err)
		}
	})
	if subtitleClassError != nil {
		return 0, subtitleClassError
	}
	title, _ := windows.UTF16PtrFromString("Lychee · 桌面字幕")
	window, _, err := subtitleCreateWindow.Call(subtitleWindowStyles, uintptr(unsafe.Pointer(subtitleClassName)), uintptr(unsafe.Pointer(title)), 0x80000000, 0, 0, 1, 1, 0, 0, uintptr(instance), 0)
	if window == 0 {
		return 0, fmt.Errorf("创建桌面字幕窗口失败：%w", err)
	}
	return window, nil
}

func (s *nativeSubtitleState) update(window uintptr, settings SubtitleSettings, frame SubtitleFrame, visible bool) error {
	if visible && !s.active {
		foreground, _, _ := subtitleForeground.Call()
		s.monitor, _, _ = subtitleMonitorFromWindow.Call(foreground, 2)
		dpi, _, _ := subtitleGetDPI.Call(foreground)
		s.scale = math.Max(1, float64(dpi)/96)
	}
	s.active = visible
	if !visible || (frame.Source == "" && frame.Translation == "") {
		subtitleKillTimer.Call(window, 1)
		subtitleShowWindow.Call(window, 0)
		s.visible = false
		s.settings, s.frame = settings, frame
		return nil
	}
	info, err := s.viewport(window)
	if err != nil {
		return err
	}
	margin := int32(math.Round(24 * s.scale))
	available := int(info.Work.Right - info.Work.Left - margin*2)
	if available <= 0 {
		return errors.New("显示器没有足够空间显示字幕")
	}
	surface, err := renderSubtitle(settings, available, s.scale, frame)
	if err != nil {
		return err
	}
	if s.surface != nil {
		s.surface.close()
	}
	s.surface = surface
	s.point = subtitlePosition(info.Work, surface.width, surface.height, margin, settings)
	if !s.visible || frame.Key != s.frame.Key {
		s.fadeStart = time.Now()
		subtitleSetTimer.Call(window, 1, 16, 0)
	}
	s.settings, s.frame, s.visible = settings, frame, true
	if err := s.paint(window); err != nil {
		return err
	}
	subtitleSetPosition.Call(window, ^uintptr(0), 0, 0, 0, 0, 0x53) // TOPMOST, SHOWWINDOW, NOACTIVATE, NOMOVE, NOSIZE.
	return nil
}

func (s *nativeSubtitleState) viewport(window uintptr) (subtitleMonitorInfo, error) {
	if s.monitor == 0 {
		foreground, _, _ := subtitleForeground.Call()
		s.monitor, _, _ = subtitleMonitorFromWindow.Call(foreground, 2)
		s.scale = 1
	}
	info := subtitleMonitorInfo{Size: uint32(unsafe.Sizeof(subtitleMonitorInfo{}))}
	if result, _, _ := subtitleGetMonitorInfo.Call(s.monitor, uintptr(unsafe.Pointer(&info))); result == 0 {
		// A disconnected display may invalidate the previous monitor handle.
		s.monitor, _, _ = subtitleMonitorFromWindow.Call(window, 2)
		if result, _, err := subtitleGetMonitorInfo.Call(s.monitor, uintptr(unsafe.Pointer(&info))); result == 0 {
			return info, fmt.Errorf("读取字幕显示器失败：%w", err)
		}
	}
	var dpiX, dpiY uint32
	if result, _, _ := subtitleMonitorDPI.Call(s.monitor, 0, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY))); result == 0 && dpiX != 0 {
		s.scale = float64(dpiX) / 96
	}
	return info, nil
}

func (s *nativeSubtitleState) page(window uintptr, settings SubtitleSettings, source, translation string) (SubtitlePage, error) {
	info, err := s.viewport(window)
	if err != nil {
		return SubtitlePage{}, err
	}
	available := int(info.Work.Right-info.Work.Left) - int(math.Round(48*s.scale))
	return paginateNativeSubtitle(settings, source, translation, available, s.scale)
}

func (s *nativeSubtitleState) paint(window uintptr) error {
	surface := s.surface
	size := subtitlePoint{int32(surface.width), int32(surface.height)}
	origin := subtitlePoint{}
	progress := math.Min(1, float64(time.Since(s.fadeStart))/float64(180*time.Millisecond))
	alpha := byte(math.Round(255 * (1 - math.Pow(1-progress, 3))))
	blend := [4]byte{0, 0, alpha, 1}
	result, _, err := subtitleUpdateLayer.Call(window, 0, uintptr(unsafe.Pointer(&s.point)), uintptr(unsafe.Pointer(&size)), surface.dc, uintptr(unsafe.Pointer(&origin)), 0, uintptr(unsafe.Pointer(&blend)), 2)
	if result == 0 {
		return fmt.Errorf("绘制桌面字幕失败：%w", err)
	}
	return nil
}

func subtitlePosition(work subtitleRect, width, height int, margin int32, settings SubtitleSettings) subtitlePoint {
	x, y := 50.0, 100.0
	if settings.Position == "top" {
		y = 0
	} else if settings.Position == "custom" {
		x, y = settings.CustomX, settings.CustomY
	}
	return subtitlePoint{
		work.Left + margin + int32(math.Round(float64(max(0, int(work.Right-work.Left-margin*2)-width))*x/100)),
		work.Top + margin + int32(math.Round(float64(max(0, int(work.Bottom-work.Top-margin*2)-height))*y/100)),
	}
}
