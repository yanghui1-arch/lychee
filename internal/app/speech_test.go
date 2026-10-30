package app

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
)

type markerDetector struct{}

func (markerDetector) Probability(frame []byte) (float32, error) {
	if frame[0] == 1 {
		return 0.9, nil
	}
	return 0.01, nil
}
func (markerDetector) Close() {}

func TestVoiceGatePreservesPrerollSilenceAndPartialTail(t *testing.T) {
	var input []byte
	for i := 0; i < 60; i++ {
		frame := make([]byte, vadFrameBytes)
		if i >= 15 && i < 18 || i >= 58 {
			frame[0] = 1
		}
		input = append(input, frame...)
	}
	input = append(input, 3, 0, 4, 0)
	var packets []speechPacket
	consume := func(packet speechPacket) error { packets = append(packets, packet); return nil }
	var gate voiceGate
	// Odd, irregular capture packets must not shift sample boundaries.
	for len(input) > 0 {
		n := min(317, len(input))
		if err := gate.Process(input[:n], markerDetector{}, consume); err != nil {
			t.Fatal(err)
		}
		input = input[n:]
	}
	if err := gate.Finish(consume); err != nil {
		t.Fatal(err)
	}
	ends, frames := 0, 0
	for _, packet := range packets {
		if packet.end {
			ends++
		} else {
			frames++
		}
	}
	if ends != 1 || frames != 54 {
		t.Fatalf("unexpected gate output: ends=%d frames=%d", ends, frames)
	}
	if !bytes.Equal(packets[len(packets)-1].pcm, []byte{3, 0, 4, 0}) {
		t.Fatal("stop lost partial PCM tail")
	}
	if packets[0].pcm[0] != 0 || packets[8].pcm[0] != 1 {
		t.Fatal("speech onset/pre-roll lost")
	}
}

func TestCaptionSnapshotsAndFragments(t *testing.T) {
	var c speechCaptions
	apply := func(event speechEvent) SubtitleFrame {
		event.SessionID = "one"
		frame, _ := c.Apply(event)
		return frame
	}
	apply(speechEvent{Type: "transcript", SentenceID: 1, Text: "We will meet tomorrow."})
	apply(speechEvent{Type: "translating", Sequence: 1, SourceText: "We will meet", SentenceIDs: []int{1}})
	apply(speechEvent{Type: "translation_preview", Sequence: 1, TranslatedText: "我们"})
	frame := apply(speechEvent{Type: "translation_preview", Sequence: 1, TranslatedText: "我们见面"})
	if frame.Translation != "我们见面" || frame.Source != "We will meet" || frame.Final {
		t.Fatalf("bad preview: %+v", frame)
	}
	apply(speechEvent{Type: "translation", Sequence: 1, TranslatedText: "我们见面"})
	apply(speechEvent{Type: "translation_preview", Sequence: 1, TranslatedText: "迟到重复"})
	apply(speechEvent{Type: "translating", Sequence: 2, SourceText: "tomorrow.", SentenceIDs: []int{1}})
	frame = apply(speechEvent{Type: "translation", Sequence: 2, TranslatedText: "明天。"})
	if frame.Translation != "明天。" || frame.Source != "tomorrow." || !frame.Final {
		t.Fatalf("fragments overwritten or duplicated: %+v", frame)
	}
	// The next sentence can arrive while the current translation is still active.
	frame = apply(speechEvent{Type: "transcript", SentenceID: 2, Text: "Next sentence"})
	if frame.Source != "tomorrow." {
		t.Fatal("next ASR sentence replaced current translation")
	}
	frame, _ = c.Apply(speechEvent{Type: "translation", SessionID: "two", Sequence: 1, TranslatedText: "Translation only"})
	if frame.Source != "" || frame.Translation != "Translation only" {
		t.Fatalf("missing source or session reset mishandled: %+v", frame)
	}
}

func TestTranscriptFinalRetractionAndLateRevision(t *testing.T) {
	var c speechCaptions
	c.Apply(speechEvent{Type: "transcript", SentenceID: 1, Text: "hypothesis"})
	frame, _ := c.Apply(speechEvent{Type: "transcript", SentenceID: 1, Text: "", IsFinal: true})
	if frame.Source != "" {
		t.Fatal("empty final did not retract hypothesis")
	}
	_, changed := c.Apply(speechEvent{Type: "transcript", SentenceID: 1, Text: "late"})
	if changed {
		t.Fatal("late interim changed a final sentence")
	}
}

func TestCaptionsWaitForMatchingTranslation(t *testing.T) {
	var captions speechCaptions
	service := newSubtitleService()
	service.window = &recordingSubtitle{}
	feed := func(event speechEvent) bool {
		event.SessionID = "paired"
		frame, publish := captions.Apply(event)
		if publish {
			service.accept(frame)
		}
		service.present()
		return publish
	}
	if feed(speechEvent{Type: "transcript", SentenceID: 1, Text: "First fragment. Still untranslated."}) {
		t.Fatal("ASR published before a translation")
	}
	if feed(speechEvent{Type: "translating", Sequence: 1, SentenceIDs: []int{1}, SourceText: "First fragment."}) {
		t.Fatal("translating published a source-only frame")
	}
	if frame := service.GetSubtitleFrame(); frame.Key != "" {
		t.Fatalf("source appeared before translation: %+v", frame)
	}
	if !feed(speechEvent{Type: "translation_preview", Sequence: 1, TranslatedText: "第一段。"}) {
		t.Fatal("first translation was withheld")
	}
	if frame := service.GetSubtitleFrame(); frame.Key != "" {
		t.Fatalf("unconfirmed preview entered the reading queue: %+v", frame)
	}
	if captions.frame().Source != "First fragment." {
		t.Fatal("untranslated ASR tail appeared in the pair")
	}
	feed(speechEvent{Type: "translation", Sequence: 1, TranslatedText: "第一段。"})
	previous := service.GetSubtitleFrame()
	if previous.Source == "" || previous.Translation == "" {
		t.Fatal("first confirmed page is not paired")
	}
	if feed(speechEvent{Type: "transcript", SentenceID: 1, Text: "First fragment. Next fragment. More untranslated text."}) {
		t.Fatal("ASR revision updated the source by itself")
	}
	if feed(speechEvent{Type: "translating", Sequence: 2, SentenceIDs: []int{1}, SourceText: "Next fragment."}) {
		t.Fatal("next source fragment ran ahead of translation")
	}
	if service.GetSubtitleFrame().Key != previous.Key {
		t.Fatal("waiting for the next translation changed the displayed pair")
	}
	feed(speechEvent{Type: "translation_preview", Sequence: 2, TranslatedText: "下一段。"})
	if len(service.playback.queue) != 1 {
		t.Fatal("next preview replaced the active reading page")
	}
	feed(speechEvent{Type: "translation", Sequence: 2, TranslatedText: "下一段。"})
	if len(service.playback.queue) != 2 || service.playback.queue[1].frame.Source != "Next fragment." || service.playback.queue[1].frame.Translation != "下一段。" {
		t.Fatal("next confirmed pair was not queued independently")
	}
	if feed(speechEvent{Type: "translation_preview", Sequence: 3, TranslatedText: "   "}) {
		t.Fatal("empty translation published a frame")
	}
	service.accept(SubtitleFrame{Key: "unpaired", Source: "must stay hidden"})
	if len(service.playback.queue) != 2 {
		t.Fatal("presenter accepted a source-only frame")
	}
}

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
			for _, event := range []speechEvent{
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
	var captions speechCaptions
	var frame SubtitleFrame
	socket, err := openSpeechSocket(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), "private-test-token", newSubtitleService().settings, func(event speechEvent) { frame, _ = captions.Apply(event) })
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
	if frame.Source != "Hello." || frame.Translation != "你好。" || !frame.Final {
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
			socket, err := openSpeechSocket(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), "test", newSubtitleService().settings, func(speechEvent) {})
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
	a.subtitles = newSubtitleService()
	a.subtitles.window = &recordingSubtitle{}
	a.list = func() ([]AudioSource, error) { return []AudioSource{{ID: "system"}}, nil }
	p := &finishingSpeech{}
	a.startSpeech = func(context.Context, SubtitleSettings, func(speechEvent), func(error)) (audioSpeech, error) {
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
	a.subtitles = newSubtitleService()
	a.subtitles.window = &recordingSubtitle{}
	a.list = func() ([]AudioSource, error) { return []AudioSource{{ID: "system"}}, nil }
	a.startSpeech = func(context.Context, SubtitleSettings, func(speechEvent), func(error)) (audioSpeech, error) {
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

func TestSpeechUtterancesDisplayInAudioOrder(t *testing.T) {
	var got []string
	output := orderedSpeechOutput{streams: map[int]*queuedSpeechOutput{}, receive: func(event speechEvent) { got = append(got, event.TranslatedText) }, failed: func(err error) { t.Error(err) }}
	first, second := output.Open(), output.Open()
	output.Accept(first, speechEvent{Type: "translation_preview", TranslatedText: "first preview"})
	output.Accept(second, speechEvent{Type: "translation_preview", Sequence: 1, TranslatedText: "second"})
	output.Accept(second, speechEvent{Type: "translation_preview", Sequence: 1, TranslatedText: "second preview"})
	output.Accept(second, speechEvent{Type: "translation", Sequence: 1, TranslatedText: "second final"})
	output.Complete(second)
	if len(got) != 1 {
		t.Fatal("later utterance overtook an unfinished earlier one")
	}
	output.Accept(first, speechEvent{Type: "translation", TranslatedText: "first final"})
	output.Complete(first)
	if strings.Join(got, ",") != "first preview,first final,second preview,second final" {
		t.Fatalf("unexpected display order: %v", got)
	}
}

func TestCaptionReadingTextBoundaries(t *testing.T) {
	if cleanSubtitleText("\x00  hello  ") != "hello" {
		t.Fatal("presentation did not sanitize text")
	}
	if subtitleReadingPrefix("你好世界", 0) != "你" {
		t.Fatal("first reading frame was not a single complete grapheme")
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
			conn.WriteJSON(speechEvent{Type: "translation", Sequence: 1, TranslatedText: session})
			conn.WriteJSON(speechEvent{Type: "completed"})
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
		p.err = p.run(markerDetector{}, "ws"+strings.TrimPrefix(server.URL, "http"), "test", newSubtitleService().settings, func(event speechEvent) {
			mu.Lock()
			defer mu.Unlock()
			if event.Type == "translation" {
				results = append(results, event.TranslatedText)
			}
		})
	}()
	var pcm []byte
	for i := 0; i < 44; i++ {
		frame := make([]byte, vadFrameBytes)
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
