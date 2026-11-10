//go:build windows && amd64

package screenshot

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

func TestScreenshotClipboardImage(t *testing.T) {
	// Include a non-zero origin and transparent pixel to catch color/row errors.
	img := image.NewNRGBA(image.Rect(3, 4, 5, 6))
	img.SetNRGBA(3, 4, color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(4, 4, color.NRGBA{G: 255, A: 255})
	img.SetNRGBA(3, 5, color.NRGBA{B: 255, A: 255})
	data := screenshotDIB(img)
	if len(data) != 56 || binary.LittleEndian.Uint32(data[:4]) != 40 || binary.LittleEndian.Uint32(data[4:8]) != 2 || int32(binary.LittleEndian.Uint32(data[8:12])) != -2 || binary.LittleEndian.Uint16(data[14:16]) != 32 {
		t.Fatal("clipboard bitmap dimensions or format incorrect")
	}
	want := []byte{0, 0, 255, 255, 0, 255, 0, 255, 255, 0, 0, 255, 255, 255, 255, 255}
	for i, b := range want {
		if data[40+i] != b {
			t.Fatalf("clipboard bitmap color/orientation mismatch at %d", i)
		}
	}
}
