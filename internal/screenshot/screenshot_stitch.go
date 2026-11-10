package screenshot

import (
	"image"
	"image/draw"
	"math"
)

// Find the repeated vertical strip; append only newly exposed content. Sample
// pixels across the viewport rather than relying on an application's scroll step.
func appendScrollFrame(long, previous, next *image.RGBA) (*image.RGBA, bool) {
	w, h := next.Bounds().Dx(), next.Bounds().Dy()
	score := func(shift int) float64 {
		var sum float64
		var n int
		for y := 0; y < h-shift; y += 7 {
			for x := 8; x < w-8; x += max(1, w/32) {
				a, b := previous.RGBAAt(x, y+shift), next.RGBAAt(x, y)
				sum += math.Abs(float64(a.R)-float64(b.R)) + math.Abs(float64(a.G)-float64(b.G)) + math.Abs(float64(a.B)-float64(b.B))
				n += 3
			}
		}
		if n == 0 {
			return 255
		}
		return sum / float64(n)
	}
	if score(0) < 1.5 {
		return long, false
	}
	best, delta := 255.0, 0
	for shift := 8; shift < h-32; shift++ {
		if value := score(shift); value < best {
			best, delta = value, shift
		}
	}
	if best > 18 || delta == 0 {
		return long, false
	}
	result := image.NewRGBA(image.Rect(0, 0, w, long.Bounds().Dy()+delta))
	draw.Draw(result, long.Bounds(), long, image.Point{}, draw.Src)
	draw.Draw(result, image.Rect(0, long.Bounds().Dy(), w, result.Bounds().Dy()), next, image.Pt(0, h-delta), draw.Src)
	return result, true
}
