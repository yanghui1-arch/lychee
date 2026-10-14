//go:build !windows || !amd64

package main

import (
	"context"
	"errors"
)

func listAudioSources() ([]AudioSource, error) {
	return nil, errors.New("应用音频监听暂仅支持 Windows 64 位")
}
func captureAudio(context.Context, AudioSource, func(), func([]byte)) error {
	return errors.New("应用音频监听暂仅支持 Windows 64 位")
}
