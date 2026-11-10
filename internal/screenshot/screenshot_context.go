package screenshot

import (
	"bytes"
	"errors"
	"image"
)

var errScrollLimit = errors.New("长截图已达到 30000 像素上限，请完成滚动")

// scrollCapture builds a long screenshot of the selected area while the user
// scrolls the application underneath it.
//
// Every frame is a capture of the tracking context: the selection's columns,
// optionally extended above and below inside the scrolled window. Output pixels
// only ever come from the selection itself (or, when one step skips past it,
// from the previous frame's context between the two positions). The context is
// used to find the scroll offset, never as content, so fixed bars outside the
// selection cannot leak into the result.
//
// Each frame is aligned against the previous accepted frame, which tolerates
// content that changes over time. When that fails (a jump back, or a step
// larger than the context), the selection is located in the captured history.
type scrollCapture struct {
	area   image.Rectangle // selection within each context frame
	width  int
	height int
	stride int

	// Reference: the last accepted frame and the content offset of its selection.
	reference      *image.RGBA
	referenceRows  []scrollRow
	referenceIndex rowIndex
	top            int
	delta          int
	lastPix        []uint8

	// Captured content of the selection's moving rows, in content coordinates.
	content      []uint8
	end          int
	history      []scrollRow
	historyIndex rowIndex
	footer       int
	footerPix    []uint8

	long *image.RGBA
}

func newScrollCapture(context *image.RGBA, area image.Rectangle) *scrollCapture {
	s := &scrollCapture{area: area, width: area.Dx(), height: area.Dy(), stride: area.Dx() * 4}
	context = atOrigin(context)
	rows := s.frameRows(context)
	band := rows[area.Min.Y:area.Max.Y]
	s.content = make([]uint8, 0, s.stride*s.height*4)
	for y := 0; y < s.height; y++ {
		s.content = append(s.content, s.bandRow(context, y)...)
	}
	s.end = s.height
	s.history = append([]scrollRow(nil), band...)
	s.historyIndex = newRowIndex(s.history)
	s.setReference(context, rows, 0)
	s.render()
	return s
}

// atOrigin views a capture with its bounds at (0, 0); area is frame-relative.
func atOrigin(frame *image.RGBA) *image.RGBA {
	if frame.Rect.Min == (image.Point{}) {
		return frame
	}
	return &image.RGBA{Pix: frame.Pix, Stride: frame.Stride, Rect: image.Rectangle{Max: frame.Rect.Size()}}
}

func (s *scrollCapture) frameRows(frame *image.RGBA) []scrollRow {
	return signatures(frame.Pix[s.area.Min.X*4:], frame.Stride, s.width, frame.Rect.Dy())
}

// bandRow returns row y of the selection in a context frame.
func (s *scrollCapture) bandRow(frame *image.RGBA, y int) []uint8 {
	offset := frame.PixOffset(s.area.Min.X, s.area.Min.Y+y)
	return frame.Pix[offset : offset+s.stride]
}

func (s *scrollCapture) setReference(frame *image.RGBA, rows []scrollRow, top int) {
	s.reference, s.referenceRows, s.top = frame, rows, top
	s.referenceIndex = newRowIndex(rows)
	s.lastPix = frame.Pix
}

func (s *scrollCapture) minimumRows() int { return max(4, min(12, s.height/10)) }

func (s *scrollCapture) append(frame *image.RGBA) (bool, error) {
	frame = atOrigin(frame)
	if frame.Rect.Size() != s.reference.Rect.Size() || bytes.Equal(frame.Pix, s.lastPix) {
		return false, nil
	}
	rows := s.frameRows(frame)
	band := rows[s.area.Min.Y:s.area.Max.Y]
	moving := band[:s.height-s.footer]
	minimum := s.minimumRows()
	previous, previousRows, previousTop := s.reference, s.referenceRows, s.top

	// Selection row r lies on previous context row r+offset; offset = Min.Y+delta.
	top, fromPrevious := 0, false
	// Context rows that did not change on screen are fixed UI (toolbars, input
	// boxes): they carry no evidence about the scroll offset.
	static := make([]bool, len(rows))
	for y := range rows {
		if y >= s.area.Min.Y && y < s.area.Max.Y {
			continue
		}
		diff, informative := rowDiff(&rows[y], &previousRows[y])
		static[y] = !informative || diff <= rowTolerance
	}
	if a, ok := locate(moving, previousRows, static, s.referenceIndex, []int{s.area.Min.Y, s.area.Min.Y + s.delta}, minimum); ok {
		top, fromPrevious = previousTop+a.offset-s.area.Min.Y, true
	} else if a, ok := locate(moving, s.history, nil, s.historyIndex, []int{previousTop, previousTop + s.delta}, minimum); ok {
		top = a.offset
	} else {
		s.lastPix = frame.Pix // unrelated frame: keep the last good reference
		return false, nil
	}
	delta := top - previousTop
	s.setReference(frame, rows, top)
	if delta == 0 {
		return false, nil
	}
	s.delta = delta

	changed := false
	if fromPrevious {
		prevBand := previousRows[s.area.Min.Y:s.area.Max.Y]
		if footer := staticFooter(band, prevBand, delta, s.height/3); footer > s.footer {
			s.setFooter(frame, footer)
			changed = true
		}
	}
	movingHeight := s.height - s.footer
	if top+movingHeight > s.end {
		if top+movingHeight > maxScrollHeight-s.footer {
			return changed, errScrollLimit
		}
		// A step longer than the selection: the skipped rows were visible in the
		// previous context below its selection.
		for y := s.end; y < top; y++ {
			row := s.area.Min.Y + y - previousTop
			if row < 0 || row >= previous.Rect.Dy() {
				return changed, nil
			}
			s.appendRow(previous, row, previousRows[row])
		}
		for y := max(s.end, top); y < top+movingHeight; y++ {
			s.appendRow(frame, s.area.Min.Y+y-top, rows[s.area.Min.Y+y-top])
		}
		changed = true
	}
	if changed {
		s.render()
	}
	return changed, nil
}

func (s *scrollCapture) appendRow(frame *image.RGBA, contextRow int, signature scrollRow) {
	offset := frame.PixOffset(s.area.Min.X, contextRow)
	s.content = append(s.content, frame.Pix[offset:offset+s.stride]...)
	// Copy the samples so history does not retain each frame's signature slab.
	signature.pixels = append([]uint8(nil), signature.pixels...)
	s.history = append(s.history, signature)
	s.historyIndex.add(s.history[s.end:], s.end)
	s.end++
}

// setFooter keeps a sticky bar at the bottom of the selection once, at the end
// of the image, and removes copies of it already appended as content.
func (s *scrollCapture) setFooter(frame *image.RGBA, footer int) {
	// Every frame showed the bar, so the newest rows appended are copies of it.
	trim := max(0, min(footer-s.footer, s.end-1))
	// Published images share the content buffer, so shrink into a new one.
	s.content = append(make([]uint8, 0, cap(s.content)), s.content[:(s.end-trim)*s.stride]...)
	s.end -= trim
	s.history = s.history[:s.end]
	s.historyIndex = newRowIndex(s.history)
	s.footer = footer
	s.footerPix = make([]uint8, 0, footer*s.stride)
	for y := s.height - footer; y < s.height; y++ {
		s.footerPix = append(s.footerPix, s.bandRow(frame, y)...)
	}
}

// render publishes the long image. Content rows are only ever appended, so the
// image can share the buffer: later appends never touch its visible rows.
func (s *scrollCapture) render() {
	if s.footer == 0 {
		s.long = &image.RGBA{Pix: s.content[: s.end*s.stride : s.end*s.stride], Stride: s.stride, Rect: image.Rect(0, 0, s.width, s.end)}
		return
	}
	pix := make([]uint8, 0, (s.end+s.footer)*s.stride)
	pix = append(append(pix, s.content[:s.end*s.stride]...), s.footerPix...)
	s.long = &image.RGBA{Pix: pix, Stride: s.stride, Rect: image.Rect(0, 0, s.width, s.end+s.footer)}
}
