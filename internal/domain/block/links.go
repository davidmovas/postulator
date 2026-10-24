package block

import (
	"slices"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type LinkAt struct {
	Href       string
	Anchor     string
	BlockHash  string
	Position   int
	InOurBlock bool
}

type PlacedKey struct {
	To        string
	Anchor    string
	BlockHash string
}

type Placed struct {
	To     string `json:"to"`
	Anchor string `json:"anchor"`
	Block  string `json:"block"`
}

type Moved struct {
	From string
	To   string
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
			out = append(out, LinkAt{
				Href: found.Href, Anchor: found.Anchor, BlockHash: body.Blocks[i].Hash, Position: i, InOurBlock: ours,
			})
		}
	}
	return out
}

func (l LinkAt) Ours(path string, held map[PlacedKey]bool) bool {
	return l.InOurBlock || held[PlacedKey{To: path, Anchor: l.Anchor, BlockHash: l.BlockHash}]
}

func HeldOurs(stored []pagemap.PageLink, last []LinkAt, resolve func(href string) (string, bool), moved []Moved) map[PlacedKey]bool {
	held := make(map[PlacedKey]bool, len(stored))
	for i := range stored {
		if stored[i].Source != pagemap.SourceRaw || !stored[i].Ours {
			continue
		}
		key := PlacedKey{To: stored[i].ToURL, Anchor: stored[i].AnchorText, BlockHash: stored[i].BlockHash}
		if key.BlockHash == "" {
			hash, found := heldIn(last, resolve, key)
			if !found {
				continue
			}
			key.BlockHash = hash
		}
		held[key] = true
	}
	for _, move := range moved {
		next := make(map[PlacedKey]bool, len(held))
		for key := range held {
			if key.BlockHash == move.From {
				key.BlockHash = move.To
			}
			next[key] = true
		}
		held = next
	}
	return held
}

func heldIn(last []LinkAt, resolve func(href string) (string, bool), key PlacedKey) (string, bool) {
	hash := ""
	for i := range last {
		if last[i].Anchor != key.Anchor {
			continue
		}
		path, ok := resolve(last[i].Href)
		if !ok || path != key.To {
			continue
		}
		if hash != "" && hash != last[i].BlockHash {
			return "", false
		}
		hash = last[i].BlockHash
	}
	return hash, hash != ""
}
