package subtitle

import (
	"testing"

	"github.com/yanghui1-arch/lychee/internal/speech"
)

func TestCaptionSnapshotsAndFragments(t *testing.T) {
	var c speechCaptions
	apply := func(event speech.Event) SubtitleFrame {
		event.SessionID = "one"
		frame, _ := c.Apply(event)
		return frame
	}
	apply(speech.Event{Type: "transcript", SentenceID: 1, Text: "We will meet tomorrow."})
	apply(speech.Event{Type: "translating", Sequence: 1, SourceText: "We will meet", SentenceIDs: []int{1}})
	apply(speech.Event{Type: "translation_preview", Sequence: 1, TranslatedText: "我们"})
	frame := apply(speech.Event{Type: "translation_preview", Sequence: 1, TranslatedText: "我们见面"})
	if frame.Translation != "我们见面" || frame.Source != "We will meet" || frame.Final {
		t.Fatalf("bad preview: %+v", frame)
	}
	apply(speech.Event{Type: "translation", Sequence: 1, TranslatedText: "我们见面"})
	apply(speech.Event{Type: "translation_preview", Sequence: 1, TranslatedText: "迟到重复"})
	apply(speech.Event{Type: "translating", Sequence: 2, SourceText: "tomorrow.", SentenceIDs: []int{1}})
	frame = apply(speech.Event{Type: "translation", Sequence: 2, TranslatedText: "明天。"})
	if frame.Translation != "明天。" || frame.Source != "tomorrow." || !frame.Final {
		t.Fatalf("fragments overwritten or duplicated: %+v", frame)
	}
	// The next sentence can arrive while the current translation is still active.
	frame = apply(speech.Event{Type: "transcript", SentenceID: 2, Text: "Next sentence"})
	if frame.Source != "tomorrow." {
		t.Fatal("next ASR sentence replaced current translation")
	}
	frame, _ = c.Apply(speech.Event{Type: "translation", SessionID: "two", Sequence: 1, TranslatedText: "Translation only"})
	if frame.Source != "" || frame.Translation != "Translation only" {
		t.Fatalf("missing source or session reset mishandled: %+v", frame)
	}
}

func TestTranscriptFinalRetractionAndLateRevision(t *testing.T) {
	var c speechCaptions
	c.Apply(speech.Event{Type: "transcript", SentenceID: 1, Text: "hypothesis"})
	frame, _ := c.Apply(speech.Event{Type: "transcript", SentenceID: 1, Text: "", IsFinal: true})
	if frame.Source != "" {
		t.Fatal("empty final did not retract hypothesis")
	}
	_, changed := c.Apply(speech.Event{Type: "transcript", SentenceID: 1, Text: "late"})
	if changed {
		t.Fatal("late interim changed a final sentence")
	}
}

func TestCaptionsWaitForMatchingTranslation(t *testing.T) {
	var captions speechCaptions
	service := newSubtitleService()
	service.window = &recordingSubtitle{}
	feed := func(event speech.Event) bool {
		event.SessionID = "paired"
		frame, publish := captions.Apply(event)
		if publish {
			service.accept(frame)
		}
		service.present()
		return publish
	}
	if feed(speech.Event{Type: "transcript", SentenceID: 1, Text: "First fragment. Still untranslated."}) {
		t.Fatal("ASR published before a translation")
	}
	if feed(speech.Event{Type: "translating", Sequence: 1, SentenceIDs: []int{1}, SourceText: "First fragment."}) {
		t.Fatal("translating published a source-only frame")
	}
	if frame := service.GetSubtitleFrame(); frame.Key != "" {
		t.Fatalf("source appeared before translation: %+v", frame)
	}
	if !feed(speech.Event{Type: "translation_preview", Sequence: 1, TranslatedText: "第一段。"}) {
		t.Fatal("first translation was withheld")
	}
	if frame := service.GetSubtitleFrame(); frame.Key != "" {
		t.Fatalf("unconfirmed preview entered the reading queue: %+v", frame)
	}
	if captions.frame().Source != "First fragment." {
		t.Fatal("untranslated ASR tail appeared in the pair")
	}
	feed(speech.Event{Type: "translation", Sequence: 1, TranslatedText: "第一段。"})
	previous := service.GetSubtitleFrame()
	if previous.Source == "" || previous.Translation == "" {
		t.Fatal("first confirmed page is not paired")
	}
	if feed(speech.Event{Type: "transcript", SentenceID: 1, Text: "First fragment. Next fragment. More untranslated text."}) {
		t.Fatal("ASR revision updated the source by itself")
	}
	if feed(speech.Event{Type: "translating", Sequence: 2, SentenceIDs: []int{1}, SourceText: "Next fragment."}) {
		t.Fatal("next source fragment ran ahead of translation")
	}
	if service.GetSubtitleFrame().Key != previous.Key {
		t.Fatal("waiting for the next translation changed the displayed pair")
	}
	feed(speech.Event{Type: "translation_preview", Sequence: 2, TranslatedText: "下一段。"})
	if len(service.playback.queue) != 1 {
		t.Fatal("next preview replaced the active reading page")
	}
	feed(speech.Event{Type: "translation", Sequence: 2, TranslatedText: "下一段。"})
	if len(service.playback.queue) != 2 || service.playback.queue[1].frame.Source != "Next fragment." || service.playback.queue[1].frame.Translation != "下一段。" {
		t.Fatal("next confirmed pair was not queued independently")
	}
	if feed(speech.Event{Type: "translation_preview", Sequence: 3, TranslatedText: "   "}) {
		t.Fatal("empty translation published a frame")
	}
	service.accept(SubtitleFrame{Key: "unpaired", Source: "must stay hidden"})
	if len(service.playback.queue) != 2 {
		t.Fatal("presenter accepted a source-only frame")
	}
}

func TestCaptionReadingTextBoundaries(t *testing.T) {
	if cleanSubtitleText("\x00  hello  ") != "hello" {
		t.Fatal("presentation did not sanitize text")
	}
	if subtitleReadingPrefix("你好世界", 0) != "你" {
		t.Fatal("first reading frame was not a single complete grapheme")
	}
}
