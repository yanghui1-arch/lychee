package screenshot

import (
	"sort"
)

// Row signatures and alignment primitives for scrolling screenshots.
//
// Each captured row is reduced to a fixed number of sampled pixels. Two rows
// match when their informative pixels (those that differ from the row's
// dominant background) agree within a small tolerance. Alignment counts
// matching and mismatching rows instead of averaging, so a few changed rows
// (hover highlights, carets, sticky bars, lazy images) do not hide the true
// scroll offset.

const (
	sampleColumns   = 96
	rowTolerance    = 12
	maxScrollHeight = 30000
	maxIndexHits    = 128
	maxCandidates   = 48
)

type scrollRow struct {
	pixels     []uint8
	textured   bool
	background [3]uint8
}

func absPixel(a, b uint8) int {
	d := int(a) - int(b)
	if d < 0 {
		return -d
	}
	return d
}

// signatures samples every row of an RGBA pixel buffer.
func signatures(pix []uint8, stride, width, height int) []scrollRow {
	columns := max(1, min(sampleColumns, width-8))
	inset := min(4, (width-columns)/2)
	span := width - 2*inset
	rows := make([]scrollRow, height)
	samples := make([]uint8, height*columns*3)
	for y := range rows {
		row := scrollRow{pixels: samples[y*columns*3 : (y+1)*columns*3 : (y+1)*columns*3]}
		var histogram [3][256]uint16
		line := pix[y*stride:]
		for x := 0; x < columns; x++ {
			// Sample bin centres, avoiding scrollbars and the selection edges.
			px := inset + (2*x+1)*span/(2*columns)
			r, g, b := line[px*4], line[px*4+1], line[px*4+2]
			row.pixels[x*3], row.pixels[x*3+1], row.pixels[x*3+2] = r, g, b
			histogram[0][r]++
			histogram[1][g]++
			histogram[2][b]++
		}
		for channel := 0; channel < 3; channel++ {
			best := uint16(0)
			for value, count := range histogram[channel] {
				if count > best {
					best = count
					row.background[channel] = uint8(value)
				}
			}
		}
		for i, v := range row.pixels {
			if absPixel(v, row.background[i%3]) > 16 {
				row.textured = true
				break
			}
		}
		rows[y] = row
	}
	return rows
}

// rowDiff returns the mean channel difference over informative pixels, and
// false when both rows are the same plain background (no evidence either way).
func rowDiff(a, b *scrollRow) (int, bool) {
	backgroundsDiffer := false
	for c := 0; c < 3; c++ {
		if absPixel(a.background[c], b.background[c]) > rowTolerance {
			backgroundsDiffer = true
		}
	}
	total, count := 0, 0
	for x := 0; x < len(a.pixels); x += 3 {
		informative := backgroundsDiffer
		for c := 0; c < 3 && !informative; c++ {
			informative = absPixel(a.pixels[x+c], a.background[c]) > rowTolerance || absPixel(b.pixels[x+c], b.background[c]) > rowTolerance
		}
		if informative {
			for c := 0; c < 3; c++ {
				total += absPixel(a.pixels[x+c], b.pixels[x+c])
			}
			count += 3
		}
	}
	if count == 0 {
		return 0, false
	}
	return total / count, true
}

func rowKey(row *scrollRow, projection int) uint64 {
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

// rowIndex maps quantized row hashes to row numbers for candidate generation.
type rowIndex map[uint64][]int

func newRowIndex(rows []scrollRow) rowIndex {
	index := make(rowIndex, len(rows)*2)
	index.add(rows, 0)
	return index
}

func (index rowIndex) add(rows []scrollRow, base int) {
	for i := range rows {
		if !rows[i].textured {
			continue
		}
		for projection := 0; projection < 4; projection++ {
			key := rowKey(&rows[i], projection)
			index[key] = append(index[key], base+i)
		}
	}
}

// alignment describes template row r lying on reference row r+offset.
type alignment struct {
	offset, matched, mismatched, overlap int
}

func (a alignment) quality() int { return a.matched - 3*a.mismatched }

// compareRows aligns template row r with reference row r+offset. Reference rows
// marked in skip (fixed UI around the selection) are left out entirely.
func compareRows(template, reference []scrollRow, skip []bool, offset int) alignment {
	start, end := max(0, -offset), min(len(template), len(reference)-offset)
	a := alignment{offset: offset}
	first, last := -1, -1
	for r := start; r < end; r++ {
		if skip != nil && skip[r+offset] {
			continue
		}
		a.overlap++
		diff, informative := rowDiff(&template[r], &reference[r+offset])
		if !informative {
			continue
		}
		if diff <= rowTolerance {
			a.matched++
			if first < 0 {
				first = r
			}
			last = r
		} else {
			a.mismatched++
		}
		// Stop once two thirds agreement is no longer reachable.
		if remaining := end - r - 1; a.mismatched*2 > a.matched+remaining {
			a.matched = 0
			return a
		}
	}
	// Matches must span the overlap, not cluster on one repeated line.
	if a.matched > 0 && last-first+1 < min(24, a.overlap/4) {
		a.matched = 0
	}
	return a
}

func acceptable(a alignment, minimum int) bool {
	return a.matched >= minimum && a.matched >= 2*a.mismatched
}

// locate finds the unique offset of template within reference. Candidates come
// from row-hash votes plus the predicted offsets; ambiguous results are refused.
func locate(template, reference []scrollRow, skip []bool, index rowIndex, predicted []int, minimum int) (alignment, bool) {
	votes := map[int]int{}
	seen := map[int]bool{}
	for r := range template {
		if !template[r].textured {
			continue
		}
		clear(seen)
		for projection := 0; projection < 4; projection++ {
			hits := index[rowKey(&template[r], projection)]
			if len(hits) > maxIndexHits {
				continue
			}
			for _, y := range hits {
				offset := y - r
				if !seen[offset] {
					seen[offset] = true
					votes[offset]++
				}
			}
		}
	}
	type candidate struct{ offset, votes int }
	candidates := make([]candidate, 0, len(votes))
	for offset, count := range votes {
		if count >= 3 {
			candidates = append(candidates, candidate{offset, count})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].votes != candidates[j].votes {
			return candidates[i].votes > candidates[j].votes
		}
		return candidates[i].offset < candidates[j].offset
	})
	offsets := make([]int, 0, maxCandidates+len(predicted))
	tried := map[int]bool{}
	for _, offset := range predicted {
		if !tried[offset] {
			tried[offset] = true
			offsets = append(offsets, offset)
		}
	}
	for _, c := range candidates[:min(maxCandidates, len(candidates))] {
		if !tried[c.offset] {
			tried[c.offset] = true
			offsets = append(offsets, c.offset)
		}
	}
	var best alignment
	found := false
	var accepted []alignment
	for _, offset := range offsets {
		a := compareRows(template, reference, skip, offset)
		if !acceptable(a, minimum) {
			continue
		}
		accepted = append(accepted, a)
		if !found || a.quality() > best.quality() {
			best, found = a, true
		}
	}
	if !found {
		return alignment{}, false
	}
	margin := max(2, best.matched/16)
	for _, a := range accepted {
		if a.offset != best.offset && a.quality() >= best.quality()-margin {
			return alignment{}, false
		}
	}
	return best, true
}

// staticFooter counts bottom rows that stayed fixed on screen while the content
// above them moved by delta, such as a sticky bar inside the selection.
func staticFooter(current, previous []scrollRow, delta, limit int) int {
	footer, run := 0, 0
	moving := -1
	for r := len(current) - 1; r >= max(0, len(current)-limit); r-- {
		diff, informative := rowDiff(&current[r], &previous[r])
		if informative && diff > rowTolerance {
			moving = r
			break
		}
		run++
		if !informative {
			continue
		}
		if p := r + delta; p >= 0 && p < len(previous) {
			if moved, ok := rowDiff(&current[r], &previous[p]); !ok || moved <= rowTolerance {
				continue // also consistent with the motion; not evidence
			}
		}
		footer = run
	}
	// Plain rows of the bar carry no detail; include those whose color differs
	// from the scrolling content above.
	if footer > 0 && moving >= 0 {
		for r := len(current) - footer - 1; r > moving; r-- {
			if !backgroundDiffers(&current[r], &current[moving]) {
				break
			}
			footer++
		}
	}
	return footer
}

func backgroundDiffers(a, b *scrollRow) bool {
	for c := 0; c < 3; c++ {
		if absPixel(a.background[c], b.background[c]) > rowTolerance {
			return true
		}
	}
	return false
}
