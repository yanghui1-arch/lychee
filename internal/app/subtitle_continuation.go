package app

import (
	"math"
	"strings"
	"time"
	"unicode"
)

func subtitleRevealDuration(source, translation string) time.Duration {
	units := math.Max(subtitleReadingUnits(source), subtitleReadingUnits(translation))
	return max(subtitleFrameInterval, time.Duration(units/subtitleReadingRate*float64(time.Second)))
}

func subtitleContiguousText(text string, offset int) bool {
	return offset > 0 && offset < len(text) && len(strings.TrimRightFunc(text[:offset], unicode.IsSpace)) == offset && len(strings.TrimLeftFunc(text[offset:], unicode.IsSpace)) == len(text)-offset
}

func joinSubtitleContext(left, right string, contiguous bool) string {
	if contiguous {
		return left + right
	}
	return joinSpeechText(left, right)
}

// Append only when the complete existing line and the next fragment fit in both
// languages. Never trim a visible prefix to make room: overflow starts a new line.
func subtitleAppendLine(previous SubtitleFrame, next subtitlePage, paginate func(string, string) (subtitlePage, error), contiguous [2]bool) (SubtitleFrame, bool, error) {
	combined := SubtitleFrame{
		Source:      joinSubtitleContext(previous.Source, next.Source, contiguous[0]),
		Translation: joinSubtitleContext(previous.Translation, next.Translation, contiguous[1]),
	}
	// An exhausted side is context, not new text to append a second time.
	if next.SourceBytes == 0 {
		combined.Source = previous.Source
	}
	if next.TranslationBytes == 0 {
		combined.Translation = previous.Translation
	}
	page, err := paginate(combined.Source, combined.Translation)
	if err != nil {
		return SubtitleFrame{}, false, err
	}
	fits := page.SourceBytes == len(combined.Source) && page.TranslationBytes == len(combined.Translation)
	return combined, fits, nil
}
