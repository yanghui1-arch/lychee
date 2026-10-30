package app

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/rivo/uniseg"
)

const (
	// One CJK grapheme counts as one reading unit, a Latin letter as 0.4.
	// This gives approximately 12 CJK characters / 30 Latin letters per second.
	// Native font measurements determine a single line's capacity.
	subtitleReadingRate   = 12.0
	subtitleFrameInterval = 40 * time.Millisecond
	subtitleLineHold      = 600 * time.Millisecond
	subtitleFinalHold     = 2 * time.Second
	subtitleQueueLimit    = 4 * 1024 * 1024
)

type subtitlePage struct {
	Source, Translation           string
	SourceBytes, TranslationBytes int
}

type subtitlePageResult struct {
	page subtitlePage
	err  error
}

type subtitleEntry struct {
	frame                             SubtitleFrame
	sourceOffset, translationOffset   int
	sourceContext, translationContext string
	units                             float64
}

type subtitleTransition struct {
	page                                subtitlePage
	frame                               SubtitleFrame
	revealedSource, revealedTranslation string
}

// The queue owns unread text. Settled fragments may share a single visible line;
// when the next fragment cannot fit, replace the whole line after its hold.
type subtitlePlayback struct {
	queue                 []*subtitleEntry
	pending               map[string]SubtitleFrame
	pendingOrder          []string
	seen                  map[string]bool
	seenOrder             []string
	page                  subtitlePage
	pageNumber            uint64
	pageFrame             SubtitleFrame
	pageDuration, elapsed time.Duration
	revealedSource        string
	revealedTranslation   string
	lastTick              time.Time
	queuedBytes           int
	bufferWarned          bool
	next                  *subtitleTransition
}

func cleanSubtitleText(text string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if r == 0 {
			return -1
		}
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, text))
}

func (p *subtitlePlayback) Accept(frame SubtitleFrame) error {
	frame.Source, frame.Translation = cleanSubtitleText(frame.Source), cleanSubtitleText(frame.Translation)
	if frame.Translation == "" || p.seen[frame.Key] {
		return nil
	}
	if p.pending == nil {
		p.pending = map[string]SubtitleFrame{}
	}
	if !frame.Final && !frame.Incomplete {
		if _, exists := p.pending[frame.Key]; !exists {
			p.pendingOrder = append(p.pendingOrder, frame.Key)
		}
		p.pending[frame.Key] = frame
		return nil
	}
	delete(p.pending, frame.Key)
	if len(p.pending) == 0 {
		p.pendingOrder = nil
	}
	if p.seen == nil {
		p.seen = map[string]bool{}
	}
	p.seen[frame.Key] = true
	p.seenOrder = append(p.seenOrder, frame.Key)
	if len(p.seenOrder) > 4096 {
		delete(p.seen, p.seenOrder[0])
		p.seenOrder = p.seenOrder[1:]
	}
	p.queue = append(p.queue, &subtitleEntry{frame: frame, units: math.Max(subtitleReadingUnits(frame.Source), subtitleReadingUnits(frame.Translation))})
	p.queuedBytes += len(frame.Source) + len(frame.Translation)
	if p.queuedBytes > subtitleQueueLimit && !p.bufferWarned {
		p.bufferWarned = true
		// Report overflow explicitly; the audio owner ends this generation.
		// Never accelerate playback or silently skip into the latest text.
		return errors.New("待阅读字幕过多，已停止监听，请稍后重新开启字幕")
	}
	return nil
}

func (p *subtitlePlayback) MarkIncomplete() {
	order := append([]string(nil), p.pendingOrder...)
	for _, key := range order {
		if frame, ok := p.pending[key]; ok {
			frame.Incomplete = true
			_ = p.Accept(frame)
		}
	}
}

func (p *subtitlePlayback) HasContent() bool { return len(p.queue) > 0 || p.pageFrame.Key != "" }

func (p *subtitlePlayback) Reflow() {
	// A font/layout change replays only the current unread page. Completed
	// pages retain their cursors and are neither replayed nor skipped.
	p.pageFrame = SubtitleFrame{}
	p.next = nil
	p.revealedSource, p.revealedTranslation = "", ""
	p.elapsed, p.lastTick = 0, time.Time{}
}

func (p *subtitlePlayback) PendingSeconds() int {
	units := 0.0
	for _, entry := range p.queue {
		units += entry.units
	}
	if p.pageFrame.Key != "" {
		progress := math.Min(1, float64(p.elapsed)/float64(p.pageDuration))
		units -= math.Max(subtitleReadingUnits(p.page.Source), subtitleReadingUnits(p.page.Translation)) * progress
	}
	return max(0, int(math.Ceil(units/subtitleReadingRate)))
}

func (p *subtitlePlayback) Tick(now time.Time, paginate func(string, string) (subtitlePage, error)) (SubtitleFrame, error) {
	if p.pageFrame.Key != "" {
		// Present the complete suffix for at least one frame before advancing.
		// Same-line appends have no pause; only whole-line replacement waits.
		revealed := p.elapsed >= p.pageDuration
		delta := now.Sub(p.lastTick)
		// A stalled UI or sleeping PC must not consume unseen reading time.
		p.elapsed += max(0, min(delta, 100*time.Millisecond))
		p.lastTick = now
		var hold time.Duration
		entry := p.queue[0]
		if len(p.queue) == 1 && entry.sourceOffset+p.page.SourceBytes == len(entry.frame.Source) && entry.translationOffset+p.page.TranslationBytes == len(entry.frame.Translation) {
			hold = subtitleFinalHold
		} else if revealed {
			if p.next == nil {
				following := *entry
				following.sourceOffset += p.page.SourceBytes
				following.translationOffset += p.page.TranslationBytes
				following.sourceContext, following.translationContext = p.page.Source, p.page.Translation
				if following.sourceOffset == len(following.frame.Source) && following.translationOffset == len(following.frame.Translation) {
					following = *p.queue[1]
				}
				var err error
				p.next, err = p.preparePage(&following, p.pageFrame, paginate)
				if err != nil {
					return SubtitleFrame{}, err
				}
			}
			// A new display key replaces the whole line instead of appending.
			if p.next.frame.Key != p.pageFrame.Key {
				hold = subtitleLineHold
			}
		}
		if revealed && p.elapsed >= p.pageDuration+hold {
			entry.sourceOffset += p.page.SourceBytes
			entry.translationOffset += p.page.TranslationBytes
			entry.sourceContext, entry.translationContext = p.page.Source, p.page.Translation
			entry.units = math.Max(subtitleReadingUnits(entry.frame.Source[entry.sourceOffset:]), subtitleReadingUnits(entry.frame.Translation[entry.translationOffset:]))
			if entry.sourceOffset == len(entry.frame.Source) && entry.translationOffset == len(entry.frame.Translation) {
				p.queuedBytes -= len(entry.frame.Source) + len(entry.frame.Translation)
				p.queue[0] = nil
				p.queue = p.queue[1:]
			}
			p.pageFrame = SubtitleFrame{}
		}
	}
	if len(p.queue) == 0 {
		return SubtitleFrame{}, nil
	}
	if p.pageFrame.Key == "" {
		entry := p.queue[0]
		next := p.next
		if next == nil {
			var err error
			next, err = p.preparePage(entry, SubtitleFrame{}, paginate)
			if err != nil {
				return SubtitleFrame{}, err
			}
		}
		p.next = nil
		p.page, p.pageFrame = next.page, next.frame
		p.revealedSource, p.revealedTranslation = next.revealedSource, next.revealedTranslation
		p.pageNumber++
		p.pageDuration = subtitleRevealDuration(next.page.Source, next.page.Translation)
		p.elapsed, p.lastTick = 0, now
	}
	frame := p.pageFrame
	progress := math.Min(1, float64(p.elapsed)/float64(p.pageDuration))
	frame.Source = p.revealedSource + subtitleReadingPrefix(strings.TrimPrefix(frame.Source, p.revealedSource), progress)
	frame.Translation = p.revealedTranslation + subtitleReadingPrefix(strings.TrimPrefix(frame.Translation, p.revealedTranslation), progress)
	frame.PendingSeconds = p.PendingSeconds()
	return frame, nil
}

func (p *subtitlePlayback) preparePage(entry *subtitleEntry, previous SubtitleFrame, paginate func(string, string) (subtitlePage, error)) (*subtitleTransition, error) {
	page, err := paginate(entry.frame.Source[entry.sourceOffset:], entry.frame.Translation[entry.translationOffset:])
	if err != nil {
		return nil, err
	}
	if page.SourceBytes == 0 && page.TranslationBytes == 0 {
		return nil, errors.New("字幕分页没有取得进展")
	}
	// Without word-level alignment, keep the shorter side as context on
	// remaining pages, rather than fabricate a new source/translation pair.
	if page.Source == "" && entry.sourceContext != "" {
		contextPage, err := paginate(entry.sourceContext, "")
		if err != nil {
			return nil, err
		}
		page.Source = contextPage.Source
	}
	if page.Translation == "" && entry.translationContext != "" {
		contextPage, err := paginate("", entry.translationContext)
		if err != nil {
			return nil, err
		}
		page.Translation = contextPage.Translation
	}
	next := SubtitleFrame{Key: fmt.Sprintf("%s:page:%d", entry.frame.Key, p.pageNumber+1), Source: page.Source, Translation: page.Translation, Final: entry.frame.Final, Incomplete: entry.frame.Incomplete}
	var revealedSource, revealedTranslation string
	if previous.Key != "" && previous.Final && !previous.Incomplete && entry.frame.Final && !entry.frame.Incomplete && (previous.Source == "") == (page.Source == "") {
		contiguous := [2]bool{subtitleContiguousText(entry.frame.Source, entry.sourceOffset), subtitleContiguousText(entry.frame.Translation, entry.translationOffset)}
		combined, fits, err := subtitleAppendLine(previous, page, paginate, contiguous)
		if err != nil {
			return nil, err
		}
		if fits {
			next.Key = previous.Key
			next.Source, next.Translation = combined.Source, combined.Translation
			revealedSource = strings.TrimSuffix(next.Source, page.Source)
			revealedTranslation = strings.TrimSuffix(next.Translation, page.Translation)
		}
	}
	// An exhausted language is already read context, not a new suffix.
	if page.SourceBytes == 0 {
		revealedSource = next.Source
	}
	if page.TranslationBytes == 0 {
		revealedTranslation = next.Translation
	}
	return &subtitleTransition{page: page, frame: next, revealedSource: revealedSource, revealedTranslation: revealedTranslation}, nil
}

type subtitleGlyph struct {
	end      int
	units    float64
	boundary bool
}

func subtitleGlyphs(text string) ([]subtitleGlyph, float64) {
	var glyphs []subtitleGlyph
	total := 0.0
	graphemes := uniseg.NewGraphemes(text)
	for graphemes.Next() {
		_, end := graphemes.Positions()
		runes := graphemes.Runes()
		r := runes[0]
		units := 0.4
		if unicode.IsSpace(r) {
			units = 0.15
		} else if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) || r > 0xffff {
			units = 1
		} else if unicode.IsPunct(r) {
			units = 0.5
		}
		total += units
		glyphs = append(glyphs, subtitleGlyph{end: end, units: total, boundary: unicode.IsSpace(r) || unicode.IsPunct(r)})
	}
	return glyphs, total
}

func subtitleReadingUnits(text string) float64 { _, units := subtitleGlyphs(text); return units }

func subtitleReadingPrefix(text string, progress float64) string {
	if text == "" || progress >= 1 {
		return text
	}
	glyphs, total := subtitleGlyphs(text)
	end := glyphs[0].end // Both available languages appear in the first frame.
	for _, glyph := range glyphs {
		if glyph.units > total*progress {
			break
		}
		end = glyph.end
	}
	return text[:end]
}

// Split from the beginning, consuming every byte exactly once. The platform
// supplies real font measurements; natural boundaries are preferred where fit.
func splitSubtitlePage(source, translation string, fits func(int, string) bool) (subtitlePage, error) {
	texts := [2]string{source, translation}
	var glyphs [2][]subtitleGlyph
	var totals [2]float64
	var capacities [2]int
	pages := 1.0
	for i, text := range texts {
		glyphs[i], totals[i] = subtitleGlyphs(text)
		if len(glyphs[i]) == 0 {
			continue
		}
		low, high := 1, len(glyphs[i])
		if !fits(i, text[:glyphs[i][0].end]) {
			return subtitlePage{}, errors.New("当前字幕区域无法容纳一个文字，请减小字号或调整显示器")
		}
		for low < high {
			middle := (low + high + 1) / 2
			if fits(i, text[:glyphs[i][middle-1].end]) {
				low = middle
			} else {
				high = middle - 1
			}
		}
		capacities[i] = low
		pages = math.Max(pages, math.Ceil(totals[i]/glyphs[i][low-1].units))
	}
	var cuts [2]int
	for i := range texts {
		if len(glyphs[i]) == 0 {
			continue
		}
		count := 1
		budget := totals[i] / pages
		for count < capacities[i] && glyphs[i][count].units <= budget {
			count++
		}
		// Keep complete words/clauses where possible, including the boundary
		// byte in the cursor so whitespace is not duplicated between pages.
		if count < len(glyphs[i]) {
			for j := count - 1; j >= count/2; j-- {
				if glyphs[i][j].boundary {
					count = j + 1
					break
				}
			}
		}
		cuts[i] = glyphs[i][count-1].end
	}
	return subtitlePage{Source: strings.TrimSpace(source[:cuts[0]]), Translation: strings.TrimSpace(translation[:cuts[1]]), SourceBytes: cuts[0], TranslationBytes: cuts[1]}, nil
}
