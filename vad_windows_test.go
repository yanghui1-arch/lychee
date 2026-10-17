//go:build windows && amd64

package main

import (
	"encoding/binary"
	"os"
	"testing"
)

func TestSileroBundledModelInference(t *testing.T) {
	vad, err := newSileroVAD()
	if err != nil {
		t.Fatal(err)
	}
	defer vad.Close()
	for i := 0; i < 32; i++ {
		probability, err := vad.Probability(make([]byte, vadFrameBytes))
		if err != nil {
			t.Fatal(err)
		}
		if probability >= 0.5 || probability < 0 {
			t.Fatalf("silence detected as speech: %v", probability)
		}
	}
	// Optional real speech smoke check, generated locally with Windows SAPI.
	if path := os.Getenv("LYCHEE_SPEECH_TEST_WAV"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) < 44 || string(data[:4]) != "RIFF" {
			t.Fatal("expected WAV")
		}
		var pcm []byte
		for pos := 12; pos+8 <= len(data); {
			size := int(binary.LittleEndian.Uint32(data[pos+4:]))
			if pos+8+size > len(data) {
				t.Fatal("invalid WAV chunk")
			}
			if string(data[pos:pos+4]) == "data" {
				pcm = data[pos+8 : pos+8+size]
				break
			}
			pos += 8 + size + (size % 2)
		}
		voiced := 0
		for len(pcm) >= vadFrameBytes {
			probability, err := vad.Probability(pcm[:vadFrameBytes])
			if err != nil {
				t.Fatal(err)
			}
			if probability >= 0.5 {
				voiced++
			}
			pcm = pcm[vadFrameBytes:]
		}
		if voiced < 5 {
			t.Fatalf("real speech not detected: %d voiced frames", voiced)
		}
		t.Logf("Silero detected %d real speech frames", voiced)
	}
}
