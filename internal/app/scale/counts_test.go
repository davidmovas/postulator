package scale_test

import (
	"strconv"
	"sync"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

const (
	mapLimit  = 50
	listLimit = 20
)

func (s *scaled) typeOf(t *testing.T, preset pagetype.Preset) string {
	t.Helper()

	types, err := sqlite.NewPageTypeRepo(s.core.Store).ListBySite(t.Context(), s.siteID)
	if err != nil {
		t.Fatalf("list the page types: %v", err)
	}
	for i := range types {
		if types[i].Preset == preset {
			return types[i].ID
		}
	}
	t.Fatalf("the site holds no %s type", preset)
	return ""
}

func (s *scaled) id(t *testing.T, path string) string {
	t.Helper()

	pageID, known := s.pageIDs[path]
	if !known {
		t.Fatalf("the generated site holds no %s", path)
	}
	return pageID
}

type operation struct {
	name string
	rows bool
	call func(t *testing.T, s *scaled) func() error
}

func siteMap(name string, request func(t *testing.T, s *scaled) sitemodel.MapRequest) operation {
	return operation{name: "map: " + name, rows: true, call: func(t *testing.T, s *scaled) func() error {
		req := request(t, s)
		req.SiteID = s.siteID
		return func() error {
			_, err := s.core.SiteModel.SiteMap(t.Context(), req)
			return err
		}
	}}
}

func findingsList(name string, request func(t *testing.T, s *scaled) findings.ListRequest) operation {
	return operation{name: "findings: " + name, call: func(t *testing.T, s *scaled) func() error {
		req := request(t, s)
		req.SiteID = s.siteID
		return func() error {
			_, err := s.core.Findings.List(t.Context(), req)
			return err
		}
	}}
}

func operations() []operation {
	return []operation{
		siteMap("the roots", func(*testing.T, *scaled) sitemodel.MapRequest { return sitemodel.MapRequest{} }),
		siteMap("the front page's children", func(t *testing.T, s *scaled) sitemodel.MapRequest {
			return sitemodel.MapRequest{ParentID: s.id(t, "/"), Limit: mapLimit}
		}),
		siteMap("a deep node's children", func(t *testing.T, s *scaled) sitemodel.MapRequest {
			return sitemodel.MapRequest{ParentID: s.id(t, "/c010/s1/t1/")}
		}),
		siteMap("a store category's children", func(t *testing.T, s *scaled) sitemodel.MapRequest {
			return sitemodel.MapRequest{ParentID: s.id(t, "/product-category/c010/s1/")}
		}),
		siteMap("a search", func(*testing.T, *scaled) sitemodel.MapRequest { return sitemodel.MapRequest{Search: "article"} }),
		siteMap("a search under a hub", func(t *testing.T, s *scaled) sitemodel.MapRequest {
			return sitemodel.MapRequest{ParentID: s.id(t, "/c010/"), Search: "article"}
		}),
		siteMap("a type filter", func(t *testing.T, s *scaled) sitemodel.MapRequest {
			return sitemodel.MapRequest{PageTypeID: s.typeOf(t, pagetype.Product)}
		}),
		siteMap("a second page", func(t *testing.T, s *scaled) sitemodel.MapRequest {
			req := sitemodel.MapRequest{SiteID: s.siteID, ParentID: s.id(t, "/c010/s1/"), Limit: 3}
			first, err := s.core.SiteModel.SiteMap(t.Context(), req)
			if err != nil || !first.Nodes.HasMore {
				t.Fatalf("the first page of a hub = %+v, %v", first.Nodes, err)
			}
			req.Cursor = first.Nodes.Next
			return req
		}),
		findingsList("by code", func(*testing.T, *scaled) findings.ListRequest {
			return findings.ListRequest{Code: string(finding.PageWithoutEntity), Limit: listLimit}
		}),
		findingsList("by page type", func(t *testing.T, s *scaled) findings.ListRequest {
			return findings.ListRequest{PageTypeID: s.typeOf(t, pagetype.Entity), Limit: listLimit}
		}),
		findingsList("under a hub", func(t *testing.T, s *scaled) findings.ListRequest {
			return findings.ListRequest{Under: s.id(t, "/c010/"), Limit: listLimit}
		}),
		findingsList("by status", func(*testing.T, *scaled) findings.ListRequest {
			return findings.ListRequest{Status: string(finding.Open), Limit: listLimit}
		}),
		findingsList("a second page", func(t *testing.T, s *scaled) findings.ListRequest {
			req := findings.ListRequest{SiteID: s.siteID, Code: string(finding.PageWithoutEntity), Limit: listLimit}
			first, err := s.core.Findings.List(t.Context(), req)
			if err != nil || !first.HasMore {
				t.Fatalf("the first page of findings = %d, more %v, %v", len(first.Items), first.HasMore, err)
			}
			req.Cursor = string(first.Next)
			return req
		}),
		{name: "pack: an article", call: func(t *testing.T, s *scaled) func() error {
			pageID := s.id(t, "/c010/s1/t1/p0/")
			return func() error {
				_, err := s.core.SiteModel.Pack(t.Context(), sitemodel.PackRequest{PageID: pageID})
				return err
			}
		}},
		{name: "estimate: five articles", call: func(t *testing.T, s *scaled) func() error {
			targets := []string{
				s.id(t, "/c010/s1/t1/p0/"), s.id(t, "/c010/s1/t1/p1/"), s.id(t, "/c011/s0/t0/p0/"),
				s.id(t, "/c012/s1/t0/p1/"), s.id(t, "/c013/s0/t1/p0/"),
			}
			return func() error {
				_, err := s.core.Runs.Estimate(t.Context(), runs.StartRequest{SiteID: s.siteID, PageIDs: targets})
				return err
			}
		}},
	}
}

func TestTheCountsHoldFromTwoThousandToFiveThousandPages(t *testing.T) {
	sizes := []int{smallBlocks, largeBlocks}
	var (
		mu      sync.Mutex
		counted = map[int]map[string]measured{}
	)
	t.Run("sizes", func(t *testing.T) {
		for _, blocks := range sizes {
			t.Run(strconv.Itoa(blocks*blockPages)+" pages", func(t *testing.T) {
				t.Parallel()

				s := generate(t, blocks)
				got := map[string]measured{}
				for _, held := range operations() {
					got[held.name] = s.measure(t, held.name, held.call(t, s))
				}
				mu.Lock()
				defer mu.Unlock()
				counted[blocks] = got
			})
		}
	})
	if t.Failed() {
		return
	}
	for _, held := range operations() {
		flat(t, held.name, counted[smallBlocks][held.name], counted[largeBlocks][held.name], held.rows)
	}
}
