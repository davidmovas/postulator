package content_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
)

func TestADescriptionIsShownWhenAVisitorCanReadItsOpening(t *testing.T) {
	t.Parallel()

	const description = `<p>The Mak liquid comes in a dark glass bottle, and it's measured with a dropper -- one dose a day.</p>` +
		`<p>A second paragraph nobody looks for.</p>`
	cases := []struct {
		name        string
		description string
		page        string
		shown       bool
	}{
		{
			name:        "the theme prints the description",
			description: description,
			page: `<html><head><title>Mak</title></head><body><h1>Mak Liquid</h1><div class="tab">` +
				`<p>The Mak liquid comes in a dark glass bottle, and it&#8217;s measured with a dropper &#8211; one dose a day.</p>` +
				`</div></body></html>`,
			shown: true,
		},
		{
			name:        "a page builder lays the page out without it",
			description: description,
			page: `<html><head><script type="application/ld+json">{"description":"The Mak liquid comes in a dark glass bottle, ` +
				`and it's measured with a dropper -- one dose a day."}</script></head><body><h1>Mak Liquid</h1>` +
				`<div class="builder">Buy now</div><style>.x{content:"The Mak liquid comes in a dark glass bottle"}</style></body></html>`,
			shown: false,
		},
		{
			name:        "the store shows its coming soon page",
			description: description,
			page:        `<html><body><h1>Coming soon</h1><p>The store opens shortly.</p></body></html>`,
			shown:       false,
		},
		{
			name:        "an empty description has nothing to show",
			description: `<p> </p>`,
			page:        `<html><body><h1>Mak Liquid</h1></body></html>`,
			shown:       true,
		},
		{
			name:        "a short description shown whole",
			description: `<p>Two words</p>`,
			page:        `<html><body><p>two words.</p></body></html>`,
			shown:       true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			shown, err := content.DescriptionShown(tc.description, tc.page)
			if err != nil {
				t.Fatalf("DescriptionShown: %v", err)
			}
			if shown != tc.shown {
				t.Errorf("shown = %t, want %t", shown, tc.shown)
			}
		})
	}
}
