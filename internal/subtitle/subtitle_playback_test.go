package subtitle

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

func readingTestPaginator(source, translation string) (SubtitlePage, error) {
	const testLineUnits = 56.0
	return splitSubtitlePage(source, translation, func(_ int, text string) bool { return subtitleReadingUnits(text) <= testLineUnits })
}

func withoutSpaces(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, text)
}

func TestReadingQueueRetainsEveryPageDuringBursts(t *testing.T) {
	inputs := []SubtitleFrame{
		{Key: "first", Source: strings.Repeat("This is the first original sentence. ", 9), Translation: strings.Repeat("第一段的开头、中间和结尾都必须显示。", 12), Final: true},
		{Key: "second", Source: "Second original.", Translation: strings.Repeat("第二段也不能抢先覆盖前一页。", 7), Final: true},
		{Key: "third", Translation: strings.Repeat("没有原文也要完整读完。", 8), Final: true},
	}
	var player subtitlePlayback
	for _, input := range inputs {
		if err := player.Accept(input); err != nil {
			t.Fatal(err)
		}
	}
	seenPages := map[uint64]bool{}
	sources, translations := map[string]string{}, map[string]string{}
	var order []string
	seenEntries := map[string]bool{}
	now := time.Unix(0, 0)
	for tick := 0; tick < 20000; tick++ {
		frame, err := player.Tick(now, readingTestPaginator)
		if err != nil {
			t.Fatal(err)
		}
		if frame.Key != "" && frame.Source == player.pageFrame.Source && frame.Translation == player.pageFrame.Translation && !seenPages[player.pageNumber] {
			seenPages[player.pageNumber] = true
			entry := player.queue[0]
			id := entry.frame.Key
			if !seenEntries[id] {
				seenEntries[id] = true
				order = append(order, id)
			}
			if player.page.SourceBytes > 0 {
				sources[id] += player.page.Source
			}
			if player.page.TranslationBytes > 0 {
				translations[id] += player.page.Translation
			}
		}
		if !player.HasContent() {
			break
		}
		now = now.Add(50 * time.Millisecond)
	}
	if player.HasContent() {
		t.Fatal("reading queue never drained")
	}
	if strings.Join(order, ",") != "first,second,third" {
		t.Fatalf("later subtitle overtook an unread one: %v", order)
	}
	for _, input := range inputs {
		if withoutSpaces(sources[input.Key]) != withoutSpaces(input.Source) || withoutSpaces(translations[input.Key]) != withoutSpaces(input.Translation) {
			t.Fatalf("unread text disappeared from %s: source=%q translation=%q", input.Key, sources[input.Key], translations[input.Key])
		}
	}
	if len(seenPages) <= len(inputs) {
		t.Fatal("large replies were not paginated")
	}
}

func TestReadingRateDoesNotAccelerateWithBacklog(t *testing.T) {
	var player subtitlePlayback
	_ = player.Accept(SubtitleFrame{Key: "one", Source: strings.Repeat("文", 40), Translation: strings.Repeat("译", 40), Final: true})
	now := time.Unix(0, 0)
	frame, err := player.Tick(now, readingTestPaginator)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Source != "文" || frame.Translation != "译" {
		t.Fatalf("first frame dumped the complete page: %+v", frame)
	}
	key := frame.Key
	for i := 0; i < 20; i++ {
		_ = player.Accept(SubtitleFrame{Key: string(rune('a' + i)), Translation: strings.Repeat("后", 50), Final: true})
	}
	for i := 0; i < 20; i++ {
		now = now.Add(50 * time.Millisecond)
		frame, err = player.Tick(now, readingTestPaginator)
		if err != nil {
			t.Fatal(err)
		}
	}
	if frame.Key != key || len([]rune(frame.Translation)) < 11 || len([]rune(frame.Translation)) > 13 {
		t.Fatalf("a burst accelerated/overwrote the reading page: %+v", frame)
	}
	// A long background pause must not skip whole unseen pages on resumption.
	frame, err = player.Tick(now.Add(time.Hour), readingTestPaginator)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Key != key || len([]rune(frame.Translation)) > 14 {
		t.Fatal("sleep/blocked UI consumed unread captions")
	}
}

func TestReadingQueueCoalescesPreviewsAndDeduplicatesFinals(t *testing.T) {
	var player subtitlePlayback
	_ = player.Accept(SubtitleFrame{Key: "one", Source: "Hello.", Translation: "你"})
	_ = player.Accept(SubtitleFrame{Key: "one", Source: "Hello.", Translation: "你好"})
	frame, err := player.Tick(time.Unix(0, 0), readingTestPaginator)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Key != "" {
		t.Fatal("mutable previews were played before confirmation")
	}
	final := SubtitleFrame{Key: "one", Source: "Hello.", Translation: "你好。", Final: true}
	_ = player.Accept(final)
	_ = player.Accept(final)
	if len(player.queue) != 1 || len(player.pending) != 0 {
		t.Fatal("preview/final snapshots duplicated the queued fragment")
	}
	_ = player.Accept(SubtitleFrame{Key: "two", Source: "Unconfirmed source.", Translation: "尚未确认"})
	player.MarkIncomplete()
	if len(player.queue) != 2 || !player.queue[1].frame.Incomplete || player.queue[1].frame.Final {
		t.Fatal("failed preview was lost or presented as confirmed")
	}
}

func TestCaptionFragmentsAppendOnNextFrameAfterFullReveal(t *testing.T) {
	var player subtitlePlayback
	_ = player.Accept(SubtitleFrame{Key: "one", Translation: "你好", Final: true})
	_ = player.Accept(SubtitleFrame{Key: "two", Translation: "下文", Final: true})
	now := time.Unix(0, 0)
	first, _ := player.Tick(now, readingTestPaginator)
	frame := advanceCaption(t, &player, &now, player.pageDuration, readingTestPaginator)
	if frame.Translation != "你好" || player.pageNumber != 1 {
		t.Fatal("advanced before presenting the complete current fragment")
	}
	frame = advanceCaption(t, &player, &now, subtitleFrameInterval, readingTestPaginator)
	if player.pageNumber != 2 || frame.Key != first.Key || frame.Translation != "你好下" {
		t.Fatalf("queued text did not append on the next frame: %+v", frame)
	}
}

func TestDiscardAndRestartRejectOldCaptionGeneration(t *testing.T) {
	service := newSubtitleService()
	window := &recordingSubtitle{}
	service.window = window
	_ = service.setVisible(true)
	oldGeneration := service.reset()
	oldFrame := SubtitleFrame{Key: "one", Source: "Source", Translation: strings.Repeat("尚未读完", 30), Final: true}
	_ = service.acceptGeneration(oldGeneration, oldFrame)
	now := time.Unix(0, 0)
	service.presentAt(now)
	service.discard()
	if window.visible || service.playback.HasContent() || service.GetSubtitleFrame().Key != "" {
		t.Fatal("discard left unread captions visible")
	}
	_ = service.acceptGeneration(oldGeneration, oldFrame)
	newGeneration := service.reset()
	_ = service.acceptGeneration(oldGeneration, oldFrame)
	_ = service.acceptGeneration(newGeneration, SubtitleFrame{Key: "new", Translation: "新视频", Final: true})
	service.presentAt(now.Add(50 * time.Millisecond))
	if len(service.playback.queue) != 1 || service.playback.queue[0].frame.Key != "new" {
		t.Fatal("old results revived after restarting capture")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service.discard()
	if _, resumed := service.resume(ctx); resumed || service.currentGeneration(newGeneration) {
		t.Fatal("a canceled capture reactivated subtitle reception")
	}
}

func TestForwardPaginationPreservesUnicodeAndEveryByte(t *testing.T) {
	source := "  👨‍👩‍👧‍👦 café with words. 中文👍🏽。 More text.  "
	translation := "第一段。第二段。第三段。最后不能丢失。"
	originalSource, originalTranslation := source, translation
	var combinedSource, combinedTranslation string
	for i := 0; i < 100 && (source != "" || translation != ""); i++ {
		page, err := splitSubtitlePage(source, translation, func(_ int, text string) bool { return uniseg.GraphemeClusterCount(text) <= 6 })
		if err != nil {
			t.Fatal(err)
		}
		if !utf8.ValidString(page.Source) || !utf8.ValidString(page.Translation) {
			t.Fatal("pagination split a UTF-8 sequence")
		}
		combinedSource += source[:page.SourceBytes]
		combinedTranslation += translation[:page.TranslationBytes]
		source = source[page.SourceBytes:]
		translation = translation[page.TranslationBytes:]
	}
	if source != "" || translation != "" || combinedSource != originalSource || combinedTranslation != originalTranslation {
		t.Fatal("forward pagination skipped or repeated input bytes")
	}
	if subtitleReadingPrefix("👨‍👩‍👧‍👦 café", 0) != "👨‍👩‍👧‍👦" {
		t.Fatal("reveal split a grapheme cluster")
	}
}

func TestReadingQueueOverflowRetainsTheReceivedFragment(t *testing.T) {
	player := subtitlePlayback{queuedBytes: subtitleQueueLimit}
	err := player.Accept(SubtitleFrame{Key: "overflow", Translation: "这段已经收到了，仍然要显示。", Final: true})
	if err == nil || len(player.queue) != 1 || player.queue[0].frame.Translation != "这段已经收到了，仍然要显示。" {
		t.Fatal("overflow silently discarded the incoming fragment")
	}
}

func TestReadingReflowDoesNotAdvanceUnreadCursors(t *testing.T) {
	var player subtitlePlayback
	_ = player.Accept(SubtitleFrame{Key: "one", Source: strings.Repeat("Original words. ", 12), Translation: strings.Repeat("字体改变也不能漏掉未读内容。", 10), Final: true})
	now := time.Unix(0, 0)
	_, _ = player.Tick(now, readingTestPaginator)
	_, _ = player.Tick(now.Add(50*time.Millisecond), readingTestPaginator)
	player.Reflow()
	if player.queue[0].sourceOffset != 0 || player.queue[0].translationOffset != 0 {
		t.Fatal("font changes consumed an unread page")
	}
	frame, err := player.Tick(now.Add(100*time.Millisecond), func(source, translation string) (SubtitlePage, error) {
		return splitSubtitlePage(source, translation, func(_ int, text string) bool { return uniseg.GraphemeClusterCount(text) <= 8 })
	})
	if err != nil || !strings.HasPrefix(player.page.Source, "O") || !strings.HasPrefix(player.page.Translation, "字") {
		t.Fatalf("reflow lost the start of the unread page: %+v %v", frame, err)
	}
}
