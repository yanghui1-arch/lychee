//go:build windows && amd64

package app

import (
	"errors"
	"sync"
	"unsafe"

	"github.com/yanghui1-arch/lychee/frontend"
)

var subtitleRobotoOnce sync.Once
var subtitleRobotoError error

func loadSubtitleRoboto() error {
	subtitleRobotoOnce.Do(func() {
		font, err := frontend.SubtitleFonts.ReadFile("src/assets/fonts/Roboto-Medium.ttf")
		if err != nil {
			subtitleRobotoError = err
			return
		}
		var count uint32
		addFont := subtitleGDI32.NewProc("AddFontMemResourceEx")
		handle, _, _ := addFont.Call(uintptr(unsafe.Pointer(&font[0])), uintptr(len(font)), 0, uintptr(unsafe.Pointer(&count)))
		if handle == 0 || count == 0 {
			subtitleRobotoError = errors.New("无法加载内置字幕字体")
		}
		// This font is private to this process and is automatically unloaded
		// by Windows at process exit. Keep it available across caption toggles.
	})
	return subtitleRobotoError
}
