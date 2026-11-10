//go:build windows && amd64

package subtitle

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Logical canvas dimensions are independent of text and font size. Keep the
// settings preview in subtitle.css at the same 0.46 scale.
const subtitleCanvasWidth = 1000
const subtitleCanvasHeight = 240

type subtitleSurface struct {
	dc, bitmap, previous uintptr
	pixels               []byte
	width, height        int
	inkRegions           []subtitleInkRegion
}

func (s *subtitleSurface) close() {
	subtitleSelectObject.Call(s.dc, s.previous)
	subtitleDeleteObject.Call(s.bitmap)
	subtitleDeleteDC.Call(s.dc)
}

func newSubtitleSurface(width, height int) (*subtitleSurface, error) {
	dc, _, err := subtitleCreateDC.Call(0)
	if dc == 0 {
		return nil, err
	}
	info := subtitleBitmapInfo{Size: 40, Width: int32(width), Height: -int32(height), Planes: 1, Bits: 32}
	var pixels unsafe.Pointer
	bitmap, _, err := subtitleCreateBitmap.Call(dc, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&pixels)), 0, 0)
	if bitmap == 0 || pixels == nil {
		if bitmap != 0 {
			subtitleDeleteObject.Call(bitmap)
		}
		subtitleDeleteDC.Call(dc)
		return nil, fmt.Errorf("创建字幕画布失败：%w", err)
	}
	previous, _, _ := subtitleSelectObject.Call(dc, bitmap)
	if previous == 0 || previous == ^uintptr(0) {
		subtitleDeleteObject.Call(bitmap)
		subtitleDeleteDC.Call(dc)
		return nil, errors.New("无法选择字幕画布")
	}
	data := unsafe.Slice((*byte)(pixels), width*height*4)
	clear(data)
	return &subtitleSurface{dc: dc, bitmap: bitmap, previous: previous, pixels: data, width: width, height: height}, nil
}

func subtitleFont(size int, language string) (uintptr, error) {
	face, weight := "Roboto Medium", 500
	switch language {
	case "zh-CN", "zh-TW":
		face, weight = "Microsoft YaHei UI", 400
	case "ja-JP":
		face, weight = "Yu Gothic UI", 400
	case "ko-KR":
		face, weight = "Malgun Gothic", 400
	default:
		if err := loadSubtitleRoboto(); err != nil {
			return 0, err
		}
	}
	name, _ := windows.UTF16PtrFromString(face)
	font, _, err := subtitleCreateFont.Call(uintptr(-int32(size)), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 4, 0, uintptr(unsafe.Pointer(name)))
	if font == 0 {
		return 0, fmt.Errorf("创建字幕字体失败：%w", err)
	}
	return font, nil
}

func drawSubtitleText(dc, font uintptr, text string, rect *subtitleRect, measure bool) int {
	previous, _, _ := subtitleSelectObject.Call(dc, font)
	defer subtitleSelectObject.Call(dc, previous)
	value, _ := windows.UTF16FromString(text)
	flags := uintptr(0x10 | 0x1 | 0x800) // WORDBREAK | CENTER | NOPREFIX.
	if measure {
		flags |= 0x400 // CALCRECT.
	}
	result, _, _ := subtitleDrawText.Call(dc, uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)-1), uintptr(unsafe.Pointer(rect)), flags)
	return int(result)
}

func renderSubtitle(settings SubtitleSettings, maxWidth int, scale float64, frame SubtitleFrame) (*subtitleSurface, error) {
	// Render glyphs at twice the destination resolution before compositing.
	// Grayscale downsampling avoids jagged edges and ClearType color fringes
	// on the translucent layered window.
	high, err := renderSubtitleGlyphs(settings, maxWidth*2, scale*2, frame)
	if err != nil {
		return nil, err
	}
	defer high.close()
	surface, err := newSubtitleSurface((high.width+1)/2, (high.height+1)/2)
	if err != nil {
		return nil, err
	}
	for y := 0; y < surface.height; y++ {
		for x := 0; x < surface.width; x++ {
			sum := 0
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					px, py := x*2+dx, y*2+dy
					if px < high.width && py < high.height {
						i := (py*high.width + px) * 4
						sum += int(max(high.pixels[i], high.pixels[i+1], high.pixels[i+2]))
					}
				}
			}
			value := byte((sum + 2) / 4)
			i := (y*surface.width + x) * 4
			surface.pixels[i], surface.pixels[i+1], surface.pixels[i+2] = value, value, value
		}
	}
	for _, region := range high.inkRegions {
		surface.inkRegions = append(surface.inkRegions, subtitleInkRegion{region.left / 2, region.top / 2, (region.right + 1) / 2, (region.bottom + 1) / 2})
	}
	composeSubtitlePixels(surface.pixels, surface.width, surface.height, float64(settings.FontSize)*scale, surface.inkRegions, subtitleStyles[settings.Style])
	return surface, nil
}

func renderSubtitleGlyphs(settings SubtitleSettings, maxWidth int, scale float64, frame SubtitleFrame) (*subtitleSurface, error) {
	sizes := [2]int{int(math.Round(float64(settings.FontSize) * scale)), int(math.Round(float64(settings.FontSize) * scale * 0.95))}
	languages := [2]string{settings.Source, settings.Target}
	texts := [2]string{frame.Source, frame.Translation}
	var fonts [2]uintptr
	for i := range fonts {
		font, err := subtitleFont(sizes[i], languages[i])
		if err != nil {
			return nil, err
		}
		fonts[i] = font
		defer subtitleDeleteObject.Call(font)
	}
	dc, _, err := subtitleCreateDC.Call(0)
	if dc == 0 {
		return nil, err
	}
	defer subtitleDeleteDC.Call(dc)
	width := min(maxWidth, int(math.Round(subtitleCanvasWidth*scale)))
	height := int(math.Round(subtitleCanvasHeight * scale))
	padX, padY, gap := min(int(36*scale), width/8), int(18*scale), int(12*scale)
	textWidth := max(1, width-padX*2)
	contentHeight := height - padY*2
	rowHeight := (contentHeight - gap) / 2
	bilingual := texts[0] != "" && texts[1] != ""
	var heights [2]int
	for i := range texts {
		if texts[i] == "" {
			continue
		}
		if strings.Contains(frame.Key, ":page:") {
			var fits bool
			heights[i], fits = measureSubtitlePageText(dc, fonts[i], texts[i], textWidth, rowHeight)
			if !fits {
				return nil, errSubtitleNeedsReflow
			}
		} else {
			texts[i], heights[i] = fitSubtitleText(dc, fonts[i], texts[i], textWidth, rowHeight)
		}
		if heights[i] == 0 {
			return nil, errors.New("无法测量字幕文字")
		}
	}
	surface, err := newSubtitleSurface(width, height)
	if err != nil {
		return nil, err
	}
	subtitleBackgroundMode.Call(surface.dc, 1)
	subtitleTextColor.Call(surface.dc, 0xFFFFFF)
	for i := range fonts {
		if texts[i] == "" {
			continue
		}
		// Anchor source to the bottom of its row and translation to the top:
		// short pairs stay close together without changing the window geometry.
		top := padY + rowHeight - heights[i]
		if i == 1 {
			top = padY + rowHeight + gap
		}
		if !bilingual {
			top = padY + (contentHeight-heights[i])/2
		}
		rect := subtitleRect{int32(padX), int32(top), int32(surface.width - padX), int32(top + heights[i])}
		if drawSubtitleText(surface.dc, fonts[i], texts[i], &rect, false) == 0 {
			surface.close()
			return nil, errors.New("无法绘制字幕文字")
		}
		surface.inkRegions = append(surface.inkRegions, subtitleInkRegion{int(rect.Left), int(rect.Top), int(rect.Right), int(rect.Bottom)})
	}
	subtitleFlush.Call()
	return surface, nil
}

func measureSubtitlePageText(dc, font uintptr, text string, width, maxHeight int) (int, bool) {
	lineRect := subtitleRect{Right: int32(width)}
	lineHeight := drawSubtitleText(dc, font, "Ag", &lineRect, true)
	rect := subtitleRect{Right: int32(width)}
	height := drawSubtitleText(dc, font, text, &rect, true)
	return height, height > 0 && height <= min(maxHeight, lineHeight) && int(rect.Right) <= width
}

func paginateNativeSubtitle(settings SubtitleSettings, source, translation string, available int, scale float64) (SubtitlePage, error) {
	// Match the renderer's supersampled geometry exactly.
	scale *= 2
	width := min(available*2, int(math.Round(subtitleCanvasWidth*scale)))
	padX := min(int(36*scale), width/8)
	rowHeight := (int(math.Round(subtitleCanvasHeight*scale)) - int(36*scale) - int(12*scale)) / 2
	dc, _, err := subtitleCreateDC.Call(0)
	if dc == 0 {
		return SubtitlePage{}, err
	}
	defer subtitleDeleteDC.Call(dc)
	var fonts [2]uintptr
	for i, language := range []string{settings.Source, settings.Target} {
		size := float64(settings.FontSize) * scale
		if i == 1 {
			size *= 0.95
		}
		font, err := subtitleFont(int(math.Round(size)), language)
		if err != nil {
			return SubtitlePage{}, err
		}
		fonts[i] = font
		defer subtitleDeleteObject.Call(font)
	}
	return splitSubtitlePage(source, translation, func(i int, text string) bool {
		_, fits := measureSubtitlePageText(dc, fonts[i], text, width-padX*2, rowHeight)
		return fits
	})
}

// Fit the latest text to one actual font line. Measuring the real
// glyphs also handles narrow monitors, large fonts, CJK, and unbroken words.
func fitSubtitleText(dc, font uintptr, text string, width, maxHeight int) (string, int) {
	lineRect := subtitleRect{Right: int32(width)}
	lineHeight := drawSubtitleText(dc, font, "Ag", &lineRect, true)
	limit := min(maxHeight, lineHeight)
	measure := func(value string) (int, bool) {
		rect := subtitleRect{Right: int32(width)}
		height := drawSubtitleText(dc, font, value, &rect, true)
		return height, height > 0 && height <= limit && int(rect.Right) <= width
	}
	text = strings.TrimSpace(text)
	if height, fits := measure(text); fits {
		return text, height
	}
	runes := []rune(text)
	low, high := 0, len(runes)
	for low < high {
		middle := (low + high) / 2
		if _, fits := measure("…" + strings.TrimSpace(string(runes[middle:]))); fits {
			high = middle
		} else {
			low = middle + 1
		}
	}
	text = "…" + strings.TrimSpace(string(runes[low:]))
	height, _ := measure(text)
	return text, min(height, limit)
}
