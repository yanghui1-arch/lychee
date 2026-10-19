//go:build windows && amd64

package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestDesktopSubtitleUsesBundledRoboto(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	font, err := subtitleFont(30, "en-US")
	if err != nil {
		t.Fatal(err)
	}
	defer subtitleDeleteObject.Call(font)
	dc, _, _ := subtitleCreateDC.Call(0)
	if dc == 0 {
		t.Fatal("cannot create font test DC")
	}
	defer subtitleDeleteDC.Call(dc)
	previous, _, _ := subtitleSelectObject.Call(dc, font)
	defer subtitleSelectObject.Call(dc, previous)
	var name [64]uint16
	result, _, _ := subtitleGDI32.NewProc("GetTextFaceW").Call(dc, uintptr(len(name)), uintptr(unsafe.Pointer(&name[0])))
	if result == 0 || !strings.Contains(windows.UTF16ToString(name[:]), "Roboto") {
		t.Fatalf("bundled font fell back to %q", windows.UTF16ToString(name[:]))
	}
}

func TestNativeDesktopSubtitleWindow(t *testing.T) {
	window := newDesktopSubtitle().(*windowsDesktopSubtitle)
	defer window.Close()
	settings := newSubtitleService().settings
	if err := window.Update(settings, sampleSubtitleFrame(settings), false); err != nil {
		t.Fatal(err)
	}
	if window.window != 0 {
		t.Fatal("hidden settings created a native window")
	}
	foreground, _, _ := subtitleForeground.Call()
	if err := window.Update(settings, sampleSubtitleFrame(settings), true); err != nil {
		t.Fatal(err)
	}
	if !windows.IsWindowVisible(windows.HWND(window.window)) {
		t.Fatal("enabled desktop subtitle window is not visible")
	}
	after, _, _ := subtitleForeground.Call()
	if foreground != after {
		t.Fatal("subtitle window stole foreground focus")
	}
	style, _, _ := applicationWindowStyle.Call(window.window, ^uintptr(19))
	if style&subtitleWindowStyles != subtitleWindowStyles {
		t.Fatal("subtitle window is not topmost, passive, and click-through")
	}
	getRect := subtitleUser32.NewProc("GetWindowRect")
	var bottom, top subtitleRect
	getRect.Call(window.window, uintptr(unsafe.Pointer(&bottom)))
	settings.FontSize, settings.Position, settings.Style = 40, "top", "paper"
	if err := window.Update(settings, sampleSubtitleFrame(settings), true); err != nil {
		t.Fatal(err)
	}
	getRect.Call(window.window, uintptr(unsafe.Pointer(&top)))
	if top.Top >= bottom.Top || top.Bottom-top.Top != bottom.Bottom-bottom.Top || top.Right-top.Left != bottom.Right-bottom.Left {
		t.Fatal("position did not update or font changes resized the fixed caption canvas")
	}
	settings.Position, settings.CustomX, settings.CustomY = "custom", 0, 50
	if err := window.Update(settings, sampleSubtitleFrame(settings), true); err != nil {
		t.Fatal(err)
	}
	var custom subtitleRect
	getRect.Call(window.window, uintptr(unsafe.Pointer(&custom)))
	if custom.Left >= top.Left || custom.Top <= top.Top || custom.Top >= bottom.Top {
		t.Fatal("custom position did not move the desktop window")
	}
	if err := window.Update(settings, sampleSubtitleFrame(settings), false); err != nil {
		t.Fatal(err)
	}
	if windows.IsWindowVisible(windows.HWND(window.window)) {
		t.Fatal("disabled desktop subtitle window remained visible")
	}
	handle := window.window
	window.Close()
	if result, _, _ := subtitleUser32.NewProc("IsWindow").Call(handle); result != 0 {
		t.Fatal("subtitle window survived shutdown")
	}
}

func TestNativeSubtitleRendering(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	settings := newSubtitleService().settings
	for _, style := range []string{"charcoal", "mist", "paper", "slate", "graphite"} {
		settings.Style = style
		if style == "charcoal" && subtitleStyles[style].background.a != 179 {
			t.Fatal("default backdrop must match the 70% opacity reference")
		}
		if style != "charcoal" && subtitleStyles[style].background.a > 51 {
			t.Fatal("lightweight alternate backdrops must not exceed 20% opacity")
		}
		surface, err := renderSubtitle(settings, 1000, 1, sampleSubtitleFrame(settings))
		if err != nil {
			t.Fatal(err)
		}
		if surface.pixels[3] != 0 {
			t.Fatal("unused fixed canvas must remain transparent")
		}
		background := (surface.width/2 + surface.width*3) * 4
		if surface.pixels[background+3] != 0 {
			t.Fatalf("%s backdrop still fills the fixed canvas", style)
		}
		glyphPixels, solidPixels, outlinePixels := 0, 0, 0
		for i := 0; i < len(surface.pixels); i += 4 {
			alpha := surface.pixels[i+3]
			if surface.pixels[i] > alpha || surface.pixels[i+1] > alpha || surface.pixels[i+2] > alpha {
				t.Fatal("layered window pixels are not premultiplied")
			}
			if alpha > subtitleStyles[style].background.a {
				glyphPixels++
			}
			if alpha >= 250 {
				solidPixels++
			}
			if alpha > 120 {
				color := subtitleStyles[style].outline
				matches := true
				for channel, value := range [3]byte{color.b, color.g, color.r} {
					difference := int(surface.pixels[i+channel]) - int(value)*int(alpha)/255
					matches = matches && difference >= -3 && difference <= 3
				}
				if matches {
					outlinePixels++
				}
			}
		}
		if glyphPixels < 100 {
			t.Fatal("subtitle glyphs are missing")
		}
		if solidPixels < 100 || outlinePixels < 50 {
			t.Fatalf("%s lost solid glyphs or contrasting outlines: solid=%d outline=%d", style, solidPixels, outlinePixels)
		}
		if directory := os.Getenv("LYCHEE_SUBTITLE_PREVIEW_DIR"); directory != "" {
			if err := os.MkdirAll(directory, 0755); err != nil {
				t.Fatal(err)
			}
			output, err := os.Create(filepath.Join(directory, style+".png"))
			if err != nil {
				t.Fatal(err)
			}
			pixels := append([]byte(nil), surface.pixels...)
			for i := 0; i < len(pixels); i += 4 {
				pixels[i], pixels[i+2] = pixels[i+2], pixels[i]
			}
			err = png.Encode(output, &image.RGBA{Pix: pixels, Stride: surface.width * 4, Rect: image.Rect(0, 0, surface.width, surface.height)})
			_ = output.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
		surface.close()
	}
	settings.Source, settings.Target = "ko-KR", "ja-JP"
	surface, err := renderSubtitle(settings, 400, 1.5, sampleSubtitleFrame(settings))
	if err != nil {
		t.Fatal(err)
	}
	defer surface.close()
	if surface.width > 400 || surface.height < 150 {
		t.Fatal("narrow, scaled subtitles did not wrap inside the available width")
	}
}

func TestNativeSubtitleBackdropFollowsFontSize(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	settings := newSubtitleService().settings
	previousArea := 0
	for _, fontSize := range []int{18, 30, 40} {
		settings.FontSize = fontSize
		surface, err := renderSubtitle(settings, 1000, 1, SubtitleFrame{Source: "Hello", Translation: "你好"})
		if err != nil {
			t.Fatal(err)
		}
		area, backdropPixels := 0, 0
		for i := 3; i < len(surface.pixels); i += 4 {
			if surface.pixels[i] != 0 {
				area++
			}
			if surface.pixels[i] == subtitleStyles[settings.Style].background.a {
				backdropPixels++
			}
		}
		if surface.width != 1000 || surface.height != 240 || area <= previousArea || area > 1000*240/8 || backdropPixels < 50 {
			t.Fatalf("backdrop does not fit text/font: font=%d area=%d previous=%d backdrop=%d", fontSize, area, previousArea, backdropPixels)
		}
		previousArea = area
		surface.close()
	}
}

func TestSubtitlePositionOnOffsetMonitor(t *testing.T) {
	work := subtitleRect{-1920, -200, 0, 840}
	settings := newSubtitleService().settings
	bottom := subtitlePosition(work, 800, 120, 24, settings)
	settings.Position = "top"
	top := subtitlePosition(work, 800, 120, 24, settings)
	settings.Position, settings.CustomX, settings.CustomY = "custom", 100, 100
	custom := subtitlePosition(work, 800, 120, 24, settings)
	if top.Y != -176 || bottom.Y != 696 || bottom.X != -1360 || custom.X != -824 || custom.Y != bottom.Y {
		t.Fatalf("incorrect placement on offset monitor: top=%+v bottom=%+v custom=%+v", top, bottom, custom)
	}
}

func TestNativeSubtitleMissingSourceKeepsFixedCanvas(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	settings := newSubtitleService().settings
	translation := SubtitleFrame{Key: "live", Translation: "这是一条真实返回的译文。"}
	one, err := renderSubtitle(settings, 1000, 1, translation)
	if err != nil {
		t.Fatal(err)
	}
	defer one.close()
	translation.Source = "This is a live translation."
	two, err := renderSubtitle(settings, 1000, 1, translation)
	if err != nil {
		t.Fatal(err)
	}
	defer two.close()
	if two.height != one.height || two.width != one.width {
		t.Fatalf("missing source resized the caption canvas: single=%dx%d bilingual=%dx%d", one.width, one.height, two.width, two.height)
	}
	window := newDesktopSubtitle().(*windowsDesktopSubtitle)
	defer window.Close()
	if err = window.Update(settings, SubtitleFrame{}, true); err != nil {
		t.Fatal(err)
	}
	if windows.IsWindowVisible(windows.HWND(window.window)) {
		t.Fatal("empty live caption displayed placeholder text")
	}
}

func TestNativeSubtitleCanvasDoesNotGrowWithText(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	settings := newSubtitleService().settings
	for _, fontSize := range []int{18, 30, 40} {
		settings.FontSize = fontSize
		for _, translation := range []string{"你", strings.Repeat("不断增长的实时字幕，", 100)} {
			for _, source := range []string{"", "Hello", strings.Repeat("A continually growing transcript with more words. ", 100)} {
				surface, err := renderSubtitle(settings, 1000, 1, SubtitleFrame{Source: source, Translation: translation})
				if err != nil {
					t.Fatal(err)
				}
				if surface.width != 1000 || surface.height != 240 {
					t.Fatalf("text/font resized canvas: %dx%d font=%d", surface.width, surface.height, fontSize)
				}
				surface.close()
			}
		}
	}
	font, err := subtitleFont(40, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	defer subtitleDeleteObject.Call(font)
	dc, _, _ := subtitleCreateDC.Call(0)
	if dc == 0 {
		t.Fatal("cannot create measurement DC")
	}
	defer subtitleDeleteDC.Call(dc)
	text := strings.Repeat("很长的字幕不能继续扩展窗口", 40) + "这是最新尾部。"
	fitted, height := fitSubtitleText(dc, font, text, 300, 96)
	if height > 96 || !strings.HasPrefix(fitted, "…") || !strings.HasSuffix(fitted, "最新尾部。") {
		t.Fatalf("text escaped its fixed row or lost latest tail: %q height=%d", fitted, height)
	}
	for _, word := range []string{strings.Repeat("W", 300), strings.Repeat("🙂", 100)} {
		fitted, height = fitSubtitleText(dc, font, word, 300, 96)
		rect := subtitleRect{Right: 300}
		drawSubtitleText(dc, font, fitted, &rect, true)
		if height > 96 || rect.Right > 300 {
			t.Fatalf("unbroken text overflowed: %q rect=%+v height=%d", fitted, rect, height)
		}
	}
}

func BenchmarkLiveSubtitleRendering(b *testing.B) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	settings := newSubtitleService().settings
	frame := sampleSubtitleFrame(settings)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		surface, err := renderSubtitle(settings, 1000, 1, frame)
		if err != nil {
			b.Fatal(err)
		}
		surface.close()
	}
}

func TestNativeReadingPagesFitWithoutDiscardingText(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for _, test := range []struct {
		font, width int
		scale       float64
	}{{30, 1000, 1}, {40, 400, 1.25}, {18, 700, 2}} {
		settings := newSubtitleService().settings
		settings.FontSize = test.font
		source := "The opening words must stay visible. " + strings.Repeat("Every word in this long paragraph needs its own reading time. ", 4) + "Final original words."
		translation := "开头的文字不能消失。" + strings.Repeat("服务器一次返回很多内容时，也必须逐页展示，不能只保留尾部。", 5) + "最后的译文也要完整显示。"
		originalSource, originalTranslation := source, translation
		var displayedSource, displayedTranslation string
		for pageIndex := 0; source != "" || translation != ""; pageIndex++ {
			if pageIndex > 1000 {
				t.Fatal("native pagination made no progress")
			}
			page, err := paginateNativeSubtitle(settings, source, translation, test.width, test.scale)
			if err != nil {
				t.Fatal(err)
			}
			frame := SubtitleFrame{Key: "test:page:1", Source: page.Source, Translation: page.Translation, Final: true}
			// Live page frames fail instead of silently clipping into a tail.
			surface, err := renderSubtitle(settings, test.width, test.scale, frame)
			if err != nil {
				t.Fatalf("page does not fit real renderer: %+v %v", test, err)
			}
			if directory := os.Getenv("LYCHEE_SUBTITLE_PREVIEW_DIR"); directory != "" && test.font == 30 && pageIndex < 2 {
				pixels := append([]byte(nil), surface.pixels...)
				for i := 0; i < len(pixels); i += 4 {
					pixels[i], pixels[i+2] = pixels[i+2], pixels[i]
				}
				name := []string{"reading-first.png", "reading-second.png"}[pageIndex]
				file, err := os.Create(filepath.Join(directory, name))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(file, &image.RGBA{Pix: pixels, Stride: surface.width * 4, Rect: image.Rect(0, 0, surface.width, surface.height)})
				file.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
			surface.close()
			displayedSource += page.Source
			displayedTranslation += page.Translation
			source = source[page.SourceBytes:]
			translation = translation[page.TranslationBytes:]
		}
		if withoutSpaces(displayedSource) != withoutSpaces(originalSource) || withoutSpaces(displayedTranslation) != withoutSpaces(originalTranslation) {
			t.Fatal("native reading pages dropped part of the reply")
		}
	}
}

func TestNativeSingleLineCaptionsAppendThenReplace(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for _, test := range []struct {
		font, width int
		scale       float64
	}{{30, 1000, 1}, {40, 400, 1.25}, {18, 700, 2}} {
		settings := newSubtitleService().settings
		settings.FontSize = test.font
		paginate := func(source, translation string) (subtitlePage, error) {
			return paginateNativeSubtitle(settings, source, translation, test.width, test.scale)
		}
		var player subtitlePlayback
		_ = player.Accept(SubtitleFrame{Key: "first", Source: "Hello.", Translation: "你好。", Final: true})
		_ = player.Accept(SubtitleFrame{Key: "short", Source: "More.", Translation: "后续。", Final: true})
		_ = player.Accept(SubtitleFrame{Key: "second", Source: strings.Repeat("New words follow the previous sentence. ", 3), Translation: strings.Repeat("后续字幕接在已有内容后面，空间不足时整行清屏。", 3), Final: true})
		now := time.Unix(0, 0)
		_, err := player.Tick(now, paginate)
		if err != nil {
			t.Fatal(err)
		}
		var checked uint64
		retained, replaced := false, false
		var previousKey string
		for i := 0; i < 5000 && player.HasContent(); i++ {
			frame := advanceCaption(t, &player, &now, 50*time.Millisecond, paginate)
			if frame.Key == "" || player.elapsed < player.pageDuration || player.pageNumber == checked {
				continue
			}
			checked = player.pageNumber
			replaced = replaced || (previousKey != "" && previousKey != frame.Key)
			previousKey = frame.Key
			retained = retained || player.revealedSource != "" || player.revealedTranslation != ""
			surface, err := renderSubtitle(settings, test.width, test.scale, frame)
			if err != nil {
				t.Fatalf("single-line caption overflowed native layout (%+v): %v", test, err)
			}
			surface.close()
		}
		if player.HasContent() || !retained || !replaced {
			t.Fatalf("single-line captions failed to append, replace, or drain: %+v", test)
		}
	}
}

func TestNativeSubtitleRejectsASecondLine(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	font, err := subtitleFont(30, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	defer subtitleDeleteObject.Call(font)
	dc, _, _ := subtitleCreateDC.Call(0)
	if dc == 0 {
		t.Fatal("cannot create single-line measurement DC")
	}
	defer subtitleDeleteDC.Call(dc)
	singleHeight, fits := measureSubtitlePageText(dc, font, "单行字幕", 190, 120)
	if !fits {
		t.Fatal("one line was rejected")
	}
	height, fits := measureSubtitlePageText(dc, font, strings.Repeat("字", 12), 190, 120)
	if fits || height <= singleHeight {
		t.Fatal("two lines were accepted as a single line")
	}
	_, height = fitSubtitleText(dc, font, strings.Repeat("字", 20), 190, 120)
	if height > singleHeight {
		t.Fatal("fallback rendering still wraps to a second line")
	}
}
