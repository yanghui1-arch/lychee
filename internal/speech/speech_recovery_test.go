package speech

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
	"github.com/yanghui1-arch/lychee/internal/vad"
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
						if len(pcm) != vad.FrameBytes || pcm[1] == 0 || pcm[1] > 44 {
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
						conn.WriteJSON(Event{Type: "translation", Sequence: 1, SourceText: "First.", TranslatedText: "第一句。"})
						conn.WriteJSON(Event{Type: "translation_preview", Sequence: 2, SourceText: "Unfinished.", TranslatedText: "未完成"})
						conn.WriteJSON(map[string]any{"type": "error", "error": map[string]string{"code": "asr_committed_text_changed", "message": "识别修改了已提交的文字，会话已停止"}})
					} else {
						conn.WriteJSON(Event{Type: "translation", Sequence: 1, SourceText: "Next.", TranslatedText: "后续字幕。"})
						conn.WriteJSON(Event{Type: "completed"})
					}
					return
				}
			}))
			defer server.Close()
			p := &speechPipeline{ctx: ctx, cancel: cancel, input: make(chan []byte, 50), done: make(chan struct{})}
			go func() {
				defer close(p.done)
				p.err = p.run(markerDetector{}, "ws"+strings.TrimPrefix(server.URL, "http"), "test", Settings{Source: "en-US", Target: "zh-CN"}, func(event Event) {
					if event.Type == "session_interrupted" {
						got = append(got, "interrupted")
					} else {
						if event.Type == "translation" {
							got = append(got, event.TranslatedText)
						}
					}
				})
			}()
			var pcm []byte
			for i := 0; i < 44; i++ {
				frame := make([]byte, vad.FrameBytes)
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
		p.err = p.run(markerDetector{}, "ws"+strings.TrimPrefix(server.URL, "http"), "test", Settings{Source: "en-US", Target: "zh-CN"}, func(Event) {})
	}()
	var pcm []byte
	for i := 0; i < 50; i++ {
		frame := make([]byte, vad.FrameBytes)
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
