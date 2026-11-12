package screenshot

import (
	"image"
	"testing"
)

func TestPinnedScreenshotOffsetsFromSelection(t *testing.T) {
	for _, test := range []struct {
		display    image.Rectangle
		x, y, w, h int
		want       image.Rectangle
	}{
		{image.Rect(0, 0, 1920, 1080), 720, 300, 640, 400, image.Rect(732, 312, 1372, 712)},
		{image.Rect(-1920, 100, 0, 1180), 300, 200, 500, 300, image.Rect(-1608, 312, -1108, 612)},
		{image.Rect(0, 0, 1920, 1080), 1800, 1000, 40, 30, image.Rect(1812, 1012, 1852, 1042)},
		{image.Rect(0, 0, 1920, 1080), 0, 0, 1920, 1080, image.Rect(0, 0, 1920, 1080)},
		{image.Rect(0, 0, 1920, 1080), 300, 200, 600, 3000, image.Rect(312, 0, 528, 1080)},
	} {
		got, err := pinnedBounds(test.display, test.x, test.y, test.w, test.h)
		if err != nil || got != test.want {
			t.Fatalf("got %v (%v), want %v", got, err, test.want)
		}
	}
}
