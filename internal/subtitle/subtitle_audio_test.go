package subtitle

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yanghui1-arch/lychee/internal/speech"
)

func TestAudioSilencePreservesReadingTimeAndRejectsLateResults(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	subtitles := newSubtitleService()
	window := &recordingSubtitle{}
	subtitles.window = window
	_ = subtitles.setVisible(true)
	now := time.Unix(0, 0)
	var contexts []context.Context
	var receivers []func(speech.Event)
	var failures []func(error)
	var pipelines []*finishingSpeech
	failuresReported, notices := 0, 0
	session := &AudioSession{
		ctx: ctx, subtitles: subtitles, now: func() time.Time { return now },
		failed: func(error) { failuresReported++ }, notice: func() { notices++ },
	}
	session.start = func(ctx context.Context, _ speech.Settings, receive func(speech.Event), failed func(error)) (speech.Pipeline, error) {
		pipeline := &finishingSpeech{}
		contexts = append(contexts, ctx)
		receivers = append(receivers, receive)
		failures = append(failures, failed)
		pipelines = append(pipelines, pipeline)
		return pipeline, nil
	}
	if err := session.open(subtitles.reset()); err != nil {
		t.Fatal(err)
	}
	defer session.Finish()
	if err := session.Push([]byte{10, 0}, 0.1); err != nil {
		t.Fatal(err)
	}
	receivers[0](speech.Event{Type: "translation", SessionID: "old", Sequence: 1, SourceText: "Old video", TranslatedText: strings.Repeat("旧视频的字幕积压。", 30)})
	receivers[0](speech.Event{Type: "translation_preview", SessionID: "old", Sequence: 2, TranslatedText: "尚未确认"})
	subtitles.presentAt(now)
	if !window.visible || !subtitles.playback.HasContent() || len(subtitles.playback.pending) != 1 {
		t.Fatal("test did not create visible and pending subtitle backlog")
	}
	now = now.Add(2 * time.Second)
	_ = session.Push([]byte{0, 0}, 0)
	if !subtitles.playback.HasContent() || contexts[0].Err() != nil {
		t.Fatal("normal short pause discarded subtitles")
	}
	now = now.Add(500 * time.Millisecond)
	_ = session.Push([]byte{0, 0}, 0)
	if !window.visible || !subtitles.playback.HasContent() || len(subtitles.playback.pending) != 0 || contexts[0].Err() == nil {
		t.Fatal("silence erased the visible chunk or failed to retire reception")
	}
	if len(subtitles.playback.queue) != 1 || len(subtitles.playback.queue[0].frame.Translation) != subtitles.playback.page.TranslationBytes {
		t.Fatal("silence retained unread backlog beyond the current chunk")
	}
	receivers[0](speech.Event{Type: "translation", SessionID: "old", Sequence: 2, TranslatedText: "迟到的旧译文"})
	failures[0](errors.New("old connection closed"))
	receivers[0](speech.Event{Type: "session_interrupted"})
	if failuresReported != 0 || notices != 0 || len(subtitles.playback.queue) != 1 {
		t.Fatal("retired callbacks resurrected old captions or reported a false failure")
	}
	for elapsed := time.Duration(0); elapsed < subtitles.playback.pageDuration+100*time.Millisecond; elapsed += 50 * time.Millisecond {
		now = now.Add(50 * time.Millisecond)
		subtitles.presentAt(now)
	}
	if !window.visible || subtitles.frame.Translation != subtitles.playback.page.Translation {
		t.Fatal("silence cut off the current chunk before it could be read")
	}
	for i := 0; i < 100; i++ {
		now = now.Add(50 * time.Millisecond)
		subtitles.presentAt(now)
	}
	if subtitles.playback.HasContent() || subtitles.frame.Key != "" {
		t.Fatal("paused caption did not clear after its final hold")
	}
	now = now.Add(time.Second)
	_ = session.Push([]byte{0, 0}, 0)
	if len(pipelines) != 1 {
		t.Fatal("silence repeatedly restarted the model")
	}
	now = now.Add(time.Second)
	// Very quiet but real audio must resume capture (the UI meter's larger
	// threshold must not be used for deciding that a video has ended).
	_ = session.Push([]byte{10, 0}, 0.0003)
	if len(pipelines) != 2 || contexts[1].Err() != nil {
		t.Fatal("sound did not reopen a fresh speech session")
	}
	receivers[0](speech.Event{Type: "translation", SessionID: "old", Sequence: 3, TranslatedText: "恢复后迟到的旧译文"})
	receivers[1](speech.Event{Type: "translation", SessionID: "new", Sequence: 1, SourceText: "New video", TranslatedText: "新视频的字幕"})
	subtitles.presentAt(now)
	if len(subtitles.playback.queue) != 1 || subtitles.playback.queue[0].frame.Translation != "新视频的字幕" || !window.visible {
		t.Fatal("resumed audio mixed old and new subtitles")
	}
	cancel()
	_ = session.Push([]byte{10, 0}, 0.1)
	session.Finish()
	for _, pipeline := range pipelines {
		pipeline.mu.Lock()
		finished := pipeline.finished
		pipeline.mu.Unlock()
		if !finished {
			t.Fatal("retired speech resources were not joined")
		}
	}
	if subtitles.playback.HasContent() || window.visible || len(pipelines) != 2 {
		t.Fatal("stop left old subtitles or restarted speech")
	}
}

func TestAudioQuietActivityResetsSilenceGrace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	subtitles := newSubtitleService()
	subtitles.window = &recordingSubtitle{}
	now := time.Unix(0, 0)
	session := &AudioSession{ctx: ctx, subtitles: subtitles, now: func() time.Time { return now }, failed: func(err error) { t.Error(err) }}
	session.start = func(context.Context, speech.Settings, func(speech.Event), func(error)) (speech.Pipeline, error) {
		return &finishingSpeech{}, nil
	}
	if err := session.open(subtitles.reset()); err != nil {
		t.Fatal(err)
	}
	defer session.Finish()
	for i := 0; i < 6; i++ {
		now = now.Add(time.Second)
		_ = session.Push([]byte{5, 0}, 0.00015)
		if session.idle {
			t.Fatal("quiet ongoing playback was mistaken for silence")
		}
	}
	now = now.Add(2 * time.Second)
	_ = session.Push([]byte{0, 0}, 0)
	if session.idle {
		t.Fatal("silence grace was not measured from the last audible block")
	}
	now = now.Add(500 * time.Millisecond)
	_ = session.Push([]byte{0, 0}, 0)
	if !session.idle {
		t.Fatal("silence never ended the current subtitle generation")
	}
}

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
