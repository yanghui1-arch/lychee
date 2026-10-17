package main

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
	// Page length is bounded separately; quicker reveal never enlarges a page.
	subtitleReadingRate = 12.0
	subtitlePageUnits   = 28.0
	subtitlePageHold    = 800 * time.Millisecond
	subtitleQueueLimit  = 4 * 1024 * 1024
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
	lastFrameKey                      string
	sourceOffset, translationOffset   int
	sourceContext, translationContext string
	units                             float64
}

// The playback queue retains complete translation fragments. Consecutive short
// fragments may share a page; only completed pages advance the byte cursors.
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
	continuable           bool
	stopped               bool
	lastTick              time.Time
	queuedBytes           int
	bufferWarned          bool
}

func cleanSubtitleText(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, "\x00", ""))
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
	p.queue = append(p.queue, &subtitleEntry{frame: frame, lastFrameKey: frame.Key, units: math.Max(subtitleReadingUnits(frame.Source), subtitleReadingUnits(frame.Translation))})
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
	p.revealedSource, p.revealedTranslation = "", ""
	p.elapsed, p.lastTick = 0, time.Time{}
}

func (p *subtitlePlayback) PendingSeconds() int {
	units := 0.0
	for i, entry := range p.queue {
		if i == 0 && (p.revealedSource != "" || p.revealedTranslation != "") {
			sourceUnits := subtitleReadingUnits(entry.frame.Source[entry.sourceOffset:]) - subtitleReadingUnits(p.revealedSource)
			translationUnits := subtitleReadingUnits(entry.frame.Translation[entry.translationOffset:]) - subtitleReadingUnits(p.revealedTranslation)
			units += math.Max(sourceUnits, translationUnits)
		} else {
			units += entry.units
		}
	}
	return max(0, int(math.Ceil(units/subtitleReadingRate))-int(p.elapsed/time.Second))
}

func (p *subtitlePlayback) Tick(now time.Time, paginate func(string, string) (subtitlePage, error)) (SubtitleFrame, error) {
	if p.pageFrame.Key != "" {
		delta := now.Sub(p.lastTick)
		// A stalled UI or sleeping PC must not consume unseen reading time.
		p.elapsed += max(0, min(delta, 100*time.Millisecond))
		p.lastTick = now
		if p.elapsed >= p.pageDuration {
			p.extendShortPage(paginate)
		}
		hold := subtitlePageHold
		if p.waitForContinuation() {
			hold += subtitleContinuationWait
		}
		if p.elapsed >= p.pageDuration+hold {
			entry := p.queue[0]
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
		page, err := paginate(entry.frame.Source[entry.sourceOffset:], entry.frame.Translation[entry.translationOffset:])
		if err != nil {
			return SubtitleFrame{}, err
		}
		if page.SourceBytes == 0 && page.TranslationBytes == 0 {
			return SubtitleFrame{}, errors.New("字幕分页没有取得进展")
		}
		// Without word-level alignment, keep the shorter side as context on
		// remaining pages, rather than fabricate a new source/translation pair.
		if page.Source == "" && entry.sourceContext != "" {
			contextPage, err := paginate(entry.sourceContext, "")
			if err != nil {
				return SubtitleFrame{}, err
			}
			page.Source = contextPage.Source
		}
		if page.Translation == "" && entry.translationContext != "" {
			contextPage, err := paginate("", entry.translationContext)
			if err != nil {
				return SubtitleFrame{}, err
			}
			page.Translation = contextPage.Translation
		}
		p.page = page
		p.revealedSource, p.revealedTranslation = "", ""
		p.continuable = subtitleCanContinue(entry, page)
		p.pageNumber++
		p.pageFrame = SubtitleFrame{Key: fmt.Sprintf("%s:page:%d", entry.frame.Key, p.pageNumber), Source: page.Source, Translation: page.Translation, Final: entry.frame.Final, Incomplete: entry.frame.Incomplete}
		p.pageDuration = subtitleRevealDuration(page.Source, page.Translation)
		p.elapsed, p.lastTick = 0, now
	}
	frame := p.pageFrame
	progress := math.Min(1, float64(p.elapsed)/float64(p.pageDuration))
	frame.Source = p.revealedSource + subtitleReadingPrefix(strings.TrimPrefix(frame.Source, p.revealedSource), progress)
	frame.Translation = p.revealedTranslation + subtitleReadingPrefix(strings.TrimPrefix(frame.Translation, p.revealedTranslation), progress)
	frame.PendingSeconds = p.PendingSeconds()
	return frame, nil
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
		low, high := 1, 1
		for high < len(glyphs[i]) && glyphs[i][high].units <= subtitlePageUnits {
			high++
		}
		if !fits(i, text[:glyphs[i][0].end]) {
			return subtitlePage{}, errors.New("当前字幕区域无法容纳一个文字，请减小字号或调整显示器")
		}
		for low < high {
			middle := (low + high + 1) / 2
			if glyphs[i][middle-1].units <= subtitlePageUnits && fits(i, text[:glyphs[i][middle-1].end]) {
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
