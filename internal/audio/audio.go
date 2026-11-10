package audio

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/yanghui1-arch/lychee/internal/speech"
	"github.com/yanghui1-arch/lychee/internal/subtitle"
)

const audioSampleRate = 16000

type AudioSource struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
	PID   uint32 `json:"pid"`
	Icon  string `json:"icon,omitempty"`
}

type AudioStatus struct {
	State   string      `json:"state"`
	Source  AudioSource `json:"source"`
	Session uint64      `json:"session"`
	Level   float64     `json:"level"`
	Frames  uint64      `json:"frames"`
	Error   string      `json:"error"`
}

type AudioLevel struct {
	Session uint64  `json:"session"`
	Level   float64 `json:"level"`
}

type AudioService struct {
	commands    sync.Mutex
	mu          sync.Mutex
	ctx         context.Context
	status      AudioStatus
	cancel      context.CancelFunc
	done        chan struct{}
	emit        func(string, any)
	list        func() ([]AudioSource, error)
	capture     func(context.Context, AudioSource, func(), func([]byte)) error
	subtitles   *subtitle.Controller
	startSpeech speech.StartFunc
	now         func() time.Time
}

func newAudioService() *AudioService {
	return &AudioService{ctx: context.Background(), status: AudioStatus{State: "idle"}, list: listAudioSources, capture: captureAudio, now: time.Now}
}

func (a *AudioService) ListAudioSources() ([]AudioSource, error) { return a.list() }

func (a *AudioService) GetAudioStatus() AudioStatus {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status
}

func (a *AudioService) publish(status AudioStatus) {
	if a.subtitles != nil {
		if err := a.subtitles.SetVisible(status.State == "listening"); err != nil && a.emit != nil {
			a.emit("subtitle:error", err.Error())
		}
	}
	if a.emit != nil {
		a.emit("audio:status", status)
	}
}

func (a *AudioService) StartAudioCapture(id string) (AudioStatus, error) {
	a.commands.Lock()
	defer a.commands.Unlock()
	sources, err := a.list()
	if err != nil {
		return a.GetAudioStatus(), err
	}
	var source AudioSource
	for _, item := range sources {
		if item.ID == id {
			source = item
			break
		}
	}
	if source.ID == "" {
		return a.GetAudioStatus(), errors.New("所选应用当前不可用，请重新选择")
	}
	a.stop()
	var generation uint64
	if a.subtitles != nil {
		generation = a.subtitles.Reset()
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancel = cancel
	a.done = make(chan struct{})
	ready := make(chan error, 1)
	a.mu.Lock()
	a.status = AudioStatus{State: "starting", Source: source, Session: a.status.Session + 1}
	status := a.status
	a.mu.Unlock()
	a.publish(status)
	go func() {
		defer close(a.done)
		started := false
		var session *subtitle.AudioSession
		speechErrors := make(chan error, 1)
		failSpeech := func(err error) {
			select {
			case speechErrors <- err:
			default:
			}
			cancel()
		}
		var err error
		if a.startSpeech != nil {
			notice := func() {
				if a.emit != nil {
					a.emit("audio:notice", speech.InterruptionNotice)
				}
			}
			session = a.subtitles.NewAudioSession(ctx, a.startSpeech, failSpeech, notice, a.now)
			err = session.Open(generation)
		}
		if err == nil {
			err = a.capture(ctx, source, func() {
				started = true
				a.mu.Lock()
				a.status.State = "listening"
				status := a.status
				a.mu.Unlock()
				a.publish(status)
				ready <- nil
			}, func(pcm []byte) {
				level := pcmLevel(pcm)
				if session != nil {
					if err := session.Push(pcm, level); err != nil {
						failSpeech(err)
						return
					}
				}
				a.mu.Lock()
				a.status.Level = level
				a.status.Frames += uint64(len(pcm) / 2)
				session := a.status.Session
				a.mu.Unlock()
				if a.emit != nil {
					// PCM stays in Go; the UI only needs the level, not raw audio.
					a.emit("audio:level", AudioLevel{Session: session, Level: level})
				}
			})
		}
		if session != nil {
			session.Finish()
		}
		var speechErr error
		select {
		case speechErr = <-speechErrors:
			err = speechErr
		default:
		}
		if err != nil && (ctx.Err() == nil || speechErr != nil) {
			a.mu.Lock()
			a.status.State, a.status.Error, a.status.Level = "error", err.Error(), 0
			status := a.status
			a.mu.Unlock()
			a.publish(status)
		}
		if !started {
			ready <- err
		}
	}()
	err = <-ready
	return a.GetAudioStatus(), err
}

func (a *AudioService) stop() {
	if a.cancel != nil {
		a.cancel()
	}
	if a.subtitles != nil {
		a.subtitles.Discard()
	}
	if a.cancel != nil {
		<-a.done
		a.cancel = nil
	}
	a.mu.Lock()
	a.status.State, a.status.Level, a.status.Error = "idle", 0, ""
	status := a.status
	a.mu.Unlock()
	a.publish(status)
}

func (a *AudioService) StopAudioCapture() AudioStatus {
	a.commands.Lock()
	defer a.commands.Unlock()
	a.stop()
	return a.GetAudioStatus()
}

func pcmLevel(pcm []byte) float64 {
	if len(pcm) < 2 {
		return 0
	}
	var sum float64
	for i := 0; i+1 < len(pcm); i += 2 {
		sample := float64(int16(binary.LittleEndian.Uint16(pcm[i:]))) / 32768
		sum += sample * sample
	}
	return math.Min(1, math.Sqrt(sum/float64(len(pcm)/2)))
}
