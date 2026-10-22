package block

import "slices"

func Landed(changes []Change, composed, stored string) []Change {
	left, right := Parse(composed).Blocks, Parse(stored).Blocks
	owners := make([]Owner, 0, len(left))
	ours := make([]string, 0, len(left))
	for i := range changes {
		if changes[i].Change == ChangeRemoved {
			continue
		}
		owners = append(owners, changes[i].Owner)
		if changes[i].Owner == OwnerOurs {
			ours = append(ours, changes[i].Hash)
		}
	}
	if len(owners) != len(left) {
		return changes
	}
	at := Align(composed, stored, ours)
	pairTextless(at, owners, left, right)
	taken := make([]bool, len(right))
	for _, j := range at {
		if j >= 0 {
			taken[j] = true
		}
	}

	out := make([]Change, 0, len(changes)+len(right))
	next, k, client := 0, 0, false
	flush := func(until int) {
		for ; next < until; next++ {
			if taken[next] {
				continue
			}
			if client {
				out = append(out, changeOf(right[next], OwnerClient, ChangeKept))
				continue
			}
			out = append(out, changeOf(right[next], OwnerSite, ChangeAdded))
		}
	}
	for i := range changes {
		if changes[i].Change == ChangeRemoved {
			out = append(out, changes[i])
			continue
		}
		j := at[k]
		k++
		if j < 0 {
			client = client || changes[i].Owner != OwnerOurs
			continue
		}
		flush(j)
		client = false
		out = append(out, changeOf(right[j], changes[i].Owner, changes[i].Change))
		next = j + 1
	}
	flush(len(right))
	return out
}

func OursOf(changes []Change) []string {
	out := make([]string, 0, len(changes))
	for i := range changes {
		if changes[i].Owner == OwnerOurs && changes[i].Change != ChangeRemoved && !slices.Contains(out, changes[i].Hash) {
			out = append(out, changes[i].Hash)
		}
	}
	return out
}

func pairTextless(at []int, owners []Owner, left, right []Block) {
	gap := make([]int, 0)
	prevRight := -1
	for k := 0; k <= len(at); k++ {
		if k < len(at) && at[k] < 0 {
			gap = append(gap, k)
			continue
		}
		nextRight := len(right)
		if k < len(at) {
			nextRight = at[k]
		}
		pairGapTextless(at, owners, left, right, gap, prevRight+1, nextRight)
		gap, prevRight = gap[:0], nextRight
	}
}

func pairGapTextless(at []int, owners []Owner, left, right []Block, gap []int, from, until int) {
	composed := make([]int, 0, len(gap))
	for _, k := range gap {
		if owners[k] != OwnerOurs {
			return
		}
		if left[k].Text() == "" {
			composed = append(composed, k)
		}
	}
	paired := make(map[int]bool, len(at))
	for _, j := range at {
		paired[j] = true
	}
	held := make([]int, 0, until-from)
	for j := from; j < until; j++ {
		if !paired[j] && right[j].Text() == "" {
			held = append(held, j)
		}
	}
	if len(composed) == 0 || len(composed) != len(held) {
		return
	}
	for n, k := range composed {
		at[k] = held[n]
	}
}
