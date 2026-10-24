package render_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/render"
)

func TestExtractSaysWhetherTheThemePrintsTheH1(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   func(t *testing.T) render.Input
		want render.ThemeH1
	}{
		{
			name: "the replica's hub with an empty raw",
			in:   func(t *testing.T) render.Input { return standInput(t, replica, "peptides", "/peptides/") },
			want: render.ThemeH1Printed,
		},
		{
			name: "the replica's page with a shortcode raw",
			in:   func(t *testing.T) render.Input { return standInput(t, replica, "bpc-157", "/bpc-157/") },
			want: render.ThemeH1Printed,
		},
		{
			name: "the replica's page with text of its own",
			in:   func(t *testing.T) render.Input { return standInput(t, replica, "tb-500", "/tb-500/") },
			want: render.ThemeH1Printed,
		},
		{
			name: "vanilla's block theme prints the post title",
			in:   func(t *testing.T) render.Input { return standInput(t, vanilla, "guide", "/guide/") },
			want: render.ThemeH1Printed,
		},
		{
			name: "a classic theme whose title shares the content's wrapper",
			in: func(*testing.T) render.Input {
				return render.Input{
					Public:   `<html><body><article><h1 class="entry-title">Lab notes</h1><p>Reconstitute slowly along the wall of the vial.</p><p>Swirl gently and never shake it.</p></article></body></html>`,
					Filtered: "<p>Reconstitute slowly along the wall of the vial.</p>\n<p>Swirl gently and never shake it.</p>",
					Body:     block.Parse("Reconstitute slowly along the wall of the vial.\n\nSwirl gently and never shake it."),
				}
			},
			want: render.ThemeH1Printed,
		},
		{
			name: "a theme that prints no heading",
			in: func(*testing.T) render.Input {
				return render.Input{
					Public:   `<html><body><main><div class="entry"><p>Reconstitute slowly along the wall of the vial.</p><p>Swirl gently and never shake it.</p></div></main></body></html>`,
					Filtered: "<p>Reconstitute slowly along the wall of the vial.</p>\n<p>Swirl gently and never shake it.</p>",
					Body:     block.Parse("Reconstitute slowly along the wall of the vial.\n\nSwirl gently and never shake it."),
				}
			},
			want: render.ThemeH1Absent,
		},
		{
			name: "the page's own heading is no theme's",
			in: func(*testing.T) render.Input {
				return render.Input{
					Public:   `<html><body><main><div class="entry"><h1>Lab notes</h1><p>Reconstitute slowly along the wall of the vial.</p></div></main></body></html>`,
					Filtered: "<h1>Lab notes</h1>\n<p>Reconstitute slowly along the wall of the vial.</p>",
					Body:     block.Parse("<h1>Lab notes</h1>\n\nReconstitute slowly along the wall of the vial."),
				}
			},
			want: render.ThemeH1Absent,
		},
		{
			name: "a root only assumed says nothing",
			in: func(*testing.T) render.Input {
				return render.Input{
					Public:   `<html><body><main><h1>Elsewhere</h1><p>Nothing of the raw is shown here.</p></main></body></html>`,
					Filtered: "<p>Reconstitute slowly along the wall of the vial.</p>",
					Body:     block.Parse("Reconstitute slowly along the wall of the vial."),
				}
			},
			want: render.ThemeH1Unknown,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := synthetic(t, tc.in(t)).ThemeH1; got != tc.want {
				t.Errorf("theme h1 %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOurH1IsOwedOnlyWhereTheThemePrintsNone(t *testing.T) {
	t.Parallel()

	record := func(h1 render.ThemeH1) render.Record {
		return render.Record{Extraction: render.Extraction{ThemeH1: h1}}
	}
	absent, printed, unknown := record(render.ThemeH1Absent), record(render.ThemeH1Printed), record(render.ThemeH1Unknown)
	cases := []struct {
		name     string
		own      *render.Record
		sameType []render.Record
		want     bool
	}{
		{name: "an existing page whose own render prints none", own: &absent, sameType: []render.Record{printed}, want: true},
		{name: "an existing page whose theme prints the title", own: &printed, sameType: []render.Record{absent}, want: false},
		{name: "an existing page whose root was only assumed", own: &unknown, sameType: []render.Record{absent}, want: false},
		{name: "a created page among pages that print none", sameType: []render.Record{absent, unknown, absent}, want: true},
		{name: "a created page where one page of its type prints the title", sameType: []render.Record{absent, printed}, want: false},
		{name: "a created page with nothing observed", want: false},
		{name: "a created page whose type was never located", sameType: []render.Record{unknown}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := render.OwesH1(tc.own, tc.sameType); got != tc.want {
				t.Errorf("owes h1 = %v, want %v", got, tc.want)
			}
		})
	}
}
