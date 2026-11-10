package app

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/yanghui1-arch/lychee/internal/audio"
	"github.com/yanghui1-arch/lychee/internal/speech"
	"github.com/yanghui1-arch/lychee/internal/subtitle"
	"github.com/yanghui1-arch/lychee/internal/translation"
)

// Run starts the desktop application with the embedded frontend assets.
func Run(assets fs.FS) error {
	parentPID := 0
	if value := os.Getenv("LYCHEE_WORKSPACE_PARENT"); value != "" {
		var err error
		parentPID, err = strconv.Atoi(value)
		if err != nil || parentPID <= 0 {
			return fmt.Errorf("无效的工作区父进程: %q", value)
		}
	}
	translationHandler, err := translation.NewHandler(translation.ConfiguredEndpoint())
	if err != nil {
		return err
	}
	subtitles := subtitle.NewController()
	audio := audio.NewController(subtitles, speech.NewPipeline)
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	window := &WindowService{
		workspace:            parentPID != 0,
		floatingPositionPath: filepath.Join(configDir, "lychee", "floating-position.json"),
	}

	var stopPresentation context.CancelFunc
	var stopTray func()
	var stopScreenshotHotkey func()
	appOptions := &options.App{
		Bind:          []interface{}{audio.AudioService, subtitles.SubtitleService, window},
		OnBeforeClose: window.beforeClose,
		OnStartup: func(ctx context.Context) {
			window.mu.Lock()
			window.ctx = ctx
			window.mu.Unlock()
			window.initScreenshot()
			if window.workspace {
				go window.watchWorkspaceParent(ctx, parentPID)
			} else {
				stopTray = window.startTray()
				stopScreenshotHotkey = window.screenshot.StartHotkey()
			}
			emit := func(event string, data any) { wailsruntime.EventsEmit(ctx, event, data) }
			audio.Start(ctx, emit)
			presentationContext, cancel := context.WithCancel(ctx)
			stopPresentation = cancel
			subtitles.Start(presentationContext, emit)
		},
		OnShutdown: func(context.Context) {
			if stopScreenshotHotkey != nil {
				stopScreenshotHotkey()
			}
			if stopTray != nil {
				stopTray()
			}
			audio.StopAudioCapture()
			if stopPresentation != nil {
				stopPresentation()
			}
			subtitles.Close()
		},
		Title:         "Lychee",
		Width:         floatingWidth,
		Height:        floatingHeight,
		MinWidth:      floatingWidth,
		MinHeight:     floatingHeight,
		DisableResize: true,
		Frameless:     true,
		AlwaysOnTop:   true,
		StartHidden:   true,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "lychee.desktop",
			OnSecondInstanceLaunch: func(options.SecondInstanceData) {
				window.showExisting()
			},
		},
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: translationHandler,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 0},
		Windows: &windows.Options{
			WindowClassName:                   mainWindowClass,
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			BackdropType:                      windows.None,
			DisableFramelessWindowDecorations: true,
		},
	}
	if window.workspace {
		appOptions.Width, appOptions.Height = workspaceWidth, workspaceHeight
		appOptions.MinWidth, appOptions.MinHeight = workspaceWidth, workspaceHeight
		appOptions.AlwaysOnTop = false
		appOptions.Windows.WindowClassName = workspaceWindowClass
		appOptions.SingleInstanceLock.UniqueId += ".workspace"
	}
	return wails.Run(appOptions)
}
