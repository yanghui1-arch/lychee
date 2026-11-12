package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/yanghui1-arch/lychee/internal/diagnostics"
)

var gate = make(chan struct{}, 1)
var persistent *workerProcess // guarded by gate; a single engine owns its inference state.
var lifetime, stopLifetime = context.WithCancel(context.Background())

type workerReport struct {
	Ready       bool               `json:"ready"`
	Error       string             `json:"error"`
	Regions     []Region           `json:"regions"`
	Timings     map[string]float64 `json:"timings_ms"`
	Metrics     map[string]float64 `json:"metrics"`
	Layout      []map[string]any   `json:"layout"`
	Groups      []map[string]any   `json:"groups"`
	RawRegions  []Region           `json:"raw_regions"`
	LayoutError string             `json:"layout_error"`
}
type workerProcess struct {
	cmd         *exec.Cmd
	encoder     *json.Encoder
	reports     chan workerResponse
	done        chan struct{}
	layoutReady bool
}

func (w *workerProcess) stop() { _ = w.cmd.Process.Kill() }

type workerResponse struct {
	report workerReport
	err    error
}

func (w *workerProcess) read(ctx context.Context) (workerReport, error) {
	select {
	case result, ok := <-w.reports:
		if !ok || result.err != nil {
			return workerReport{}, fmt.Errorf("OCR 进程连接中断，请重试")
		}
		return result.report, nil
	case <-ctx.Done():
		w.stop()
		return workerReport{}, ctx.Err()
	}
}

func startWorker(parent context.Context) (result *workerProcess, resultErr error) {
	started := time.Now()
	stages := map[string]float64{}
	var metrics map[string]float64
	layoutOutcome := "success"
	defer func() {
		outcome := "success"
		if resultErr != nil {
			outcome = "error"
			layoutOutcome = "unavailable"
		}
		diagnostics.Record(diagnostics.ScreenshotEvent{Event: "ocr_preload", DurationMS: float64(time.Since(started).Microseconds()) / 1000, Stages: stages, Outcome: outcome, Metrics: metrics})
		diagnostics.Record(diagnostics.ScreenshotEvent{Event: "layout_preload", DurationMS: stages["layout_load_ms"], Metrics: metrics, Outcome: layoutOutcome})
	}()
	python := os.Getenv("LYCHEE_OCR_PYTHON")
	if python == "" {
		exe, _ := os.Executable()
		for _, candidate := range []string{filepath.Join(filepath.Dir(exe), "ocr-runtime", "Scripts", "python.exe"), filepath.Join(".venv-ocr", "Scripts", "python.exe")} {
			if _, err := os.Stat(candidate); err == nil {
				python, _ = filepath.Abs(candidate)
				break
			}
		}
	}
	if python == "" {
		return nil, fmt.Errorf("OCR 运行环境未安装，请运行 scripts/setup-ocr.ps1")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	modelDir, err := prepareModels(cache)
	if err != nil {
		return nil, err
	}
	stages["prepare_ms"] = float64(time.Since(started).Microseconds()) / 1000
	dir, err := os.MkdirTemp("", "lychee-ocr-worker-")
	if err != nil {
		return nil, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			os.RemoveAll(dir)
		}
	}()
	script := filepath.Join(dir, "worker.py")
	if err = os.WriteFile(script, []byte(worker), 0600); err != nil {
		return nil, err
	}
	if err = os.WriteFile(filepath.Join(dir, "grouping.py"), []byte(grouping), 0600); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(lifetime, python, "-u", script, modelDir)
	hideWindow(cmd)
	cmd.Env = append(os.Environ(), "PYTHONUTF8=1")
	cmd.Stderr = io.Discard // Never persist uncontrolled SDK output or image content.
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		input.Close()
		output.Close()
		return nil, err
	}
	cleanup = false
	w := &workerProcess{cmd: cmd, encoder: json.NewEncoder(input), reports: make(chan workerResponse, 1), done: make(chan struct{})}
	go func() {
		defer close(w.done)
		defer close(w.reports)
		defer os.RemoveAll(dir)
		defer input.Close()
		decoder := json.NewDecoder(output)
		for {
			var report workerReport
			err := decoder.Decode(&report)
			if err != nil {
				w.stop()
				break
			}
			select {
			case w.reports <- workerResponse{report: report}:
			case <-lifetime.Done():
				w.stop()
			}
		}
		_ = cmd.Wait()
	}()
	ctx, cancel := context.WithTimeout(parent, 3*time.Minute)
	defer cancel()
	report, err := w.read(ctx)
	if err != nil || !report.Ready {
		w.stop()
		return nil, fmt.Errorf("OCR 预加载失败，请查看运行环境或重试")
	}
	for key, value := range report.Timings {
		stages[key] = value
	}
	metrics = report.Metrics
	w.layoutReady = report.LayoutError == ""
	if report.LayoutError != "" {
		layoutOutcome = "fallback"
	}
	return w, nil
}

// Warmup is called in a background goroutine by the floating-ball process only.
func Warmup() {
	select {
	case gate <- struct{}{}:
	case <-lifetime.Done():
		return
	}
	defer func() { <-gate }()
	if persistent == nil {
		persistent, _ = startWorker(lifetime)
	}
}
func Close() {
	stopLifetime()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case gate <- struct{}{}:
		defer func() { <-gate }()
		if persistent != nil {
			persistent.stop()
			select {
			case <-persistent.done:
			case <-timer.C:
			}
			persistent = nil
		}
	case <-timer.C:
	}
}
