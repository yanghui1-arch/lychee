package main

import "math"

// composeSubtitlePixels keeps the picture visible through a light or absent
// backdrop. An outline and soft shadow preserve glyph contrast on moving video.
// UpdateLayeredWindow requires all output colors to be premultiplied by alpha.
func composeSubtitlePixels(pixels []byte, width, height int, scale float64, style subtitleStyle) {
	mask := make([]byte, width*height)
	for i := range mask {
		mask[i] = max(pixels[i*4], pixels[i*4+1], pixels[i*4+2])
	}
	strokeRadius := 1.2 * scale
	if style.background.a >= 128 {
		strokeRadius = 0.55 * scale
	}
	outline := outlineSubtitleMask(mask, width, height, strokeRadius)
	shadow := blurSubtitleMask(mask, width, height, max(1, int(math.Round(2*scale))))
	shadowOffset := max(1, int(math.Round(scale)))
	backgroundAlpha := float64(style.background.a) / 255
	// Channels are BGRA in a Windows DIB. Each column is a compositing layer.
	channels := [3][3]byte{
		{style.background.b, style.outline.b, style.foreground.b},
		{style.background.g, style.outline.g, style.foreground.g},
		{style.background.r, style.outline.r, style.foreground.r},
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := y*width + x
			i := index * 4
			shadowAlpha := 0.0
			if y >= shadowOffset {
				shadowAlpha = float64(shadow[(y-shadowOffset)*width+x]) / 255 * float64(style.outline.a) / 255 * 0.55
			}
			outlineAlpha := float64(outline[index]) / 255 * float64(style.outline.a) / 255
			foregroundAlpha := float64(mask[index]) / 255
			shadowCompositeAlpha := shadowAlpha + backgroundAlpha*(1-shadowAlpha)
			outlineCompositeAlpha := outlineAlpha + shadowCompositeAlpha*(1-outlineAlpha)
			alpha := foregroundAlpha + outlineCompositeAlpha*(1-foregroundAlpha)
			for channel, colors := range channels {
				shadowColor := float64(colors[1])*shadowAlpha + float64(colors[0])*backgroundAlpha*(1-shadowAlpha)
				outlineColor := float64(colors[1])*outlineAlpha + shadowColor*(1-outlineAlpha)
				color := float64(colors[2])*foregroundAlpha + outlineColor*(1-foregroundAlpha)
				pixels[i+channel] = byte(math.Round(color))
			}
			pixels[i+3] = byte(math.Round(alpha * 255))
		}
	}
}

func outlineSubtitleMask(mask []byte, width, height int, radius float64) []byte {
	type sample struct {
		x, y   int
		weight float64
	}
	var kernel []sample
	limit := int(math.Ceil(radius))
	for y := -limit; y <= limit; y++ {
		for x := -limit; x <= limit; x++ {
			weight := math.Max(0, math.Min(1, radius+0.5-math.Hypot(float64(x), float64(y))))
			if weight > 0 {
				kernel = append(kernel, sample{x, y, weight})
			}
		}
	}
	outline := make([]byte, len(mask))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := 0.0
			for _, sample := range kernel {
				px, py := x+sample.x, y+sample.y
				if px >= 0 && px < width && py >= 0 && py < height {
					value = math.Max(value, float64(mask[py*width+px])*sample.weight)
				}
			}
			outline[y*width+x] = byte(math.Round(value))
		}
	}
	return outline
}

// blurSubtitleMask uses separable running sums, keeping shadow work linear in
// the bitmap size even on monitors with high DPI scaling.
func blurSubtitleMask(mask []byte, width, height, radius int) []byte {
	horizontal, blurred := make([]byte, len(mask)), make([]byte, len(mask))
	window := radius*2 + 1
	for y := 0; y < height; y++ {
		sum := 0
		for x := 0; x <= min(radius, width-1); x++ {
			sum += int(mask[y*width+x])
		}
		for x := 0; x < width; x++ {
			horizontal[y*width+x] = byte(sum / window)
			if x-radius >= 0 {
				sum -= int(mask[y*width+x-radius])
			}
			if x+radius+1 < width {
				sum += int(mask[y*width+x+radius+1])
			}
		}
	}
	for x := 0; x < width; x++ {
		sum := 0
		for y := 0; y <= min(radius, height-1); y++ {
			sum += int(horizontal[y*width+x])
		}
		for y := 0; y < height; y++ {
			blurred[y*width+x] = byte(sum / window)
			if y-radius >= 0 {
				sum -= int(horizontal[(y-radius)*width+x])
			}
			if y+radius+1 < height {
				sum += int(horizontal[(y+radius+1)*width+x])
			}
		}
	}
	return blurred
}
