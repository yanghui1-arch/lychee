package subtitle

import (
	"math"
	"testing"
)

type recordingSubtitle struct {
	settings SubtitleSettings
	visible  bool
	updates  int
	closed   bool
}

func sampleSubtitleFrame(settings SubtitleSettings) SubtitleFrame {
	return SubtitleFrame{Key: "test", Source: subtitlePlaceholderText(settings.Source), Translation: subtitlePlaceholderText(settings.Target), Final: true}
}

func (w *recordingSubtitle) Update(settings SubtitleSettings, _ SubtitleFrame, visible bool) error {
	w.settings, w.visible = settings, visible
	w.updates++
	return nil
}
func (w *recordingSubtitle) Close() { w.closed = true; w.visible = false }
func (*recordingSubtitle) Page(_ SubtitleSettings, source, translation string) (SubtitlePage, error) {
	return readingTestPaginator(source, translation)
}

func TestSubtitleSettingsValidation(t *testing.T) {
	subtitles := newSubtitleService()
	window := &recordingSubtitle{}
	subtitles.window = window
	for _, change := range []func(*SubtitleSettings){
		func(s *SubtitleSettings) { s.Source = "invalid" },
		func(s *SubtitleSettings) { s.Target = "invalid" },
		func(s *SubtitleSettings) { s.FontSize = 100 },
		func(s *SubtitleSettings) { s.Style = "invalid" },
		func(s *SubtitleSettings) { s.Position = "invalid" },
		func(s *SubtitleSettings) { s.CustomX = math.NaN() },
		func(s *SubtitleSettings) { s.CustomX = math.Inf(1) },
		func(s *SubtitleSettings) { s.CustomY = math.Inf(-1) },
		func(s *SubtitleSettings) { s.CustomX = -1 },
		func(s *SubtitleSettings) { s.CustomY = 101 },
	} {
		settings := subtitles.settings
		change(&settings)
		if err := subtitles.Configure(settings); err == nil {
			t.Fatalf("invalid settings accepted: %+v", settings)
		}
	}
	if window.updates != 0 {
		t.Fatal("invalid settings reached the renderer")
	}
}
