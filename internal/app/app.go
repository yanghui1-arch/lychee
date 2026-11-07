package app

import (
	"context"
	"io/fs"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Run starts the desktop application with the embedded frontend assets.
func Run(assets fs.FS) error {
	translationHandler, err := newTranslationHandler(configuredTranslationEndpoint())
	if err != nil {
		return err
	}
	audio := newAudioService()
	window := &WindowService{
		closePreferencePath:  defaultClosePreferencePath(),
		floatingPositionPath: filepath.Join(filepath.Dir(defaultClosePreferencePath()), "floating-position.json"),
	}
	subtitles := newSubtitleService()
	audio.subtitles = subtitles
	audio.startSpeech = newSpeechPipeline
	var stopPresentation context.CancelFunc
	return wails.Run(&options.App{
		Bind:          []interface{}{audio, subtitles, window},
		OnBeforeClose: window.beforeClose,
		OnStartup: func(ctx context.Context) {
			window.mu.Lock()
			window.ctx = ctx
			window.mu.Unlock()
			audio.ctx = ctx
			audio.emit = func(event string, data any) { wailsruntime.EventsEmit(ctx, event, data) }
			presentationContext, cancel := context.WithCancel(ctx)
			stopPresentation = cancel
			subtitles.start(presentationContext, audio.emit)
		},
		OnShutdown: func(context.Context) {
			audio.StopAudioCapture()
			if stopPresentation != nil {
				stopPresentation()
			}
			subtitles.close()
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
