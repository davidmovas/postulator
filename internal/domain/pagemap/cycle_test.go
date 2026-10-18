package pagemap_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func under(page pagemap.Page, parent string) pagemap.Page {
	page.ParentPageID = &parent
	return page
}

func TestCyclesNamesEachLoopOnceInPathOrder(t *testing.T) {
	t.Parallel()

	pages := []pagemap.Page{
		sitePage("home", "/", pagemap.WPPage, 1),
		under(sitePage("b", "/b/", pagemap.WPPage, 2), "a"),
		under(sitePage("a", "/a/", pagemap.WPPage, 3), "b"),
		under(sitePage("leaf", "/leaf/", pagemap.WPPage, 4), "a"),
		under(sitePage("z", "/z/", pagemap.WPPage, 5), "y"),
		under(sitePage("y", "/y/", pagemap.WPPage, 6), "x"),
		under(sitePage("x", "/x/", pagemap.WPPage, 7), "z"),
		under(sitePage("self", "/self/", pagemap.WPPage, 8), "self"),
		under(sitePage("fine", "/fine/", pagemap.WPPage, 9), "home"),
	}

	got := make([][]string, 0)
	for _, cycle := range pagemap.Cycles(pages) {
		ids := make([]string, 0, len(cycle))
		for _, page := range cycle {
			ids = append(ids, page.ID)
		}
		got = append(got, ids)
	}
	want := [][]string{{"a", "b"}, {"self"}, {"x", "y", "z"}}
	if !slices.EqualFunc(got, want, slices.Equal[[]string]) {
		t.Errorf("Cycles = %v, want %v", got, want)
	}
}

func TestCyclesIsEmptyForATree(t *testing.T) {
	t.Parallel()

	pages := []pagemap.Page{
		sitePage("home", "/", pagemap.WPPage, 1),
		under(sitePage("a", "/a/", pagemap.WPPage, 2), "home"),
		under(sitePage("b", "/a/b/", pagemap.WPPage, 3), "a"),
		under(sitePage("orphan", "/orphan/", pagemap.WPPage, 4), "missing"),
	}
	if cycles := pagemap.Cycles(pages); len(cycles) != 0 {
		t.Errorf("Cycles = %v, want none", cycles)
	}
}
