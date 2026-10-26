//go:build !race

package scale_test

import (
	"strconv"
	"sync"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/dto"
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

func pagesList(name string, request func(t *testing.T, s *scaled) pages.ListRequest) operation {
	return operation{name: "pages: " + name, rows: true, call: func(t *testing.T, s *scaled) func() error {
		req := request(t, s)
		req.SiteID, req.Limit = s.siteID, listLimit
		if req.Sort == nil {
			req.Sort = &dto.Sort{Field: string(pagemap.SortPath)}
		}
		return func() error {
			_, err := s.core.Pages.List(t.Context(), req)
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
		siteMap("the picker's level", func(t *testing.T, s *scaled) sitemodel.MapRequest {
			return sitemodel.MapRequest{ParentID: s.id(t, "/c010/s0/")}
		}),
		{name: "picker: the parents written first", rows: true, call: func(t *testing.T, s *scaled) func() error {
			selected := []string{
				s.id(t, "/c010/s0/t0/new-0/"), s.id(t, "/c010/s0/t0/p0/"), s.id(t, "/c011/s1/t1/p1/"), s.id(t, "/c012/s0/"),
				s.id(t, "/product/item-013-1-4/"),
			}
			return func() error {
				_, err := s.core.Runs.Parents(t.Context(), runs.ParentsRequest{SiteID: s.siteID, PageIDs: selected})
				return err
			}
		}},
		findingsList("by code", func(*testing.T, *scaled) findings.ListRequest {
			return findings.ListRequest{Code: string(finding.PageWithoutEntity), Limit: listLimit}
		}),
		findingsList("by page type", func(t *testing.T, s *scaled) findings.ListRequest {
			return findings.ListRequest{PageTypeID: s.typeOf(t, pagetype.Guide), Limit: listLimit}
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
		{name: "classify: the facts", call: func(t *testing.T, s *scaled) func() error {
			return func() error {
				_, err := s.core.SiteModel.ClassificationFacts(t.Context(), s.siteID)
				return err
			}
		}},
		{name: "classify: the preview", call: func(t *testing.T, s *scaled) func() error {
			return func() error {
				_, err := s.core.Classification.Preview(t.Context(), pagetypes.PreviewRequest{SiteID: s.siteID})
				return err
			}
		}},
		pagesList("a type's first page", func(t *testing.T, s *scaled) pages.ListRequest {
			return pages.ListRequest{PageFilter: pages.PageFilter{PageTypeID: s.typeOf(t, pagetype.Product)}}
		}),
		pagesList("a search", func(*testing.T, *scaled) pages.ListRequest {
			return pages.ListRequest{PageFilter: pages.PageFilter{Search: "article"}}
		}),
		pagesList("by title", func(*testing.T, *scaled) pages.ListRequest {
			return pages.ListRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: string(pagemap.SortTitle)}}}
		}),
		pagesList("under a hub", func(t *testing.T, s *scaled) pages.ListRequest {
			return pages.ListRequest{PageFilter: pages.PageFilter{Under: s.id(t, "/c010/")}}
		}),
		{name: "pages: a type's count", rows: true, call: func(t *testing.T, s *scaled) func() error {
			req := pages.CountRequest{SiteID: s.siteID, PageFilter: pages.PageFilter{PageTypeID: s.typeOf(t, pagetype.Guide)}}
			return func() error {
				_, err := s.core.Pages.Count(t.Context(), req)
				return err
			}
		}},
		{name: "pages: a hub's subtree selected", rows: true, call: func(t *testing.T, s *scaled) func() error {
			selection := pages.Selection{SiteID: s.siteID, Subtrees: []pages.Subtree{{PageID: s.id(t, "/c010/")}}}
			return func() error {
				_, err := s.core.Pages.Resolve(t.Context(), selection)
				return err
			}
		}},
		siteMap("the roots within a type", func(t *testing.T, s *scaled) sitemodel.MapRequest {
			return sitemodel.MapRequest{Within: s.typeOf(t, pagetype.Guide), Limit: mapLimit}
		}),
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

type walked struct {
	fixed   float64
	perPage float64
}

func (s *scaled) walked(t *testing.T) walked {
	t.Helper()

	five := s.once(t, "walk: 5% touched", func() error {
		s.walk(t, s.plan.touched(false))
		return nil
	})
	ten := s.once(t, "walk: 10% touched", func() error {
		s.walk(t, s.plan.touched(true))
		return nil
	})
	touchedFive, touchedTen := len(s.plan.touched(false)), len(s.plan.touched(true))
	out := walked{perPage: float64(ten.Statements-five.Statements) / float64(touchedTen-touchedFive)}
	out.fixed = float64(five.Statements) - out.perPage*float64(touchedFive)
	t.Logf("walk at %d pages: %.1f fixed statements, %.2f per touched page", s.pages(), out.fixed, out.perPage)
	return out
}

func TestTheCountsHoldFromTwoThousandToFiveThousandPages(t *testing.T) {
	sizes := []int{smallBlocks, largeBlocks}
	var (
		mu      sync.Mutex
		counted = map[int]map[string]measured{}
		walks   = map[int]walked{}
		applied = map[int]map[string]measured{}
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
				walk := s.walked(t)
				retyped := s.retyped(t)
				s.scansNothingLarge(t)
				mu.Lock()
				defer mu.Unlock()
				counted[blocks], walks[blocks], applied[blocks] = got, walk, retyped
			})
		}
	})
	if t.Failed() {
		return
	}
	for _, held := range operations() {
		flat(t, held.name, counted[smallBlocks][held.name], counted[largeBlocks][held.name], held.rows)
	}
	for name, small := range applied[smallBlocks] {
		flat(t, name, small, applied[largeBlocks][name], false)
	}
	if small, large := walks[smallBlocks], walks[largeBlocks]; small != large {
		t.Errorf("a walk runs %.1f + %.2f statements per touched page at %d pages and %.1f + %.2f at %d",
			small.fixed, small.perPage, smallBlocks*blockPages, large.fixed, large.perPage, largeBlocks*blockPages)
	}
}
