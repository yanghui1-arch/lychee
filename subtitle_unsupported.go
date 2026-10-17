//go:build !windows || !amd64

package main

import "errors"

type unsupportedDesktopSubtitle struct{}

func newDesktopSubtitle() desktopSubtitle { return &unsupportedDesktopSubtitle{} }
func (*unsupportedDesktopSubtitle) Update(_ SubtitleSettings, _ SubtitleFrame, visible bool) error {
	if visible {
		return errors.New("桌面悬浮字幕暂仅支持 Windows 64 位")
	}
	return nil
}
func (*unsupportedDesktopSubtitle) Close() {}
func (*unsupportedDesktopSubtitle) Page(_ SubtitleSettings, source, translation string) (subtitlePage, error) {
	return splitSubtitlePage(source, translation, func(_ int, text string) bool { return subtitleReadingUnits(text) <= subtitlePageUnits })
}
