package screenshot

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func TestScrollingScreenshotStitchesOnlyNewContent(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 240, 1200))
	for y := 0; y < 1200; y++ {
		for x := 0; x < 240; x++ {
			page.SetRGBA(x, y, color.RGBA{uint8((x*7 + y*17) % 251), uint8((x*3 + y*11) % 253), uint8((x + y*23) % 249), 255})
		}
	}
	frame := func(top int) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, 240, 400))
		draw.Draw(img, img.Bounds(), page, image.Pt(0, top), draw.Src)
		return img
	}
	first, next := frame(0), frame(180)
	merged, added := appendScrollFrame(first, first, next)
	if !added || merged.Bounds().Dy() != 580 {
		t.Fatalf("expected 580px stitched screenshot, got %v, added=%t", merged.Bounds(), added)
	}
	for y := 0; y < 580; y++ {
		if merged.RGBAAt(100, y) != page.RGBAAt(100, y) {
			t.Fatalf("seam or repeated pixels at row %d", y)
		}
	}
	unchanged, added := appendScrollFrame(merged, next, next)
	if added || unchanged != merged {
		t.Fatal("bottom of page should not append duplicate content")
	}
}
