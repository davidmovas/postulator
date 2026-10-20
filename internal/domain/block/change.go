package block

import "slices"

type Owner string

const (
	OwnerOurs   Owner = "ours"
	OwnerClient Owner = "client"
)

type ChangeKind string

const (
	ChangeKept    ChangeKind = "kept"
	ChangeAdded   ChangeKind = "added"
	ChangeRemoved ChangeKind = "removed"
)

type Change struct {
	Kind   BlockKind  `json:"kind"`
	Hash   string     `json:"hash"`
	Raw    string     `json:"raw"`
	Owner  Owner      `json:"owner"`
	Change ChangeKind `json:"change"`
}

func Diff(before, after string, written []string) []Change {
	left, right := Parse(before).Blocks, Parse(after).Blocks
	pairs := align(hashes(left), hashes(right))
	out := make([]Change, 0, len(left)+len(right))
	i, j := 0, 0
	for _, matched := range append(pairs, pair{left: len(left), right: len(right)}) {
		for ; i < matched.left; i++ {
			out = append(out, changeOf(left[i], ownerOf(left[i], written), ChangeRemoved))
		}
		for ; j < matched.right; j++ {
			out = append(out, changeOf(right[j], OwnerOurs, ChangeAdded))
		}
		if matched.left < len(left) && matched.right < len(right) {
			out = append(out, changeOf(right[j], ownerOf(left[i], written), ChangeKept))
			i, j = i+1, j+1
		}
	}
	return out
}

func hashes(blocks []Block) []string {
	out := make([]string, 0, len(blocks))
	for i := range blocks {
		out = append(out, blocks[i].Hash)
	}
	return out
}

func ownerOf(held Block, written []string) Owner {
	if slices.Contains(written, held.Hash) {
		return OwnerOurs
	}
	return OwnerClient
}

func changeOf(held Block, owner Owner, kind ChangeKind) Change {
	return Change{Kind: held.Kind, Hash: held.Hash, Raw: held.Raw, Owner: owner, Change: kind}
}
