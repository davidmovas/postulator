package block_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
)

func TestAlignAnswersMinusOneForABlockTheSiteDidNotKeep(t *testing.T) {
	t.Parallel()

	const (
		client   = "<p>Fresh roasted daily.</p>"
		second   = "<p>Beans from three farms.</p>"
		ours     = `<p>We roast every <a href="/coffee/espresso/">espresso</a> blend we sell.</p>`
		filtered = `<p>We roast every <a rel="noopener" href="/coffee/espresso/">espresso</a> blend we sell.</p>`
		sentence = "<p>Read more about filter coffee.</p>"
		injected = "<p>[toc]</p>"
	)
	join := func(blocks ...string) string {
		out := ""
		for i := range blocks {
			if i > 0 {
				out += "\n\n"
			}
			out += blocks[i]
		}
		return out
	}

	cases := []struct {
		name     string
		composed string
		stored   string
		ours     []string
		want     []int
	}{
		{
			name:     "the site reformatted our block",
			composed: join(client, ours),
			stored:   join(client, filtered),
			ours:     []string{block.Hash(ours)},
			want:     []int{0, 1},
		},
		{
			name:     "two of ours around the client's block",
			composed: join(ours, client, sentence),
			stored:   join(filtered, client, sentence),
			ours:     []string{block.Hash(ours), block.Hash(sentence)},
			want:     []int{0, 1, 2},
		},
		{
			name:     "a block prepended and ours dropped, the count unchanged",
			composed: join(ours, client),
			stored:   join(injected, client),
			ours:     []string{block.Hash(ours)},
			want:     []int{-1, 1},
		},
		{
			name:     "one of ours dropped from a run of two",
			composed: join(client, ours, sentence),
			stored:   join(client, sentence),
			ours:     []string{block.Hash(ours), block.Hash(sentence)},
			want:     []int{0, -1, -1},
		},
		{
			name:     "the client's block repeated",
			composed: join(client, client, ours),
			stored:   join(client, client, filtered),
			ours:     []string{block.Hash(ours)},
			want:     []int{0, 1, 2},
		},
		{
			name:     "an editor body",
			composed: "<!-- wp:paragraph -->\n" + client + "\n<!-- /wp:paragraph -->\n\n<!-- wp:html -->\n" + ours + "\n<!-- /wp:html -->",
			stored:   "<!-- wp:paragraph -->\n" + client + "\n<!-- /wp:paragraph -->\n\n<!-- wp:html -->\n" + filtered + "\n<!-- /wp:html -->",
			ours:     []string{block.Hash("<!-- wp:html -->\n" + ours + "\n<!-- /wp:html -->")},
			want:     []int{0, 1},
		},
		{
			name:     "the client's block changed by the site",
			composed: join(client, ours),
			stored:   join(second, filtered),
			ours:     []string{block.Hash(ours)},
			want:     []int{-1, 1},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := block.Align(tc.composed, tc.stored, tc.ours); !slices.Equal(got, tc.want) {
				t.Errorf("Align = %v, want %v", got, tc.want)
			}
		})
	}
}
