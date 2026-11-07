package app

import "testing"

func TestFloatingUpdatesCannotAffectWorkspaceOrUnshownWindow(t *testing.T) {
	for _, service := range []*WindowService{
		{workspace: true, shown: true},
		{},
	} {
		if err := service.SetFloatingExpanded(true, 1); err != nil {
			t.Fatal(err)
		}
		if service.expanded {
			t.Fatal("a stale hover update changed a workspace or unshown window")
		}
	}
}

func TestLateHoverCallbackCannotHideNewerToolbar(t *testing.T) {
	service := &WindowService{shown: true, expanded: true, hoverRevision: 4}
	if err := service.SetFloatingExpanded(false, 3); err != nil {
		t.Fatal(err)
	}
	if !service.expanded || service.hoverRevision != 4 {
		t.Fatal("an outdated leave callback changed the current hover state")
	}
}
