package main

import (
	"errors"
	"sync"
)

type queuedSpeechOutput struct {
	events    []speechEvent
	completed bool
}

// Consecutive VAD utterances use independent server sessions. A later session
// may translate faster, so keep its results bounded and display in audio order.
type orderedSpeechOutput struct {
	mu            sync.Mutex
	current, next int
	streams       map[int]*queuedSpeechOutput
	receive       func(speechEvent)
	failed        func(error)
}

func (o *orderedSpeechOutput) Open() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.next++
	if o.current == 0 {
		o.current = o.next
	}
	o.streams[o.next] = &queuedSpeechOutput{}
	return o.next
}

func (o *orderedSpeechOutput) Accept(id int, event speechEvent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if id == o.current {
		o.receive(event)
		return
	}
	stream := o.streams[id]
	if stream == nil {
		return
	}
	// Coalesce replaceable snapshots, but never discard settled translations.
	if event.Type == "translation_preview" || event.Type == "transcript" && !event.IsFinal {
		for i := len(stream.events) - 1; i >= 0; i-- {
			previous := stream.events[i]
			if previous.Type == event.Type && previous.Sequence == event.Sequence && previous.SentenceID == event.SentenceID && !previous.IsFinal {
				stream.events[i] = event
				return
			}
		}
	}
	if len(stream.events) >= 256 {
		o.failed(errors.New("字幕响应积压过多，已停止语音会话"))
		return
	}
	stream.events = append(stream.events, event)
}

func (o *orderedSpeechOutput) Complete(id int) {
	o.complete(id, false)
}

func (o *orderedSpeechOutput) Interrupt(id int) {
	o.complete(id, true)
}

func (o *orderedSpeechOutput) complete(id int, interrupted bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if stream := o.streams[id]; stream != nil {
		if interrupted {
			// Keep the interruption after this session's previews and before the
			// next session, so only its unconfirmed text is marked incomplete.
			stream.events = append(stream.events, speechEvent{Type: "session_interrupted"})
		}
		stream.completed = true
	}
	for {
		stream := o.streams[o.current]
		if stream == nil {
			return
		}
		for _, event := range stream.events {
			o.receive(event)
		}
		stream.events = nil
		if !stream.completed {
			return
		}
		delete(o.streams, o.current)
		o.current++
	}
}
