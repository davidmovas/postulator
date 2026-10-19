package block_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
)

func hashesOf(raw string) []string {
	body := block.Parse(raw)
	out := make([]string, 0, len(body.Blocks))
	for i := range body.Blocks {
		out = append(out, body.Blocks[i].Hash)
	}
	return out
}

func TestKeepPutsOurPartAfterTheClientsText(t *testing.T) {
	t.Parallel()

	const (
		client    = "<p>Step-by-step guides for handling research peptides.</p>\n\n<p>Ask before you mix.</p>"
		shortcode = `[products category="healing-peptides" limit="4"]`
		ours      = "<h2>Storage</h2><p>Keep them cold.</p>"
		earlier   = "<h2>Old</h2><p>What we wrote before.</p>"
	)
	cases := []struct {
		name    string
		current string
		written []string
		want    string
		kept    bool
	}{
		{name: "an empty body is ours alone", current: "", want: ours},
		{name: "the client's text first, ours after it", current: client, want: client + "\n\n" + ours, kept: true},
		{name: "ours before a lone shortcode, so the grid stays last", current: shortcode, want: ours + "\n\n" + shortcode, kept: true},
		{
			name: "ours after the text and before a trailing shortcode", current: client + "\n\n" + shortcode,
			want: client + "\n\n" + ours + "\n\n" + shortcode, kept: true,
		},
		{
			name: "what we wrote before is replaced, the client's text kept", current: client + "\n\n" + earlier,
			written: hashesOf(earlier), want: client + "\n\n" + ours, kept: true,
		},
		{name: "a body that is all ours is ours again", current: earlier, written: hashesOf(earlier), want: ours},
		{
			name: "the client's bytes stay as they were, line endings included", current: "<p>One\r\nline.</p>\r\n\r\n<p>Two.</p>",
			want: "<p>One\r\nline.</p>\r\n\r\n<p>Two.</p>\n\n" + ours, kept: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, kept := block.Keep(tc.current, tc.written, ours)
			if got != tc.want || kept != tc.kept {
				t.Errorf("Keep = %q, %t\nwant %q, %t", got, kept, tc.want, tc.kept)
			}
			if tc.kept && !strings.Contains(got, ours) {
				t.Errorf("our part is lost: %q", got)
			}
		})
	}
}

func TestClientTextIsWhatWeDidNotWrite(t *testing.T) {
	t.Parallel()

	const ours = "<p>Ours.</p>"
	cases := []struct {
		name    string
		current string
		written []string
		want    bool
	}{
		{name: "nothing", current: "", want: false},
		{name: "the client's paragraph", current: "<p>Theirs.</p>", want: true},
		{name: "only ours", current: ours, written: hashesOf(ours), want: false},
		{name: "a shortcode carries no text", current: "[products limit=\"4\"]", want: false},
		{name: "theirs beside ours", current: "<p>Theirs.</p>\n\n" + ours, written: hashesOf(ours), want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := block.ClientText(tc.current, tc.written); got != tc.want {
				t.Errorf("ClientText(%q) = %t, want %t", tc.current, got, tc.want)
			}
		})
	}
}

func TestTheClientsBlocksAreWhatIsLeftOfOurs(t *testing.T) {
	t.Parallel()

	const theirs = "<p>Theirs.</p>\n\n"
	ours := "<p>Ours.</p>"
	if got := block.Client(theirs+ours, hashesOf(ours)); got != theirs {
		t.Errorf("Client = %q, want only the client's block", got)
	}
	if got := block.Client(ours, hashesOf(ours)); got != "" {
		t.Errorf("Client of our own body = %q, want nothing", got)
	}
}

func TestOursAreTheBlocksTheClientDidNotHave(t *testing.T) {
	t.Parallel()

	const client = "<p>Theirs.</p>"
	stored := client + "\n\n<h2>Storage</h2><p>Keep them cold.</p>"
	got := block.Ours(stored, client)
	if len(got) == 0 {
		t.Fatal("no block counted as ours")
	}
	for _, hash := range got {
		if hash == block.Hash("<p>Theirs.</p>") {
			t.Errorf("the client's paragraph was counted as ours")
		}
	}
	if all := block.Ours(stored, ""); len(all) != len(hashesOf(stored)) {
		t.Errorf("with no client text every block is ours: %v", all)
	}
}
