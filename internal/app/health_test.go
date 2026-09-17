package app_test

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/app"
)

func TestPingReportsTheVersion(t *testing.T) {
	t.Parallel()

	if got := app.NewHealthService().Ping(); got != app.Version {
		t.Fatalf("Ping() = %q, want %q", got, app.Version)
	}
}

func TestBuildInfoReportsEveryStamp(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  string
		want string
	}{
		{name: "version", got: app.Build().Version, want: app.Version},
		{name: "commit", got: app.Build().Commit, want: app.Commit},
		{name: "buildDate", got: app.Build().BuildDate, want: app.BuildDate},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got != tc.want {
				t.Fatalf("Build().%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestServiceBuildInfoMatchesBuild(t *testing.T) {
	t.Parallel()

	if got := app.NewHealthService().BuildInfo(); got != app.Build() {
		t.Fatalf("BuildInfo() = %+v, want %+v", got, app.Build())
	}
}

func TestStampsHaveSentinelDefaultsWithoutLdflags(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  string
	}{
		{name: "Version", got: app.Version},
		{name: "Commit", got: app.Commit},
		{name: "BuildDate", got: app.BuildDate},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got == "" {
				t.Fatalf("%s is empty; an uninjected stamp must still carry a sentinel", tc.name)
			}
		})
	}

	if app.Version != "2.0.0-dev" {
		t.Fatalf("Version = %q, want the uninjected sentinel 2.0.0-dev", app.Version)
	}
	if app.Commit != "unknown" || app.BuildDate != "unknown" {
		t.Fatalf("Commit/BuildDate = %q/%q, want unknown/unknown", app.Commit, app.BuildDate)
	}
}

func TestBuildInfoMarshalsCamelCase(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(app.BuildInfo{Version: "2.0.0", Commit: "abc1234", BuildDate: "2026-09-17T10:30:00Z"})
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	const want = `{"version":"2.0.0","commit":"abc1234","buildDate":"2026-09-17T10:30:00Z"}`
	if string(encoded) != want {
		t.Fatalf("Marshal() = %s, want %s", encoded, want)
	}
}
