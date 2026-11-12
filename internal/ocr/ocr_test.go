package ocr

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRejectInvalidImageBeforeStartingRuntime(t *testing.T) {
	for _, data := range []string{"", "data:image/jpeg;base64,AAAA", "data:image/png;base64,???", "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("not a PNG"))} {
		if _, err := Recognize(context.Background(), data); err == nil {
			t.Fatalf("accepted invalid image %q", data)
		}
	}
}

func TestBundledModelsRestoreMissingAndCorruptFiles(t *testing.T) {
	cache := t.TempDir()
	root, err := prepareModels(cache)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "PP-OCRv6_small_det", "inference.yml")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = prepareModels(cache); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(path)
	if err != nil || string(original) != string(restored) {
		t.Fatal("model corruption was not repaired")
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err = prepareModels(cache); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatal("missing model was not restored", err)
	}
}

func TestSmallModelIntegration(t *testing.T) {
	path := os.Getenv("LYCHEE_OCR_TEST_IMAGE")
	if path == "" {
		t.Skip("set LYCHEE_OCR_TEST_IMAGE to run real PP-OCRv6 Small inference")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	Warmup()
	process := persistent
	if process == nil {
		t.Fatal("preload failed")
	}
	if !process.layoutReady {
		t.Fatal("layout model failed to load")
	}
	defer func() {
		Close()
		select {
		case <-process.done:
		default:
			t.Error("OCR worker did not exit on close")
		}
	}()
	for i := 0; i < 2; i++ {
		if _, err := Recognize(context.Background(), "data:image/png;base64,"+base64.StdEncoding.EncodeToString(raw)); err != nil {
			t.Fatal(err)
		}
		if persistent != process {
			t.Fatal("worker was reloaded between requests")
		}
	}
	regions, err := Recognize(context.Background(), "data:image/png;base64,"+base64.StdEncoding.EncodeToString(raw))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, region := range regions {
		t.Logf("%q at (%d,%d) %dx%d score %.3f", region.Text, region.X, region.Y, region.Width, region.Height, region.Score)
		if region.Width <= 0 || region.Height <= 0 {
			t.Fatal("empty text region")
		}
		if strings.Contains(strings.ToLower(region.Text), "lychee") {
			found = true
		}
	}
	if !found {
		t.Fatal("test image must contain recognizable 'Lychee' text")
	}
}
