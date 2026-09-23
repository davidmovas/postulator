package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestBuildTreeAttachesToTheNearestExistingAncestor(t *testing.T) {
	t.Parallel()

	tree := pagemap.BuildTree([]pagemap.Page{
		page(pageD, "/shop/bags/leather/", nil),
		page(pageA, "/shop/", nil),
		page(pageB, "/blog/", nil),
		page(pageC, "/shop/shoes/", nil),
	})

	if len(tree) != 2 || tree[0].Page.Path != "/blog/" || tree[1].Page.Path != "/shop/" {
		t.Fatalf("roots = %+v", tree)
	}
	shop := tree[1]
	if len(shop.Children) != 2 || shop.Children[0].Page.Path != "/shop/bags/leather/" || shop.Children[1].Page.Path != "/shop/shoes/" {
		t.Errorf("shop children = %+v", shop.Children)
	}
	if len(tree[0].Children) != 0 {
		t.Errorf("blog children = %+v", tree[0].Children)
	}
}

func TestBuildTreeWithARootPage(t *testing.T) {
	t.Parallel()

	tree := pagemap.BuildTree([]pagemap.Page{page(pageA, "/", nil), page(pageB, "/a/", nil), page(pageC, "/a/b/", nil)})
	if len(tree) != 1 || tree[0].Page.Path != "/" || len(tree[0].Children) != 1 || len(tree[0].Children[0].Children) != 1 {
		t.Errorf("tree = %+v", tree)
	}
	if len(pagemap.BuildTree(nil)) != 0 {
		t.Error("no pages, no tree")
	}
}
