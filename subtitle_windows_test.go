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
	if err := window.Update(settings, false); err != nil {
		t.Fatal(err)
	}
	if window.window != 0 {
		t.Fatal("hidden settings created a native window")
	}
	foreground, _, _ := subtitleForeground.Call()
	if err := window.Update(settings, true); err != nil {
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
	if err := window.Update(settings, true); err != nil {
		t.Fatal(err)
	}
	getRect.Call(window.window, uintptr(unsafe.Pointer(&top)))
	if top.Top >= bottom.Top || top.Bottom-top.Top <= bottom.Bottom-bottom.Top {
		t.Fatal("live font size and position were not applied")
	}
	settings.Position, settings.CustomX, settings.CustomY = "custom", 0, 50
	if err := window.Update(settings, true); err != nil {
		t.Fatal(err)
	}
	var custom subtitleRect
	getRect.Call(window.window, uintptr(unsafe.Pointer(&custom)))
	if custom.Left >= top.Left || custom.Top <= top.Top || custom.Top >= bottom.Top {
		t.Fatal("custom position did not move the desktop window")
	}
	if err := window.Update(settings, false); err != nil {
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
		surface, err := renderSubtitle(settings, 1000, 1)
		if err != nil {
			t.Fatal(err)
		}
		if surface.pixels[3] != subtitleStyles[style].background.a {
			t.Fatal("caption backdrop must use straight corners and the preset alpha")
		}
		background := (surface.width/2 + surface.width*3) * 4
		if surface.pixels[background+3] != subtitleStyles[style].background.a {
			t.Fatalf("background alpha does not match %s preset", style)
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
	surface, err := renderSubtitle(settings, 400, 1.5)
	if err != nil {
		t.Fatal(err)
	}
	defer surface.close()
	if surface.width > 400 || surface.height < 150 {
		t.Fatal("narrow, scaled subtitles did not wrap inside the available width")
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
