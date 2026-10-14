package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	translationHandler, err := newTranslationHandler(configuredTranslationEndpoint())
	if err != nil {
		println("Error:", err.Error())
		return
	}
	audio := newAudioService()
	err = wails.Run(&options.App{
		Bind: []interface{}{audio},
		OnStartup: func(ctx context.Context) {
			audio.ctx = ctx
			audio.emit = func(event string, data any) { wailsruntime.EventsEmit(ctx, event, data) }
		},
		OnShutdown:    func(context.Context) { audio.StopAudioCapture() },
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

	if err != nil {
		println("Error:", err.Error())
	}
}
