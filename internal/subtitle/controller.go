package subtitle

import (
	"context"
	"time"

	"github.com/yanghui1-arch/lychee/internal/speech"
)

// Controller owns the subtitle lifecycle; only SubtitleService is bound to Wails.
type Controller struct{ *SubtitleService }

func NewController(renderer ...DesktopSubtitle) *Controller {
	service := newSubtitleService()
	if len(renderer) > 0 {
		service.window = renderer[0]
	}
	return &Controller{service}
}
func (c *Controller) Start(ctx context.Context, emit func(string, any)) { c.start(ctx, emit) }
func (c *Controller) Close()                                            { c.close() }
func (c *Controller) SetVisible(visible bool) error                     { return c.setVisible(visible) }
func (c *Controller) Reset() uint64                                     { return c.reset() }
func (c *Controller) Discard()                                          { c.discard() }
func (c *Controller) NewAudioSession(ctx context.Context, start speech.StartFunc, failed func(error), notice func(), now func() time.Time) *AudioSession {
	return &AudioSession{ctx: ctx, subtitles: c.SubtitleService, start: start, failed: failed, notice: notice, now: now}
}
func (s *AudioSession) Open(generation uint64) error { return s.open(generation) }

// SnapshotSettings returns the current settings under the service lock.
func (c *Controller) SnapshotSettings() SubtitleSettings { return c.snapshotSettings() }

// PresentAt advances the renderer using the supplied presentation clock.
func (c *Controller) PresentAt(now time.Time) { c.presentAt(now) }
