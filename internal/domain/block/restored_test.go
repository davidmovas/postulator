package block_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
)

func TestRestoredNamesOurBlocksAfterAPutBack(t *testing.T) {
	t.Parallel()

	const (
		client   = "<p>The client's own paragraph.</p>"
		earlier  = "<p>What the earlier run wrote.</p>"
		rewrite  = "<p>What the reverted run wrote over it.</p>"
		linked   = `<p>We sell <a href="/espresso/">espresso</a> and ristretto.</p>`
		unlinked = "<p>We sell espresso and ristretto.</p>"
		figure   = `<figure><img src="/roast.jpg" alt="The roastery"></figure>`
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
		name      string
		body      string
		written   []string
		moved     []block.Moved
		uncovered []string
		want      []string
	}{
		{
			name:      "the earlier run's block comes back ours through the diff",
			body:      join(client, earlier),
			written:   []string{block.Hash(rewrite)},
			uncovered: []string{block.Hash(earlier)},
			want:      []string{block.Hash(earlier)},
		},
		{
			name:    "our block put back through its lineage stays ours",
			body:    join(client, unlinked),
			written: []string{block.Hash(linked)},
			moved:   []block.Moved{{From: block.Hash(linked), To: block.Hash(unlinked)}},
			want:    []string{block.Hash(unlinked)},
		},
		{
			name:    "a block of ours the put-back left alone stays ours",
			body:    join(client, figure),
			written: []string{block.Hash(figure), block.Hash(rewrite)},
			want:    []string{block.Hash(figure)},
		},
		{
			name:    "the client's blocks are never ours",
			body:    join(client, unlinked),
			written: []string{block.Hash(rewrite)},
			moved:   []block.Moved{{From: block.Hash(client), To: block.Hash(unlinked)}},
			want:    []string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := block.Restored(tc.body, tc.written, tc.moved, tc.uncovered); !slices.Equal(got, tc.want) {
				t.Errorf("Restored = %v, want %v", got, tc.want)
			}
		})
	}
}
