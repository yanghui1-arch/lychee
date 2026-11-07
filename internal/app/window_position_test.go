package app

import (
	"path/filepath"
	"testing"
)

func TestFloatingPositionPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lychee", "floating-position.json")
	if readFloatingPosition(path) != nil {
		t.Fatal("first launch should use the default position")
	}
	for _, x := range []int{1000, 1200} {
		if err := writeFloatingPosition(path, x, 270); err != nil {
			t.Fatal(err)
		}
		if got := readFloatingPosition(path); len(got) != 2 || got[0] != x || got[1] != 270 {
			t.Fatalf("saved position was not restored: %v", got)
		}
	}
}
