package main

import "testing"

func TestSubtitleBackdropUsesInkBoundsAndFontPadding(t *testing.T) {
	const width, height = 100, 100
	ink := make([]byte, width*height)
	ink[45*width+40], ink[54*width+59] = 255, 255
	region := []subtitleInkRegion{{0, 0, width, height}}
	for _, font := range []float64{18, 40} {
		mask := subtitleBackdropMask(ink, width, height, font, region)
		if mask[0] != 0 || mask[99*width+50] != 0 || mask[50*width+50] != 255 {
			t.Fatal("background is not limited to ink bounds")
		}
		// 40px type needs 14px horizontal padding, 18px type only 6px.
		if (mask[50*width+27] == 255) != (font == 40) {
			t.Fatal("backdrop padding does not follow font size")
		}
	}
	clear(ink)
	for _, alpha := range subtitleBackdropMask(ink, width, height, 30, region) {
		if alpha != 0 {
			t.Fatal("empty text painted a backdrop")
		}
	}
}
