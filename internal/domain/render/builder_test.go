package render_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/render"
)

func TestExtractDetectsABuilder(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   func(t *testing.T) render.Input
		want string
	}{
		{
			name: "elementor's data attribute on the wrapper the content landed in",
			in: func(*testing.T) render.Input {
				return render.Input{
					Public: `<html><body><header><h2>Shop</h2></header><main><div data-elementor-type="wp-page" data-elementor-id="7">` +
						`<section><div><p>Built with blocks of a builder.</p></div></section></div></main></body></html>`,
					Filtered: `<div data-elementor-type="wp-page" data-elementor-id="7"><p>Built with blocks of a builder.</p></div>`,
					Body:     block.Parse(`<p>Built with blocks of a builder.</p>`),
					Builders: defaultBuilders(),
				}
			},
			want: "elementor",
		},
		{
			name: "wpbakery's shortcode in the body",
			in: func(*testing.T) render.Input {
				return render.Input{
					Public:   `<html><body><main><div><p>Columns of text.</p></div></main></body></html>`,
					Filtered: `<div><p>Columns of text.</p></div>`,
					Body:     block.Parse(`[vc_row][vc_column]Columns of text.[/vc_column][/vc_row]`),
					Builders: defaultBuilders(),
				}
			},
			want: "wpbakery",
		},
		{
			name: "divi's class prefix inside the root",
			in: func(*testing.T) render.Input {
				return render.Input{
					Public:   `<html><body><main><div><div class="et_pb_section et_pb_section_0"><p>Divi text here.</p></div><p>More text.</p></div></main></body></html>`,
					Filtered: `<div class="et_pb_section et_pb_section_0"><p>Divi text here.</p></div><p>More text.</p>`,
					Body:     block.Parse("Divi text here.\n\nMore text."),
					Builders: defaultBuilders(),
				}
			},
			want: "divi",
		},
		{
			name: "a builder that only made the theme's header is not the content's",
			in: func(*testing.T) render.Input {
				return render.Input{
					Public: `<html><body><header data-elementor-type="header"><p>Shop name</p></header>` +
						`<main><h1>Title</h1><div><p>Plain words of the page.</p></div></main></body></html>`,
					Filtered: `<p>Plain words of the page.</p>`,
					Body:     block.Parse(`<p>Plain words of the page.</p>`),
					Builders: defaultBuilders(),
				}
			},
			want: "",
		},
		{
			name: "an empty rule matches nothing",
			in: func(*testing.T) render.Input {
				return render.Input{
					Public:   `<html><body><main><p>Words.</p></main></body></html>`,
					Body:     block.Parse(`<p>Words.</p>`),
					Builders: []render.Builder{{Name: "blank", DataAttributes: []string{""}, ClassPrefixes: []string{""}, ShortcodePrefixes: []string{""}}},
				}
			},
			want: "",
		},
		{
			name: "the replica's hub was built by none",
			in: func(t *testing.T) render.Input {
				return standInput(t, replica, "bpc-157", "/bpc-157/")
			},
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := synthetic(t, tc.in(t))
			if got.Builder != tc.want {
				t.Errorf("builder %q, want %q", got.Builder, tc.want)
			}
		})
	}
}

func TestExtractReadsRobots(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		head        string
		wantRobots  string
		wantNoIndex bool
	}{
		{"a noindex page", `<meta name="robots" content="noindex, follow">`, "noindex, follow", true},
		{"an upper-case directive and name", `<meta name="ROBOTS" content=" NOINDEX,NOFOLLOW ">`, "noindex,nofollow", true},
		{"an indexed page", `<meta name="robots" content="index, follow, max-snippet:-1">`, "index, follow, max-snippet:-1", false},
		{"no robots meta", `<meta name="description" content="noindex">`, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			public := `<html><head>` + tc.head + `</head><body><main><p>Words.</p></main></body></html>`
			got := synthetic(t, render.Input{Public: public, Body: block.Parse("")})
			if got.Robots != tc.wantRobots || got.NoIndex() != tc.wantNoIndex {
				t.Errorf("robots %q (noindex %t), want %q (%t)", got.Robots, got.NoIndex(), tc.wantRobots, tc.wantNoIndex)
			}
		})
	}

	stand := extractStand(t, replica, "tb-500", "/tb-500/")
	if stand.NoIndex() || stand.Robots != "index, follow, max-snippet:-1, max-video-preview:-1, max-image-preview:large" {
		t.Errorf("the replica's robots %q", stand.Robots)
	}
}
