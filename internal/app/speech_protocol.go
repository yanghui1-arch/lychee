package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type speechEvent struct {
	Type           string `json:"type"`
	SessionID      string `json:"session_id"`
	Sequence       int    `json:"sequence"`
	SentenceID     int    `json:"sentence_id"`
	SentenceIDs    []int  `json:"sentence_ids"`
	Text           string `json:"text"`
	SourceText     string `json:"source_text"`
	TranslatedText string `json:"translated_text"`
	IsFinal        bool   `json:"is_final"`
	Error          struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func configuredSpeechEndpoint() (string, error) {
	endpoint := os.Getenv("LYCHEE_SPEECH_URL")
	if endpoint == "" {
		target, err := url.Parse(configuredTranslationEndpoint())
		if err != nil {
			return "", errors.New("翻译服务地址无效")
		}
		target.Path = strings.TrimSuffix(target.Path, "/") + "/speech"
		endpoint = target.String()
	}
	target, err := url.Parse(endpoint)
	if err != nil || target.Host == "" || target.User != nil {
		return "", errors.New("语音服务地址无效")
	}
	switch target.Scheme {
	case "http":
		target.Scheme = "ws"
	case "https":
		target.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", errors.New("LYCHEE_SPEECH_URL 必须为完整的 WS(S) 地址")
	}
	return target.String(), nil
}

type speechSocket struct {
	conn      *websocket.Conn
	readDone  chan struct{}
	terminal  error // Written by the reader, published by closing readDone.
	stopWatch func() bool
	nextFrame time.Time
}

func openSpeechSocket(ctx context.Context, endpoint, token string, settings SubtitleSettings, receive func(speechEvent)) (*speechSocket, error) {
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second, Proxy: http.ProxyFromEnvironment}
	// Native clients can authenticate in the header and need no browser Origin.
	conn, response, err := dialer.DialContext(ctx, endpoint, http.Header{"Authorization": {"Bearer " + token}})
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		return nil, errors.New("无法连接语音翻译服务，请检查服务器地址与网络")
	}
	s := &speechSocket{conn: conn}
	s.stopWatch = context.AfterFunc(ctx, func() { conn.Close() })
	fail := func(err error) (*speechSocket, error) { s.Close(); return nil, err }
	conn.SetReadLimit(2 * 1024 * 1024)
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	if err := s.writeJSON(map[string]any{"type": "start", "target_language": settings.Target, "source_language": settings.Source, "sample_rate": 16000, "audio_format": "pcm_s16le", "channels": 1}); err != nil {
		return fail(err)
	}
	var ready speechEvent
	if err := conn.ReadJSON(&ready); err != nil {
		return fail(errors.New("语音服务启动失败或超时"))
	}
	if ready.Type == "error" {
		return fail(speechEventError(ready))
	}
	if ready.Type != "ready" || ready.SessionID == "" {
		return fail(errors.New("语音服务返回了无效的 ready 消息"))
	}
	s.readDone = make(chan struct{})
	go func() {
		defer close(s.readDone)
		for {
			// Covers long model responses, but still bounds a stalled connection.
			conn.SetReadDeadline(time.Now().Add(130 * time.Second))
			var event speechEvent
			if err := conn.ReadJSON(&event); err != nil {
				s.terminal = errors.New("语音连接中断，请重新开启字幕")
				return
			}
			if event.SessionID == "" {
				event.SessionID = ready.SessionID
			}
			switch event.Type {
			case "error":
				s.terminal = speechEventError(event)
				return
			case "completed":
				return
			default:
				receive(event)
			}
		}
	}()
	return s, nil
}

type speechServiceError struct {
	code, message string
}

func (e *speechServiceError) Error() string {
	return fmt.Sprintf("%s（%s）", e.message, e.code)
}

func speechEventError(event speechEvent) error {
	message := strings.TrimSpace(event.Error.Message)
	if message == "" {
		message = "语音服务返回错误"
	}
	return &speechServiceError{code: event.Error.Code, message: message}
}

// The terminal result is persistent: a failed write/finish must not hide an
// earlier, more specific server error or consume it before recovery handles it.
func (s *speechSocket) completion() (error, bool) {
	select {
	case <-s.readDone:
		return s.terminal, true
	default:
		return nil, false
	}
}

func (s *speechSocket) writeFailure(fallback error) error {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-s.readDone:
		if s.terminal != nil {
			return s.terminal
		}
	case <-timer.C:
	}
	return fallback
}

func (s *speechSocket) writeJSON(value any) error {
	s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := s.conn.WriteJSON(value); err != nil {
		return errors.New("发送语音控制消息失败")
	}
	return nil
}

func (s *speechSocket) Send(ctx context.Context, pcm []byte) error {
	if err, complete := s.completion(); complete {
		if err == nil {
			return errors.New("语音服务提前结束")
		}
		return err
	}
	// Pacing also applies to the pre-roll. Never burst buffered audio at ASR.
	if delay := time.Until(s.nextFrame); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	s.nextFrame = time.Now().Add(time.Duration(len(pcm)) * time.Second / 32000)
	if err, complete := s.completion(); complete {
		if err != nil {
			return err
		}
		return errors.New("语音服务提前结束")
	}
	s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := s.conn.WriteMessage(websocket.BinaryMessage, pcm); err != nil {
		return s.writeFailure(errors.New("发送音频失败，语音连接已中断"))
	}
	return nil
}

func (s *speechSocket) Finish(ctx context.Context) error {
	defer s.Close()
	if err, complete := s.completion(); complete {
		return err
	}
	if err := s.writeJSON(map[string]string{"type": "finish"}); err != nil {
		return s.writeFailure(err)
	}
	timer := time.NewTimer(125 * time.Second)
	defer timer.Stop()
	select {
	case <-s.readDone:
		return s.terminal
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return errors.New("等待语音尾部翻译超时，请重新开启字幕")
	}
}

func (s *speechSocket) Close() {
	if s.stopWatch != nil {
		s.stopWatch()
	}
	s.conn.Close()
	if s.readDone != nil {
		<-s.readDone
	}
}
