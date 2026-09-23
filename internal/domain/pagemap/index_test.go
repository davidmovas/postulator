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

	if index.Len() != 4 {
		t.Fatalf("Len = %d", index.Len())
	}
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
