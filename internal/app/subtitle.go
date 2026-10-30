package app

import (
	"context"
	"errors"
	"sync"
	"time"
)

// SubtitleSettings describes the appearance and placement of desktop captions.
type SubtitleSettings struct {
	Source   string  `json:"source"`
	Target   string  `json:"target"`
	FontSize int     `json:"fontSize"`
	Position string  `json:"position"`
	CustomX  float64 `json:"customX"`
	CustomY  float64 `json:"customY"`
	Style    string  `json:"style"`
}

type desktopSubtitle interface {
	Update(SubtitleSettings, SubtitleFrame, bool) error
	Page(SubtitleSettings, string, string) (subtitlePage, error)
	Close()
}

var errSubtitleNeedsReflow = errors.New("字幕区域改变，需要重新分页")

// SubtitleService keeps the native caption window in sync with user settings.
type SubtitleService struct {
	mu         sync.Mutex
	settings   SubtitleSettings
	visible    bool
	window     desktopSubtitle
	frame      SubtitleFrame
	playback   subtitlePlayback
	generation uint64
	accepting  bool
	emit       func(string, any)
}

func newSubtitleService() *SubtitleService {
	return &SubtitleService{
		settings: SubtitleSettings{
			Source: "en-US", Target: "zh-CN", FontSize: 30,
			Position: "bottom", CustomX: 50, CustomY: 82, Style: "charcoal",
		},
		window: newDesktopSubtitle(),
	}
}

// Configure applies caption settings, including to an already visible window.
func (s *SubtitleService) Configure(settings SubtitleSettings) error {
	if err := settings.validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	reflow := settings.FontSize != s.settings.FontSize || settings.Source != s.settings.Source || settings.Target != s.settings.Target
	if reflow && s.playback.HasContent() {
		s.settings = settings
		s.playback.Reflow()
		return nil
	}
	if err := s.window.Update(settings, s.frame, s.visible || s.playback.HasContent()); err != nil {
		return err
	}
	s.settings = settings
	return nil
}

func (s *SubtitleService) setVisible(visible bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.visible == visible {
		return nil
	}
	if err := s.window.Update(s.settings, s.frame, visible || s.playback.HasContent()); err != nil {
		return err
	}
	s.visible = visible
	return nil
}

func (s *SubtitleService) GetSubtitleFrame() SubtitleFrame {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.frame
}

func (s *SubtitleService) snapshotSettings() SubtitleSettings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

func (s *SubtitleService) accept(frame SubtitleFrame) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.playback.Accept(frame)
}

func (s *SubtitleService) reset() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearLocked(true)
	return s.generation
}

func (s *SubtitleService) resume(ctx context.Context) (uint64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil {
		return 0, false
	}
	s.clearLocked(true)
	return s.generation, true
}

func (s *SubtitleService) discard() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearLocked(false)
}

// Silence retires reception and drops the backlog, but the visible line still
// finishes revealing and gets its full reading hold. Manual stop uses discard.
func (s *SubtitleService) pause() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.generation++
	s.accepting = false
	s.playback.pending, s.playback.pendingOrder = nil, nil
	s.playback.next = nil
	if s.playback.pageFrame.Key == "" {
		s.playback = subtitlePlayback{}
		return
	}
	entry := s.playback.queue[0]
	entry.frame.Source = entry.frame.Source[:entry.sourceOffset+s.playback.page.SourceBytes]
	entry.frame.Translation = entry.frame.Translation[:entry.translationOffset+s.playback.page.TranslationBytes]
	entry.units = max(subtitleReadingUnits(entry.frame.Source[entry.sourceOffset:]), subtitleReadingUnits(entry.frame.Translation[entry.translationOffset:]))
	s.playback.queue = []*subtitleEntry{entry}
	s.playback.queuedBytes = len(entry.frame.Source) + len(entry.frame.Translation)
}

func (s *SubtitleService) clearLocked(accepting bool) {
	s.generation++
	s.accepting = accepting
	// Clear settled pages, previews, line state and estimates together.
	s.frame, s.playback = SubtitleFrame{}, subtitlePlayback{}
	if err := s.window.Update(s.settings, s.frame, false); err != nil && s.emit != nil {
		s.emit("subtitle:error", err.Error())
	}
	if s.emit != nil {
		s.emit("subtitle:frame", s.frame)
	}
}

func (s *SubtitleService) currentGeneration(generation uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.accepting && s.generation == generation
}

func (s *SubtitleService) acceptGeneration(generation uint64, frame SubtitleFrame) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.accepting || generation != s.generation {
		return nil
	}
	return s.playback.Accept(frame)
}

func (s *SubtitleService) interruptGeneration(generation uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.accepting || generation != s.generation {
		return false
	}
	s.playback.MarkIncomplete()
	return true
}

func (s *SubtitleService) start(ctx context.Context, emit func(string, any)) {
	s.emit = emit
	go func() {
		ticker := time.NewTicker(subtitleFrameInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.present()
			}
		}
	}()
}

func (s *SubtitleService) present() {
	s.presentAt(time.Now())
}

func (s *SubtitleService) presentAt(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next, err := s.playback.Tick(now, func(source, translation string) (subtitlePage, error) {
		return s.window.Page(s.settings, source, translation)
	})
	if err != nil {
		if s.emit != nil {
			s.emit("subtitle:error", err.Error())
		}
		return
	}
	if next == s.frame {
		return
	}
	if err := s.window.Update(s.settings, next, s.visible || s.playback.HasContent()); err != nil {
		if errors.Is(err, errSubtitleNeedsReflow) {
			s.playback.Reflow()
			return
		}
		if s.emit != nil {
			s.emit("subtitle:error", err.Error())
		}
		return
	}
	s.frame = next
	if s.emit != nil {
		s.emit("subtitle:frame", s.frame)
	}
}

func (s *SubtitleService) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.window.Close()
	s.visible = false
}

func (s SubtitleSettings) validate() error {
	if subtitlePlaceholderText(s.Source) == "" || subtitlePlaceholderText(s.Target) == "" {
		return errors.New("字幕语言无效")
	}
	if s.FontSize < 18 || s.FontSize > 40 {
		return errors.New("字幕字号应在 18 到 40 之间")
	}
	if s.Position != "top" && s.Position != "bottom" && s.Position != "custom" {
		return errors.New("字幕位置无效")
	}
	if !validSubtitlePercentage(s.CustomX) || !validSubtitlePercentage(s.CustomY) {
		return errors.New("自定义字幕位置应在 0 到 100 之间")
	}
	if _, ok := subtitleStyles[s.Style]; !ok {
		return errors.New("字幕样式无效")
	}
	return nil
}

func validSubtitlePercentage(value float64) bool {
	// These comparisons also reject NaN and both infinities.
	return value >= 0 && value <= 100
}

type subtitleColor struct{ r, g, b, a byte }
type subtitleStyle struct{ background, foreground, outline subtitleColor }

// Keep these colors aligned with the subtitle-caption presets in subtitle.css.
var subtitleStyles = map[string]subtitleStyle{
	"charcoal": {subtitleColor{0, 0, 0, 179}, subtitleColor{255, 255, 255, 255}, subtitleColor{0, 0, 0, 96}},
	"mist":     {subtitleColor{15, 23, 42, 26}, subtitleColor{255, 255, 255, 255}, subtitleColor{0, 0, 0, 204}},
	"paper":    {subtitleColor{255, 255, 255, 38}, subtitleColor{32, 36, 44, 255}, subtitleColor{255, 255, 255, 230}},
	"slate":    {subtitleColor{0, 0, 0, 0}, subtitleColor{255, 233, 163, 255}, subtitleColor{0, 0, 0, 230}},
	"graphite": {subtitleColor{15, 23, 42, 51}, subtitleColor{230, 237, 247, 255}, subtitleColor{0, 0, 0, 220}},
}

// subtitlePlaceholderText returns the same sample sentence as the settings preview.
func subtitlePlaceholderText(language string) string {
	switch language {
	case "zh-CN":
		return "未来属于那些不断探索的人。"
	case "zh-TW":
		return "未來屬於那些不斷探索的人。"
	case "en-US", "en-GB":
		return "The future belongs to those who keep exploring."
	case "ja-JP":
		return "未来は、探求し続ける人のものです。"
	case "ko-KR":
		return "미래는 끊임없이 탐구하는 사람들의 것입니다."
	case "fr-FR":
		return "L’avenir appartient à ceux qui continuent d’explorer."
	case "de-DE":
		return "Die Zukunft gehört denen, die immer weiter entdecken."
	default:
		return ""
	}
}
