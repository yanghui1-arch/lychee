//go:build windows && amd64

package app

import (
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var applicationUser32 = windows.NewLazySystemDLL("user32.dll")
var applicationWindowStyle = applicationUser32.NewProc("GetWindowLongPtrW")
var applicationWindowClass = applicationUser32.NewProc("GetClassNameW")
var applicationWindowText = applicationUser32.NewProc("GetWindowTextW")
var applicationWindowAttribute = windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmGetWindowAttribute")

type desktopApplicationWindow struct {
	pid   uint32
	title string
}

type desktopApplicationSet struct {
	processes map[uint32]bool
	windows   []desktopApplicationWindow
}

var enumerateDesktopApplication = syscall.NewCallback(func(window windows.HWND, applications *desktopApplicationSet) uintptr {
	if !windows.IsWindowVisible(window) {
		return 1
	}
	var cloaked uint32
	result, _, _ := applicationWindowAttribute.Call(uintptr(window), 14, uintptr(unsafe.Pointer(&cloaked)), unsafe.Sizeof(cloaked)) // DWMWA_CLOAKED.
	if result == 0 && cloaked != 0 {
		return 1
	}
	style, _, _ := applicationWindowStyle.Call(uintptr(window), ^uintptr(19)) // GWL_EXSTYLE = -20.
	if style&0x80 != 0 {
		return 1
	} // WS_EX_TOOLWINDOW: tooltips and utility windows.
	var class [128]uint16
	applicationWindowClass.Call(uintptr(window), uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
	switch windows.UTF16ToString(class[:]) {
	case "Progman", "WorkerW", "Shell_TrayWnd", "Shell_SecondaryTrayWnd":
		return 1
	}
	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(window, &pid); err == nil && pid != 0 {
		applications.processes[pid] = true
		var title [1024]uint16
		applicationWindowText.Call(uintptr(window), uintptr(unsafe.Pointer(&title[0])), uintptr(len(title)))
		applications.windows = append(applications.windows, desktopApplicationWindow{pid: pid, title: strings.TrimSpace(windows.UTF16ToString(title[:]))})
	}
	return 1
})

func desktopApplicationProcesses() (*desktopApplicationSet, error) {
	applications := &desktopApplicationSet{processes: make(map[uint32]bool)}
	err := windows.EnumWindows(enumerateDesktopApplication, unsafe.Pointer(applications))
	runtime.KeepAlive(applications)
	return applications, err
}

// Filter infrastructure from both window discovery and active audio sessions.
// Do not filter all Microsoft-signed software or all WindowsApps packages:
// those also contain user applications such as Teams and Store-installed apps.
func userAudioApplication(exe, path, windowsDirectory string) bool {
	switch strings.ToLower(exe) {
	case "applicationframehost.exe", "textinputhost.exe", "inputapp.exe",
		"explorer.exe", "shellexperiencehost.exe", "startmenuexperiencehost.exe",
		"searchhost.exe", "searchapp.exe", "searchui.exe", "runtimebroker.exe",
		"systemsettings.exe", "lockapp.exe", "dwm.exe", "ctfmon.exe",
		"sihost.exe", "taskhostw.exe", "dllhost.exe", "svchost.exe",
		"audiodg.exe", "rtkuwp.exe", "rtkngui64.exe", "rtkngui.exe":
		return false
	}
	if windowsDirectory != "" {
		for _, directory := range []string{"SystemApps", "ImmersiveControlPanel"} {
			prefix := strings.ToLower(filepath.Join(windowsDirectory, directory)) + string(filepath.Separator)
			if strings.HasPrefix(strings.ToLower(filepath.Clean(path)), prefix) {
				return false
			}
		}
	}
	return true
}
