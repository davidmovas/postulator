package main

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/internal/application/events"
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

type recordingRelay struct {
	seen []events.Type
	sent []events.FilesDroppedPayload
}

func (r *recordingRelay) Publish(eventType events.Type, payload any) error {
	r.seen = append(r.seen, eventType)
	if dropped, ok := payload.(events.FilesDroppedPayload); ok {
		r.sent = append(r.sent, dropped)
	}
	return nil
}

func TestADropIsRelayedAsPaths(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		paths []string
		want  int
	}{
		{name: "nothing dropped", paths: nil},
		{name: "one file", paths: []string{`C:\Users\admin\Downloads\map.xlsx`}, want: 1},
		{name: "several files", paths: []string{`C:\a.csv`, `C:\b.csv`}, want: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			relay := &recordingRelay{}
			publishDrop(relay, tc.paths)

			if len(relay.seen) != tc.want {
				t.Fatalf("published %v, want %d event(s)", relay.seen, tc.want)
			}
			if tc.want == 0 {
				return
			}
			if relay.seen[0] != events.FilesDropped {
				t.Fatalf("published %q, want %q", relay.seen[0], events.FilesDropped)
			}
			if len(relay.sent[0].Paths) != len(tc.paths) {
				t.Fatalf("relayed %v, want %v", relay.sent[0].Paths, tc.paths)
			}
			for index, path := range tc.paths {
				if relay.sent[0].Paths[index] != path {
					t.Fatalf("relayed %v, want %v", relay.sent[0].Paths, tc.paths)
				}
			}
		})
	}
}

func TestTheBuildOpensTheDevtoolsEndpointWhenAsked(t *testing.T) {
	cases := []struct {
		name string
		port string
		want []string
	}{
		{name: "unset changes nothing", port: ""},
		{name: "a port is passed to the webview", port: "9333", want: []string{"--remote-debugging-port=9333"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(devtoolsVariable, tc.port)

			got := options(application.Options{Name: "Postulator"}).Windows.AdditionalBrowserArgs
			if len(got) != len(tc.want) {
				t.Fatalf("AdditionalBrowserArgs = %v, want %v", got, tc.want)
			}
			for index, arg := range tc.want {
				if got[index] != arg {
					t.Fatalf("AdditionalBrowserArgs = %v, want %v", got, tc.want)
				}
			}
		})
	}
}
