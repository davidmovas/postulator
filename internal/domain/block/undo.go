package block

import (
	"slices"
	"strings"
)

const defaultSeparator = "\n\n"

type Undone struct {
	Body   string
	Undone int
	Kept   int
}

type undoGroup struct {
	before     string
	removed    []Change
	added      []Change
	occurrence int
}

type segment struct {
	lead string
	core string
	sep  string
}

type undoSplice struct {
	with       []segment
	at, length int
}

func undoGroups(changes []Change) []undoGroup {
	groups := make([]undoGroup, 0)
	anchor := ""
	seen := make(map[string]int)
	var current *undoGroup
	for i := range changes {
		switch changes[i].Change {
		case ChangeKept:
			if current != nil {
				groups = append(groups, *current)
				current = nil
			}
			anchor = changes[i].Hash
			seen[anchor]++
		default:
			if current == nil {
				current = &undoGroup{before: anchor, occurrence: seen[anchor]}
			}
			if changes[i].Change == ChangeAdded {
				current.added = append(current.added, changes[i])
			} else {
				current.removed = append(current.removed, changes[i])
			}
		}
	}
	if current != nil {
		groups = append(groups, *current)
	}
	return groups
}

func segmentOf(raw string) segment {
	held := split(raw)
	return segment{lead: held.lead, core: held.core, sep: held.trail}
}

func Undo(live string, changes []Change) Undone {
	blocks := Parse(live).Blocks
	segments := make([]segment, 0, len(blocks))
	hashes := make([]string, 0, len(blocks))
	for i := range blocks {
		segments = append(segments, segmentOf(blocks[i].Raw))
		hashes = append(hashes, blocks[i].Hash)
	}
	ending := ""
	if len(segments) > 0 {
		ending = segments[len(segments)-1].sep
	}

	splices := make([]undoSplice, 0)
	used := make([]bool, len(hashes))
	result := Undone{}
	for _, group := range undoGroups(changes) {
		at, found := locate(hashes, used, group)
		if !found {
			result.Kept++
			continue
		}
		restored := make([]segment, 0, len(group.removed))
		for i := range group.removed {
			restored = append(restored, segmentOf(group.removed[i].Raw))
		}
		for i := at; i < at+len(group.added); i++ {
			used[i] = true
		}
		splices = append(splices, undoSplice{at: at, length: len(group.added), with: restored})
		result.Undone++
	}
	slices.SortStableFunc(splices, func(a, b undoSplice) int { return b.at - a.at })
	for _, s := range splices {
		segments = slices.Replace(segments, s.at, s.at+s.length, s.with...)
	}
	result.Body = assemble(segments, ending)
	return result
}

func locate(hashes []string, used []bool, group undoGroup) (int, bool) {
	from := 0
	if group.before != "" {
		anchor := occurrenceOf(hashes, group.before, group.occurrence)
		if anchor < 0 && len(group.added) == 0 {
			return 0, false
		}
		if anchor >= 0 {
			from = anchor + 1
		}
	}
	if len(group.added) == 0 {
		return from, true
	}
	for _, start := range []int{from, 0} {
		for at := start; at+len(group.added) <= len(hashes); at++ {
			if runMatches(hashes, used, at, group.added) {
				return at, true
			}
		}
	}
	return 0, false
}

func occurrenceOf(hashes []string, hash string, occurrence int) int {
	last, seen := -1, 0
	for i := range hashes {
		if hashes[i] != hash {
			continue
		}
		last, seen = i, seen+1
		if seen == occurrence {
			return i
		}
	}
	return last
}

func runMatches(hashes []string, used []bool, at int, added []Change) bool {
	for i := range added {
		if used[at+i] || hashes[at+i] != added[i].Hash {
			return false
		}
	}
	return true
}

func assemble(segments []segment, ending string) string {
	var builder strings.Builder
	for i := range segments {
		if i == 0 || segments[i].lead != "" {
			builder.WriteString(segments[i].lead)
		}
		builder.WriteString(segments[i].core)
		switch {
		case i == len(segments)-1:
			builder.WriteString(ending)
		case segments[i].sep != "" || segments[i+1].lead != "":
			builder.WriteString(segments[i].sep)
		default:
			builder.WriteString(defaultSeparator)
		}
	}
	return builder.String()
}
