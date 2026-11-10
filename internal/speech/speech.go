package speech

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/yanghui1-arch/lychee/internal/vad"
)

type StartFunc func(context.Context, Settings, func(Event), func(error)) (Pipeline, error)

type Pipeline interface {
	Push([]byte) error
	Finish() error
}

// speechPipeline owns VAD and the socket writer on one worker. The audio thread
// only copies into a bounded queue; a slow network cannot stall WASAPI buffers.
type speechPipeline struct {
	ctx    context.Context
	cancel context.CancelFunc
	input  chan []byte
	done   chan struct{}
	err    error
	once   sync.Once
}

func NewPipeline(ctx context.Context, settings Settings, receive func(Event), failed func(error)) (Pipeline, error) {
	endpoint, err := configuredSpeechEndpoint()
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(os.Getenv("LYCHEE_ACCESS_TOKEN"))
	if token == "" {
		return nil, errors.New("请先配置本机 LYCHEE_ACCESS_TOKEN 登录凭证，再开启字幕")
	}
	detector, err := vad.NewSileroVAD()
	if err != nil {
		return nil, err
	}
	pctx, cancel := context.WithCancel(ctx)
	p := &speechPipeline{ctx: pctx, cancel: cancel, input: make(chan []byte, 50), done: make(chan struct{})}
	go func() {
		defer close(p.done)
		defer detector.Close()
		defer cancel()
		p.err = p.run(detector, endpoint, token, settings, receive)
		if p.err != nil && !errors.Is(p.err, context.Canceled) && ctx.Err() == nil {
			failed(p.err)
		}
	}()
	return p, nil
}

func (p *speechPipeline) Push(pcm []byte) error {
	select {
	case <-p.done:
		return p.err
	default:
	}
	select {
	case p.input <- append([]byte(nil), pcm...):
		return nil
	default:
		p.cancel()
		return errors.New("语音处理积压超过 10 秒，已停止监听，请检查网络后重试")
	}
}

func (p *speechPipeline) Finish() error {
	p.once.Do(func() { close(p.input) })
	// Stop sends finish and waits for completed, so final words are not lost.
	timer := time.NewTimer(130 * time.Second)
	defer timer.Stop()
	select {
	case <-p.done:
		return p.err
	case <-timer.C:
		p.cancel()
		<-p.done
		return errors.New("语音结束确认超时")
	}
}

func (p *speechPipeline) run(detector vad.Detector, endpoint, token string, settings Settings, receive func(Event)) (result error) {
	var gate vad.Gate
	var socket *speechSocket
	var streamID int
	var draining sync.WaitGroup
	// Up to two old utterances may be settling while the next one streams.
	// Audio capture must not wait for the previous model's final response.
	slots := make(chan struct{}, 3)
	errorsCh := make(chan error, 1)
	report := func(err error) {
		// Closing sibling connections during cancellation is cleanup, not a
		// new failure. Keep only a cause reported before cancellation started.
		if err != nil && p.ctx.Err() == nil {
			select {
			case errorsCh <- err:
			default:
			}
			p.cancel()
		}
	}
	output := orderedSpeechOutput{streams: map[int]*queuedSpeechOutput{}, receive: receive, failed: report}
	var recovery speechRecovery
	recoverSession := func(id int, err error) error {
		if p.ctx.Err() != nil {
			return err
		}
		allowed, failure := recovery.allow(err, time.Now())
		if !allowed {
			return failure
		}
		output.Interrupt(id)
		return nil
	}
	defer func() {
		if result != nil {
			p.cancel()
		}
		if socket != nil {
			socket.Close()
		}
		draining.Wait()
		// A draining session can fail while another Send is waiting for its
		// pacing timer. Preserve that cause instead of showing context.Canceled.
		select {
		case err := <-errorsCh:
			result = err
		default:
		}
	}()
	finish := func() {
		ending, id := socket, streamID
		socket = nil
		draining.Add(1)
		go func() {
			defer draining.Done()
			defer func() { <-slots }()
			if err := ending.Finish(p.ctx); err != nil {
				report(recoverSession(id, err))
				return
			}
			output.Complete(id)
		}()
	}
	interrupt := func(err error) error {
		socket.Close()
		socket = nil
		<-slots
		return recoverSession(streamID, err)
	}
	send := func(packet vad.Packet) error {
		if packet.End {
			if socket != nil {
				finish()
			}
			return nil
		}
		if socket == nil {
			select {
			case slots <- struct{}{}:
			default:
				return errors.New("待完成的语音片段过多，请等待服务器恢复后重试")
			}
			streamID = output.Open()
			id := streamID
			var err error
			socket, err = openSpeechSocket(p.ctx, endpoint, token, settings, func(event Event) { output.Accept(id, event) })
			if err != nil {
				return err
			}
		}
		if err := socket.Send(p.ctx, packet.PCM); err != nil {
			// This packet may already have reached the server. Resume from the
			// next live packet instead of retransmitting and duplicating audio.
			return interrupt(err)
		}
		return nil
	}
	for {
		var socketDone <-chan struct{}
		if socket != nil {
			socketDone = socket.readDone
		}
		select {
		case <-socketDone:
			err, _ := socket.completion()
			if err == nil {
				err = errors.New("语音服务提前结束")
			}
			if err := interrupt(err); err != nil {
				return err
			}
		case <-p.ctx.Done():
			select {
			case err := <-errorsCh:
				return err
			default:
			}
			return p.ctx.Err()
		case pcm, ok := <-p.input:
			if !ok {
				if err := gate.Finish(send); err != nil {
					return err
				}
				if socket != nil {
					finish()
				}
				draining.Wait()
				select {
				case err := <-errorsCh:
					return err
				default:
				}
				return nil
			}
			if err := gate.Process(pcm, detector, send); err != nil {
				return err
			}
		}
	}
}
