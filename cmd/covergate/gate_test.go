package main

import (
	"math"
	"strings"
	"testing"
)

const module = "github.com/davidmovas/postulator"

func profileText(lines ...string) string {
	return "mode: atomic\n" + strings.Join(lines, "\n") + "\n"
}

func TestParse(t *testing.T) {
	t.Parallel()

	raw := profileText(
		module+"/internal/domain/site/site.go:10.20,12.3 2 1",
		module+"/internal/domain/site/site.go:14.2,16.4 3 0",
		module+"/internal/kernel/id/id.go:5.1,6.2 1 7",
	)

	profile, err := parse(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("parse() error: %v", err)
	}
	if len(profile.blocks) != 3 {
		t.Fatalf("parsed %d blocks, want 3", len(profile.blocks))
	}
	if profile.blocks[0].statements != 2 || !profile.blocks[0].covered {
		t.Fatalf("first block = %+v", profile.blocks[0])
	}
	if profile.blocks[1].covered {
		t.Fatal("a block with count 0 must not count as covered")
	}
}

func TestParseRejectsMalformedProfiles(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
	}{
		{name: "empty", raw: ""},
		{name: "no mode line", raw: module + "/a.go:1.1,2.2 1 1\n"},
		{name: "missing fields", raw: profileText(module + "/a.go:1.1,2.2 1")},
		{name: "statements not a number", raw: profileText(module + "/a.go:1.1,2.2 many 1")},
		{name: "count not a number", raw: profileText(module + "/a.go:1.1,2.2 1 lots")},
		{name: "no file separator", raw: profileText("1.1,2.2 1 1")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parse(strings.NewReader(tc.raw)); err == nil {
				t.Fatal("parse() must reject a malformed profile")
			}
		})
	}
}

func TestParseIgnoresBlankLines(t *testing.T) {
	t.Parallel()

	raw := "mode: atomic\n\n" + module + "/a.go:1.1,2.2 4 1\n\n"
	profile, err := parse(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("parse() error: %v", err)
	}
	if len(profile.blocks) != 1 {
		t.Fatalf("parsed %d blocks, want 1", len(profile.blocks))
	}
}

func TestRate(t *testing.T) {
	t.Parallel()

	profile, err := parse(strings.NewReader(profileText(
		module+"/internal/domain/site/site.go:1.1,2.2 6 1",
		module+"/internal/domain/site/site.go:3.1,4.2 4 0",
		module+"/internal/application/sites/create.go:1.1,2.2 5 1",
		module+"/internal/application/sites/create.go:3.1,4.2 5 0",
		module+"/internal/kernel/id/id.go:1.1,2.2 10 1",
	)))
	if err != nil {
		t.Fatalf("parse() error: %v", err)
	}

	cases := []struct {
		name       string
		prefixes   []string
		statements int
		covered    int
		percent    float64
	}{
		{name: "everything", prefixes: nil, statements: 30, covered: 21, percent: 70},
		{name: "domain only", prefixes: []string{"internal/domain/"}, statements: 10, covered: 6, percent: 60},
		{name: "domain and application", prefixes: []string{"internal/domain/", "internal/application/"}, statements: 20, covered: 11, percent: 55},
		{name: "nothing matches", prefixes: []string{"internal/runtime/"}, statements: 0, covered: 0, percent: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := profile.rate(tc.prefixes)
			if got.statements != tc.statements || got.covered != tc.covered {
				t.Fatalf("rate() = %+v, want %d/%d", got, tc.covered, tc.statements)
			}
			if math.Abs(got.percent()-tc.percent) > 1e-9 {
				t.Fatalf("percent() = %.2f, want %.2f", got.percent(), tc.percent)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		lines    []string
		wantPass bool
		wantSkip bool
	}{
		{
			name: "both gates met",
			lines: []string{
				module + "/internal/domain/site/site.go:1.1,2.2 9 1",
				module + "/internal/domain/site/site.go:3.1,4.2 1 0",
				module + "/internal/kernel/id/id.go:1.1,2.2 9 1",
				module + "/internal/kernel/id/id.go:3.1,4.2 1 0",
			},
			wantPass: true,
		},
		{
			name: "core below its gate",
			lines: []string{
				module + "/internal/domain/site/site.go:1.1,2.2 7 1",
				module + "/internal/domain/site/site.go:3.1,4.2 3 0",
				module + "/internal/kernel/id/id.go:1.1,2.2 10 1",
			},
			wantPass: false,
		},
		{
			name: "total below its gate",
			lines: []string{
				module + "/internal/domain/site/site.go:1.1,2.2 10 1",
				module + "/internal/kernel/id/id.go:1.1,2.2 1 1",
				module + "/internal/kernel/id/id.go:3.1,4.2 19 0",
			},
			wantPass: false,
		},
		{
			name: "core gate is skipped while the tree is empty",
			lines: []string{
				module + "/internal/kernel/id/id.go:1.1,2.2 8 1",
				module + "/internal/kernel/id/id.go:3.1,4.2 2 0",
			},
			wantPass: true,
			wantSkip: true,
		},
		{
			name:     "an empty profile passes nothing and fails nothing",
			lines:    []string{},
			wantPass: true,
			wantSkip: true,
		},
	}

	thresholds := gates{core: 80, total: 70}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			raw := "mode: atomic\n"
			if len(tc.lines) > 0 {
				raw = profileText(tc.lines...)
			}
			profile, err := parse(strings.NewReader(raw))
			if err != nil {
				t.Fatalf("parse() error: %v", err)
			}

			report := evaluate(profile, thresholds)
			if report.pass() != tc.wantPass {
				t.Fatalf("pass() = %v, want %v: %s", report.pass(), tc.wantPass, report)
			}
			if report.core.skipped != tc.wantSkip {
				t.Fatalf("core skipped = %v, want %v", report.core.skipped, tc.wantSkip)
			}
		})
	}
}

func TestReportRendersEveryGate(t *testing.T) {
	t.Parallel()

	profile, err := parse(strings.NewReader(profileText(
		module+"/internal/domain/site/site.go:1.1,2.2 8 1",
		module+"/internal/domain/site/site.go:3.1,4.2 2 0",
	)))
	if err != nil {
		t.Fatalf("parse() error: %v", err)
	}

	rendered := evaluate(profile, gates{core: 80, total: 70}).String()
	for _, want := range []string{"domain+application", "total", "80.0", "80.00%"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("report %q does not mention %q", rendered, want)
		}
	}
}

func TestSkippedGateIsRendered(t *testing.T) {
	t.Parallel()

	profile, err := parse(strings.NewReader(profileText(module + "/internal/kernel/id/id.go:1.1,2.2 1 1")))
	if err != nil {
		t.Fatalf("parse() error: %v", err)
	}

	rendered := evaluate(profile, gates{core: 80, total: 70}).String()
	if !strings.Contains(rendered, "no statements") {
		t.Fatalf("report %q does not explain the skipped gate", rendered)
	}
}
