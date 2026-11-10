package screenshot

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

// textPage draws sparse, distinct "text lines" like a document or chat log.
func textPage(width, height int, seed uint32) *image.RGBA {
	page := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(page, page.Bounds(), image.NewUniform(color.RGBA{250, 250, 250, 255}), image.Point{}, draw.Src)
	for line := 0; line*22+16 < height; line++ {
		glyphs := 8 + int((uint32(line)*2654435761^seed)%uint32(max(1, width/8-10)))
		for glyph := 0; glyph < glyphs; glyph++ {
			hash := (uint32(line+1)*2654435761 ^ uint32(glyph+1)*2246822519) + seed
			for y := 0; y < 9; y++ {
				for x := 0; x < 5; x++ {
					if (hash>>uint((x+y*5)%31))&1 != 0 {
						page.SetRGBA(12+glyph*7+x, line*22+6+y, color.RGBA{30, 30, 34, 255})
					}
				}
			}
		}
	}
	return page
}

func fill(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
}

func samePixels(t *testing.T, got *image.RGBA, gotRect image.Rectangle, want *image.RGBA, wantMin image.Point) {
	t.Helper()
	for y := gotRect.Min.Y; y < gotRect.Max.Y; y++ {
		for x := gotRect.Min.X; x < gotRect.Max.X; x++ {
			if got.RGBAAt(x, y) != want.RGBAAt(wantMin.X+x-gotRect.Min.X, wantMin.Y+y-gotRect.Min.Y) {
				t.Fatalf("missing or duplicated content at %d,%d", x, y)
			}
		}
	}
}

// A chat-style window: fixed header and input box around a scrolling list.
// The tracking context spans the whole window, the selection a short strip.
func TestScrollCaptureIgnoresFixedBarsAndFillsLongSteps(t *testing.T) {
	const width, screenHeight, header, footer = 360, 620, 64, 80
	page := textPage(width, 4000, 7)
	screen := func(top int) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, width, screenHeight))
		draw.Draw(img, image.Rect(0, header, width, screenHeight-footer), page, image.Pt(0, top), draw.Src)
		fill(img, image.Rect(0, 0, width, header), color.RGBA{40, 44, 52, 255})
		fill(img, image.Rect(20, 20, 200, 40), color.RGBA{230, 230, 240, 255})
		fill(img, image.Rect(0, screenHeight-footer, width, screenHeight), color.RGBA{236, 238, 242, 255})
		fill(img, image.Rect(16, screenHeight-60, 300, screenHeight-24), color.RGBA{255, 255, 255, 255})
		return img
	}
	for _, tc := range []struct {
		name  string
		area  image.Rectangle
		steps []int
	}{
		{"short strip, steps longer than the strip", image.Rect(0, 230, width, 350), []int{40, 130, 90, 160, 30, 150, 140, 0, 125}},
		{"tiny strip", image.Rect(0, 300, width, 348), []int{20, 70, 65, 90, 10, 80, 75}},
		{"strip touching the input box", image.Rect(0, 380, width, screenHeight-footer), []int{60, 120, 100, 110, 45, 120}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			capture := newScrollCapture(screen(0), tc.area)
			top, furthest := 0, 0
			for _, step := range tc.steps {
				top += step
				furthest = max(furthest, top)
				if _, err := capture.append(screen(top)); err != nil {
					t.Fatal(err)
				}
				if capture.top != top || capture.long.Bounds().Dy() != tc.area.Dy()+furthest {
					t.Fatalf("scrolled to %d: tracked=%d height=%d want %d", top, capture.top, capture.long.Bounds().Dy(), tc.area.Dy()+furthest)
				}
			}
			start := tc.area.Min.Y - header
			samePixels(t, capture.long, capture.long.Bounds(), page, image.Pt(0, start))
		})
	}
}

func TestScrollCaptureKeepsStickyFooterOnce(t *testing.T) {
	const width, height, bar = 300, 300, 30
	page := textPage(width, 2000, 3)
	frame := func(top int) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(img, image.Rect(0, 0, width, height-bar), page, image.Pt(0, top), draw.Src)
		fill(img, image.Rect(0, height-bar, width, height), color.RGBA{24, 90, 200, 255})
		fill(img, image.Rect(30, height-20, 120, height-10), color.RGBA{255, 255, 255, 255})
		return img
	}
	capture := newScrollCapture(frame(0), image.Rect(0, 0, width, height))
	for _, top := range []int{50, 120, 200, 140, 260} {
		if _, err := capture.append(frame(top)); err != nil {
			t.Fatal(err)
		}
		if capture.top != top {
			t.Fatalf("scrolled to %d: tracked %d", top, capture.top)
		}
	}
	content := height - bar + 260
	if capture.long.Bounds().Dy() != content+bar {
		t.Fatalf("height=%d want %d", capture.long.Bounds().Dy(), content+bar)
	}
	samePixels(t, capture.long, image.Rect(0, 0, width, content), page, image.Point{})
	samePixels(t, capture.long, image.Rect(0, content, width, content+bar), frame(0), image.Pt(0, height-bar))
}

// Hover highlights under the cursor and content that changes after it was
// captured (lazy images) must not stop tracking.
func TestScrollCaptureToleratesHoverAndChangedHistory(t *testing.T) {
	const width, height = 320, 260
	page := textPage(width, 3000, 11)
	frame := func(top int) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(img, img.Bounds(), page, image.Pt(0, top), draw.Src)
		// The row under a stationary cursor gets a highlight.
		for y := 120; y < 142; y++ {
			for x := 0; x < width; x++ {
				c := img.RGBAAt(x, y)
				c.B = max(c.B, 255-uint8(top%7))
				c.R -= c.R / 8
				img.SetRGBA(x, y, c)
			}
		}
		return img
	}
	capture := newScrollCapture(frame(0), image.Rect(0, 0, width, height))
	furthest := 0
	for i, top := range []int{35, 80, 150, 60, 20, 170, 240, 310, 270, 380} {
		if i == 3 {
			// An image finishes loading in content that was already captured.
			fill(page, image.Rect(40, 30, 280, 140), color.RGBA{180, 120, 60, 255})
		}
		furthest = max(furthest, top)
		if _, err := capture.append(frame(top)); err != nil {
			t.Fatal(err)
		}
		if capture.top != top || capture.long.Bounds().Dy() != height+furthest {
			t.Fatalf("scrolled to %d: tracked=%d height=%d", top, capture.top, capture.long.Bounds().Dy())
		}
	}
}

func TestScrollCapturePublishedImagesStayImmutable(t *testing.T) {
	page := textPage(200, 2000, 5)
	frame := func(top int) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, 200, 240))
		draw.Draw(img, img.Bounds(), page, image.Pt(0, top), draw.Src)
		return img
	}
	capture := newScrollCapture(frame(0), image.Rect(0, 0, 200, 240))
	var published []*image.RGBA
	for top := 60; top <= 900; top += 60 {
		if _, err := capture.append(frame(top)); err != nil {
			t.Fatal(err)
		}
		published = append(published, capture.long)
	}
	for _, img := range published {
		samePixels(t, img, img.Bounds(), page, image.Point{})
	}
}

func BenchmarkScrollCaptureAppend(b *testing.B) {
	const width, height = 900, 1300
	page := textPage(width, 40000, 9)
	frame := func(top int) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(img, img.Bounds(), page, image.Pt(0, top), draw.Src)
		return img
	}
	frames := make([]*image.RGBA, 64)
	for i := range frames {
		frames[i] = frame(i * 45)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%len(frames) == 0 {
			b.StopTimer()
			capture := newScrollCapture(frames[0], image.Rect(0, 400, width, 900))
			b.StartTimer()
			for _, f := range frames[1:] {
				_, _ = capture.append(f)
			}
		}
	}
}
