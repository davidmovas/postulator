package block_test

import (
	"reflect"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
)

func TestADiffMarksEachBlockKeptAddedOrRemovedAndWhoseItIs(t *testing.T) {
	t.Parallel()

	const (
		theirs = "<p>The owner's paragraph.</p>"
		grid   = `[products limit="4"]`
		ourOld = "<p>Our first article.</p>"
		ourNew = "<p>Our second article.</p>"
	)
	before := theirs + "\n\n" + ourOld + "\n\n" + grid
	after := theirs + "\n\n" + ourNew + "\n\n" + grid

	got := block.Diff(before, after, []string{block.Hash(ourOld)})
	want := []block.Change{
		{Kind: block.Paragraph, Hash: block.Hash(theirs), Raw: theirs + "\n\n", Owner: block.OwnerClient, Change: block.ChangeKept},
		{Kind: block.Paragraph, Hash: block.Hash(ourOld), Raw: ourOld + "\n\n", Owner: block.OwnerOurs, Change: block.ChangeRemoved},
		{Kind: block.Paragraph, Hash: block.Hash(ourNew), Raw: ourNew + "\n\n", Owner: block.OwnerOurs, Change: block.ChangeAdded},
		{Kind: block.Shortcode, Hash: block.Hash(grid), Raw: grid, Owner: block.OwnerClient, Change: block.ChangeKept},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Diff =\n%+v\nwant\n%+v", got, want)
	}
}

func TestADiffOfAnUnchangedBodyKeepsEveryBlock(t *testing.T) {
	t.Parallel()

	const body = "<!-- wp:paragraph -->\n<p>One.</p>\n<!-- /wp:paragraph -->\n\n<!-- wp:heading -->\n<h2>Two</h2>\n<!-- /wp:heading -->"
	for _, change := range block.Diff(body, body, nil) {
		if change.Change != block.ChangeKept || change.Owner != block.OwnerClient {
			t.Errorf("an unchanged block = %+v, want kept and the client's", change)
		}
	}
	if got := len(block.Diff(body, body, nil)); got != 2 {
		t.Errorf("the diff holds %d blocks, want 2", got)
	}
}

func TestADiffOfAnEmptyBodyAddsEveryBlock(t *testing.T) {
	t.Parallel()

	got := block.Diff("", "<p>Ours.</p>", nil)
	if len(got) != 1 || got[0].Change != block.ChangeAdded || got[0].Owner != block.OwnerOurs {
		t.Fatalf("Diff of a written empty page = %+v, want one block of ours added", got)
	}
}
