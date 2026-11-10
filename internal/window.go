package app

import (
	"context"
	"os"
	"sync"
	"sync/atomic"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/yanghui1-arch/lychee/internal/screenshot"
)

const (
	floatingSize         = 56
	floatingOffsetX      = 62
	floatingOffsetY      = 10
	floatingWidth        = floatingOffsetX + floatingSize
	floatingHeight       = floatingOffsetY + floatingSize
	workspaceWidth       = 860
	workspaceHeight      = 600
	mainWindowClass      = "LycheeMainWindow"
	workspaceWindowClass = "LycheeWorkspaceWindow"
)

// WindowService keeps the floating ball separate from the reusable workspace.
type WindowService struct {
	screenshot           *screenshot.Service
	ctx                  context.Context
	mu                   sync.Mutex
	expanded             bool
	workspace            bool
	workspaceProcess     *os.Process
	shown                bool
	hoverRevision        int
	quitAllowed          atomic.Bool
	floatingPositionPath string
}

func (s *WindowService) showExisting() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx != nil {
		wailsruntime.WindowUnminimise(s.ctx)
		wailsruntime.WindowShow(s.ctx)
	}
}

// ShowFloating waits for the SVG to load before revealing the transparent window.
func (s *WindowService) ShowFloating() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.screenshot.Active() || s.workspace || s.shown {
		return nil
	}
	if err := prepareFloatingWindow(); err != nil {
		return err
	}
	wailsruntime.WindowSetSize(s.ctx, floatingWidth, floatingHeight)
	if err := s.restoreFloatingPosition(); err != nil {
		return err
	}
	if err := setFloatingRegion(false); err != nil {
		return err
	}
	s.shown = true
	wailsruntime.WindowShow(s.ctx)
	return nil
}

func (s *WindowService) OpenWorkspace() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.workspace {
		return nil
	}
	if s.screenshot.Active() {
		return nil
	}
	return s.openWorkspace()
}

func (s *WindowService) IsWorkspace() bool { return s.workspace }

// Reveal the workspace only after its React view has painted.
func (s *WindowService) ShowWorkspace() {
	if s.workspace {
		wailsruntime.WindowShow(s.ctx)
	}
}

// SetFloatingExpanded only changes clipping. Hover must never move or resize the
// WebView: doing so exposes stale compositor frames and causes ghost images.
func (s *WindowService) SetFloatingExpanded(expanded bool, revision int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.screenshot.Active() || s.workspace || !s.shown || revision <= s.hoverRevision {
		return nil
	}
	s.hoverRevision = revision
	if s.expanded == expanded {
		return nil
	}
	if err := setFloatingRegion(expanded); err != nil {
		return err
	}
	s.expanded = expanded
	return nil
}

func (s *WindowService) BeginScreenshot() error {
	if s.workspace {
		return nil
	}
	return s.screenshot.BeginScreenshot()
}
func (s *WindowService) CancelScreenshot() error          { return s.screenshot.CancelScreenshot() }
func (s *WindowService) ScreenshotReady()                 { s.screenshot.ScreenshotReady() }
func (s *WindowService) CopyScreenshot(data string) error { return s.screenshot.CopyScreenshot(data) }
func (s *WindowService) SaveScreenshot(data string) (string, error) {
	return s.screenshot.SaveScreenshot(data)
}
func (s *WindowService) PinScreenshot(width, height int) error {
	return s.screenshot.PinScreenshot(width, height)
}
func (s *WindowService) ScrollScreenshot(x, y, width, height int) (screenshot.ScreenshotFrame, error) {
	return s.screenshot.ScrollScreenshot(x, y, width, height)
}
