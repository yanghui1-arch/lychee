package subtitle

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func TestEncodeWallpaperPreviewDownscales(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 400, 200))
	for y := range 200 {
		for x := range 400 {
			source.Set(x, y, color.RGBA{R: 200, G: 120, B: 40, A: 255})
		}
	}
	var raw bytes.Buffer
	if err := png.Encode(&raw, source); err != nil {
		t.Fatal(err)
	}
	encoded, err := encodeWallpaperPreview(raw.Bytes(), 100)
	if err != nil {
		t.Fatal(err)
	}
	const prefix = "data:image/jpeg;base64,"
	if !strings.HasPrefix(encoded, prefix) {
		t.Fatalf("unexpected data URL prefix: %.40s", encoded)
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encoded, prefix))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if got := decoded.Bounds().Size(); got != image.Pt(100, 50) {
		t.Fatalf("size = %v, want 100x50", got)
	}
	r, g, b, _ := decoded.At(50, 25).RGBA()
	if diff := func(a uint32, want int) int { d := int(a>>8) - want; return max(d, -d) }; diff(r, 200) > 6 || diff(g, 120) > 6 || diff(b, 40) > 6 {
		t.Fatalf("color = %d,%d,%d, want about 200,120,40", r>>8, g>>8, b>>8)
	}
}

func TestEncodeWallpaperPreviewRejectsUnknownFormat(t *testing.T) {
	if _, err := encodeWallpaperPreview([]byte("not an image"), 100); err == nil {
		t.Fatal("expected decode error")
	}
}
