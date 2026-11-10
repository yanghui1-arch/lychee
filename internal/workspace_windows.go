//go:build windows && amd64

package app

import (
	"context"
	"os"
	"os/exec"
	"strconv"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

// Wails v2 owns one WebView window per process. Keep its existing bindings in a
// single reusable workspace process; the original window remains the ball/editor.
// The caller holds s.mu, including while recording a newly started process.
func (s *WindowService) openWorkspace() error {
	if s.workspaceProcess != nil {
		hwnd := processWindowHandle(s.workspaceProcess.Pid, workspaceWindowClass)
		if hwnd != 0 {
			command := uintptr(5) // SW_SHOW preserves an already visible window.
			if minimized, _, _ := windowUser32.NewProc("IsIconic").Call(uintptr(hwnd)); minimized != 0 {
				command = 9 // SW_RESTORE
			}
			windowUser32.NewProc("ShowWindow").Call(uintptr(hwnd), command)
			windowUser32.NewProc("SetForegroundWindow").Call(uintptr(hwnd))
		}
		return nil // The first request may still be loading its WebView.
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	command := exec.Command(executable)
	command.Env = append(os.Environ(), "LYCHEE_WORKSPACE_PARENT="+strconv.Itoa(os.Getpid()))
	if err := command.Start(); err != nil {
		return err
	}
	s.workspaceProcess = command.Process
	go func() {
		err := command.Wait()
		s.mu.Lock()
		s.workspaceProcess = nil
		s.mu.Unlock()
		if err != nil {
			wailsruntime.LogError(s.ctx, "工作区已退出: "+err.Error())
		}
	}()
	return nil
}

// Exiting from the owner's tray also ends the workspace, including when the
// owner crashes. Watching the process handle avoids PID-reuse races.
func (s *WindowService) watchWorkspaceParent(ctx context.Context, pid int) {
	parent, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err == nil {
		defer windows.CloseHandle(parent)
		for {
			status, err := windows.WaitForSingleObject(parent, 500)
			if err != nil || status != uint32(windows.WAIT_TIMEOUT) {
				break
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}
	s.quitAllowed.Store(true)
	wailsruntime.Quit(ctx)
}
