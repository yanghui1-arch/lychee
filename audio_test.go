package main

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"sync/atomic"
	"testing"
	"time"
)

func TestPCMLevel(t *testing.T) {
	for _, test := range []struct {
		samples  []int16
		expected float64
	}{
		{nil, 0}, {[]int16{0, 0}, 0}, {[]int16{-32768, -32768}, 1}, {[]int16{16384, -16384}, 0.5},
	} {
		pcm := make([]byte, len(test.samples)*2)
		for i, sample := range test.samples {
			binary.LittleEndian.PutUint16(pcm[i*2:], uint16(sample))
		}
		if got := pcmLevel(pcm); math.Abs(got-test.expected) > 1e-9 {
			t.Fatalf("level=%v want %v", got, test.expected)
		}
	}
}

func TestAudioServiceLifecycle(t *testing.T) {
	a := newAudioService()
	a.list = func() ([]AudioSource, error) { return []AudioSource{{ID: "one"}, {ID: "two"}, {ID: "failed"}}, nil }
	var active, stopped atomic.Int32
	a.capture = func(ctx context.Context, source AudioSource, ready func(), chunk func([]byte)) error {
		if source.ID == "failed" {
			return errors.New("test device error")
		}
		if active.Add(1) != 1 {
			t.Error("two capture sessions active")
		}
		defer active.Add(-1)
		ready()
		chunk(make([]byte, 6400))
		<-ctx.Done()
		stopped.Add(1)
		return nil
	}
	defer a.StopAudioCapture()
	if _, err := a.StartAudioCapture("one"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.StartAudioCapture("missing"); err == nil {
		t.Fatal("invalid source accepted")
	}
	if a.GetAudioStatus().Source.ID != "one" || active.Load() != 1 {
		t.Fatal("invalid selection changed running source")
	}
	if _, err := a.StartAudioCapture("two"); err != nil {
		t.Fatal(err)
	}
	if stopped.Load() != 1 || a.GetAudioStatus().Source.ID != "two" {
		t.Fatal("old source was not stopped before switching")
	}
	a.StopAudioCapture()
	a.StopAudioCapture()
	if active.Load() != 0 || a.GetAudioStatus().State != "idle" {
		t.Fatal("capture was not stopped")
	}
	if _, err := a.StartAudioCapture("failed"); err == nil {
		t.Fatal("capture error lost")
	}
	deadline := time.Now().Add(time.Second)
	for a.GetAudioStatus().State != "error" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if a.GetAudioStatus().Error == "" {
		t.Fatal("failure not exposed")
	}
}
