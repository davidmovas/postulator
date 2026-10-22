package block_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
)

type landedEntry struct {
	hash   string
	owner  block.Owner
	change block.ChangeKind
}

func entriesOf(changes []block.Change) []landedEntry {
	out := make([]landedEntry, 0, len(changes))
	for i := range changes {
		out = append(out, landedEntry{hash: changes[i].Hash, owner: changes[i].Owner, change: changes[i].Change})
	}
	return out
}

func TestLandedCarriesTheDiffOntoWhatTheSiteStored(t *testing.T) {
	t.Parallel()

	const (
		intro    = "<p>Fresh roasted daily.</p>"
		middle   = "<p>Beans from three farms.</p>"
		altered  = "<p>Beans from three farms and a fourth.</p>"
		edited   = "<p>Fresh roasted every morning.</p>"
		ours     = `<p>We roast every <a href="/coffee/espresso/">espresso</a> blend we sell.</p>`
		filtered = `<p>We roast every <a rel="noopener" href="/coffee/espresso/">espresso</a> blend we sell.</p>`
		sentence = "<p>Read more about filter coffee.</p>"
		injected = "<p>[toc]</p>"
		figure   = `<figure><a href="/coffee/"><img src="/wp-content/uploads/roast.jpg" alt="The roastery"></a></figure>`
		figured  = `<figure><a rel="noopener" href="/coffee/"><img src="/wp-content/uploads/roast.jpg" alt="The roastery"></a></figure>`
	)
	hash := block.Hash
	kept := func(raw string, owner block.Owner) landedEntry {
		return landedEntry{hash: hash(raw), owner: owner, change: block.ChangeKept}
	}
	added := func(raw string, owner block.Owner) landedEntry {
		return landedEntry{hash: hash(raw), owner: owner, change: block.ChangeAdded}
	}
	removed := func(raw string) landedEntry {
		return landedEntry{hash: hash(raw), owner: block.OwnerClient, change: block.ChangeRemoved}
	}

	cases := []struct {
		name     string
		previous string
		composed string
		stored   string
		live     string
		want     []landedEntry
		undone   string
	}{
		{
			name:     "the site reformatted ours and the client edited his block since",
			previous: joined(intro, middle),
			composed: joined(intro, ours),
			stored:   joined(intro, filtered),
			live:     joined(edited, filtered),
			want:     []landedEntry{kept(intro, block.OwnerClient), removed(middle), added(filtered, block.OwnerOurs)},
			undone:   joined(edited, middle),
		},
		{
			name:     "the site stored what was sent",
			previous: joined(intro),
			composed: joined(intro, ours),
			stored:   joined(intro, ours),
			live:     joined(intro, ours),
			want:     []landedEntry{kept(intro, block.OwnerClient), added(ours, block.OwnerOurs)},
			undone:   joined(intro),
		},
		{
			name:     "the site added a block among ours",
			previous: joined(intro),
			composed: joined(intro, ours),
			stored:   joined(intro, injected, ours),
			live:     joined(intro, injected, ours),
			want:     []landedEntry{kept(intro, block.OwnerClient), added(injected, block.OwnerSite), added(ours, block.OwnerOurs)},
			undone:   joined(intro),
		},
		{
			name:     "the site dropped one of ours",
			previous: joined(intro),
			composed: joined(intro, ours, sentence),
			stored:   joined(intro, sentence),
			live:     joined(intro, sentence),
			want:     []landedEntry{kept(intro, block.OwnerClient), added(sentence, block.OwnerOurs)},
			undone:   joined(intro),
		},
		{
			name:     "the save altered a block of the client's",
			previous: joined(intro, middle),
			composed: joined(intro, middle, ours),
			stored:   joined(intro, altered, ours),
			live:     joined(intro, altered, ours),
			want:     []landedEntry{kept(intro, block.OwnerClient), kept(altered, block.OwnerClient), added(ours, block.OwnerOurs)},
			undone:   joined(intro, altered),
		},
		{
			name:     "a textless block of ours whose link the site changed",
			previous: joined(intro),
			composed: joined(intro, ours, figure),
			stored:   joined(intro, filtered, figured),
			live:     joined(intro, filtered, figured),
			want:     []landedEntry{kept(intro, block.OwnerClient), added(filtered, block.OwnerOurs), added(figured, block.OwnerOurs)},
			undone:   joined(intro),
		},
		{
			name:     "a block prepended while ours was dropped is never ours",
			previous: joined(intro),
			composed: joined(ours, intro),
			stored:   joined(injected, intro),
			live:     joined(injected, intro),
			want:     []landedEntry{added(injected, block.OwnerSite), kept(intro, block.OwnerClient)},
			undone:   joined(intro),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			landed := block.Landed(block.Diff(tc.previous, tc.composed, nil), tc.composed, tc.stored)
			if got := entriesOf(landed); !slices.Equal(got, tc.want) {
				t.Errorf("Landed =\n%+v\nwant\n%+v", got, tc.want)
			}
			if got := block.Undo(tc.live, landed); got.Body != tc.undone {
				t.Errorf("Undo over the landed diff =\n%q\nwant\n%q", got.Body, tc.undone)
			}
		})
	}
}

func TestLandedKeepsOurEarlierBlocksOurs(t *testing.T) {
	t.Parallel()

	const (
		intro    = "<p>Fresh roasted daily.</p>"
		earlier  = "<p>Our earlier section on roasting.</p>"
		ours     = `<p>We roast every <a href="/coffee/espresso/">espresso</a> blend we sell.</p>`
		filtered = `<p>We roast every <a rel="noopener" href="/coffee/espresso/">espresso</a> blend we sell.</p>`
	)
	composed := joined(intro, earlier, ours)
	landed := block.Landed(block.Diff(joined(intro, earlier), composed, []string{block.Hash(earlier)}), composed, joined(intro, earlier, filtered))
	want := []landedEntry{
		{hash: block.Hash(intro), owner: block.OwnerClient, change: block.ChangeKept},
		{hash: block.Hash(earlier), owner: block.OwnerOurs, change: block.ChangeKept},
		{hash: block.Hash(filtered), owner: block.OwnerOurs, change: block.ChangeAdded},
	}
	if got := entriesOf(landed); !slices.Equal(got, want) {
		t.Errorf("Landed =\n%+v\nwant\n%+v", got, want)
	}
	if raw := landed[2].Raw; raw != filtered {
		t.Errorf("our block's raw = %q, want the bytes the site stored", raw)
	}
}
