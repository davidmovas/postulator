package main

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type stubWindow struct {
	steps []string
}

func (w *stubWindow) Restore() {
	w.steps = append(w.steps, "restore")
}

func (w *stubWindow) Show() application.Window {
	w.steps = append(w.steps, "show")
	return nil
}

func (w *stubWindow) Focus() {
	w.steps = append(w.steps, "focus")
}

func TestRaiseBringsAWindowBack(t *testing.T) {
	t.Parallel()

	window := &stubWindow{}
	raise(window)

	want := []string{"restore", "show", "focus"}
	if len(window.steps) != len(want) {
		t.Fatalf("raise did %v, want %v", window.steps, want)
	}
	for index, step := range want {
		if window.steps[index] != step {
			t.Fatalf("raise did %v, want %v", window.steps, want)
		}
	}
}

func TestRaiseSurvivesHavingNoWindow(t *testing.T) {
	t.Parallel()

	raise(nil)
}

func TestFirstWindowAnswersNothingWithoutAnApplication(t *testing.T) {
	t.Parallel()

	if found := firstWindow(); found != nil {
		t.Fatalf("firstWindow = %v, want nothing outside a running application", found)
	}
}

func TestTheSecondInstanceCallbackIsWired(t *testing.T) {
	t.Parallel()

	only := onlyInstance("com.example.test")
	if only.UniqueID != "com.example.test" {
		t.Fatalf("UniqueID = %q", only.UniqueID)
	}
	if only.OnSecondInstanceLaunch == nil {
		t.Fatal("a second launch has nothing to call")
	}
	only.OnSecondInstanceLaunch(application.SecondInstanceData{})
}

func TestTheProductionBuildTakesTheStableId(t *testing.T) {
	only := options(application.Options{Name: "Postulator"}).SingleInstance
	if only == nil {
		t.Fatal("the build declares no single instance")
	}
	if only.UniqueID == "" {
		t.Fatal("the single instance has no id, so a second launch would not be noticed")
	}
}
