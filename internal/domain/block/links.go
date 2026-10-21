package block

import (
	"slices"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type LinkAt struct {
	Href       string
	Anchor     string
	Position   int
	InOurBlock bool
}

type LinkKey struct {
	To     string
	Anchor string
}

func LinksOf(raw string, written []string) []LinkAt {
	body := Parse(raw)
	out := make([]LinkAt, 0)
	for i := range body.Blocks {
		if body.Blocks[i].Dynamic {
			continue
		}
		doc, err := content.Parse(body.Blocks[i].Raw)
		if err != nil {
			continue
		}
		ours := slices.Contains(written, body.Blocks[i].Hash)
		for _, found := range doc.Links() {
			out = append(out, LinkAt{Href: found.Href, Anchor: found.Anchor, Position: i, InOurBlock: ours})
		}
	}
	return out
}

func (l LinkAt) Ours(path string, held map[LinkKey]bool) bool {
	return l.InOurBlock || held[LinkKey{To: path, Anchor: l.Anchor}]
}

func HeldOurs(stored []pagemap.PageLink) map[LinkKey]bool {
	held := make(map[LinkKey]bool, len(stored))
	for i := range stored {
		if stored[i].Source == pagemap.SourceRaw && stored[i].Ours {
			held[LinkKey{To: stored[i].ToURL, Anchor: stored[i].AnchorText}] = true
		}
	}
	return held
}
