package field_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/field"
)

func TestSameTextDecodesOnlyPlainFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		key  field.Key
		a, b string
		want bool
	}{
		{name: "a title kses stored with its ampersand escaped", key: field.Title, a: "Tea & coffee", b: "Tea &amp; coffee", want: true},
		{name: "an excerpt with a numeric entity", key: field.Excerpt, a: "Don&#8217;t shake it", b: "Don’t shake it", want: true},
		{name: "an SEO title with escaped brackets", key: field.SEOTitle, a: "Doses &lt;5 mg&gt;", b: "Doses <5 mg>", want: true},
		{name: "an SEO description with spaces around it", key: field.SEODescription, a: " Pull a shot. ", b: "Pull a shot.", want: true},
		{name: "a title that moved", key: field.Title, a: "Tea & coffee", b: "Tea and coffee"},
		{name: "a title compared without folding its case", key: field.Title, a: "Café", b: "CAFÉ"},
		{name: "a body is markup, never decoded", key: field.Body, a: "<p>Tea & coffee</p>", b: "<p>Tea &amp; coffee</p>"},
		{name: "a product's description is markup too", key: field.ProductDescription, a: "Tea & coffee", b: "Tea &amp; coffee"},
		{name: "equal text is the same for any field", key: field.Body, a: "<p>Tea</p>", b: "<p>Tea</p>", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := field.SameText(tc.key, tc.a, tc.b); got != tc.want {
				t.Errorf("SameText(%s, %q, %q) = %t, want %t", tc.key, tc.a, tc.b, got, tc.want)
			}
		})
	}
}
