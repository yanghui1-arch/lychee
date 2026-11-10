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
	stitch := newScrollStitcher(first)
	added, err := stitch.append(next)
	if err != nil {
		t.Fatal(err)
	}
	merged := stitch.long
	if !added || merged.Bounds().Dy() != 580 {
		t.Fatalf("expected 580px stitched screenshot, got %v, added=%t", merged.Bounds(), added)
	}
	for y := 0; y < 580; y++ {
		if merged.RGBAAt(100, y) != page.RGBAAt(100, y) {
			t.Fatalf("seam or repeated pixels at row %d", y)
		}
	}
	added, err = stitch.append(next)
	if err != nil {
		t.Fatal(err)
	}
	unchanged := stitch.long
	if added || unchanged != merged {
		t.Fatal("bottom of page should not append duplicate content")
	}
}

func TestScrollingScreenshotTracksGentleContinuousScroll(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 160, 700))
	for y := 0; y < 700; y++ {
		for x := 0; x < 160; x++ {
			page.SetRGBA(x, y, color.RGBA{uint8((x*13 + y*37) % 251), uint8((x*7 + y*19) % 253), uint8((x*11 + y*31) % 249), 255})
		}
	}
	frame := func(top int) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, 160, 200))
		draw.Draw(img, img.Bounds(), page, image.Pt(0, top), draw.Src)
		return img
	}
	stitch := newScrollStitcher(frame(0))
	for _, top := range []int{3, 7, 12, 80, 180, 320} {
		next := frame(top)
		added, err := stitch.append(next)
		if err != nil {
			t.Fatal(err)
		}
		merged := stitch.long
		if !added || merged.Bounds().Dy() != 200+top {
			t.Fatalf("scroll to %d: bounds=%v, added=%t", top, merged.Bounds(), added)
		}
	}
	for y := 0; y < stitch.long.Bounds().Dy(); y++ {
		for x := 0; x < 160; x++ {
			if stitch.long.RGBAAt(x, y) != page.RGBAAt(x, y) {
				t.Fatalf("incorrect pixel at %d,%d", x, y)
			}
		}
	}
}

func TestScrollingScreenshotDoesNotAppendRevisitedContent(t *testing.T) {
	// Similar-looking text lines make a wrong-direction overlap look plausible.
	// Each line still has a distinct pattern, so its real position is identifiable.
	page := image.NewRGBA(image.Rect(0, 0, 240, 1600))
	for y := 0; y < 1600; y++ {
		for x := 0; x < 240; x++ {
			pixel := color.RGBA{250, 250, 250, 255}
			hash := uint32(y/20+1)*2654435761 ^ uint32(x/3+1)*2246822519
			if y%20 >= 6 && y%20 <= 10 && x >= 24 && x < 180 && hash%3 == 0 {
				pixel = color.RGBA{35, 35, 35, 255}
			}
			page.SetRGBA(x, y, pixel)
		}
	}
	const start, height = 400, 240
	frame := func(top int) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, 240, height))
		draw.Draw(img, img.Bounds(), page, image.Pt(0, start+top), draw.Src)
		return img
	}
	for _, tc := range []struct {
		name      string
		positions []int
	}{
		{"down then revisit then extend", []int{100, 200, 300, 200, 100, 0, 100, 200, 300, 350, 220, 120, 220, 320, 410}},
		{"jump back across several viewports", []int{100, 200, 300, 400, 500, 0, 100, 200, 500, 600}},
		{"up first then revisit then extend", []int{-100, -200, -100, 0, 100, 0, -100, 0, 100, 200}},
		{"turn around between captured positions", []int{73, 151, 247, 182, 91, 28, 101, 199, 274, 274, 171, 274, 340}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stitch := newScrollStitcher(frame(0))
			furthest := 0
			for _, top := range tc.positions {
				old := stitch.long
				added, err := stitch.append(frame(top))
				if err != nil {
					t.Fatal(err)
				}
				wantAdded := top > furthest
				furthest = max(furthest, top)
				if stitch.top != top || added != wantAdded || stitch.long.Bounds().Dy() != height+furthest {
					t.Fatalf("viewport=%d: tracked=%d added=%t want=%t height=%d want=%d", top, stitch.top, added, wantAdded, stitch.long.Bounds().Dy(), height+furthest)
				}
				if !wantAdded && stitch.long != old {
					t.Fatal("revisit changed the captured image")
				}
				for y := 0; y < stitch.long.Bounds().Dy(); y++ {
					for x := 0; x < 240; x++ {
						if stitch.long.RGBAAt(x, y) != page.RGBAAt(x, start+y) {
							t.Fatalf("duplicated or missing content at %d,%d after viewport=%d", x, y, top)
						}
					}
				}
			}
		})
	}
}

func TestScrollingScreenshotSparseTextWithNoiseAndRejectedFrames(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 600, 2200))
	draw.Draw(page, page.Bounds(), image.NewUniform(color.RGBA{250, 250, 250, 255}), image.Point{}, draw.Src)
	for line := 0; line < 110; line++ {
		for glyph := 0; glyph < 12; glyph++ {
			hash := uint32(line+1)*2654435761 ^ uint32(glyph+1)*2246822519
			for y := 0; y < 7; y++ {
				for x := 0; x < 4; x++ {
					if (hash>>uint((x+y*4)%31))&1 != 0 {
						page.SetRGBA(13+glyph*6+x, line*20+6+y, color.RGBA{36, 36, 36, 255})
					}
				}
			}
		}
	}
	const height = 300
	frame := func(top, noise int) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, 600, height))
		draw.Draw(img, img.Bounds(), page, image.Pt(0, 400+top), draw.Src)
		if noise != 0 {
			for y := 0; y < height; y++ {
				for x := 0; x < 600; x++ {
					c := img.RGBAAt(x, y)
					c.R += uint8(noise)
					c.G += uint8(noise)
					c.B += uint8(noise)
					img.SetRGBA(x, y, c)
				}
			}
		}
		return img
	}
	stitch := newScrollStitcher(frame(0, 0))
	furthest := 0
	for i, top := range []int{11, 90, 170, 80, 0, 170, 240, 120, 240, 310} {
		added, err := stitch.append(frame(top, i%2))
		if err != nil {
			t.Fatal(err)
		}
		want := top > furthest
		furthest = max(furthest, top)
		if added != want || stitch.top != top || stitch.long.Bounds().Dy() != height+furthest {
			t.Fatalf("top=%d tracked=%d added=%t want=%t height=%d", top, stitch.top, added, want, stitch.long.Bounds().Dy())
		}
		// An unrelated animation frame must not move the anchor or grow the image.
		unrelated := image.NewRGBA(image.Rect(0, 0, 600, height))
		draw.Draw(unrelated, unrelated.Bounds(), image.NewUniform(color.RGBA{80, 30, 150, 255}), image.Point{}, draw.Src)
		before := stitch.long
		if added, err := stitch.append(unrelated); added || err != nil || stitch.long != before || stitch.top != top {
			t.Fatal("unrelated frame corrupted the capture")
		}
	}
	for y := 0; y < stitch.long.Bounds().Dy(); y++ {
		for x := 0; x < 600; x++ {
			a, b := stitch.long.RGBAAt(x, y), page.RGBAAt(x, 400+y)
			if absPixel(a.R, b.R) > 1 || absPixel(a.G, b.G) > 1 || absPixel(a.B, b.B) > 1 {
				t.Fatalf("missing or duplicated content at %d,%d", x, y)
			}
		}
	}
}

func TestScrollingScreenshotRejectsAmbiguousRepeatedContent(t *testing.T) {
	page := image.NewRGBA(image.Rect(0, 0, 160, 800))
	for y := 0; y < 800; y++ {
		for x := 0; x < 160; x++ {
			page.SetRGBA(x, y, color.RGBA{uint8((x*7 + (y%40)*13) % 251), uint8((x*3 + (y%40)*7) % 253), uint8((x + (y%40)*17) % 249), 255})
		}
	}
	frame := func(top int) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, 160, 200))
		draw.Draw(img, img.Bounds(), page, image.Pt(0, top), draw.Src)
		return img
	}
	stitch := newScrollStitcher(frame(0))
	for _, top := range []int{40, 80, 20, 0, 60, 100} {
		before := stitch.long
		added, err := stitch.append(frame(top))
		if added || err != nil || stitch.long != before {
			t.Fatalf("ambiguous repeated pattern was appended at %d", top)
		}
	}
}
