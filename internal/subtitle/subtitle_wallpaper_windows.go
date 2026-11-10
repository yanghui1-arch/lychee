//go:build windows

package subtitle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	wallpaperUser32         = windows.NewLazySystemDLL("user32.dll")
	wallpaperSystemParamsW  = wallpaperUser32.NewProc("SystemParametersInfoW")
	wallpaperGetSystemColor = wallpaperUser32.NewProc("GetSysColor")
)

const (
	spiGetDeskWallpaper = 0x0073
	colorDesktop        = 1
)

// wallpaperPaths lists candidate wallpaper files, best first. The configured
// path may be a format the standard decoders do not read (BMP, WebP), so the
// JPEG Windows transcodes for display is kept as a fallback.
func wallpaperPaths() []string {
	var paths []string
	buffer := make([]uint16, 1024)
	if ok, _, _ := wallpaperSystemParamsW.Call(spiGetDeskWallpaper, uintptr(len(buffer)), uintptr(unsafe.Pointer(&buffer[0])), 0); ok != 0 {
		path := windows.UTF16ToString(buffer)
		if path == "" {
			// A solid-color desktop; a leftover transcoded file would be stale.
			return nil
		}
		paths = append(paths, path)
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		transcoded := filepath.Join(appData, "Microsoft", "Windows", "Themes", "TranscodedWallpaper")
		if len(paths) == 0 || !strings.EqualFold(paths[0], transcoded) {
			paths = append(paths, transcoded)
		}
	}
	return paths
}

func desktopColor() string {
	value, _, _ := wallpaperGetSystemColor.Call(colorDesktop)
	return fmt.Sprintf("#%02x%02x%02x", value&0xff, value>>8&0xff, value>>16&0xff)
}
