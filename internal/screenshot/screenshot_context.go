package screenshot

import (
	"errors"
	"image"
	"image/draw"
)

// The tracking viewport can be much taller than the selected output. This keeps
// overlap when a wheel step skips over a short selection. Only the user's crop
// from the first viewport through the furthest observed crop is exported.
type scrollCapture struct {
	tracker *scrollStitcher
	area    image.Rectangle
	end     int
	long    *image.RGBA
}

func newScrollCapture(context *image.RGBA, area image.Rectangle) *scrollCapture {
	s := &scrollCapture{tracker: newScrollStitcher(context), area: area, end: area.Max.Y}
	s.tracker.limit = 30000 + context.Bounds().Dy()
	s.render()
	return s
}

func (s *scrollCapture) render() {
	s.long = image.NewRGBA(image.Rect(0, 0, s.area.Dx(), s.end-s.area.Min.Y))
	draw.Draw(s.long, s.long.Bounds(), s.tracker.long, s.area.Min, draw.Src)
}

func (s *scrollCapture) append(frame *image.RGBA) (bool, error) {
	if _, err := s.tracker.append(frame); err != nil {
		return false, err
	}
	end := s.tracker.top + s.area.Max.Y
	if end <= s.end {
		return false, nil
	}
	if end-s.area.Min.Y > 30000 {
		return false, errors.New("长截图已达到 30000 像素上限，请完成滚动")
	}
	s.end = end
	s.render()
	return true, nil
}
