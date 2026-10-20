package block

import (
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ReasonBlockNotEditable = "block_not_editable"
	ReasonBlockOutOfRange  = "block_out_of_range"
	ReasonNothingToRewrite = "nothing_to_rewrite"

	blockSpace = " \t\r\n"
)

const (
	detailFused = "the change would merge a block with its neighbors"
	detailKind  = "the change would turn the body between the block editor and the classic one"
)

type piece struct {
	lead  string
	core  string
	trail string
	fresh bool
}

func split(raw string) piece {
	rest := strings.TrimLeft(raw, blockSpace)
	core := strings.TrimRight(rest, blockSpace)
	return piece{lead: raw[:len(raw)-len(rest)], core: core, trail: rest[len(core):]}
}

func added(raw string) piece {
	return piece{core: strings.Trim(raw, blockSpace), fresh: true}
}

func (b Body) pieces() []piece {
	out := make([]piece, 0, len(b.Blocks)+1)
	for i := range b.Blocks {
		out = append(out, split(b.Blocks[i].Raw))
	}
	return out
}

func (b Body) separator() string {
	for i := range b.Blocks {
		if strings.Contains(b.Blocks[i].Raw, "\r\n") {
			return "\r\n\r\n"
		}
	}
	return "\n\n"
}

func (b Body) Replace(i int, raw string) (Body, error) {
	if err := b.within(i); err != nil {
		return Body{}, err
	}
	if err := addable(raw); err != nil {
		return Body{}, err
	}
	pieces := b.pieces()
	pieces[i] = piece{lead: pieces[i].lead, core: strings.Trim(raw, blockSpace), trail: pieces[i].trail, fresh: true}
	return b.rebuild(pieces, i-1, i)
}

func (b Body) InsertAfter(i int, raw string) (Body, error) {
	if i < -1 || i >= len(b.Blocks) {
		return Body{}, outOfRange(i, len(b.Blocks))
	}
	if err := addable(raw); err != nil {
		return Body{}, err
	}
	pieces := b.pieces()
	insert := added(raw)
	switch {
	case len(pieces) == 0:
	case i == len(pieces)-1:
		insert.trail, pieces[i].trail = pieces[i].trail, b.separator()
	default:
		insert.trail = b.separator()
	}
	return b.rebuild(slices.Insert(pieces, i+1, insert), i, i+1)
}

func (b Body) Delete(i int) (Body, error) {
	if err := b.within(i); err != nil {
		return Body{}, err
	}
	return b.rebuild(removed(b.pieces(), i), i-1, i-1)
}

func (b Body) RewriteRange(from, to int, raw string) (Body, error) {
	if from < 0 || to > len(b.Blocks) || from >= to {
		return Body{}, outOfRange(from, len(b.Blocks)).WithDetail("to", to)
	}
	if err := addable(raw); err != nil {
		return Body{}, err
	}
	prose := make([]int, 0, to-from)
	for k := from; k < to; k++ {
		if b.Blocks[k].Prose() {
			prose = append(prose, k)
		}
	}
	if len(prose) == 0 {
		return Body{}, errors.New(errors.Invalid, "the range holds no text block to rewrite").
			WithDetail("reason", ReasonNothingToRewrite).WithDetail("from", from).WithDetail("to", to)
	}
	pieces := b.pieces()
	first := prose[0]
	pieces[first] = piece{lead: pieces[first].lead, core: strings.Trim(raw, blockSpace), trail: pieces[first].trail, fresh: true}
	for _, k := range slices.Backward(prose[1:]) {
		pieces = removed(pieces, k)
	}
	return b.rebuild(pieces, first-1, to-len(prose))
}

func removed(pieces []piece, i int) []piece {
	switch {
	case len(pieces) == 1:
	case i == len(pieces)-1:
		pieces[i-1].trail = pieces[i].trail
	case i == 0:
		pieces[1].lead = pieces[0].lead + pieces[1].lead
	}
	return slices.Delete(pieces, i, i+1)
}

func (b Body) rebuild(pieces []piece, lo, hi int) (Body, error) {
	next, detail := b.assembled(pieces)
	if detail == "" {
		return next, nil
	}
	if detail == detailFused && widened(pieces, lo, hi, b.separator()) {
		if next, detail = b.assembled(pieces); detail == "" {
			return next, nil
		}
	}
	return Body{}, errors.New(errors.Invalid, detail).WithDetail("reason", ReasonBlockNotEditable)
}

func widened(pieces []piece, lo, hi int, separator string) bool {
	changed := false
	for k := max(lo, 0); k <= hi && k < len(pieces)-1; k++ {
		if !blankLine.MatchString(pieces[k].trail) {
			pieces[k].trail = separator
			changed = true
		}
	}
	return changed
}

func (b Body) assembled(pieces []piece) (next Body, detail string) {
	var builder strings.Builder
	for i := range pieces {
		builder.WriteString(pieces[i].lead)
		builder.WriteString(pieces[i].core)
		builder.WriteString(pieces[i].trail)
	}
	next = Parse(builder.String())
	switch {
	case b.Kind == Blocks && next.Kind != Blocks:
		return Body{}, detailKind
	case b.Kind == Classic && len(b.Blocks) > 0 && next.Kind != Classic:
		return Body{}, detailKind
	}

	at := 0
	for i := range pieces {
		want := 1
		switch {
		case pieces[i].core == "":
			want = 0
		case pieces[i].fresh:
			want = len(parseAs(next.Kind, pieces[i].core).Blocks)
		}
		if at+want > len(next.Blocks) || strings.Trim(joinRaw(next.Blocks[at:at+want]), blockSpace) != pieces[i].core {
			return Body{}, detailFused
		}
		at += want
	}
	if at != len(next.Blocks) {
		return Body{}, detailFused
	}
	return next, ""
}

func parseAs(kind Kind, raw string) Body {
	if kind == Blocks {
		return finish(Blocks, raw, parseBlocks(raw))
	}
	return finish(Classic, raw, parseClassic(raw))
}

func (b Body) within(i int) error {
	if i < 0 || i >= len(b.Blocks) {
		return outOfRange(i, len(b.Blocks))
	}
	return nil
}

func outOfRange(i, count int) *errors.Error {
	return errors.New(errors.Invalid, "the body has no such block").
		WithDetail("reason", ReasonBlockOutOfRange).WithDetail("block", i).WithDetail("blocks", count)
}

func addable(raw string) error {
	if strings.Trim(raw, blockSpace) == "" {
		return errors.New(errors.Invalid, "an empty block is no block").WithDetail("reason", ReasonBlockNotEditable)
	}
	return nil
}
