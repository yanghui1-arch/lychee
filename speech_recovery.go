package main

import (
	"errors"
	"sync"
	"time"
)

const speechInterruptionNotice = "识别服务修订了已翻译内容，当前片段可能不完整；已保留字幕，继续接收后续音频"

// Only this explicit upstream consistency failure is recoverable. Never retry
// authentication/balance/network failures, and never replay already sent PCM.
type speechRecovery struct {
	mu     sync.Mutex
	recent []time.Time
}

func (r *speechRecovery) allow(err error, now time.Time) (bool, error) {
	var serviceError *speechServiceError
	if !errors.As(err, &serviceError) || serviceError.code != "asr_committed_text_changed" {
		return false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for len(r.recent) > 0 && now.Sub(r.recent[0]) >= 30*time.Second {
		r.recent = r.recent[1:]
	}
	if len(r.recent) >= 3 {
		return false, errors.New("识别服务在 30 秒内多次修改已翻译内容，已停止监听，请稍后重试")
	}
	r.recent = append(r.recent, now)
	return true, nil
}
