package main

import (
	"context"
	"errors"
	"math"
	"testing"
)

type recordingSubtitle struct {
	settings SubtitleSettings
	visible  bool
	updates  int
	closed   bool
}

func (w *recordingSubtitle) Update(settings SubtitleSettings, visible bool) error {
	w.settings, w.visible = settings, visible
	w.updates++
	return nil
}
func (w *recordingSubtitle) Close() { w.closed = true; w.visible = false }

func TestDesktopSubtitleFollowsAudioLifecycle(t *testing.T) {
	window := &recordingSubtitle{}
	subtitles := newSubtitleService()
	subtitles.window = window
	audio := newAudioService()
	audio.subtitles = subtitles
	audio.list = func() ([]AudioSource, error) { return []AudioSource{{ID: "system", Kind: "system"}}, nil }
	fail := make(chan struct{})
	audio.capture = func(ctx context.Context, _ AudioSource, ready func(), _ func([]byte)) error {
		ready()
		select {
		case <-ctx.Done():
			return nil
		case <-fail:
			return errors.New("device disconnected")
		}
	}
	defer audio.StopAudioCapture()
	defer subtitles.close()
	settings := subtitles.settings
	settings.FontSize, settings.Style, settings.Position = 38, "paper", "top"
	if err := subtitles.Configure(settings); err != nil {
		t.Fatal(err)
	}
	if window.visible {
		t.Fatal("configuration showed captions before the switch was enabled")
	}
	if _, err := audio.StartAudioCapture("system"); err != nil {
		t.Fatal(err)
	}
	if !window.visible || window.settings != settings {
		t.Fatal("starting audio did not show captions with saved settings")
	}
	settings.Style, settings.Position, settings.CustomX, settings.CustomY = "mist", "custom", 20, 35
	if err := subtitles.Configure(settings); err != nil {
		t.Fatal(err)
	}
	if !window.visible || window.settings != settings {
		t.Fatal("live configuration did not update visible captions")
	}
	close(fail)
	<-audio.done
	if window.visible {
		t.Fatal("captions remained visible after capture failure")
	}
	if _, err := audio.StartAudioCapture("system"); err != nil {
		t.Fatal(err)
	}
	audio.StopAudioCapture()
	if window.visible {
		t.Fatal("stopping audio did not hide captions")
	}
	subtitles.close()
	if !window.closed {
		t.Fatal("shutdown did not release caption resources")
	}
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
