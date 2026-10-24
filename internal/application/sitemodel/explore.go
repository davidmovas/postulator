package sitemodel

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type View string

const (
	ViewTree  View = "tree"
	ViewLinks View = "links"
	ViewGaps  View = "gaps"
	ViewPack  View = "pack"

	exploreDefaultLimit = 50
	exploreMaxLimit     = 200
)

type ExploreRequest struct {
	SiteID string          `json:"siteId"`
	View   View            `json:"view"`
	PageID string          `json:"pageId"`
	Path   string          `json:"path"`
	Field  string          `json:"field"`
	Gap    contextpack.Gap `json:"gap"`
	Cursor paging.Cursor   `json:"cursor"`
	Limit  int             `json:"limit"`
	Budget int             `json:"budget"`
}

type ExploreResponse struct {
	View  View                             `json:"view"`
	Tree  *TreeView                        `json:"tree,omitempty"`
	Links *reports.LinkAuditPageResponse   `json:"links,omitempty"`
	Gaps  *paging.List[contextpack.GapRow] `json:"gaps,omitempty"`
	Pack  *PackText                        `json:"pack,omitempty"`
}

type TreeView struct {
	Parent *contextpack.Ref              `json:"parent"`
	Nodes  paging.List[contextpack.Node] `json:"nodes"`
}

type PackText struct {
	PageID  string `json:"pageId"`
	Path    string `json:"path"`
	Field   string `json:"field"`
	Budget  int    `json:"budget"`
	Tokens  int    `json:"tokens"`
	Text    string `json:"text"`
	Dropped []Drop `json:"dropped"`
}

func (s *Service) Explore(ctx context.Context, req ExploreRequest) (ExploreResponse, error) {
	switch req.View {
	case ViewTree:
		return s.exploreTree(ctx, req)
	case ViewLinks:
		return s.exploreLinks(ctx, req)
	case ViewGaps:
		return s.exploreGaps(ctx, req)
	case ViewPack:
		return s.explorePack(ctx, req)
	default:
		return ExploreResponse{}, errors.New(errors.Invalid, "the view must be tree, links, gaps or pack; a page is read by pages_get").
			WithDetail("field", "view").WithDetail("value", string(req.View))
	}
}

func (s *Service) exploreLinks(ctx context.Context, req ExploreRequest) (ExploreResponse, error) {
	var audit reports.LinkAuditPageResponse
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		page, err := s.named(c, req, true)
		if err != nil {
			return err
		}
		audit, err = s.deps.Audits.LinkAuditPage(c, reports.LinkAuditPageRequest{PageID: page.ID})
		return err
	})
	return ExploreResponse{View: ViewLinks, Links: &audit}, err
}

func (s *Service) explorePack(ctx context.Context, req ExploreRequest) (ExploreResponse, error) {
	page, err := s.namedOutside(ctx, req)
	if err != nil {
		return ExploreResponse{}, err
	}
	budget := req.Budget
	if budget <= 0 {
		budget = s.deps.Budgets.InputBudget(roleOf(field.Key(req.Field)))
	}
	picture, err := s.Picture(ctx, PackRequest{PageID: page.ID, Field: req.Field, Budget: budget})
	if err != nil {
		return ExploreResponse{}, err
	}
	view := packView(picture)
	return ExploreResponse{View: ViewPack, Pack: &PackText{
		PageID: view.PageID, Path: view.Path, Field: view.Field, Budget: view.Budget, Tokens: view.Tokens,
		Text: picture.Fitted.Text(), Dropped: view.Dropped,
	}}, nil
}

func (s *Service) exploreTree(ctx context.Context, req ExploreRequest) (ExploreResponse, error) {
	var tree TreeView
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		owner, err := s.deps.Sites.Get(c, req.SiteID)
		if err != nil {
			return err
		}
		snapshot, err := s.base(c, owner.ID)
		if err != nil {
			return err
		}
		parentID := ""
		if req.PageID != "" || req.Path != "" {
			parent, namedErr := s.named(c, req, false)
			if namedErr != nil {
				return namedErr
			}
			ref := snapshot.Ref(parent)
			tree.Parent, parentID = &ref, parent.ID
		}
		tree.Nodes, err = rows(snapshot.Nodes(parentID), func(node contextpack.Node) contextpack.Ref { return node.Page }, req)
		return err
	})
	return ExploreResponse{View: ViewTree, Tree: &tree}, err
}

func (s *Service) exploreGaps(ctx context.Context, req ExploreRequest) (ExploreResponse, error) {
	if req.Gap != "" && !slices.Contains(contextpack.Gaps(), req.Gap) {
		return ExploreResponse{}, errors.New(errors.Invalid, "the gap must be one of planned, empty, unmapped, orphan, missing_links or findings").
			WithDetail("field", "gap").WithDetail("value", string(req.Gap))
	}
	var listed paging.List[contextpack.GapRow]
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		snapshot, audit, _, err := s.whole(c, req.SiteID)
		if err != nil {
			return err
		}
		found := snapshot.Gaps(linkFacts(audit))
		if req.Gap != "" {
			found = slices.DeleteFunc(found, func(row contextpack.GapRow) bool {
				return !slices.Contains(row.Gaps, req.Gap)
			})
		}
		listed, err = rows(found, func(row contextpack.GapRow) contextpack.Ref { return row.Page }, req)
		return err
	})
	return ExploreResponse{View: ViewGaps, Gaps: &listed}, err
}

func (s *Service) whole(ctx context.Context, siteID string) (contextpack.Snapshot, reports.LinkAuditResponse, site.Site, error) {
	owner, err := s.deps.Sites.Get(ctx, siteID)
	if err != nil {
		return contextpack.Snapshot{}, reports.LinkAuditResponse{}, site.Site{}, err
	}
	snapshot, err := s.base(ctx, owner.ID)
	if err != nil {
		return contextpack.Snapshot{}, reports.LinkAuditResponse{}, site.Site{}, err
	}
	bodies, err := s.deps.Bodies.ListBySite(ctx, owner.ID)
	if err != nil {
		return contextpack.Snapshot{}, reports.LinkAuditResponse{}, site.Site{}, err
	}
	for i := range bodies {
		if bodies[i].Side == block.Current {
			snapshot.Bodies[bodies[i].PageID] = bodies[i]
		}
	}
	if snapshot.Findings, err = s.deps.Findings.ListBySite(ctx, owner.ID, finding.Open); err != nil {
		return contextpack.Snapshot{}, reports.LinkAuditResponse{}, site.Site{}, err
	}
	audit, err := s.deps.Audits.LinkAudit(ctx, reports.LinkAuditRequest{SiteID: owner.ID})
	if err != nil {
		return contextpack.Snapshot{}, reports.LinkAuditResponse{}, site.Site{}, err
	}
	return snapshot, audit, owner, nil
}

func linkFacts(audit reports.LinkAuditResponse) map[string]contextpack.LinkFacts {
	out := make(map[string]contextpack.LinkFacts, len(audit.Pages))
	for i := range audit.Pages {
		row := &audit.Pages[i]
		out[row.PageID] = contextpack.LinkFacts{
			Orphan: row.Orphan, MissingRequired: row.MissingRequired, Inbound: row.Inbound, Outbound: row.ByType.Contextual,
		}
	}
	return out
}

func (s *Service) named(ctx context.Context, req ExploreRequest, needed bool) (pagemap.Page, error) {
	switch {
	case req.PageID != "":
		page, err := s.deps.Pages.Get(ctx, req.PageID)
		if err != nil {
			return pagemap.Page{}, err
		}
		if page.SiteID != req.SiteID {
			return pagemap.Page{}, errors.New(errors.NotFound, "page not found").WithDetail("pageId", req.PageID)
		}
		return page, nil
	case req.Path != "":
		path, err := pagemap.NormalizePath(strings.TrimSpace(req.Path))
		if err != nil {
			return pagemap.Page{}, errors.Wrap(err, errors.Invalid, "the path is not a page's path")
		}
		pages, err := s.deps.Pages.ListBySite(ctx, req.SiteID)
		if err != nil {
			return pagemap.Page{}, err
		}
		if page, found := pagemap.NewIndex(pages).ByPath(path); found {
			return page, nil
		}
		return pagemap.Page{}, errors.New(errors.NotFound, "no page has this path").WithDetail("path", req.Path)
	case needed:
		return pagemap.Page{}, errors.New(errors.Invalid, "the "+string(req.View)+" view needs a page: give pageId or path").WithDetail("field", "pageId")
	default:
		return pagemap.Page{}, nil
	}
}

func (s *Service) namedOutside(ctx context.Context, req ExploreRequest) (pagemap.Page, error) {
	var page pagemap.Page
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		var err error
		page, err = s.named(c, req, true)
		return err
	})
	return page, err
}

func rows[T any](all []T, ref func(T) contextpack.Ref, req ExploreRequest) (paging.List[T], error) {
	return byPath(all, ref, req.Cursor, clamped(req.Limit, exploreDefaultLimit, exploreMaxLimit))
}

func byPath[T any](all []T, ref func(T) contextpack.Ref, cursor paging.Cursor, limit int) (paging.List[T], error) {
	keyset := paging.Keyset[T]{
		IDColumn: "id",
		ID:       func(row T) string { return ref(row).PageID },
		Keys: []paging.SortKey[T]{
			paging.TextKey[T]("path", "path", func(row T) any { return ref(row).Path }),
		},
	}
	return keyset.Page(all, paging.Request{After: cursor, Limit: limit})
}

func clamped(limit, fallback, ceiling int) int {
	switch {
	case limit <= 0:
		return fallback
	case limit > ceiling:
		return ceiling
	default:
		return limit
	}
}
