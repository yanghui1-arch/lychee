package main

// Silero v6 consumes 512 samples (32 ms) at 16 kHz. Keep this independent
// from WASAPI packet boundaries and retain real audio around each utterance.
const vadFrameBytes = 512 * 2

type voiceDetector interface {
	Probability([]byte) (float32, error)
	Close()
}

type speechPacket struct {
	pcm []byte
	end bool
}
type voiceGate struct {
	pending        []byte
	preroll        [][]byte
	active         bool
	onset, silence int
}

func (g *voiceGate) Process(pcm []byte, detector voiceDetector, send func(speechPacket) error) error {
	g.pending = append(g.pending, pcm...)
	for len(g.pending) >= vadFrameBytes {
		frame := append([]byte(nil), g.pending[:vadFrameBytes]...)
		g.pending = g.pending[vadFrameBytes:]
		probability, err := detector.Probability(frame)
		if err != nil {
			return err
		}
		if !g.active {
			g.preroll = append(g.preroll, frame)
			if len(g.preroll) > 10 {
				g.preroll = g.preroll[1:]
			}
			if probability >= 0.5 {
				g.onset++
			} else {
				g.onset = 0
			}
			if g.onset < 2 {
				continue
			}
			g.active, g.silence = true, 0
			for _, previous := range g.preroll {
				if err := send(speechPacket{pcm: previous}); err != nil {
					return err
				}
			}
			g.preroll = nil
			continue
		}
		if err := send(speechPacket{pcm: frame}); err != nil {
			return err
		}
		if probability < 0.35 {
			g.silence++
		} else {
			g.silence = 0
		}
		// 1.28 s of genuine trailing silence lets upstream ASR finalize its
		// sentence. Never concatenate speech across a removed silence gap.
		if g.silence >= 40 {
			if err := send(speechPacket{end: true}); err != nil {
				return err
			}
			g.active, g.onset, g.silence = false, 0, 0
		}
	}
	return nil
}

func (g *voiceGate) Finish(send func(speechPacket) error) error {
	if g.active && len(g.pending) > 0 {
		if err := send(speechPacket{pcm: append([]byte(nil), g.pending...)}); err != nil {
			return err
		}
	}
	return nil
}
