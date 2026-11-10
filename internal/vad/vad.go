package vad

// Silero v6 consumes 512 samples (32 ms) at 16 kHz. Keep this independent
// from WASAPI packet boundaries and retain real audio around each utterance.
const FrameBytes = 512 * 2

type Detector interface {
	Probability([]byte) (float32, error)
	Close()
}
