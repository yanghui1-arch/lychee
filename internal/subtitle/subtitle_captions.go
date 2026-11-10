package subtitle

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/yanghui1-arch/lychee/internal/speech"
)

type SubtitleFrame struct {
	Key            string `json:"key"`
	Source         string `json:"source"`
	Translation    string `json:"translation"`
	Final          bool   `json:"final"`
	Incomplete     bool   `json:"incomplete"`
	PendingSeconds int    `json:"pendingSeconds"`
}

type transcriptSnapshot struct {
	text  string
	final bool
}
type captionFragment struct {
	source, translation string
	sentences           []int
	final               bool
}

// Server messages are snapshots, not deltas. Keep sentence revisions separate
// from append-only translation fragments, keyed within the current WS session.
type speechCaptions struct {
	session     string
	transcripts map[int]transcriptSnapshot
	fragments   map[int]captionFragment
	current     int
	translated  bool
}

func (c *speechCaptions) Apply(event speech.Event) (SubtitleFrame, bool) {
	if event.SessionID != c.session || c.transcripts == nil {
		*c = speechCaptions{session: event.SessionID, transcripts: map[int]transcriptSnapshot{}, fragments: map[int]captionFragment{}}
	}
	publish := false
	switch event.Type {
	case "transcript":
		if event.SentenceID <= 0 {
			return SubtitleFrame{}, false
		}
		previous := c.transcripts[event.SentenceID]
		if previous.final {
			return SubtitleFrame{}, false
		}
		c.transcripts[event.SentenceID] = transcriptSnapshot{event.Text, event.IsFinal}
		// Recognition can be ahead of translation. Cache it without exposing a
		// source-only frame or changing the source of an already visible pair.
	case "translating", "translation_preview", "translation":
		if event.Sequence <= 0 {
			return SubtitleFrame{}, false
		}
		part := c.fragments[event.Sequence]
		if part.final {
			return SubtitleFrame{}, false
		}
		if len(event.SentenceIDs) > 0 {
			part.sentences = event.SentenceIDs
		}
		if event.SourceText != "" {
			part.source = event.SourceText
		}
		if event.Type != "translating" {
			if strings.TrimSpace(event.TranslatedText) == "" {
				return c.frame(), false
			}
			// Ignore regressing previews; final snapshots may legitimately repair
			// punctuation. Never append repeated complete snapshots to the text.
			if event.Type == "translation_preview" && !strings.HasPrefix(event.TranslatedText, part.translation) {
				return SubtitleFrame{}, false
			}
			part.translation, part.final = event.TranslatedText, event.Type == "translation"
			if event.Sequence >= c.current || !c.translated {
				c.current, c.translated = event.Sequence, true
			}
			publish = true
		}
		c.fragments[event.Sequence] = part
	default:
		return SubtitleFrame{}, false
	}
	frame := c.frame()
	if publish {
		frame = c.fragmentFrame(event.Sequence)
	}
	// Only retain nearby recognition and translation context in a long monologue.
	for id := range c.transcripts {
		if id < event.SentenceID-64 {
			delete(c.transcripts, id)
		}
	}
	for sequence := range c.fragments {
		if sequence < c.current-128 {
			delete(c.fragments, sequence)
		}
	}
	return frame, publish
}

func (c *speechCaptions) frame() SubtitleFrame {
	if !c.translated {
		return SubtitleFrame{}
	}
	return c.fragmentFrame(c.current)
}

func (c *speechCaptions) fragmentFrame(sequence int) SubtitleFrame {
	part := c.fragments[sequence]
	// A queue item is one server sequence, never the cumulative sentence.
	// This avoids replaying earlier translated fragments on each new result.
	frame := SubtitleFrame{Key: fmt.Sprintf("%s:sequence:%d", c.session, sequence), Source: part.source, Translation: part.translation, Final: part.final}
	sentenceIDs := map[int]bool{}
	for _, id := range part.sentences {
		sentenceIDs[id] = true
	}
	ids := make([]int, 0, len(sentenceIDs))
	for id := range sentenceIDs {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	var transcript string
	for _, id := range ids {
		transcript = joinSpeechText(transcript, c.transcripts[id].text)
	}
	// A full ASR sentence may include a tail that has not been translated yet.
	// Prefer the source fragments belonging to the displayed translation.
	if frame.Source == "" && transcript != "" {
		frame.Source = transcript
	}
	return frame
}

func joinSpeechText(left, right string) string {
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	l, r := []rune(left), []rune(right)
	a, b := l[len(l)-1], r[0]
	if a < unicode.MaxASCII && b < unicode.MaxASCII && (unicode.IsLetter(a) || unicode.IsDigit(a) || strings.ContainsRune(".!?,;:", a)) && (unicode.IsLetter(b) || unicode.IsDigit(b)) {
		return left + " " + right
	}
	return left + right
}
