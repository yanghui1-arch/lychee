package ocr

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/yanghui1-arch/lychee/internal/diagnostics"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed worker.py
var worker string

//go:embed grouping.py
var grouping string

type Box struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type EraseRun struct {
	X     int   `json:"x"`
	Y     int   `json:"y"`
	Width int   `json:"width"`
	Color []int `json:"color"`
}
type Region struct {
	Erase  []EraseRun `json:"erase"`
	Text   string     `json:"text"`
	Score  float64    `json:"score"`
	X      int        `json:"x"`
	Y      int        `json:"y"`
	Width  int        `json:"width"`
	Height int        `json:"height"`
	Boxes  []Box      `json:"boxes,omitempty"`
}

func Recognize(ctx context.Context, data string) ([]Region, error) {
	return RecognizeSession(ctx, data, "")
}

func RecognizeSession(ctx context.Context, data, session string) (regions []Region, resultErr error) {
	started := time.Now()
	stages := map[string]float64{}
	defer func() {
		outcome := "success"
		if resultErr != nil {
			outcome = "error"
		}
		if errors.Is(resultErr, context.DeadlineExceeded) {
			outcome = "timeout"
		}
		if errors.Is(resultErr, context.Canceled) {
			outcome = "cancelled"
		}
		diagnostics.Record(diagnostics.ScreenshotEvent{Session: session, Event: "ocr", DurationMS: float64(time.Since(started).Microseconds()) / 1000, Regions: len(regions), Outcome: outcome, Stages: stages, Details: regions})
	}()
	if !strings.HasPrefix(data, "data:image/png;base64,") || len(data) > 48<<20 {
		return nil, fmt.Errorf("无效或过大的截图")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, "data:image/png;base64,"))
	if err != nil {
		return nil, fmt.Errorf("无效的截图编码")
	}
	config, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil || config.Width > 30000 || config.Height > 30000 || int64(config.Width)*int64(config.Height) > 60000000 {
		return nil, fmt.Errorf("截图尺寸超出 OCR 限制")
	}

	dir, err := os.MkdirTemp("", "lychee-ocr-image-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	imagePath := filepath.Join(dir, "capture.png")
	if err = os.WriteFile(imagePath, raw, 0600); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	waitStarted := time.Now()
	select {
	case gate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-gate }()
	if persistent == nil {
		persistent, err = startWorker(ctx)
		if err != nil {
			return nil, err
		}
	}
	stages["wait_ready_ms"] = float64(time.Since(waitStarted).Microseconds()) / 1000
	processStarted := time.Now()
	if err = persistent.encoder.Encode(map[string]string{"image": imagePath}); err != nil {
		persistent.stop()
		persistent = nil
		return nil, fmt.Errorf("OCR 进程已退出，请重试")
	}
	report, err := persistent.read(ctx)
	stages["request_ms"] = float64(time.Since(processStarted).Microseconds()) / 1000
	if err != nil {
		persistent.stop()
		persistent = nil
		return nil, err
	}
	if report.Error != "" {
		return nil, fmt.Errorf("OCR 推理失败: %s", report.Error)
	}
	regions = report.Regions
	layoutOutcome := "success"
	if report.LayoutError != "" {
		layoutOutcome = "fallback"
	}
	if report.Metrics == nil {
		report.Metrics = map[string]float64{}
	}
	report.Metrics["raw_regions_count"] = float64(len(report.RawRegions))
	report.Metrics["translation_groups_count"] = float64(len(regions))
	report.Metrics["image_width_px"] = float64(config.Width)
	report.Metrics["image_height_px"] = float64(config.Height)
	diagnostics.Record(diagnostics.ScreenshotEvent{Session: session, Event: "layout_analysis", DurationMS: report.Timings["layout_ms"], Regions: len(report.Layout), Outcome: layoutOutcome, Metrics: report.Metrics, Layout: report.Layout, Groups: report.Groups, Stages: map[string]float64{"grouping_ms": report.Timings["grouping_ms"]}, Details: map[string]any{"raw_regions": report.RawRegions, "grouped_regions": regions, "layout_error": report.LayoutError}})
	for name, value := range report.Timings {
		stages[name] = value
	}
	return regions, nil
}
