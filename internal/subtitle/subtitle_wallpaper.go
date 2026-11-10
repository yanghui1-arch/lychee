package subtitle

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"sync"
	"time"
)

// DesktopBackground describes the user's desktop so the settings preview can
// show captions over what they will actually sit on.
type DesktopBackground struct {
	// Image is a downscaled JPEG data URL, or empty when no image is available.
	Image string `json:"image"`
	// Color is the solid desktop color as #rrggbb, or empty when unknown.
	Color string `json:"color"`
}

// wallpaperPreviewWidth bounds the encoded preview; the workspace preview is
// at most ~860 CSS pixels wide, so this stays sharp on 2x displays.
const wallpaperPreviewWidth = 1600

var wallpaperCache struct {
	sync.Mutex
	path     string
	modified time.Time
	size     int64
	image    string
}

// DesktopBackground returns the current desktop wallpaper for the preview.
func (s *SubtitleService) DesktopBackground() (DesktopBackground, error) {
	background := DesktopBackground{Color: desktopColor()}
	for _, path := range wallpaperPaths() {
		if encoded, err := cachedWallpaper(path); err == nil {
			background.Image = encoded
			break
		}
	}
	return background, nil
}

func cachedWallpaper(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() || info.Size() == 0 {
		return "", fmt.Errorf("wallpaper %q is not a file", path)
	}
	wallpaperCache.Lock()
	defer wallpaperCache.Unlock()
	if wallpaperCache.path == path && wallpaperCache.size == info.Size() && wallpaperCache.modified.Equal(info.ModTime()) {
		return wallpaperCache.image, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	encoded, err := encodeWallpaperPreview(raw, wallpaperPreviewWidth)
	if err != nil {
		return "", err
	}
	wallpaperCache.path, wallpaperCache.size, wallpaperCache.modified = path, info.Size(), info.ModTime()
	wallpaperCache.image = encoded
	return encoded, nil
}

// encodeWallpaperPreview decodes an image, box-downscales it to maxWidth and
// returns it as a JPEG data URL.
func encodeWallpaperPreview(raw []byte, maxWidth int) (string, error) {
	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	scaled := downscale(decoded, maxWidth)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, scaled, &jpeg.Options{Quality: 86}); err != nil {
		return "", err
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(out.Bytes()), nil
}

func downscale(source image.Image, maxWidth int) image.Image {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= maxWidth || width == 0 || height == 0 {
		return source
	}
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(rgba, rgba.Bounds(), source, bounds.Min, draw.Src)
	outWidth := maxWidth
	outHeight := max(1, height*maxWidth/width)
	out := image.NewRGBA(image.Rect(0, 0, outWidth, outHeight))
	for y := range outHeight {
		y0, y1 := y*height/outHeight, max((y+1)*height/outHeight, y*height/outHeight+1)
		for x := range outWidth {
			x0, x1 := x*width/outWidth, max((x+1)*width/outWidth, x*width/outWidth+1)
			var r, g, b, a, count uint32
			for sy := y0; sy < y1; sy++ {
				row := rgba.Pix[sy*rgba.Stride+x0*4 : sy*rgba.Stride+x1*4]
				for i := 0; i < len(row); i += 4 {
					r += uint32(row[i])
					g += uint32(row[i+1])
					b += uint32(row[i+2])
					a += uint32(row[i+3])
					count++
				}
			}
			offset := y*out.Stride + x*4
			out.Pix[offset] = uint8(r / count)
			out.Pix[offset+1] = uint8(g / count)
			out.Pix[offset+2] = uint8(b / count)
			out.Pix[offset+3] = uint8(a / count)
		}
	}
	return out
}
