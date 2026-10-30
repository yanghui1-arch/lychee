//go:build windows && amd64

package app

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestApplicationIcons(t *testing.T) {
	// Shell initializes per-thread USER resources; measure icon cleanup on one thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	system, err := windows.GetSystemDirectory()
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{filepath.Join(system, "WindowsPowerShell", "v1.0", "powershell.exe"), filepath.Join(system, "cmd.exe")}
	icons := make([]string, 0, len(paths))
	for _, path := range paths {
		data := applicationIconFromPath(path)
		if !strings.HasPrefix(data, "data:image/png;base64,") {
			t.Fatalf("icon missing for %s", path)
		}
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, "data:image/png;base64,"))
		if err != nil {
			t.Fatal(err)
		}
		icon, err := png.Decode(bytes.NewReader(decoded))
		if err != nil {
			t.Fatal(err)
		}
		if icon.Bounds().Dx() != 32 || icon.Bounds().Dy() != 32 {
			t.Fatal("unexpected icon dimensions")
		}
		visible, transparent := false, false
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				_, _, _, alpha := icon.At(x, y).RGBA()
				visible = visible || alpha > 0
				transparent = transparent || alpha == 0
			}
		}
		if !visible || !transparent {
			t.Fatal("icon pixels or transparency lost")
		}
		if data != applicationIconFromPath(path) {
			t.Fatal("cached icon changed")
		}
		icons = append(icons, data)
	}
	if icons[0] == icons[1] {
		t.Fatal("different applications have the same icon")
	}
	processCommand := exec.Command(paths[0], "-NoProfile", "-NonInteractive", "-Command", "Start-Sleep -Seconds 10")
	processCommand.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	if err := processCommand.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = processCommand.Process.Kill(); _ = processCommand.Wait() })
	source, ok := applicationAudioSource(uint32(processCommand.Process.Pid), filepath.Base(paths[0]), "")
	if !ok || source.Icon != icons[0] {
		t.Fatal("application audio source icon does not match its executable")
	}
	if applicationIconFromPath(filepath.Join(t.TempDir(), "missing.exe")) != "" {
		t.Fatal("missing icon did not fall back")
	}
	getResources := iconUser32.NewProc("GetGuiResources")
	process := windows.CurrentProcess()
	beforeGDI, _, _ := getResources.Call(uintptr(process), 0)
	beforeUSER, _, _ := getResources.Call(uintptr(process), 1)
	for i := 0; i < 20; i++ {
		if readApplicationIcon(paths[0]) == "" {
			t.Fatal("repeated extraction failed")
		}
	}
	afterGDI, _, _ := getResources.Call(uintptr(process), 0)
	afterUSER, _, _ := getResources.Call(uintptr(process), 1)
	if afterGDI > beforeGDI || afterUSER > beforeUSER {
		t.Fatalf("icon handles leaked: GDI %d->%d USER %d->%d", beforeGDI, afterGDI, beforeUSER, afterUSER)
	}
	if unsafe.Sizeof(iconBitmapInfo{}) != 44 {
		t.Fatal("bitmap info layout does not match Win32")
	}
}
