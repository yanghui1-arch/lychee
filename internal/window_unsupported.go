//go:build !windows || !amd64

package app

import (
	"github.com/yanghui1-arch/lychee/internal/screenshot"
)

func prepareFloatingWindow() error         { return nil }
func setFloatingRegion(bool) error         { return nil }
func clearFloatingRegion() error           { return nil }
func moveFloatingWindow(int, int) error    { return nil }
func (s *WindowService) startTray() func() { return func() {} }

func (s *WindowService) initScreenshot() {
	s.screenshot = screenshot.New(s.ctx, &s.mu, screenshot.WindowHooks{})
}
