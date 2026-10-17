package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAudioSilenceClearsBacklogAndRejectsLateResults(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	subtitles := newSubtitleService()
	window := &recordingSubtitle{}
	subtitles.window = window
	_ = subtitles.setVisible(true)
	now := time.Unix(0, 0)
	var contexts []context.Context
	var receivers []func(speechEvent)
	var failures []func(error)
	var pipelines []*finishingSpeech
	failuresReported, notices := 0, 0
	session := &audioSubtitleSession{
		ctx: ctx, subtitles: subtitles, now: func() time.Time { return now },
		failed: func(error) { failuresReported++ }, notice: func() { notices++ },
	}
	session.start = func(ctx context.Context, _ SubtitleSettings, receive func(speechEvent), failed func(error)) (audioSpeech, error) {
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
	receivers[0](speechEvent{Type: "translation", SessionID: "old", Sequence: 1, SourceText: "Old video", TranslatedText: strings.Repeat("旧视频的字幕积压。", 30)})
	receivers[0](speechEvent{Type: "translation_preview", SessionID: "old", Sequence: 2, TranslatedText: "尚未确认"})
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
	if window.visible || subtitles.playback.HasContent() || len(subtitles.playback.pending) != 0 || subtitles.GetSubtitleFrame().Key != "" || contexts[0].Err() == nil {
		t.Fatal("sustained silence did not clear all queues and cancel old speech")
	}
	receivers[0](speechEvent{Type: "translation", SessionID: "old", Sequence: 2, TranslatedText: "迟到的旧译文"})
	failures[0](errors.New("old connection closed"))
	receivers[0](speechEvent{Type: "session_interrupted"})
	if failuresReported != 0 || notices != 0 || subtitles.playback.HasContent() {
		t.Fatal("retired callbacks resurrected old captions or reported a false failure")
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
	receivers[0](speechEvent{Type: "translation", SessionID: "old", Sequence: 3, TranslatedText: "恢复后迟到的旧译文"})
	receivers[1](speechEvent{Type: "translation", SessionID: "new", Sequence: 1, SourceText: "New video", TranslatedText: "新视频的字幕"})
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
	session := &audioSubtitleSession{ctx: ctx, subtitles: subtitles, now: func() time.Time { return now }, failed: func(err error) { t.Error(err) }}
	session.start = func(context.Context, SubtitleSettings, func(speechEvent), func(error)) (audioSpeech, error) {
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

type lateFinishingSpeech struct {
	finishingSpeech
	receive func(speechEvent)
}

func (p *lateFinishingSpeech) Finish() error {
	p.receive(speechEvent{Type: "translation", SessionID: "old", Sequence: 2, TranslatedText: "停止时迟到的译文"})
	return p.finishingSpeech.Finish()
}

func TestAudioManualStopClearsAndRejectsFinishingResult(t *testing.T) {
	a := newAudioService()
	a.subtitles = newSubtitleService()
	window := &recordingSubtitle{}
	a.subtitles.window = window
	a.list = func() ([]AudioSource, error) { return []AudioSource{{ID: "system"}}, nil }
	var receive func(speechEvent)
	a.startSpeech = func(_ context.Context, _ SubtitleSettings, callback func(speechEvent), _ func(error)) (audioSpeech, error) {
		receive = callback
		return &lateFinishingSpeech{receive: callback}, nil
	}
	queued := make(chan struct{})
	a.capture = func(ctx context.Context, _ AudioSource, ready func(), _ func([]byte)) error {
		ready()
		receive(speechEvent{Type: "translation", SessionID: "old", Sequence: 1, TranslatedText: strings.Repeat("旧视频", 50)})
		close(queued)
		<-ctx.Done()
		return nil
	}
	if _, err := a.StartAudioCapture("system"); err != nil {
		t.Fatal(err)
	}
	<-queued
	a.subtitles.present()
	if !window.visible {
		t.Fatal("test did not show the queued subtitle")
	}
	a.StopAudioCapture()
	if a.GetAudioStatus().State != "idle" || a.subtitles.GetSubtitleFrame().Key != "" || a.subtitles.playback.HasContent() || window.visible {
		t.Fatal("manual stop drained old captions or accepted a late finish result")
	}
}
