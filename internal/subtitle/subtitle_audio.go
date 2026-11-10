package subtitle

import (
	"context"
	"sync"
	"time"

	"github.com/yanghui1-arch/lychee/internal/speech"
)

const (
	audioSubtitleIdleTimeout = 2500 * time.Millisecond
	// Use captured audio energy, not VAD: music and quiet speech still mean
	// playback is active. This is well below the UI meter's sound threshold.
	audioSubtitleQuietLevel = 0.0001
)

// The capture goroutine owns Push/Finish and pipeline replacement. Reader
// callbacks carry a subtitle generation, so retired sockets cannot repopulate
// the screen even when their final responses race with pause or resume.
type AudioSession struct {
	ctx       context.Context
	subtitles *SubtitleService
	start     speech.StartFunc
	failed    func(error)
	notice    func()
	now       func() time.Time
	lastSound time.Time
	idle      bool
	pipeline  speech.Pipeline
	cancel    context.CancelFunc
	retiring  sync.WaitGroup
}

func (s *AudioSession) open(generation uint64) error {
	ctx, cancel := context.WithCancel(s.ctx)
	var captions speechCaptions
	failed := func(err error) {
		if ctx.Err() == nil && s.subtitles.currentGeneration(generation) {
			s.failed(err)
		}
	}
	settings := s.subtitles.snapshotSettings()
	pipeline, err := s.start(ctx, speech.Settings{Source: settings.Source, Target: settings.Target}, func(event speech.Event) {
		if ctx.Err() != nil {
			return
		}
		if event.Type == "session_interrupted" {
			if s.subtitles.interruptGeneration(generation) && s.notice != nil {
				s.notice()
			}
			return
		}
		if frame, changed := captions.Apply(event); changed {
			if err := s.subtitles.acceptGeneration(generation, frame); err != nil {
				failed(err)
			}
		}
	}, failed)
	if err != nil {
		cancel()
		return err
	}
	s.pipeline, s.cancel = pipeline, cancel
	s.lastSound = s.now()
	return nil
}

func (s *AudioSession) retire() {
	if s.pipeline == nil {
		return
	}
	pipeline := s.pipeline
	s.pipeline = nil
	s.cancel()
	s.cancel = nil
	// Cancellation closes the old sockets immediately. Joining on another
	// goroutine keeps WASAPI's callback free to deliver subsequent audio.
	s.retiring.Add(1)
	go func() {
		defer s.retiring.Done()
		_ = pipeline.Finish()
	}()
}

func (s *AudioSession) Push(pcm []byte, level float64) error {
	if s.ctx.Err() != nil {
		return nil // Capture may flush a final partial block during manual stop.
	}
	now := s.now()
	if level > audioSubtitleQuietLevel {
		s.lastSound = now
		if s.idle {
			s.idle = false
			generation, active := s.subtitles.resume(s.ctx)
			if !active {
				return nil
			}
			if err := s.open(generation); err != nil {
				return err
			}
		}
	} else if !s.idle && now.Sub(s.lastSound) >= audioSubtitleIdleTimeout {
		s.idle = true
		s.subtitles.pause()
		s.retire()
	}
	if s.idle || s.pipeline == nil {
		return nil
	}
	return s.pipeline.Push(pcm)
}

func (s *AudioSession) Finish() {
	s.subtitles.discard()
	s.retire()
	// Push can no longer run: capture has returned. No new retirement can race
	// with this wait, and all model/socket resources are joined before shutdown.
	s.retiring.Wait()
}
