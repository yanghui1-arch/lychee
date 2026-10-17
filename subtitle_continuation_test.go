package main

import (
	"strings"
	"testing"
	"time"

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

func TestShortCaptionWaitsAndAppendsWithoutRetyping(t *testing.T) {
	var player subtitlePlayback
	first := SubtitleFrame{Key: "session:sequence:1", Source: "Today", Translation: "今天", Final: true}
	second := SubtitleFrame{Key: "session:sequence:2", Source: "we meet.", Translation: "我们见面。", Final: true}
	third := SubtitleFrame{Key: "session:sequence:3", Source: "Again.", Translation: "再见。", Final: true}
	_ = player.Accept(first)
	now := time.Unix(0, 0)
	initial, _ := player.Tick(now, readingTestPaginator)
	frame := advanceCaption(t, &player, &now, 1500*time.Millisecond, readingTestPaginator)
	if frame.Key != initial.Key || frame.Source != first.Source || frame.Translation != first.Translation {
		t.Fatal("short caption cleared before its continuation window ended")
	}
	_ = player.Accept(second)
	frame = advanceCaption(t, &player, &now, 50*time.Millisecond, readingTestPaginator)
	if frame.Key != initial.Key || !strings.HasPrefix(frame.Source, first.Source) || !strings.HasPrefix(frame.Translation, first.Translation) || len(player.queue) != 1 {
		t.Fatalf("continuation erased/restarted the visible prefix: %+v", frame)
	}
	_ = player.Accept(second) // Settled fragments remain deduplicated after merging.
	_ = player.Accept(third)
	previous := frame
	for i := 0; i < 20; i++ {
		frame = advanceCaption(t, &player, &now, 50*time.Millisecond, readingTestPaginator)
		if frame.Key != initial.Key || !strings.HasPrefix(frame.Source, previous.Source) || !strings.HasPrefix(frame.Translation, previous.Translation) {
			t.Fatalf("appending another fragment rewound the text or restarted the fade: before=%+v after=%+v", previous, frame)
		}
		previous = frame
	}
	if frame.Source != "Today we meet. Again." || frame.Translation != "今天我们见面。再见。" || len(player.queue) != 1 {
		t.Fatalf("lost/duplicated a continuation: %+v", frame)
	}
	if player.queuedBytes != len(frame.Source)+len(frame.Translation) {
		t.Fatal("joining fragments corrupted queue byte accounting")
	}
	player.stopped = true
	advanceCaption(t, &player, &now, time.Second, readingTestPaginator)
	if player.HasContent() || player.queuedBytes != 0 {
		t.Fatal("joined text did not drain completely")
	}
}

func TestShortCaptionWaitIsBoundedAndStopSkipsExtraWait(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		player := subtitlePlayback{stopped: stopped}
		_ = player.Accept(SubtitleFrame{Key: "session:sequence:1", Translation: "好", Final: true})
		now := time.Unix(0, 0)
		_, _ = player.Tick(now, readingTestPaginator)
		frame := advanceCaption(t, &player, &now, 1200*time.Millisecond, readingTestPaginator)
		if (frame.Key == "") != stopped {
			t.Fatalf("stop/wait policy is wrong: stopped=%v frame=%+v", stopped, frame)
		}
		if !stopped {
			frame = advanceCaption(t, &player, &now, 850*time.Millisecond, readingTestPaginator)
			if frame.Translation != "好" {
				t.Fatal("short caption disappeared before the extra 900ms elapsed")
			}
			frame = advanceCaption(t, &player, &now, 50*time.Millisecond, readingTestPaginator)
			if frame.Key != "" || player.HasContent() {
				t.Fatal("short caption waited indefinitely without a continuation")
			}
		}
	}
}

func TestShortCaptionContinuationHonorsPairAndLayoutBoundaries(t *testing.T) {
	for _, test := range []struct {
		name string
		next SubtitleFrame
		fits int
	}{
		{"new session", SubtitleFrame{Key: "other:sequence:2", Source: "next", Translation: "后续", Final: true}, 100},
		{"sequence gap", SubtitleFrame{Key: "session:sequence:3", Source: "next", Translation: "后续", Final: true}, 100},
		{"missing original", SubtitleFrame{Key: "session:sequence:2", Translation: "后续", Final: true}, 100},
		{"incomplete", SubtitleFrame{Key: "session:sequence:2", Source: "next", Translation: "未完成", Incomplete: true}, 100},
		{"page budget", SubtitleFrame{Key: "session:sequence:2", Source: "next", Translation: strings.Repeat("字", 28), Final: true}, 100},
		{"narrow layout", SubtitleFrame{Key: "session:sequence:2", Source: "next", Translation: "后续", Final: true}, 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			var player subtitlePlayback
			_ = player.Accept(SubtitleFrame{Key: "session:sequence:1", Source: "Here", Translation: "这里", Final: true})
			now := time.Unix(0, 0)
			paginate := func(source, translation string) (subtitlePage, error) {
				return splitSubtitlePage(source, translation, func(_ int, text string) bool { return uniseg.GraphemeClusterCount(text) <= test.fits })
			}
			initial, _ := player.Tick(now, paginate)
			_ = player.Accept(test.next)
			frame := advanceCaption(t, &player, &now, 1100*time.Millisecond, paginate)
			if frame.Key != initial.Key || frame.Source != "Here" || frame.Translation != "这里" || len(player.queue) != 2 {
				t.Fatalf("unrelated or oversized text merged into this page: %+v", frame)
			}
			frame = advanceCaption(t, &player, &now, 100*time.Millisecond, paginate)
			if frame.Key == initial.Key || !strings.HasPrefix(frame.Key, test.next.Key+":page:") {
				t.Fatal("non-continuation did not advance through ordinary pagination")
			}
		})
	}
}

func TestShortCaptionReflowRetainsAllMergedText(t *testing.T) {
	var player subtitlePlayback
	_ = player.Accept(SubtitleFrame{Key: "session:sequence:1", Source: "First", Translation: "第一段", Final: true})
	_ = player.Accept(SubtitleFrame{Key: "session:sequence:2", Source: "next", Translation: "后续内容", Final: true})
	now := time.Unix(0, 0)
	_, _ = player.Tick(now, readingTestPaginator)
	advanceCaption(t, &player, &now, 600*time.Millisecond, readingTestPaginator)
	if len(player.queue) != 1 {
		t.Fatal("test did not append the short continuation")
	}
	player.Reflow()
	frame := advanceCaption(t, &player, &now, 50*time.Millisecond, readingTestPaginator)
	if frame.Source != "F" || frame.Translation != "第" || player.page.Source != "First next" || player.page.Translation != "第一段后续内容" {
		t.Fatalf("reflow retained a stale suffix clock or lost merged text: %+v", frame)
	}
}
