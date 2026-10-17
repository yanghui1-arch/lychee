package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSpeechRecoveryOnlyAcceptsBoundedASRRevisions(t *testing.T) {
	var recovery speechRecovery
	now := time.Now()
	for _, err := range []error{errors.New("asr_committed_text_changed"), &speechServiceError{code: "invalid_credentials"}, &speechServiceError{code: "insufficient_credits"}, context.Canceled} {
		if ok, result := recovery.allow(err, now); ok || result != err {
			t.Fatalf("unexpectedly recovered unrelated error: %v", err)
		}
	}
	revision := fmt.Errorf("wrapped: %w", &speechServiceError{code: "asr_committed_text_changed"})
	for i := 0; i < 4; i++ {
		ok, err := recovery.allow(revision, now)
		if ok != (i < 3) || i == 3 && (err == nil || !strings.Contains(err.Error(), "30 秒")) {
			t.Fatalf("recovery rate limit failed: attempt=%d ok=%v err=%v", i, ok, err)
		}
	}
	if ok, err := recovery.allow(revision, now.Add(30*time.Second)); !ok || err != nil {
		t.Fatal("recovery budget did not expire")
	}
}

func TestSpeechPipelineRecoversASRRevision(t *testing.T) {
	for _, phase := range []string{"streaming", "finishing"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			var connections atomic.Int32
			var mu sync.Mutex
			var got []string
			var uploaded [45]int
			upgrader := websocket.Upgrader{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()
				conn.SetReadDeadline(time.Now().Add(7 * time.Second))
				id := connections.Add(1)
				var start any
				if conn.ReadJSON(&start) != nil {
					return
				}
				conn.WriteJSON(map[string]string{"type": "ready", "session_id": fmt.Sprint(id)})
				for {
					kind, pcm, err := conn.ReadMessage()
					if err != nil {
						return
					}
					if kind == websocket.BinaryMessage {
						if len(pcm) != vadFrameBytes || pcm[1] == 0 || pcm[1] > 44 {
							t.Error("invalid live audio packet")
							return
						}
						mu.Lock()
						uploaded[pcm[1]]++
						mu.Unlock()
						if id != 1 || phase != "streaming" {
							continue
						}
					}
					if id == 1 {
						conn.WriteJSON(speechEvent{Type: "translation", Sequence: 1, SourceText: "First.", TranslatedText: "第一句。"})
						conn.WriteJSON(speechEvent{Type: "translation_preview", Sequence: 2, SourceText: "Unfinished.", TranslatedText: "未完成"})
						conn.WriteJSON(map[string]any{"type": "error", "error": map[string]string{"code": "asr_committed_text_changed", "message": "识别修改了已提交的文字，会话已停止"}})
					} else {
						conn.WriteJSON(speechEvent{Type: "translation", Sequence: 1, SourceText: "Next.", TranslatedText: "后续字幕。"})
						conn.WriteJSON(speechEvent{Type: "completed"})
					}
					return
				}
			}))
			defer server.Close()
			p := &speechPipeline{ctx: ctx, cancel: cancel, input: make(chan []byte, 50), done: make(chan struct{})}
			var captions speechCaptions
			var playback subtitlePlayback
			go func() {
				defer close(p.done)
				p.err = p.run(markerDetector{}, "ws"+strings.TrimPrefix(server.URL, "http"), "test", newSubtitleService().settings, func(event speechEvent) {
					if event.Type == "session_interrupted" {
						got = append(got, "interrupted")
						playback.MarkIncomplete()
					} else if frame, changed := captions.Apply(event); changed {
						if event.Type == "translation" {
							got = append(got, event.TranslatedText)
						}
						_ = playback.Accept(frame)
					}
				})
			}()
			var pcm []byte
			for i := 0; i < 44; i++ {
				frame := make([]byte, vadFrameBytes)
				if phase == "streaming" || i < 2 || i >= 42 {
					frame[0] = 1
				}
				frame[1] = byte(i + 1)
				pcm = append(pcm, frame...)
			}
			if err := p.Push(pcm); err != nil {
				t.Fatal(err)
			}
			if err := p.Finish(); err != nil {
				t.Fatalf("ASR revision stopped listening: %v", err)
			}
			if connections.Load() != 2 || strings.Join(got, ",") != "第一句。,interrupted,后续字幕。" {
				t.Fatalf("lost or unordered recovery: connections=%d got=%v", connections.Load(), got)
			}
			if len(playback.queue) != 3 || !playback.queue[1].frame.Incomplete || playback.queue[2].frame.Incomplete {
				t.Fatal("failed preview lost or later settled caption marked incomplete")
			}
			mu.Lock()
			defer mu.Unlock()
			for index, count := range uploaded {
				if count > 1 {
					t.Fatalf("replayed old audio frame %d: %d uploads", index, count)
				}
			}
		})
	}
}

func TestSpeechInterruptionDoesNotStopCapture(t *testing.T) {
	a := newAudioService()
	a.subtitles = newSubtitleService()
	a.subtitles.window = &recordingSubtitle{}
	a.list = func() ([]AudioSource, error) { return []AudioSource{{ID: "system"}}, nil }
	var receive func(speechEvent)
	a.startSpeech = func(_ context.Context, _ SubtitleSettings, callback func(speechEvent), _ func(error)) (audioSpeech, error) {
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
		receive(speechEvent{Type: "translation_preview", SessionID: "failed", Sequence: 1, TranslatedText: "未完成的译文"})
		receive(speechEvent{Type: "session_interrupted"})
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
	a.subtitles.mu.Lock()
	defer a.subtitles.mu.Unlock()
	if len(a.subtitles.playback.queue) != 1 || !a.subtitles.playback.queue[0].frame.Incomplete {
		t.Fatal("unconfirmed subtitle not preserved")
	}
}

func TestSpeechPipelineKeepsFatalErrorDuringConcurrentUpload(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	secondAudio := make(chan struct{})
	var connections atomic.Int32
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetReadDeadline(time.Now().Add(7 * time.Second))
		id := connections.Add(1)
		var start any
		if conn.ReadJSON(&start) != nil {
			return
		}
		conn.WriteJSON(map[string]string{"type": "ready", "session_id": fmt.Sprint(id)})
		notified := false
		for {
			kind, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if kind == websocket.BinaryMessage {
				if id == 2 && !notified {
					close(secondAudio)
					notified = true
				}
				continue
			}
			select {
			case <-secondAudio:
			case <-ctx.Done():
				return
			}
			conn.WriteJSON(map[string]any{"type": "error", "error": map[string]string{"code": "insufficient_credits", "message": "积分不足"}})
			return
		}
	}))
	defer server.Close()
	p := &speechPipeline{ctx: ctx, cancel: cancel, input: make(chan []byte, 50), done: make(chan struct{})}
	go func() {
		defer close(p.done)
		p.err = p.run(markerDetector{}, "ws"+strings.TrimPrefix(server.URL, "http"), "test", newSubtitleService().settings, func(speechEvent) {})
	}()
	var pcm []byte
	for i := 0; i < 50; i++ {
		frame := make([]byte, vadFrameBytes)
		if i < 2 || i >= 42 {
			frame[0] = 1
		}
		pcm = append(pcm, frame...)
	}
	if err := p.Push(pcm); err != nil {
		t.Fatal(err)
	}
	err := p.Finish()
	var serviceError *speechServiceError
	if !errors.As(err, &serviceError) || serviceError.code != "insufficient_credits" {
		t.Fatalf("cancellation hid the actual service error: %v", err)
	}
}
