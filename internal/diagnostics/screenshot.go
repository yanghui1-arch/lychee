package diagnostics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ScreenshotEvent contains only explicit diagnostics fields, never credentials or images.
type ScreenshotEvent struct {
	Session    string             `json:"session"`
	Event      string             `json:"event"`
	DurationMS float64            `json:"duration_ms"`
	Region     int                `json:"region"`
	Regions    int                `json:"regions,omitempty"`
	Completed  int                `json:"completed,omitempty"`
	Failed     int                `json:"failed,omitempty"`
	Outcome    string             `json:"outcome,omitempty"`
	Stages     map[string]float64 `json:"stages_ms,omitempty"`
	Text       string             `json:"text,omitempty"`
	Result     string             `json:"result,omitempty"`
	Details    any                `json:"details,omitempty"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
	Layout     []map[string]any   `json:"layout,omitempty"`
	Groups     []map[string]any   `json:"groups,omitempty"`
}

var mu sync.Mutex

func encode(event ScreenshotEvent, debug bool) ([]byte, error) {
	if !debug {
		event.Text = ""
		event.Result = ""
		event.Details = nil
	}
	return json.Marshal(struct {
		Time string `json:"time"`
		ScreenshotEvent
	}{time.Now().Format(time.RFC3339Nano), event})
}

// One bounded file per process/day avoids contention with the workspace process.
func Record(event ScreenshotEvent) {
	if len(event.Session) > 80 || len(event.Event) > 80 {
		return
	}
	line, err := encode(event, os.Getenv("LYCHEE_SCREENSHOT_DEBUG") == "1")
	if err != nil {
		return
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	dir := filepath.Join(cache, "Lychee", "logs")
	if os.MkdirAll(dir, 0700) != nil {
		return
	}
	path := filepath.Join(dir, fmt.Sprintf("screenshot-%s-%d.jsonl", time.Now().Format("2006-01-02"), os.Getpid()))
	if info, err := os.Stat(path); err == nil && info.Size() > 20<<20 {
		os.Remove(path + ".1")
		if os.Rename(path, path+".1") != nil {
			return
		}
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Write(append(line, '\n'))
}
