package sitemodel

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type pageName struct {
	SiteID string
	PageID string
	Path   string
}

type PackTextRequest struct {
	SiteID string
	PageID string
	Path   string
	Field  string
	Budget int
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

func (s *Service) PackText(ctx context.Context, req PackTextRequest) (PackText, error) {
	page, err := s.namedOutside(ctx, pageName{SiteID: req.SiteID, PageID: req.PageID, Path: req.Path})
	if err != nil {
		return PackText{}, err
	}
	budget := req.Budget
	if budget <= 0 {
		budget = s.deps.Budgets.InputBudget(roleOf(field.Key(req.Field)))
	}
	picture, err := s.Picture(ctx, PackRequest{PageID: page.ID, Field: req.Field, Budget: budget})
	if err != nil {
		return PackText{}, err
	}
	view := packView(picture)
	return PackText{
		PageID: view.PageID, Path: view.Path, Field: view.Field, Budget: view.Budget, Tokens: view.Tokens,
		Text: picture.Fitted.Text(), Dropped: view.Dropped,
	}, nil
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

func (s *Service) named(ctx context.Context, req pageName) (pagemap.Page, error) {
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
	default:
		return pagemap.Page{}, errors.New(errors.Invalid, "name the page: give pageId or path").WithDetail("field", "pageId")
	}
}

func (s *Service) namedOutside(ctx context.Context, req pageName) (pagemap.Page, error) {
	var page pagemap.Page
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		var err error
		page, err = s.named(c, req)
		return err
	})
	return page, err
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
