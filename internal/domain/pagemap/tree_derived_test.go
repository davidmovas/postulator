package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func childOf(id, path, parent string) pagemap.Page {
	page := pagemap.Page{ID: id, SiteID: "site", Path: path, WPType: pagemap.WPPage, Status: pagemap.StatusPublished}
	if parent != "" {
		page.ParentPageID = &parent
	}
	return page
}

func TestBuildTreeFollowsTheDerivedParent(t *testing.T) {
	t.Parallel()

	roots := pagemap.BuildTree([]pagemap.Page{
		childOf("home", "/", ""),
		childOf("peptides", "/peptides/", "home"),
		childOf("bpc", "/bpc-157/", "peptides"),
		childOf("guide", "/guide/", ""),
		childOf("setup", "/guide/setup/", ""),
	})
	if len(roots) != 1 || roots[0].Page.ID != "home" {
		t.Fatalf("roots = %v", ids(roots))
	}
	home := roots[0]
	if got := ids(home.Children); len(got) != 2 || got[0] != "guide" || got[1] != "peptides" {
		t.Fatalf("home's children = %v", got)
	}
	guide, peptides := home.Children[0], home.Children[1]
	if len(peptides.Children) != 1 || peptides.Children[0].Page.ID != "bpc" {
		t.Errorf("a flat URL is nested under its derived parent, got %v", ids(peptides.Children))
	}
	if len(guide.Children) != 1 || guide.Children[0].Page.ID != "setup" {
		t.Errorf("a page without a derived parent falls back to its path ancestor, got %v", ids(guide.Children))
	}
}

func TestBuildTreeBreaksACycleAtTheFirstPath(t *testing.T) {
	t.Parallel()

	roots := pagemap.BuildTree([]pagemap.Page{
		childOf("b", "/b/", "a"),
		childOf("a", "/a/", "b"),
	})
	if len(roots) != 1 || roots[0].Page.ID != "a" {
		t.Fatalf("roots = %v, want a alone", ids(roots))
	}
	if len(roots[0].Children) != 1 || roots[0].Children[0].Page.ID != "b" || len(roots[0].Children[0].Children) != 0 {
		t.Errorf("the cycle is not broken under a: %v", ids(roots[0].Children))
	}
}

func ids(nodes []pagemap.Node) []string {
	out := make([]string, 0, len(nodes))
	for i := range nodes {
		out = append(out, nodes[i].Page.ID)
	}
	return out
}
