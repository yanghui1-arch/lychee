package app

import (
	"context"
	"io/fs"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Run starts the desktop application with the embedded frontend assets.
func Run(assets fs.FS) error {
	translationHandler, err := newTranslationHandler(configuredTranslationEndpoint())
	if err != nil {
		return err
	}
	audio := newAudioService()
	subtitles := newSubtitleService()
	audio.subtitles = subtitles
	audio.startSpeech = newSpeechPipeline
	var stopPresentation context.CancelFunc
	return wails.Run(&options.App{
		Bind: []interface{}{audio, subtitles},
		OnStartup: func(ctx context.Context) {
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
		Width:         860,
		Height:        600,
		MinWidth:      860,
		MinHeight:     600,
		DisableResize: true,
		Frameless:     true,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: translationHandler,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 255},
	})
}
