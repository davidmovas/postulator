package contextpack_test

import (
	"reflect"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestAProductsCandidatesAreItsHubsProductsThenItsRelatedEntitiesProducts(t *testing.T) {
	t.Parallel()

	hub := mapped(onSitePage("hub", "/peptides/", "Peptides", 10, pagemap.WPPage), "e-hub")
	other := mapped(onSitePage("other", "/capsules/", "Capsules", 11, pagemap.WPPage), "e-other")
	target := mapped(under(onSitePage("bpc", "/product/bpc/", "BPC-157", 20, pagemap.WPProduct), "hub", pagemap.ParentJSONLD), "e-bpc")
	sibling := under(onSitePage("tb", "/product/tb/", "TB-500", 21, pagemap.WPProduct), "hub", pagemap.ParentJSONLD)
	draft := under(onSitePage("ghk", "/product/ghk/", "GHK-Cu", 22, pagemap.WPProduct), "hub", pagemap.ParentJSONLD)
	draft.Status = pagemap.StatusPlanned
	guide := under(onSitePage("guide", "/peptides/guide/", "A guide", 23, pagemap.WPPage), "hub", pagemap.ParentJSONLD)
	related := mapped(under(onSitePage("caps", "/product/caps/", "BPC-157 Capsules", 30, pagemap.WPProduct), "other", pagemap.ParentJSONLD), "e-caps")

	f := fixture{
		pages: []pagemap.Page{hub, other, target, sibling, draft, guide, related},
		entities: []graph.Entity{
			{ID: "e-hub", SiteID: siteID, Name: "Peptides", Source: graph.SourceUser},
			{ID: "e-other", SiteID: siteID, Name: "Capsules", Source: graph.SourceUser},
			{ID: "e-bpc", SiteID: siteID, Name: "BPC-157", Source: graph.SourceUser},
			{ID: "e-caps", SiteID: siteID, Name: "BPC-157 Capsules", Source: graph.SourceUser},
		},
		edges: []graph.Edge{{
			ID: "r1", SiteID: siteID, FromEntityID: "e-bpc", ToEntityID: "e-caps", Kind: graph.EdgeRelated, Weight: 0.7,
			Source: graph.SourceUser, Status: graph.StatusApproved,
		}},
	}
	s := f.snapshot(t)

	facts, err := s.Facts("bpc", content.LinkContext{})
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}
	want := []contextpack.Candidate{
		{WPID: 21, Name: "TB-500", Path: "/product/tb/"},
		{WPID: 30, Name: "BPC-157 Capsules", Path: "/product/caps/"},
	}
	if !reflect.DeepEqual(facts.Candidates, want) {
		t.Errorf("candidates = %+v, want %+v: the hub's published products, then the related entities', never a draft or a page", facts.Candidates, want)
	}

	hubFacts, err := s.Facts("hub", content.LinkContext{})
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}
	if len(hubFacts.Candidates) != 0 {
		t.Errorf("a page has the candidates %+v, want none: only a product has upsells", hubFacts.Candidates)
	}
}
