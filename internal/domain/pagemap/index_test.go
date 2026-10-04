package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestIndex(t *testing.T) {
	t.Parallel()

	index := pagemap.NewIndex([]pagemap.Page{
		page(pageC, "/shop/bags/", ptr(entA)),
		page(pageA, "/", nil),
		page(pageB, "/shop/", ptr(entA)),
		page(pageD, "/blog/", ptr(entB)),
	})

	if p, found := index.ByID(pageB); !found || p.Path != "/shop/" {
		t.Errorf("ByID = %+v, %v", p, found)
	}
	if _, found := index.ByID("nope"); found {
		t.Error("unknown id found")
	}
	if p, found := index.ByPath("/Shop/Bags"); !found || p.ID != pageC {
		t.Errorf("ByPath must normalise its input, got %+v, %v", p, found)
	}
	if _, found := index.ByPath("/a b/"); found {
		t.Error("an invalid path must not be found")
	}
	byEntity := index.ByEntity(entA)
	if len(byEntity) != 2 || byEntity[0].Path != "/shop/" || byEntity[1].Path != "/shop/bags/" {
		t.Errorf("ByEntity = %+v", byEntity)
	}
	if len(index.ByEntity("nope")) != 0 {
		t.Error("unknown entity must have no pages")
	}
	pages := index.Pages()
	if len(pages) != 4 || pages[0].Path != "/" || pages[1].Path != "/blog/" || pages[3].Path != "/shop/bags/" {
		t.Errorf("Pages = %+v", pages)
	}
}

func TestAPageSitsUnderThePathAboveItAndAStoreItemNever(t *testing.T) {
	t.Parallel()

	root := page(pageA, "/", nil)
	shop := page(pageB, "/shop/", nil)
	index := pagemap.NewIndex([]pagemap.Page{root, shop})

	product := page(pageC, "/shop/bags/", nil)
	product.WPType = pagemap.WPProduct
	category := page(pageD, "/shop/totes/", nil)
	category.WPType = pagemap.WPProductCategory
	post := page(pageC, "/shop/news/", nil)
	post.WPType = pagemap.WPPost

	cases := []struct {
		name  string
		child pagemap.Page
		want  string
	}{
		{name: "a page under a page", child: page(pageC, "/shop/bags/", nil), want: pageB},
		{name: "a post under a page", child: post, want: pageB},
		{name: "a page written without its slash", child: page(pageC, "/Shop/Bags", nil), want: pageB},
		{name: "a page whose parent path is not mapped", child: page(pageC, "/blog/news/", nil)},
		{name: "the page above itself", child: shop, want: pageA},
		{name: "the root", child: root},
		{name: "a product", child: product},
		{name: "a product category", child: category},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parentID := index.PathParentID(tc.child)
			if (parentID != nil) != (tc.want != "") || (parentID != nil && *parentID != tc.want) {
				t.Errorf("PathParentID = %v, want %q", parentID, tc.want)
			}
		})
	}
}
