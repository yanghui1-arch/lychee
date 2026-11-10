package speech

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

type markerDetector struct{}

func (markerDetector) Probability(frame []byte) (float32, error) {
	if frame[0] == 1 {
		return 0.9, nil
	}
	return 0.01, nil
}
func (markerDetector) Close() {}

func TestSpeechSocketRealProtocolAndFinish(t *testing.T) {
	var received bytes.Buffer
	var start map[string]any
	serverErr := make(chan error, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer private-test-token" || strings.Contains(r.URL.String(), "token") {
			serverErr <- errors.New("unsafe authentication")
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close()
		if err = conn.ReadJSON(&start); err != nil {
			serverErr <- err
			return
		}
		conn.WriteJSON(map[string]string{"type": "ready", "session_id": "mock"})
		for {
			kind, data, err := conn.ReadMessage()
			if err != nil {
				serverErr <- err
				return
			}
			if kind == websocket.BinaryMessage {
				received.Write(data)
				continue
			}
			var control map[string]string
			if json.Unmarshal(data, &control) != nil || control["type"] != "finish" {
				serverErr <- errors.New("expected finish")
				return
			}
			for _, event := range []Event{
				{Type: "transcript", SentenceID: 1, Text: "Hello.", IsFinal: true},
				{Type: "translating", Sequence: 1, SourceText: "Hello.", SentenceIDs: []int{1}},
				{Type: "translation_preview", Sequence: 1, TranslatedText: "你"},
				{Type: "translation", Sequence: 1, TranslatedText: "你好。"},
				{Type: "completed"},
			} {
				if err = conn.WriteJSON(event); err != nil {
					serverErr <- err
					return
				}
			}
			serverErr <- nil
			return
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var frame Event
	var source string
	socket, err := openSpeechSocket(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), "private-test-token", Settings{Source: "en-US", Target: "zh-CN"}, func(event Event) {
		if event.Type == "translating" {
			source = event.SourceText
		}
		if event.Type == "translation" {
			frame = event
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	pcm := make([]byte, 1024)
	pcm[0] = 12
	if err = socket.Send(ctx, pcm); err != nil {
		t.Fatal(err)
	}
	if err = socket.Finish(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-serverErr; err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(received.Bytes(), pcm) || start["sample_rate"] != float64(16000) || start["target_language"] != "zh-CN" {
		t.Fatalf("invalid audio/start: %v", start)
	}
	if source != "Hello." || frame.TranslatedText != "你好。" || frame.Type != "translation" {
		t.Fatalf("lost final response: %+v", frame)
	}
}

func TestSpeechRejectsAuthenticationAndDisconnect(t *testing.T) {
	for _, failure := range []string{"auth", "disconnect"} {
		t.Run(failure, func(t *testing.T) {
			upgrader := websocket.Upgrader{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()
				var start any
				conn.ReadJSON(&start)
				if failure == "auth" {
					conn.WriteJSON(map[string]any{"type": "error", "error": map[string]string{"code": "invalid_credentials", "message": "凭证无效"}})
					return
				}
				conn.WriteJSON(map[string]string{"type": "ready", "session_id": "mock"})
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			socket, err := openSpeechSocket(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), "test", Settings{Source: "en-US", Target: "zh-CN"}, func(Event) {})
			if failure == "auth" {
				if err == nil || !strings.Contains(err.Error(), "invalid_credentials") {
					t.Fatalf("auth error lost: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer socket.Close()
			if err = socket.Finish(ctx); err == nil {
				t.Fatal("disconnect accepted as completion")
			}
		})
	}
}

func TestSpeechUtterancesDisplayInAudioOrder(t *testing.T) {
	var got []string
	output := orderedSpeechOutput{streams: map[int]*queuedSpeechOutput{}, receive: func(event Event) { got = append(got, event.TranslatedText) }, failed: func(err error) { t.Error(err) }}
	first, second := output.Open(), output.Open()
	output.Accept(first, Event{Type: "translation_preview", TranslatedText: "first preview"})
	output.Accept(second, Event{Type: "translation_preview", Sequence: 1, TranslatedText: "second"})
	output.Accept(second, Event{Type: "translation_preview", Sequence: 1, TranslatedText: "second preview"})
	output.Accept(second, Event{Type: "translation", Sequence: 1, TranslatedText: "second final"})
	output.Complete(second)
	if len(got) != 1 {
		t.Fatal("later utterance overtook an unfinished earlier one")
	}
	output.Accept(first, Event{Type: "translation", TranslatedText: "first final"})
	output.Complete(first)
	if strings.Join(got, ",") != "first preview,first final,second preview,second final" {
		t.Fatalf("unexpected display order: %v", got)
	}
}

func TestSpeechPipelineUploadsNextUtteranceDuringDrain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	releaseFirst := make(chan struct{})
	defer close(releaseFirst)
	secondAudio := make(chan struct{}, 1)
	var connections atomic.Int32
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		id := connections.Add(1)
		var start any
		if conn.ReadJSON(&start) != nil {
			return
		}
		session := "first"
		if id == 2 {
			session = "second"
		}
		conn.WriteJSON(map[string]string{"type": "ready", "session_id": session})
		for {
			kind, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if kind == websocket.BinaryMessage {
				if id == 2 {
					select {
					case secondAudio <- struct{}{}:
					default:
					}
				}
				continue
			}
			var control map[string]string
			if json.Unmarshal(data, &control) != nil || control["type"] != "finish" {
				return
			}
			if id == 1 {
				select {
				case <-releaseFirst:
				case <-ctx.Done():
					return
				}
			}
			conn.WriteJSON(Event{Type: "translation", Sequence: 1, TranslatedText: session})
			conn.WriteJSON(Event{Type: "completed"})
			return
		}
	}))
	defer server.Close()
	var mu sync.Mutex
	var results []string
	pctx, stop := context.WithCancel(ctx)
	defer stop()
	p := &speechPipeline{ctx: pctx, cancel: stop, input: make(chan []byte, 50), done: make(chan struct{})}
	go func() {
		defer close(p.done)
		p.err = p.run(markerDetector{}, "ws"+strings.TrimPrefix(server.URL, "http"), "test", Settings{Source: "en-US", Target: "zh-CN"}, func(event Event) {
			mu.Lock()
			defer mu.Unlock()
			if event.Type == "translation" {
				results = append(results, event.TranslatedText)
			}
		})
	}()
	var pcm []byte
	for i := 0; i < 44; i++ {
		frame := make([]byte, vad.FrameBytes)
		if i < 2 || i >= 42 {
			frame[0] = 1
		}
		pcm = append(pcm, frame...)
	}
	if err := p.Push(pcm); err != nil {
		t.Fatal(err)
	}
	select {
	case <-secondAudio:
	case <-ctx.Done():
		t.Fatal("next utterance waited for previous final translation")
	}
	// Allow both streams to finish now; the buffered later result stays ordered.
	releaseFirst <- struct{}{}
	if err := p.Finish(); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(results, ",") != "first,second" {
		t.Fatalf("lost or reordered results: %v", results)
	}
}
