//go:build windows && amd64

package app

import (
	"runtime"

	"github.com/energye/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	buildassets "github.com/yanghui1-arch/lychee/build"
)

func (s *WindowService) startTray() func() {
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		systray.Run(func() {
			systray.SetIcon(buildassets.Icon)
			systray.SetTooltip("荔枝")
			systray.SetOnClick(func(systray.IMenu) { s.showExisting() })
			systray.AddMenuItem("显示荔枝", "").Click(s.showExisting)
			systray.AddMenuItem("退出荔枝", "").Click(func() {
				s.mu.Lock()
				err := s.saveFloatingPositionLocked()
				s.mu.Unlock()
				if err != nil {
					wailsruntime.LogError(s.ctx, err.Error())
				}
				s.quitAllowed.Store(true)
				wailsruntime.Quit(s.ctx)
			})
		}, func() { close(done) })
	}()
	return func() {
		systray.Quit()
		<-done
	}
}
