//go:build !windows || !amd64

package screenshot

import (
	"errors"
)

type screenshotSession struct{}

func (s *Service) CopyScreenshot(string) error { return s.BeginScreenshot() }

func (s *Service) BeginScreenshot() error {
	return errors.New("截图暂仅支持 Windows 64 位")
}
func (s *Service) CancelScreenshot() error               { return nil }
func (s *Service) ScreenshotReady()                      {}
func (s *Service) PinScreenshot(int, int) error          { return s.BeginScreenshot() }
func (s *Service) SaveScreenshot(string) (string, error) { return "", s.BeginScreenshot() }
func (s *Service) ScrollScreenshot(int, int, int, int) (ScreenshotFrame, error) {
	return ScreenshotFrame{}, s.BeginScreenshot()
}
func (s *Service) StartHotkey() func() { return func() {} }

func (s *Service) StopScrollScreenshot() (ScreenshotFrame, error) {
	return ScreenshotFrame{}, s.BeginScreenshot()
}

func (s *Service) SetScrollScreenshotOverlay(int, int, int, int, int, int, int, int, int) error {
	return s.BeginScreenshot()
}
