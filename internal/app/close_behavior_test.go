package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCloseBehaviorPersistenceAndInvalidData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings", "close.json")
	if readCloseBehavior(path) != closeAsk {
		t.Fatal("first launch must ask")
	}
	for _, action := range []closeBehavior{closeMinimize, closeQuit, closeAsk} {
		if err := writeCloseBehavior(path, action); err != nil {
			t.Fatal(err)
		}
		if readCloseBehavior(path) != action {
			t.Fatalf("did not persist %s", action)
		}
	}
	for _, data := range []string{`broken json`, `{"action":"delete"}`, `{"action":42}`} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if readCloseBehavior(path) != closeAsk {
			t.Fatal("invalid preferences must ask instead of exiting")
		}
	}
	if err := writeCloseBehavior(path, "invalid"); err == nil {
		t.Fatal("accepted an unknown preference")
	}
}

func TestCloseChoiceOnlyRemembersConfirmedCheckedSelection(t *testing.T) {
	for _, test := range []struct {
		name     string
		decision closeDecision
		stored   closeBehavior
	}{
		{"close once", closeDecision{closeQuit, false}, closeAsk},
		{"minimize once", closeDecision{closeMinimize, false}, closeAsk},
		{"remember close", closeDecision{closeQuit, true}, closeQuit},
		{"remember minimize", closeDecision{closeMinimize, true}, closeMinimize},
		{"cancel checked", closeDecision{"", true}, closeAsk},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "close.json")
			action, err := resolveCloseBehavior(path, func() (closeDecision, error) { return test.decision, nil })
			if err != nil || action != test.decision.action {
				t.Fatalf("wrong selection: %s %v", action, err)
			}
			if readCloseBehavior(path) != test.stored {
				t.Fatal("remembered an unchecked or cancelled selection")
			}
			if test.stored != closeAsk {
				_, err := resolveCloseBehavior(path, func() (closeDecision, error) {
					t.Fatal("a remembered action prompted again")
					return closeDecision{}, nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
