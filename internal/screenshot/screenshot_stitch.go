package screenshot

import (
	"errors"
	"image"
	"image/draw"
	"math"
	"sort"
)

// Every viewport is located against the immutable captured content, rather than
// integrating frame-to-frame offsets (which accumulate errors after reversals).
type scrollStitcher struct {
	long, previous *image.RGBA
	top            int
	limit          int
	rows           []scrollRow
	index          map[uint64][]int
}

type scrollRow struct {
	pixels     []uint8
	textured   bool
	background [3]uint8
}

func newScrollStitcher(first *image.RGBA) *scrollStitcher {
	s := &scrollStitcher{long: first, previous: first, index: make(map[uint64][]int), limit: 30000}
	s.addRows(scrollRows(first))
	return s
}

func scrollRows(img *image.RGBA) []scrollRow {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	columns := min(96, w-8)
	rows := make([]scrollRow, h)
	for y := range rows {
		row := scrollRow{pixels: make([]uint8, columns*3)}
		var histogram [3][256]int
		for x := 0; x < columns; x++ {
			// Sample bin centres, avoiding the scrollbar and the selection edges.
			px := 4 + (2*x+1)*(w-8)/(2*columns)
			c := img.RGBAAt(px, y)
			row.pixels[x*3], row.pixels[x*3+1], row.pixels[x*3+2] = c.R, c.G, c.B
			histogram[0][c.R]++
			histogram[1][c.G]++
			histogram[2][c.B]++
		}
		for channel := 0; channel < 3; channel++ {
			best := 0
			for value, count := range histogram[channel] {
				if count > best {
					best = count
					row.background[channel] = uint8(value)
				}
			}
		}
		for x := 0; x < columns; x++ {
			for channel := 0; channel < 3; channel++ {
				if absPixel(row.pixels[x*3+channel], row.background[channel]) > 16 {
					row.textured = true
				}
			}
		}
		rows[y] = row
	}
	return rows
}

func absPixel(a, b uint8) int {
	d := int(a) - int(b)
	if d < 0 {
		return -d
	}
	return d
}

func rowKey(row scrollRow, projection int) uint64 {
	hash := uint64(1469598103934665603)
	columns := len(row.pixels) / 3
	for i := projection; i < columns; i += 4 {
		for channel := 0; channel < 3; channel++ {
			hash ^= uint64(row.pixels[i*3+channel] >> 4)
			hash *= 1099511628211
		}
	}
	return hash ^ uint64(projection)
}

func (s *scrollStitcher) addRows(rows []scrollRow) {
	for _, row := range rows {
		y := len(s.rows)
		s.rows = append(s.rows, row)
		if !row.textured {
			continue
		}
		for projection := 0; projection < 4; projection++ {
			key := rowKey(row, projection)
			s.index[key] = append(s.index[key], y)
		}
	}
}

// Background pixels do not dilute the score of mismatched text. Small rendering
// noise is tolerated, and a few changed rows (caret/cursor) may be discarded.
func viewportScore(known, next []scrollRow, top int) (float64, bool) {
	start, end := max(0, -top), min(len(next), len(known)-top)
	scores := make([]float64, 0, end-start)
	first, last := -1, -1
	for y := start; y < end; y++ {
		a, b := known[y+top], next[y]
		var difference float64
		count := 0
		for x := 0; x < len(b.pixels); x += 3 {
			informative := false
			for c := 0; c < 3; c++ {
				if absPixel(a.pixels[x+c], a.background[c]) > 12 || absPixel(b.pixels[x+c], b.background[c]) > 12 || absPixel(a.background[c], b.background[c]) > 12 {
					informative = true
				}
			}
			if informative {
				for c := 0; c < 3; c++ {
					difference += float64(absPixel(a.pixels[x+c], b.pixels[x+c]))
					count++
				}
			}
		}
		if count > 0 {
			scores = append(scores, difference/float64(count))
			if first < 0 {
				first = y
			}
			last = y
		}
	}
	if len(scores) < 4 || last-first < (end-start)/3 {
		return 255, false
	}
	sort.Float64s(scores)
	keep := len(scores) - len(scores)/20 // at most 5% transient rows
	var total float64
	for _, v := range scores[:keep] {
		total += v
	}
	score := total / float64(keep)
	return score, score <= 6
}

func (s *scrollStitcher) append(next *image.RGBA) (bool, error) {
	if next.Bounds().Size() != s.previous.Bounds().Size() {
		return false, nil
	}
	rows := scrollRows(next)
	h := len(rows)
	minimum := max(24, min(64, h/8))
	type candidate struct {
		top, votes, lastRow int
		score               float64
	}
	votes := map[int]*candidate{}
	for y, row := range rows {
		if !row.textured {
			continue
		}
		for projection := 0; projection < 4; projection++ {
			hits := s.index[rowKey(row, projection)]
			// Repeated/blank patterns cannot establish a unique position.
			if len(hits) > 128 {
				continue
			}
			for _, knownY := range hits {
				top := knownY - y
				if min(len(s.rows), top+h)-max(0, top) < minimum {
					continue
				}
				c := votes[top]
				if c == nil {
					c = &candidate{top: top, lastRow: -1}
					votes[top] = c
				}
				if c.lastRow != y {
					c.votes++
					c.lastRow = y
				}
			}
		}
	}
	candidates := make([]*candidate, 0, len(votes))
	for _, c := range votes {
		if c.votes >= 4 {
			candidates = append(candidates, c)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].votes != candidates[j].votes {
			return candidates[i].votes > candidates[j].votes
		}
		return math.Abs(float64(candidates[i].top-s.top)) < math.Abs(float64(candidates[j].top-s.top))
	})
	var best *candidate
	ambiguous := false
	for _, c := range candidates[:min(64, len(candidates))] {
		score, ok := viewportScore(s.rows, rows, c.top)
		if !ok {
			continue
		}
		c.score = score
		if best == nil || score < best.score-0.5 {
			best = c
			ambiguous = false
		} else if math.Abs(score-best.score) <= 0.5 && c.top != best.top {
			ambiguous = true
		}
	}
	// Never grow the image on an ambiguous alignment. A later, distinguishable
	// viewport can recover directly against history without corrupting the result.
	if best == nil || ambiguous {
		return false, nil
	}
	top := best.top
	bottom := top + h
	if bottom > s.limit {
		return false, errors.New("长截图已达到 30000 像素上限，请完成滚动")
	}
	s.top, s.previous = top, next
	if bottom <= s.long.Bounds().Dy() {
		return false, nil
	}
	oldHeight := s.long.Bounds().Dy()
	result := image.NewRGBA(image.Rect(0, 0, next.Bounds().Dx(), bottom))
	draw.Draw(result, s.long.Bounds(), s.long, image.Point{}, draw.Src)
	draw.Draw(result, image.Rect(0, oldHeight, result.Bounds().Dx(), bottom), next, image.Pt(0, oldHeight-top), draw.Src)
	s.long = result
	s.addRows(rows[oldHeight-top:])
	return true, nil
}
