package paging_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func held(desc bool) []row {
	base := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	rows := []row{
		{createdAt: base, id: "a"},
		{createdAt: base.Add(time.Minute), id: "b"},
		{createdAt: base.Add(time.Minute), id: "c"},
		{createdAt: base.Add(2 * time.Minute), id: "d"},
		{createdAt: base.Add(3 * time.Minute), id: "e"},
	}
	if desc {
		slices.Reverse(rows)
	}
	return rows
}

func ids(list paging.List[row]) string {
	out := make([]string, 0, len(list.Items))
	for _, r := range list.Items {
		out = append(out, r.id)
	}
	return strings.Join(out, "")
}

func walk(t *testing.T, k paging.Keyset[row], rows []row, limit int) []paging.List[row] {
	t.Helper()

	pages := make([]paging.List[row], 0)
	request := paging.Request{Limit: limit}
	for range len(rows) + 1 {
		page, err := k.Page(rows, request)
		if err != nil {
			t.Fatalf("Page(%+v): %v", request, err)
		}
		pages = append(pages, page)
		if page.Next == "" {
			return pages
		}
		request = paging.Request{After: page.Next, Limit: limit}
	}
	t.Fatal("the walk never ended")
	return nil
}

func TestPageWalksHeldRowsByCursor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		desc bool
		want []string
	}{
		{name: "ascending, a tie broken by id", want: []string{"ab", "cd", "e"}},
		{name: "descending", desc: true, want: []string{"ed", "cb", "a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pages := walk(t, rowKeyset(tc.desc), held(tc.desc), 2)
			got := make([]string, 0, len(pages))
			for _, page := range pages {
				got = append(got, ids(page))
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("pages = %v, want %v", got, tc.want)
			}
			if pages[0].Prev != "" || pages[1].Prev == "" || pages[len(pages)-1].HasMore {
				t.Errorf("cursors = %+v, want no prev on the first page, one on the second and the last with nothing more", pages)
			}
		})
	}
}

func TestPageGoesBackByThePrevCursor(t *testing.T) {
	t.Parallel()

	k := rowKeyset(false)
	rows := held(false)
	pages := walk(t, k, rows, 2)
	back, err := k.Page(rows, paging.Request{Before: pages[2].Prev, Limit: 2})
	if err != nil {
		t.Fatalf("Page before: %v", err)
	}
	if ids(back) != "cd" || back.Next == "" || back.Prev == "" {
		t.Errorf("the page before the last = %q %+v, want cd with both cursors", ids(back), back.Cursors)
	}
	first, err := k.Page(rows, paging.Request{Before: back.Prev, Limit: 2})
	if err != nil {
		t.Fatalf("Page before: %v", err)
	}
	if ids(first) != "ab" || first.Prev != "" || first.HasMore {
		t.Errorf("the first page reached backward = %q %+v, want ab and nothing before it", ids(first), first)
	}
}

func TestPageRefusesACursorOfAnotherOrder(t *testing.T) {
	t.Parallel()

	desc := rowKeyset(true)
	cursor := mustCursor(t, desc, held(true)[0])
	if _, err := rowKeyset(false).Page(held(false), paging.Request{After: cursor}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("Page with a descending cursor = %v, want Invalid", err)
	}
	empty, err := rowKeyset(false).Page(nil, paging.Request{})
	if err != nil || len(empty.Items) != 0 || empty.HasMore {
		t.Errorf("Page of nothing = %+v, %v", empty, err)
	}
	if _, err := (paging.Keyset[row]{}).Page(held(false), paging.Request{}); !errors.IsCode(err, errors.Internal) {
		t.Errorf("Page with no keys = %v, want Internal", err)
	}
}

func TestPageComparesTextAndIntKeys(t *testing.T) {
	t.Parallel()

	k := paging.Keyset[row]{
		IDColumn: "id",
		ID:       func(r row) string { return r.id },
		Keys: []paging.SortKey[row]{
			paging.TextKey[row]("name", "name", func(r row) any { return r.name }),
			paging.IntKey[row]("score", "score", func(r row) any { return r.score }),
		},
	}
	rows := []row{
		{id: "1", name: "alpha", score: 1}, {id: "2", name: "alpha", score: 2},
		{id: "3", name: "beta", score: 1}, {id: "4", name: "beta", score: 1}, {id: "5", name: "gamma", score: 0},
	}
	pages := walk(t, k, rows, 2)
	got := make([]string, 0, len(pages))
	for _, page := range pages {
		got = append(got, ids(page))
	}
	if want := []string{"12", "34", "5"}; !slices.Equal(got, want) {
		t.Errorf("pages = %v, want %v", got, want)
	}
}
