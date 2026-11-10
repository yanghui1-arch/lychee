package audio

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yanghui1-arch/lychee/internal/speech"
	"github.com/yanghui1-arch/lychee/internal/subtitle"
)

type finishingSpeech struct {
	mu       sync.Mutex
	audio    []byte
	finished bool
}

func (p *finishingSpeech) Push(pcm []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.audio = append(p.audio, pcm...)
	return nil
}
func (p *finishingSpeech) Finish() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.finished = true
	return nil
}

func TestAudioStopsAndFinishesSpeech(t *testing.T) {
	a := newAudioService()
	a.subtitles = subtitle.NewController()
	a.subtitles = subtitle.NewController(&recordingSubtitle{})
	a.list = func() ([]AudioSource, error) { return []AudioSource{{ID: "system"}}, nil }
	p := &finishingSpeech{}
	a.startSpeech = func(context.Context, speech.Settings, func(speech.Event), func(error)) (speech.Pipeline, error) {
		return p, nil
	}
	a.capture = func(ctx context.Context, _ AudioSource, ready func(), consume func([]byte)) error {
		ready()
		consume([]byte{1, 0, 2, 0})
		<-ctx.Done()
		return nil
	}
	if _, err := a.StartAudioCapture("system"); err != nil {
		t.Fatal(err)
	}
	a.StopAudioCapture()
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.finished || len(p.audio) != 4 {
		t.Fatal("stop did not drain speech pipeline")
	}
}

func TestAudioSpeechStartupFailureLeavesErrorState(t *testing.T) {
	a := newAudioService()
	a.subtitles = subtitle.NewController()
	a.subtitles = subtitle.NewController(&recordingSubtitle{})
	a.list = func() ([]AudioSource, error) { return []AudioSource{{ID: "system"}}, nil }
	a.startSpeech = func(context.Context, speech.Settings, func(speech.Event), func(error)) (speech.Pipeline, error) {
		return nil, errors.New("请配置登录凭证")
	}
	a.capture = func(context.Context, AudioSource, func(), func([]byte)) error {
		t.Error("capture started before speech initialization succeeded")
		return nil
	}
	status, err := a.StartAudioCapture("system")
	if err == nil || status.State != "error" || status.Error != "请配置登录凭证" {
		t.Fatalf("startup failure left stale UI state: %+v %v", status, err)
	}
	a.StopAudioCapture()
}

func TestSpeechInterruptionDoesNotStopCapture(t *testing.T) {
	a := newAudioService()
	a.subtitles = subtitle.NewController()
	a.subtitles = subtitle.NewController(&recordingSubtitle{})
	a.list = func() ([]AudioSource, error) { return []AudioSource{{ID: "system"}}, nil }
	var receive func(speech.Event)
	a.startSpeech = func(_ context.Context, _ speech.Settings, callback func(speech.Event), _ func(error)) (speech.Pipeline, error) {
		receive = callback
		return &finishingSpeech{}, nil
	}
	notice := make(chan string, 1)
	a.emit = func(event string, value any) {
		if event == "audio:notice" {
			notice <- value.(string)
		}
	}
	a.capture = func(ctx context.Context, _ AudioSource, ready func(), _ func([]byte)) error {
		ready()
		receive(speech.Event{Type: "translation_preview", SessionID: "failed", Sequence: 1, TranslatedText: "未完成的译文"})
		receive(speech.Event{Type: "session_interrupted"})
		<-ctx.Done()
		return nil
	}
	defer a.StopAudioCapture()
	if _, err := a.StartAudioCapture("system"); err != nil {
		t.Fatal(err)
	}
	select {
	case message := <-notice:
		if !strings.Contains(message, "可能不完整") {
			t.Fatal("recovery hid the possible transcript gap")
		}
	case <-time.After(time.Second):
		t.Fatal("missing nonfatal recovery notice")
	}
	if status := a.GetAudioStatus(); status.State != "listening" || status.Error != "" {
		t.Fatalf("single-session failure stopped capture: %+v", status)
	}
	a.subtitles.PresentAt(time.Now())
	if !a.subtitles.GetSubtitleFrame().Incomplete {
		t.Fatal("unconfirmed subtitle not preserved")
	}
}

type lateFinishingSpeech struct {
	finishingSpeech
	receive func(speech.Event)
}

func (p *lateFinishingSpeech) Finish() error {
	p.receive(speech.Event{Type: "translation", SessionID: "old", Sequence: 2, TranslatedText: "停止时迟到的译文"})
	return p.finishingSpeech.Finish()
}

func TestAudioManualStopClearsAndRejectsFinishingResult(t *testing.T) {
	a := newAudioService()
	a.subtitles = subtitle.NewController()
	window := &recordingSubtitle{}
	a.subtitles = subtitle.NewController(window)
	a.list = func() ([]AudioSource, error) { return []AudioSource{{ID: "system"}}, nil }
	var receive func(speech.Event)
	a.startSpeech = func(_ context.Context, _ speech.Settings, callback func(speech.Event), _ func(error)) (speech.Pipeline, error) {
		receive = callback
		return &lateFinishingSpeech{receive: callback}, nil
	}
	queued := make(chan struct{})
	a.capture = func(ctx context.Context, _ AudioSource, ready func(), _ func([]byte)) error {
		ready()
		receive(speech.Event{Type: "translation", SessionID: "old", Sequence: 1, TranslatedText: strings.Repeat("旧视频", 50)})
		close(queued)
		<-ctx.Done()
		return nil
	}
	if _, err := a.StartAudioCapture("system"); err != nil {
		t.Fatal(err)
	}
	<-queued
	a.subtitles.PresentAt(time.Now())
	if !window.visible {
		t.Fatal("test did not show the queued subtitle")
	}
	a.StopAudioCapture()
	if a.GetAudioStatus().State != "idle" || a.subtitles.GetSubtitleFrame().Key != "" || window.visible {
		t.Fatal("manual stop drained old captions or accepted a late finish result")
	}
}
func TestDesktopSubtitleFollowsAudioLifecycle(t *testing.T) {
	window := &recordingSubtitle{}
	subtitles := subtitle.NewController(window)
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
	defer subtitles.Close()
	settings := subtitles.SnapshotSettings()
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
	subtitles.Close()
	if !window.closed {
		t.Fatal("shutdown did not release caption resources")
	}
}

type recordingSubtitle struct {
	settings        subtitle.SubtitleSettings
	visible, closed bool
}

func (w *recordingSubtitle) Update(settings subtitle.SubtitleSettings, _ subtitle.SubtitleFrame, visible bool) error {
	w.settings = settings
	w.visible = visible
	return nil
}
func (w *recordingSubtitle) Close() { w.closed = true; w.visible = false }
func (*recordingSubtitle) Page(_ subtitle.SubtitleSettings, source, translation string) (subtitle.SubtitlePage, error) {
	return subtitle.SubtitlePage{Source: source, Translation: translation, SourceBytes: len(source), TranslationBytes: len(translation)}, nil
}
