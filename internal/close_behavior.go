package app

import (
	"context"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Closing a window keeps the app running; only the tray exit permits quitting.
func (s *WindowService) RequestClose() error {
	s.mu.Lock()
	if s.screenshot.Active() {
		s.mu.Unlock()
		return s.CancelScreenshot()
	}
	defer s.mu.Unlock()
	if s.ctx == nil {
		return nil
	}
	if s.workspace {
		wailsruntime.WindowHide(s.ctx)
		return nil
	}
	err := s.saveFloatingPositionLocked()
	wailsruntime.WindowMinimise(s.ctx)
	return err
}

func (s *WindowService) beforeClose(ctx context.Context) bool {
	if s.quitAllowed.Load() {
		return false
	}
	go func() {
		if err := s.RequestClose(); err != nil {
			wailsruntime.LogError(ctx, err.Error())
		}
	}()
	return true
}
