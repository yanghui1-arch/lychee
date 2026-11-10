package audio

import (
	"context"

	"github.com/yanghui1-arch/lychee/internal/speech"
	"github.com/yanghui1-arch/lychee/internal/subtitle"
)

// Controller owns audio startup; only AudioService is bound to Wails.
type Controller struct{ *AudioService }

func NewController(subtitles *subtitle.Controller, start speech.StartFunc) *Controller {
	service := newAudioService()
	service.subtitles = subtitles
	service.startSpeech = start
	return &Controller{service}
}
func (c *Controller) Start(ctx context.Context, emit func(string, any)) { c.ctx = ctx; c.emit = emit }
