//go:build windows && amd64

package vad

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
	"github.com/yanghui1-arch/lychee/models"
)

var vadRuntimeOnce sync.Once
var vadRuntimeError error

type sileroVAD struct {
	session                         *ort.AdvancedSession
	input, state, output, nextState *ort.Tensor[float32]
	rate                            *ort.Tensor[int64]
}

func NewSileroVAD() (Detector, error) {
	vadRuntimeOnce.Do(func() {
		cache, err := os.UserCacheDir()
		if err != nil {
			vadRuntimeError = err
			return
		}
		digest := sha256.Sum256(models.ONNXRuntimeDLL)
		dir := filepath.Join(cache, "Lychee", fmt.Sprintf("onnxruntime-1.22-%x", digest[:8]))
		if err = os.MkdirAll(dir, 0700); err != nil {
			vadRuntimeError = err
			return
		}
		path := filepath.Join(dir, "onnxruntime.dll")
		// Keep upstream notices with the extracted runtime, including when the
		// user receives only the standalone desktop executable.
		for _, name := range []string{"silero-LICENSE", "onnxruntime-LICENSE", "onnxruntime-NOTICES.txt"} {
			data, readErr := models.Notices.ReadFile(name)
			if readErr != nil {
				vadRuntimeError = readErr
				return
			}
			if err = os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
				vadRuntimeError = err
				return
			}
		}
		existing, _ := os.ReadFile(path)
		if sha256.Sum256(existing) != digest {
			file, createErr := os.CreateTemp(dir, "runtime-*.dll")
			if createErr != nil {
				vadRuntimeError = createErr
				return
			}
			name := file.Name()
			defer os.Remove(name)
			_, err = file.Write(models.ONNXRuntimeDLL)
			closeErr := file.Close()
			if err == nil {
				err = closeErr
			}
			if err == nil {
				err = os.Rename(name, path)
			}
			if err != nil {
				vadRuntimeError = err
				return
			}
		}
		ort.SetSharedLibraryPath(path)
		// The environment lives for the process; sessions/tensors are per capture.
		vadRuntimeError = ort.InitializeEnvironment()
	})
	if vadRuntimeError != nil {
		return nil, fmt.Errorf("无法加载 Silero VAD：%w", vadRuntimeError)
	}
	v := &sileroVAD{}
	var err error
	defer func() {
		if err != nil {
			v.Close()
		}
	}()
	v.input, err = ort.NewEmptyTensor[float32](ort.NewShape(1, 576))
	if err != nil {
		return nil, err
	}
	v.state, err = ort.NewEmptyTensor[float32](ort.NewShape(2, 1, 128))
	if err != nil {
		return nil, err
	}
	v.rate, err = ort.NewTensor[int64](ort.NewShape(1), []int64{16000})
	if err != nil {
		return nil, err
	}
	v.output, err = ort.NewEmptyTensor[float32](ort.NewShape(1, 1))
	if err != nil {
		return nil, err
	}
	v.nextState, err = ort.NewEmptyTensor[float32](ort.NewShape(2, 1, 128))
	if err != nil {
		return nil, err
	}
	options, err := ort.NewSessionOptions()
	if err != nil {
		return nil, err
	}
	defer options.Destroy()
	if err = options.SetIntraOpNumThreads(1); err != nil {
		return nil, err
	}
	if err = options.SetInterOpNumThreads(1); err != nil {
		return nil, err
	}
	v.session, err = ort.NewAdvancedSessionWithONNXData(models.SileroModel, []string{"input", "state", "sr"}, []string{"output", "stateN"}, []ort.Value{v.input, v.state, v.rate}, []ort.Value{v.output, v.nextState}, options)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (v *sileroVAD) Probability(pcm []byte) (float32, error) {
	input := v.input.GetData()
	for i := 0; i < 512; i++ {
		input[64+i] = float32(int16(binary.LittleEndian.Uint16(pcm[i*2:]))) / 32768
	}
	if err := v.session.Run(); err != nil {
		return 0, err
	}
	copy(v.state.GetData(), v.nextState.GetData())
	copy(input[:64], input[512:576]) // Silero's required 64-sample context.
	return v.output.GetData()[0], nil
}

func (v *sileroVAD) Close() {
	if v.session != nil {
		v.session.Destroy()
	}
	for _, tensor := range []*ort.Tensor[float32]{v.input, v.state, v.output, v.nextState} {
		if tensor != nil {
			tensor.Destroy()
		}
	}
	if v.rate != nil {
		v.rate.Destroy()
	}
}
