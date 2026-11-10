package vad

import (
	"bytes"
	"testing"
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
		frame := make([]byte, FrameBytes)
		if i >= 15 && i < 18 || i >= 58 {
			frame[0] = 1
		}
		input = append(input, frame...)
	}
	input = append(input, 3, 0, 4, 0)
	var packets []Packet
	consume := func(packet Packet) error { packets = append(packets, packet); return nil }
	var gate Gate
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
		if packet.End {
			ends++
		} else {
			frames++
		}
	}
	if ends != 1 || frames != 54 {
		t.Fatalf("unexpected gate output: ends=%d frames=%d", ends, frames)
	}
	if !bytes.Equal(packets[len(packets)-1].PCM, []byte{3, 0, 4, 0}) {
		t.Fatal("stop lost partial PCM tail")
	}
	if packets[0].PCM[0] != 0 || packets[8].PCM[0] != 1 {
		t.Fatal("speech onset/pre-roll lost")
	}
}
