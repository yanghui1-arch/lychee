//go:build !windows || !amd64

package vad

import (
	"errors"
)

func NewSileroVAD() (Detector, error) {
	return nil, errors.New("Silero VAD 当前支持 Windows 64 位桌面端")
}
