//go:build !windows || !amd64

package app

import "errors"

func newSileroVAD() (voiceDetector, error) {
	return nil, errors.New("Silero VAD 当前支持 Windows 64 位桌面端")
}
