package screenshot

import (
	"errors"
	"image"
	"math"
)

// Coordinates are capture-relative physical pixels, matching OCR and PNG export.
func pinnedBounds(display image.Rectangle, x, y, width, height int) (image.Rectangle, error) {
	if display.Empty() || x < 0 || y < 0 || x >= display.Dx() || y >= display.Dy() || width < 1 || height < 1 || width > 32768 || height > 32768 {
		return image.Rectangle{}, errors.New("置顶截图位置或大小无效")
	}
	scale := math.Min(1, math.Min(float64(display.Dx())/float64(width), float64(display.Dy())/float64(height)))
	w, h := max(1, int(float64(width)*scale)), max(1, int(float64(height)*scale))
	const offset = 12
	left := min(display.Min.X+x+offset, display.Max.X-w)
	top := min(display.Min.Y+y+offset, display.Max.Y-h)
	return image.Rect(left, top, left+w, top+h), nil
}
