//go:build !windows || !amd64

package app

func prepareFloatingWindow() error         { return nil }
func setFloatingRegion(bool) error         { return nil }
func clearFloatingRegion() error           { return nil }
func moveFloatingWindow(int, int) error    { return nil }
func (s *WindowService) startTray() func() { return func() {} }
