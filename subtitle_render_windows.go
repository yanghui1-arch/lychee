//go:build windows && amd64

package main

import (
	"errors"
	"fmt"
	"math"
	"unsafe"

	"golang.org/x/sys/windows"
)

type subtitleSurface struct {
	dc, bitmap, previous uintptr
	pixels               []byte
	width, height        int
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
	return &subtitleSurface{dc, bitmap, previous, data, width, height}, nil
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

func renderSubtitle(settings SubtitleSettings, maxWidth int, scale float64) (*subtitleSurface, error) {
	// Render glyphs at twice the destination resolution before compositing.
	// Grayscale downsampling avoids jagged edges and ClearType color fringes
	// on the translucent layered window.
	high, err := renderSubtitleGlyphs(settings, maxWidth*2, scale*2)
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
	composeSubtitlePixels(surface.pixels, surface.width, surface.height, scale, subtitleStyles[settings.Style])
	return surface, nil
}

func renderSubtitleGlyphs(settings SubtitleSettings, maxWidth int, scale float64) (*subtitleSurface, error) {
	sizes := [2]int{int(math.Round(float64(settings.FontSize) * scale)), int(math.Round(float64(settings.FontSize) * scale * 0.95))}
	languages := [2]string{settings.Source, settings.Target}
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
	padX, padY, gap := int(36*scale), int(12*scale), int(6*scale)
	textWidth := max(1, maxWidth-padX*2)
	var rects [2]subtitleRect
	var heights [2]int
	width := 0
	for i := range rects {
		rects[i].Right = int32(textWidth)
		heights[i] = drawSubtitleText(dc, fonts[i], subtitlePlaceholderText(languages[i]), &rects[i], true)
		if heights[i] == 0 {
			return nil, errors.New("无法测量字幕文字")
		}
		width = max(width, int(rects[i].Right))
	}
	lineHeight := [2]int{max(heights[0], int(float64(sizes[0])*1.5)), max(heights[1], int(float64(sizes[1])*1.5))}
	surface, err := newSubtitleSurface(min(maxWidth, width+padX*2), lineHeight[0]+lineHeight[1]+gap+padY*2)
	if err != nil {
		return nil, err
	}
	subtitleBackgroundMode.Call(surface.dc, 1)
	subtitleTextColor.Call(surface.dc, 0xFFFFFF)
	y := padY
	for i := range fonts {
		top := y + (lineHeight[i]-heights[i])/2
		rect := subtitleRect{int32(padX), int32(top), int32(surface.width - padX), int32(top + heights[i])}
		if drawSubtitleText(surface.dc, fonts[i], subtitlePlaceholderText(languages[i]), &rect, false) == 0 {
			surface.close()
			return nil, errors.New("无法绘制字幕文字")
		}
		y += lineHeight[i] + gap
	}
	subtitleFlush.Call()
	return surface, nil
}
