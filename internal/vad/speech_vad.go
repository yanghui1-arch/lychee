package vad

type Packet struct {
	PCM []byte
	End bool
}
type Gate struct {
	pending        []byte
	preroll        [][]byte
	active         bool
	onset, silence int
}

func (g *Gate) Process(pcm []byte, detector Detector, send func(Packet) error) error {
	g.pending = append(g.pending, pcm...)
	for len(g.pending) >= FrameBytes {
		frame := append([]byte(nil), g.pending[:FrameBytes]...)
		g.pending = g.pending[FrameBytes:]
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
				if err := send(Packet{PCM: previous}); err != nil {
					return err
				}
			}
			g.preroll = nil
			continue
		}
		if err := send(Packet{PCM: frame}); err != nil {
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
			if err := send(Packet{End: true}); err != nil {
				return err
			}
			g.active, g.onset, g.silence = false, 0, 0
		}
	}
	return nil
}

func (g *Gate) Finish(send func(Packet) error) error {
	if g.active && len(g.pending) > 0 {
		if err := send(Packet{PCM: append([]byte(nil), g.pending...)}); err != nil {
			return err
		}
	}
	return nil
}
