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

func TestExtractRecordsTheVisibleH1OfEveryCapture(t *testing.T) {
	t.Parallel()

	cases := []struct {
		stand, slug, document string
		archive               bool
		want                  string
		source                render.H1Source
	}{
		{stand: replica, slug: "peptides", document: "/peptides/", want: "Peptides", source: render.H1Theme},
		{stand: replica, slug: "bpc-157", document: "/bpc-157/", want: "BPC-157", source: render.H1Theme},
		{stand: replica, slug: "tb-500", document: "/tb-500/", want: "TB-500", source: render.H1Theme},
		{stand: replica, slug: "hello-world", document: "/hello-world/", want: "Hello world!", source: render.H1Theme},
		{stand: replica, slug: "cart", document: "/cart/", want: "Cart", source: render.H1Theme},
		{stand: replica, slug: "my-account", document: "/my-account/", want: "My account", source: render.H1Theme},
		{stand: replica, slug: "bpc-157-peptide", document: "/product/bpc-157-peptide/", want: "BPC-157 Peptide", source: render.H1Theme},
		{stand: replica, slug: "shop", document: "/shop/", archive: true, want: "Shop", source: render.H1Theme},
		{stand: replica, slug: "healing-peptides", document: "/product-category/peptides/healing-peptides/", archive: true, want: "Healing Peptides", source: render.H1Theme},
		{stand: vanilla, slug: "guide", document: "/guide/", want: "Guide", source: render.H1Theme},
		{stand: vanilla, slug: "setup", document: "/guide/setup/", want: "Setup", source: render.H1Theme},
		{stand: vanilla, slug: "powder", document: "/product/powder/", want: "Powder", source: render.H1Theme},
		{stand: vanilla, slug: "shop", document: "/shop/", archive: true, want: "Shop", source: render.H1Theme},
		{stand: vanilla, slug: "koffein", document: "/product-category/koffein/", archive: true, want: "Koffein", source: render.H1Theme},
	}
	for _, tc := range cases {
		t.Run(tc.stand+"/"+tc.slug, func(t *testing.T) {
			t.Parallel()

			in := standInput(t, tc.stand, tc.slug, tc.document)
			in.Archive = tc.archive
			got := synthetic(t, in)
			if got.H1 != tc.want || got.H1Source != tc.source {
				t.Errorf("h1 %q from %q, want %q from %q", got.H1, got.H1Source, tc.want, tc.source)
			}
		})
	}
}

func TestExtractTellsTheH1sSource(t *testing.T) {
	t.Parallel()

	const raw = "Reconstitute slowly along the wall of the vial.\n\nSwirl gently and never shake it."
	const filtered = "<p>Reconstitute slowly along the wall of the vial.</p>\n<p>Swirl gently and never shake it.</p>"
	const paragraphs = `<p>Reconstitute slowly along the wall of the vial.</p><p>Swirl gently and never shake it.</p>`
	cases := []struct {
		name   string
		in     render.Input
		want   string
		source render.H1Source
	}{
		{
			name: "the page's own heading inside the content",
			in: render.Input{
				Public:   `<html><body><main><div class="entry"><h1>Lab <em>notes</em></h1><p>Reconstitute slowly along the wall of the vial.</p></div></main></body></html>`,
				Filtered: "<h1>Lab <em>notes</em></h1>\n<p>Reconstitute slowly along the wall of the vial.</p>",
				Body:     block.Parse("<h1>Lab <em>notes</em></h1>\n\nReconstitute slowly along the wall of the vial."),
			},
			want: "Lab notes", source: render.H1Content,
		},
		{
			name: "a title the theme prints before the content",
			in: render.Input{
				Public: `<html><body><main><header class="entry-header"><h1 class="page-title">Lab notes</h1></header><div class="entry">` + paragraphs + `</div></main></body></html>`,
				Filtered: filtered, Body: block.Parse(raw),
			},
			want: "Lab notes", source: render.H1Theme,
		},
		{
			name: "the site's name in the banner is not the page's heading",
			in: render.Input{
				Public: `<html><body><header class="site-header"><h1 class="site-title">Peptide Lab</h1><nav><h1>Menu</h1></nav></header><main><h1 class="page-title">Lab notes</h1><div class="entry">` + paragraphs + `</div></main><aside><h1>Recent</h1></aside><footer><h1>Contact</h1></footer></body></html>`,
				Filtered: filtered, Body: block.Parse(raw),
			},
			want: "Lab notes", source: render.H1Theme,
		},
		{
			name: "a theme that prints no heading",
			in: render.Input{
				Public:   `<html><body><main><div class="entry">` + paragraphs + `</div></main></body></html>`,
				Filtered: filtered, Body: block.Parse(raw),
			},
			want: "", source: render.H1None,
		},
		{
			name: "only the banner holds an h1",
			in: render.Input{
				Public:   `<html><body><div role="banner"><h1>Peptide Lab</h1></div><main><div class="entry">` + paragraphs + `</div></main></body></html>`,
				Filtered: filtered, Body: block.Parse(raw),
			},
			want: "", source: render.H1None,
		},
		{
			name: "an empty heading is skipped",
			in: render.Input{
				Public:   `<html><body><main><h1> </h1><h1>Lab notes</h1><div class="entry">` + paragraphs + `</div></main></body></html>`,
				Filtered: filtered, Body: block.Parse(raw),
			},
			want: "Lab notes", source: render.H1Theme,
		},
		{
			name: "an empty render shows none",
			in:   render.Input{Filtered: filtered, Body: block.Parse(raw)},
			want: "", source: render.H1None,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := synthetic(t, tc.in)
			if got.H1 != tc.want || got.H1Source != tc.source {
				t.Errorf("h1 %q from %q, want %q from %q", got.H1, got.H1Source, tc.want, tc.source)
			}
		})
	}
}

func TestTheVisibleH1PrefersTheRenderAndNeverGuesses(t *testing.T) {
	t.Parallel()

	record := func(h1 string, source render.H1Source) *render.Record {
		return &render.Record{Extraction: render.Extraction{H1: h1, H1Source: source}}
	}
	cases := []struct {
		name    string
		record  *render.Record
		content string
		want    string
		source  render.H1Source
	}{
		{name: "the theme's title over an empty content", record: record("BPC-157", render.H1Theme), want: "BPC-157", source: render.H1Theme},
		{name: "the render's own heading", record: record("Lab notes", render.H1Content), content: "Lab notes", want: "Lab notes", source: render.H1Content},
		{name: "a render with none keeps none", record: record("", render.H1None), content: "Stale", want: "", source: render.H1None},
		{name: "no render falls back to the content", content: "Lab notes", want: "Lab notes", source: render.H1Content},
		{name: "a render of unknown source falls back to the content", record: record("", render.H1Unknown), content: "Lab notes", want: "Lab notes", source: render.H1Content},
		{name: "nothing at all is none", want: "", source: render.H1None},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, source := render.VisibleH1(tc.record, tc.content)
			if got != tc.want || source != tc.source {
				t.Errorf("visible h1 %q from %q, want %q from %q", got, source, tc.want, tc.source)
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
