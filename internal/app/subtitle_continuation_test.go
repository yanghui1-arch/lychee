package app

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

func advanceCaption(t *testing.T, player *subtitlePlayback, now *time.Time, duration time.Duration, paginate func(string, string) (subtitlePage, error)) SubtitleFrame {
	t.Helper()
	var frame SubtitleFrame
	for duration > 0 {
		step := min(50*time.Millisecond, duration)
		*now = now.Add(step)
		var err error
		frame, err = player.Tick(*now, paginate)
		if err != nil {
			t.Fatal(err)
		}
		duration -= step
	}
	return frame
}

func nextCaptionPage(t *testing.T, player *subtitlePlayback, now *time.Time, paginate func(string, string) (subtitlePage, error)) SubtitleFrame {
	t.Helper()
	number := player.pageNumber
	for i := 0; i < 200; i++ {
		frame := advanceCaption(t, player, now, 50*time.Millisecond, paginate)
		if player.pageNumber != number {
			return frame
		}
	}
	t.Fatal("caption did not advance")
	return SubtitleFrame{}
}

func TestCaptionAppendsAcrossSpeechSessionsWithoutRetyping(t *testing.T) {
	var player subtitlePlayback
	first := SubtitleFrame{Key: "session:sequence:1", Source: "The first sentence.", Translation: "第一句话需要足够长的阅读时间。", Final: true}
	second := SubtitleFrame{Key: "other:sequence:8", Source: "Here comes more.", Translation: "下一句接在后面。", Final: true}
	_ = player.Accept(first)
	now := time.Unix(0, 0)
	initial, _ := player.Tick(now, readingTestPaginator)
	frame := advanceCaption(t, &player, &now, player.pageDuration+subtitleFinalHold-200*time.Millisecond, readingTestPaginator)
	if frame.Key != initial.Key || frame.Source != first.Source || frame.Translation != first.Translation {
		t.Fatalf("caption disappeared before a late continuation: %+v", frame)
	}
	_ = player.Accept(second)
	frame = nextCaptionPage(t, &player, &now, readingTestPaginator)
	if frame.Key != initial.Key || !strings.HasPrefix(frame.Source, first.Source) || !strings.HasPrefix(frame.Translation, first.Translation) {
		t.Fatalf("next session erased/retyped the previous sentence: %+v", frame)
	}
	_ = player.Accept(second)
	frame = advanceCaption(t, &player, &now, player.pageDuration, readingTestPaginator)
	if frame.Source != joinSpeechText(first.Source, second.Source) || frame.Translation != joinSpeechText(first.Translation, second.Translation) || len(player.queue) != 1 {
		t.Fatalf("lost or duplicated a continuation: %+v", frame)
	}
}

func TestCaptionFinalTextHasTwoSecondHold(t *testing.T) {
	var player subtitlePlayback
	_ = player.Accept(SubtitleFrame{Key: "one", Translation: "好", Final: true})
	now := time.Unix(0, 0)
	_, _ = player.Tick(now, readingTestPaginator)
	frame := advanceCaption(t, &player, &now, player.pageDuration+2*time.Second-50*time.Millisecond, readingTestPaginator)
	if frame.Translation != "好" {
		t.Fatal("last caption disappeared before its two-second hold")
	}
	frame = advanceCaption(t, &player, &now, 50*time.Millisecond, readingTestPaginator)
	if frame.Key != "" || player.HasContent() || player.queuedBytes != 0 {
		t.Fatal("last caption never drained")
	}
}

func TestCaptionFinalHoldDoesNotDelayNewArrival(t *testing.T) {
	var player subtitlePlayback
	_ = player.Accept(SubtitleFrame{Key: "first", Translation: "前文", Final: true})
	now := time.Unix(0, 0)
	first, _ := player.Tick(now, readingTestPaginator)
	advanceCaption(t, &player, &now, time.Second, readingTestPaginator)
	_ = player.Accept(SubtitleFrame{Key: "next", Translation: "后续", Final: true})
	frame := advanceCaption(t, &player, &now, subtitleFrameInterval, readingTestPaginator)
	if player.pageNumber != 2 || frame.Key != first.Key || frame.Translation != "前文后" {
		t.Fatalf("new arrival was blocked by the final hold: %+v", frame)
	}
}

func TestShortCaptionBurstsDoNotAccumulateArtificialPauses(t *testing.T) {
	var player subtitlePlayback
	const text = "一二三四五六七八"
	for _, char := range text {
		_ = player.Accept(SubtitleFrame{Key: string(char), Translation: string(char), Final: true})
	}
	now := time.Unix(0, 0)
	_, _ = player.Tick(now, readingTestPaginator)
	var frame SubtitleFrame
	for i := 0; i < 50; i++ {
		frame = advanceCaption(t, &player, &now, subtitleFrameInterval, readingTestPaginator)
	}
	if frame.Translation != text || len(player.queue) != 1 {
		t.Fatalf("short captions accumulated extra delay: %+v", frame)
	}
}

func TestSingleLineOverflowHoldsThenReplacesWholeLine(t *testing.T) {
	var player subtitlePlayback
	_ = player.Accept(SubtitleFrame{Key: "first", Translation: "一二三四五六七八", Final: true})
	_ = player.Accept(SubtitleFrame{Key: "next", Translation: "九十后续", Final: true})
	paginate := func(source, translation string) (subtitlePage, error) {
		return splitSubtitlePage(source, translation, func(_ int, text string) bool { return uniseg.GraphemeClusterCount(text) <= 8 })
	}
	now := time.Unix(0, 0)
	first, _ := player.Tick(now, paginate)
	frame := advanceCaption(t, &player, &now, player.pageDuration+subtitleLineHold-50*time.Millisecond, paginate)
	if frame.Key != first.Key || frame.Translation != "一二三四五六七八" || player.queue[0].translationOffset != 0 {
		t.Fatalf("line changed before its reading hold: %+v", frame)
	}
	frame = advanceCaption(t, &player, &now, 50*time.Millisecond, paginate)
	if frame.Key == first.Key || frame.Translation != "九" || player.revealedTranslation != "" {
		t.Fatalf("overflow retained old text instead of replacing the line: %+v", frame)
	}
	frame = advanceCaption(t, &player, &now, player.pageDuration, paginate)
	if frame.Translation != "九十后续" || len(player.queue) != 1 {
		t.Fatal("line replacement lost or repeated new content")
	}
}

func TestSingleLineNormalizesServerLineBreaks(t *testing.T) {
	var player subtitlePlayback
	_ = player.Accept(SubtitleFrame{Key: "breaks", Source: "First\r\nsecond\tthird", Translation: "第一段\n第二段\u2028第三段\x00", Final: true})
	entry := player.queue[0]
	if entry.frame.Source != "First  second third" || entry.frame.Translation != "第一段 第二段 第三段" {
		t.Fatalf("server line breaks escaped single-line normalization: %+v", entry.frame)
	}
}

func TestSingleLineReflowAndSilenceCancelPreparedReplacement(t *testing.T) {
	service := newSubtitleService()
	service.window = &recordingSubtitle{}
	_ = service.accept(SubtitleFrame{Key: "long", Translation: strings.Repeat("字", 100), Final: true})
	now := time.Unix(0, 0)
	service.presentAt(now)
	for i := 0; i < 200 && service.playback.next == nil; i++ {
		now = now.Add(50 * time.Millisecond)
		service.presentAt(now)
	}
	if service.playback.next == nil {
		t.Fatal("test did not prepare an overflowing replacement")
	}
	service.playback.Reflow()
	if service.playback.next != nil || service.playback.queue[0].translationOffset != 0 {
		t.Fatal("reflow retained stale layout or consumed unread text")
	}
	service.presentAt(now)
	for i := 0; i < 200 && service.playback.next == nil; i++ {
		now = now.Add(50 * time.Millisecond)
		service.presentAt(now)
	}
	if service.playback.next == nil {
		t.Fatal("test did not reprepare the replacement")
	}
	pageNumber := service.playback.pageNumber
	service.pause()
	if service.playback.next != nil {
		t.Fatal("silence retained a prepared replacement")
	}
	for i := 0; i < 100; i++ {
		now = now.Add(50 * time.Millisecond)
		service.presentAt(now)
	}
	if service.playback.HasContent() || service.playback.pageNumber != pageNumber {
		t.Fatal("silence played the stale replacement or failed to clear")
	}
}

func TestCaptionSeparatesMissingSourceAndIncompleteText(t *testing.T) {
	for _, next := range []SubtitleFrame{
		{Key: "next", Translation: "缺失原文", Final: true},
		{Key: "next", Source: "Unfinished", Translation: "尚未完成", Incomplete: true},
	} {
		var player subtitlePlayback
		_ = player.Accept(SubtitleFrame{Key: "first", Source: "First", Translation: "前文", Final: true})
		_ = player.Accept(next)
		now := time.Unix(0, 0)
		initial, _ := player.Tick(now, readingTestPaginator)
		frame := nextCaptionPage(t, &player, &now, readingTestPaginator)
		if frame.Key == initial.Key || player.pageFrame.Source != next.Source || player.pageFrame.Translation != next.Translation {
			t.Fatalf("incompatible captions combined: %+v", frame)
		}
	}
}

func TestLineAppendHonorsBothLanguageWidthsAndGraphemes(t *testing.T) {
	paginate := func(source, translation string) (subtitlePage, error) {
		return splitSubtitlePage(source, translation, func(_ int, text string) bool { return uniseg.GraphemeClusterCount(text) <= 8 })
	}
	previous := SubtitleFrame{Source: "👨‍👩‍👧‍👦é!", Translation: "前文"}
	next := subtitlePage{Source: "new", Translation: "后续", SourceBytes: 3, TranslationBytes: 6}
	combined, fits, err := subtitleAppendLine(previous, next, paginate, [2]bool{})
	if err != nil || !fits || combined.Source != "👨‍👩‍👧‍👦é! new" || combined.Translation != "前文后续" {
		t.Fatalf("fitting line did not append intact: %+v %v", combined, err)
	}
	if !utf8.ValidString(combined.Source) || uniseg.GraphemeClusterCount(combined.Source) != 7 {
		t.Fatal("append damaged a grapheme")
	}
	previous.Translation = "一二三四五六七八"
	_, fits, err = subtitleAppendLine(previous, next, paginate, [2]bool{})
	if err != nil || fits {
		t.Fatal("translation overflow was ignored because the source fits")
	}
	previous.Source, previous.Translation = "abcdefgh", "短"
	_, fits, err = subtitleAppendLine(previous, next, paginate, [2]bool{})
	if err != nil || fits {
		t.Fatal("source overflow was ignored because the translation fits")
	}
	previous = SubtitleFrame{Source: "hel", Translation: "测"}
	next = subtitlePage{Source: "lo", Translation: "试", SourceBytes: 2, TranslationBytes: 3}
	combined, fits, err = subtitleAppendLine(previous, next, paginate, [2]bool{subtitleContiguousText("hello", 3), false})
	if err != nil || !fits || combined.Source != "hello" || combined.Translation != "测试" {
		t.Fatalf("a fragment boundary inserted text inside a word: %+v", combined)
	}
}

func TestCaptionReflowPreservesUnreadSuffix(t *testing.T) {
	var player subtitlePlayback
	_ = player.Accept(SubtitleFrame{Key: "one", Translation: "前文", Final: true})
	_ = player.Accept(SubtitleFrame{Key: "two", Translation: "不能漏掉未读文字。", Final: true})
	now := time.Unix(0, 0)
	_, _ = player.Tick(now, readingTestPaginator)
	nextCaptionPage(t, &player, &now, readingTestPaginator)
	offset, page := player.queue[0].translationOffset, player.page.Translation
	if player.revealedTranslation == "" {
		t.Fatal("test did not retain context")
	}
	player.Reflow()
	frame := advanceCaption(t, &player, &now, 50*time.Millisecond, readingTestPaginator)
	if player.queue[0].translationOffset != offset || player.page.Translation != page || frame.Translation != string([]rune(page)[0]) {
		t.Fatal("reflow skipped unread text or replayed retained context")
	}
}

func TestCaptionDoesNotDuplicateExhaustedLanguage(t *testing.T) {
	var player subtitlePlayback
	_ = player.Accept(SubtitleFrame{Key: "uneven", Source: "X", Translation: strings.Repeat("译文比原文长很多。", 8), Final: true})
	now := time.Unix(0, 0)
	_, _ = player.Tick(now, readingTestPaginator)
	for i := 0; i < 2000 && player.HasContent(); i++ {
		frame := advanceCaption(t, &player, &now, 50*time.Millisecond, readingTestPaginator)
		if frame.Key != "" && frame.Source != "X" {
			t.Fatalf("short source was retyped or duplicated: %+v", frame)
		}
	}
	if player.HasContent() {
		t.Fatal("uneven bilingual text never drained")
	}
}
