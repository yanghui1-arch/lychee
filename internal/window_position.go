package app

import (
	"encoding/json"
	"os"
	"path/filepath"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func readFloatingPosition(path string) []int {
	data, _ := os.ReadFile(path)
	var position []int
	if json.Unmarshal(data, &position) != nil || len(position) != 2 {
		return nil
	}
	return position
}

func writeFloatingPosition(path string, x, y int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, _ := json.Marshal([]int{x, y})
	// Replace only after writing succeeds, preserving the previous position.
	temporary := path + ".tmp"
	defer os.Remove(temporary)
	if err := os.WriteFile(temporary, data, 0600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func (s *WindowService) restoreFloatingPosition() error {
	position := readFloatingPosition(s.floatingPositionPath)
	screens, err := wailsruntime.ScreenGetAll(s.ctx)
	if err != nil {
		return err
	}
	if len(screens) == 0 {
		return nil
	}
	screen := screens[0]
	scale := float64(screen.PhysicalSize.Width) / float64(screen.Size.Width)
	// A saved location may be outside the screen after a resolution change.
	// Validate the visible ball, allowing its transparent toolbar margin off-screen.
	if position == nil ||
		position[0]+int(float64(floatingOffsetX)*scale) < 0 ||
		position[1]+int(float64(floatingOffsetY)*scale) < 0 ||
		position[0]+int(float64(floatingWidth)*scale) > screen.PhysicalSize.Width ||
		position[1]+int(float64(floatingHeight)*scale) > screen.PhysicalSize.Height {
		position = []int{
			screen.PhysicalSize.Width*3/4 - int(float64(floatingOffsetX+floatingSize/2)*scale),
			screen.PhysicalSize.Height/4 - int(float64(floatingOffsetY+floatingSize/2)*scale),
		}
	}
	return moveFloatingWindow(position[0], position[1])
}

func (s *WindowService) saveFloatingPositionLocked() error {
	if s.screenshot.Active() || !s.shown || s.workspace || wailsruntime.WindowIsMinimised(s.ctx) {
		return nil
	}
	x, y := wailsruntime.WindowGetPosition(s.ctx)
	return writeFloatingPosition(s.floatingPositionPath, x, y)
}
