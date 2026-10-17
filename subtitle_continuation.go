package main

import (
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	subtitleShortPageUnits   = 12.0
	subtitleContinuationWait = 900 * time.Millisecond
)

func subtitleRevealDuration(source, translation string) time.Duration {
	units := math.Max(subtitleReadingUnits(source), subtitleReadingUnits(translation))
	return max(400*time.Millisecond, time.Duration(units/subtitleReadingRate*float64(time.Second)))
}

func subtitlePageReadingUnits(page subtitlePage) float64 {
	return math.Max(subtitleReadingUnits(page.Source), subtitleReadingUnits(page.Translation))
}

func subtitleCanContinue(entry *subtitleEntry, page subtitlePage) bool {
	if !entry.frame.Final || entry.frame.Incomplete || entry.sourceOffset != 0 || entry.translationOffset != 0 {
		return false
	}
	if page.SourceBytes != len(entry.frame.Source) || page.TranslationBytes != len(entry.frame.Translation) {
		return false
	}
	_, _, sequenceKey := subtitleSequenceKey(entry.lastFrameKey)
	return sequenceKey && subtitlePageReadingUnits(page) <= subtitleShortPageUnits
}

// Keys are created by speechCaptions, never by the server. Only consecutive
// sequences from the same connection may continue a caption; reconnects and
// missing-source changes start a new page rather than invent a bilingual pair.
func subtitleSequenceKey(key string) (string, int, bool) {
	const separator = ":sequence:"
	index := strings.LastIndex(key, separator)
	if index <= 0 {
		return "", 0, false
	}
	sequence, err := strconv.Atoi(key[index+len(separator):])
	return key[:index], sequence, err == nil && sequence > 0
}

func (p *subtitlePlayback) waitForContinuation() bool {
	return !p.stopped && p.continuable && len(p.queue) == 1 && subtitlePageReadingUnits(p.page) <= subtitleShortPageUnits
}

func (p *subtitlePlayback) extendShortPage(paginate func(string, string) (subtitlePage, error)) bool {
	if !p.continuable || len(p.queue) < 2 {
		return false
	}
	entry, next := p.queue[0], p.queue[1]
	if !next.frame.Final || next.frame.Incomplete || (entry.frame.Source == "") != (next.frame.Source == "") {
		return false
	}
	leftSession, leftSequence, leftValid := subtitleSequenceKey(entry.lastFrameKey)
	rightSession, rightSequence, rightValid := subtitleSequenceKey(next.frame.Key)
	if !leftValid || !rightValid || leftSession != rightSession || rightSequence != leftSequence+1 {
		return false
	}
	source := joinSpeechText(p.pageFrame.Source, next.frame.Source)
	translation := joinSpeechText(p.pageFrame.Translation, next.frame.Translation)
	if math.Max(subtitleReadingUnits(source), subtitleReadingUnits(translation)) > subtitlePageUnits {
		return false
	}
	page, err := paginate(source, translation)
	if err != nil || page.SourceBytes != len(source) || page.TranslationBytes != len(translation) {
		p.continuable = false // Avoid repeating native layout work each tick.
		return false          // Leave both fragments intact for ordinary pagination.
	}

	// Retain the page key and the fully visible prefix. Only the appended suffix
	// gets a fresh reveal clock, so neither renderer fades/retypes the old text.
	p.revealedSource, p.revealedTranslation = p.pageFrame.Source, p.pageFrame.Translation
	p.pageDuration = subtitleRevealDuration(strings.TrimPrefix(source, p.revealedSource), strings.TrimPrefix(translation, p.revealedTranslation))
	p.page, p.pageFrame.Source, p.pageFrame.Translation = page, source, translation
	p.elapsed = 0
	oldBytes := len(entry.frame.Source) + len(entry.frame.Translation) + len(next.frame.Source) + len(next.frame.Translation)
	entry.frame.Source, entry.frame.Translation, entry.lastFrameKey = source, translation, next.lastFrameKey
	entry.units = subtitlePageReadingUnits(page)
	p.queuedBytes += len(source) + len(translation) - oldBytes
	copy(p.queue[1:], p.queue[2:])
	p.queue[len(p.queue)-1] = nil
	p.queue = p.queue[:len(p.queue)-1]
	return true
}
