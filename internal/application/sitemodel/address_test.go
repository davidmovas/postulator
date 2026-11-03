package sitemodel_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func addressTree(t *testing.T, m *sitemodeltest.Memory, req sitemodel.AddressRequest) []sitemodel.AddressNode {
	t.Helper()

	req.SiteID = m.Site.ID
	var out []sitemodel.AddressNode
	for {
		got, err := m.Service().AddressTree(t.Context(), req)
		if err != nil {
			t.Fatalf("AddressTree(%+v): %v", req, err)
		}
		out = append(out, got.Nodes.Items...)
		if !got.Nodes.HasMore {
			return out
		}
		req.Cursor = got.Nodes.Next
	}
}

func TestThePagesTreeFollowsTheAddressNotTheBreadcrumbs(t *testing.T) {
	t.Parallel()

	m := wide()
	m.Pages = append(m.Pages, sitePage("p1", "/product/p1/", "P1", 90000, pagemap.WPProduct, pagetype.Product, "hub-03"))

	root := addressTree(t, m, sitemodel.AddressRequest{})
	if len(root) != 1 || root[0].Path != "/" || root[0].Page == nil || root[0].Page.Page.PageID != "home" || root[0].Pages != len(m.Pages) {
		t.Fatalf("the root = %+v, want the front page holding all %d pages", root, len(m.Pages))
	}

	top := addressTree(t, m, sitemodel.AddressRequest{Folder: "/", Limit: 7})
	if len(top) != hubs+1 {
		t.Fatalf("under the front page %d folders, want %d hubs and /product/", len(top), hubs+1)
	}
	last := top[len(top)-1]
	if last.Path != "/product/" || last.Page != nil || last.Children != 1 || last.Pages != 1 {
		t.Errorf("the last folder = %+v, want /product/ with no page of its own holding the product", last)
	}
	if top[0].Path != "/hub-00/" || top[0].Page == nil || top[0].Children != childrenOf || top[0].Pages != childrenOf+1 {
		t.Errorf("the first folder = %+v, want /hub-00/ with its page and its %d children", top[0], childrenOf)
	}

	products := addressTree(t, m, sitemodel.AddressRequest{Folder: "/product/"})
	if len(products) != 1 || products[0].Page == nil || products[0].Page.Page.PageID != "p1" {
		t.Errorf("under /product/ = %+v, want the product its breadcrumbs put under a hub", products)
	}

	hubsOnly := addressTree(t, m, sitemodel.AddressRequest{Folder: "/", Within: typeOf(pagetype.Hub)})
	if len(hubsOnly) != hubs || hubsOnly[0].Pages != 1 {
		t.Errorf("the hub folders = %d, first %+v; want %d folders of one hub each", len(hubsOnly), hubsOnly[0], hubs)
	}

	for _, req := range []sitemodel.AddressRequest{{SiteID: m.Site.ID, Folder: "/hub-00"}, {Folder: "/"}} {
		if _, err := m.Service().AddressTree(t.Context(), req); !errors.IsCode(err, errors.Invalid) {
			t.Errorf("AddressTree(%+v) = %v, want invalid", req, err)
		}
	}
}
