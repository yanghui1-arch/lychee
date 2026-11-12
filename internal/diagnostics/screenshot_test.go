package diagnostics

import (
	"strings"
	"testing"
)

func TestScreenshotContentOnlyInDebugLogs(t *testing.T) {
	event := ScreenshotEvent{Session: "s1", Event: "translation_region", DurationMS: 120, Text: "private source", Result: "private result", Details: []string{"private OCR"}}
	normal, _ := encode(event, false)
	if strings.Contains(string(normal), "private") {
		t.Fatal("content leaked into normal logs")
	}
	if !strings.Contains(string(normal), `"duration_ms":120`) {
		t.Fatal("timing missing")
	}
	debug, _ := encode(event, true)
	if !strings.Contains(string(debug), "private source") || !strings.Contains(string(debug), "private result") {
		t.Fatal("debug content missing")
	}
}
