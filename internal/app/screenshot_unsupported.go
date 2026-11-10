//go:build !windows || !amd64

package app

import "errors"

type screenshotSession struct{}

func (s *WindowService) CopyScreenshot(string) error { return s.BeginScreenshot() }

func (s *WindowService) BeginScreenshot() error {
	return errors.New("截图暂仅支持 Windows 64 位")
}
func (s *WindowService) CancelScreenshot() error               { return nil }
func (s *WindowService) ScreenshotReady()                      {}
func (s *WindowService) PinScreenshot(int, int) error          { return s.BeginScreenshot() }
func (s *WindowService) SaveScreenshot(string) (string, error) { return "", s.BeginScreenshot() }
func (s *WindowService) ScrollScreenshot(int, int, int, int) (ScreenshotFrame, error) {
	return ScreenshotFrame{}, s.BeginScreenshot()
}
func (s *WindowService) startScreenshotHotkey() func() { return func() {} }
