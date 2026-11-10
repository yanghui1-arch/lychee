package app

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

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
		floatingPositionPath: filepath.Join(configDir, "lychee", "floating-position.json"),
	}

	var stopPresentation context.CancelFunc
	var stopTray func()
	var stopScreenshotHotkey func()
	return wails.Run(&options.App{
		Bind:          []interface{}{audio.AudioService, subtitles.SubtitleService, window},
		OnBeforeClose: window.beforeClose,
		OnStartup: func(ctx context.Context) {
			window.mu.Lock()
			window.ctx = ctx
			window.mu.Unlock()
			stopTray = window.startTray()
			window.initScreenshot()
			stopScreenshotHotkey = window.screenshot.StartHotkey()
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
	})
}
