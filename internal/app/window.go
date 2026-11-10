package app

import (
	"context"
	"sync"
	"sync/atomic"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	floatingSize    = 56
	floatingOffsetX = 62
	floatingOffsetY = 10
	floatingWidth   = floatingOffsetX + floatingSize
	floatingHeight  = floatingOffsetY + floatingSize
	workspaceWidth  = 860
	workspaceHeight = 600
	mainWindowClass = "LycheeMainWindow"
)

// WindowService expands the startup floating ball into the existing workspace.
type WindowService struct {
	capture              *screenshotSession
	ctx                  context.Context
	mu                   sync.Mutex
	expanded             bool
	workspace            bool
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
	if s.capture != nil || s.workspace || s.shown {
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
	if err := s.saveFloatingPositionLocked(); err != nil {
		return err
	}
	// Windows refreshes taskbar membership when a window is hidden and shown.
	wailsruntime.WindowHide(s.ctx)
	defer wailsruntime.WindowShow(s.ctx)
	if err := clearFloatingRegion(); err != nil {
		return err
	}
	s.workspace = true
	wailsruntime.WindowSetAlwaysOnTop(s.ctx, false)
	wailsruntime.WindowSetSize(s.ctx, workspaceWidth, workspaceHeight)
	wailsruntime.WindowSetMinSize(s.ctx, workspaceWidth, workspaceHeight)
	wailsruntime.WindowCenter(s.ctx)
	return nil
}

// SetFloatingExpanded only changes clipping. Hover must never move or resize the
// WebView: doing so exposes stale compositor frames and causes ghost images.
func (s *WindowService) SetFloatingExpanded(expanded bool, revision int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.capture != nil || s.workspace || !s.shown || revision <= s.hoverRevision {
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
