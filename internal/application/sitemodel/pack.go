package sitemodel

import (
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Claim struct {
	Type  *pagetype.Type
	Scope field.Scope
	Links *content.LinkContext
}

type PackRequest struct {
	PageID string `json:"pageId"`
	Field  string `json:"field"`
	Budget int    `json:"budget"`
	Claim  *Claim `json:"-"`
}

type Picture struct {
	Facts  contextpack.Facts
	Pack   contextpack.Pack
	Fitted contextpack.Fitted
	Links  content.LinkContext
	Scope  field.Scope
}

func (s *Service) Picture(ctx context.Context, req PackRequest) (Picture, error) {
	if req.PageID == "" {
		return Picture{}, errors.New(errors.Invalid, "a pack is built for a page").WithDetail("field", "pageId")
	}
	var picture Picture
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		facts, scope, plan, err := s.read(c, req.PageID, req.Field, req.Claim)
		if err != nil {
			return err
		}
		pack := contextpack.Build(facts, scope)
		fitted, err := contextpack.Fit(pack, req.Budget)
		if err != nil {
			return err
		}
		picture = Picture{Facts: facts, Pack: pack, Fitted: fitted, Links: plan, Scope: scope}
		return nil
	})
	return picture, err
}

func (s *Service) read(ctx context.Context, pageID, fieldName string, claim *Claim) (contextpack.Facts, field.Scope, content.LinkContext, error) {
	snapshot, page, err := s.around(ctx, pageID)
	if err != nil {
		return contextpack.Facts{}, field.Scope{}, content.LinkContext{}, err
	}
	key, err := fieldOf(fieldName, page, snapshot.Types)
	if err != nil {
		return contextpack.Facts{}, field.Scope{}, content.LinkContext{}, err
	}
	scope, plan := s.claimed(snapshot, page, key, claim)
	if fillErr := s.fill(ctx, &snapshot, page, plan); fillErr != nil {
		return contextpack.Facts{}, field.Scope{}, content.LinkContext{}, fillErr
	}
	facts, err := snapshot.Facts(page.ID, plan)
	if err != nil {
		return contextpack.Facts{}, field.Scope{}, content.LinkContext{}, err
	}
	if claim != nil && claim.Type != nil {
		facts.Page.Type, facts.Page.Preset = claim.Type.Name, claim.Type.Preset
	}
	return facts, scope, plan, nil
}

func (s *Service) Pack(ctx context.Context, req PackRequest) (PackView, error) {
	req.Claim = nil
	if req.Budget <= 0 {
		req.Budget = s.deps.Budgets.InputBudget(roleOf(field.Key(req.Field)))
	}
	picture, err := s.Picture(ctx, req)
	if err != nil {
		return PackView{}, err
	}
	return packView(picture), nil
}

func roleOf(key field.Key) llm.Role {
	switch key {
	case field.SEOTitle, field.SEODescription, field.Excerpt:
		return llm.RoleEditor
	default:
		return llm.RoleWriter
	}
}

func fieldOf(name string, page pagemap.Page, types pagetype.Set) (field.Key, error) {
	if name == "" {
		return pagetype.LinkField(types.TypeOf(page).Preset), nil
	}
	key := field.Key(name)
	spec, known := field.Lookup(key.Kind())
	switch {
	case !known:
		return "", errors.New(errors.Invalid, "the field is not in the catalog").WithDetail("field", "field").WithDetail("value", name)
	case spec.Denied || !spec.Text:
		return "", errors.New(errors.Invalid, "the writer never writes this field").WithDetail("field", "field").WithDetail("value", name)
	default:
		return key, nil
	}
}

func (s *Service) claimed(snapshot contextpack.Snapshot, page pagemap.Page, key field.Key, claim *Claim) (field.Scope, content.LinkContext) {
	scope, _ := snapshot.Types.Scope(snapshot.Types.TypeOf(page).ID, key)
	if claim != nil {
		scope = claim.Scope
		if claim.Links != nil {
			return scope, *claim.Links
		}
	}
	policy := scope.Links
	if !scope.Enabled {
		policy = field.NoLinks()
	}
	subject := content.Subject{Site: snapshot.Site, PageID: page.ID, PagePath: page.Path, Absolute: snapshot.Absolute}
	if page.EntityID != nil {
		subject.EntityID = *page.EntityID
	}
	return scope, content.PlanLinks(snapshot.Graph, snapshot.Index, subject, policy).Context
}

func (s *Service) around(ctx context.Context, pageID string) (contextpack.Snapshot, pagemap.Page, error) {
	page, err := s.deps.Pages.Get(ctx, pageID)
	if err != nil {
		return contextpack.Snapshot{}, pagemap.Page{}, err
	}
	snapshot, err := s.base(ctx, page.SiteID)
	if err != nil {
		return contextpack.Snapshot{}, pagemap.Page{}, err
	}
	own, err := s.deps.Links.ListFromPages(ctx, []string{page.ID})
	if err != nil {
		return contextpack.Snapshot{}, pagemap.Page{}, err
	}
	snapshot.Links = own
	return snapshot, page, nil
}

func (s *Service) base(ctx context.Context, siteID string) (contextpack.Snapshot, error) {
	owner, err := s.deps.Sites.Get(ctx, siteID)
	if err != nil {
		return contextpack.Snapshot{}, err
	}
	pages, err := s.deps.Pages.ListBySite(ctx, siteID)
	if err != nil {
		return contextpack.Snapshot{}, err
	}
	entities, err := s.deps.Entities.ListBySite(ctx, siteID)
	if err != nil {
		return contextpack.Snapshot{}, err
	}
	edges, err := s.deps.Edges.ListBySite(ctx, siteID)
	if err != nil {
		return contextpack.Snapshot{}, err
	}
	g, err := graph.New(entities, edges)
	if err != nil {
		return contextpack.Snapshot{}, err
	}
	types, err := s.deps.Types.Set(ctx, siteID)
	if err != nil {
		return contextpack.Snapshot{}, err
	}
	terms, err := s.deps.Terms.ListTermsBySite(ctx, siteID)
	if err != nil {
		return contextpack.Snapshot{}, err
	}
	absolute, err := s.absolute(ctx, owner)
	if err != nil {
		return contextpack.Snapshot{}, err
	}
	return contextpack.Snapshot{
		Site: pagemap.NewSite(owner.BaseURL), Absolute: absolute, Index: pagemap.NewIndex(pages), Types: types, Graph: g,
		Terms: terms, Fields: map[string][]field.Value{}, Bodies: map[string]block.Snapshot{}, Written: map[string][]string{},
		Renders: map[string]render.Record{}, Links: []pagemap.PageLink{}, Findings: []finding.Finding{},
	}, nil
}

func (s *Service) Absolute(ctx context.Context, siteID string) (bool, error) {
	var absolute bool
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		owner, err := s.deps.Sites.Get(c, siteID)
		if err != nil {
			return err
		}
		absolute, err = s.absolute(c, owner)
		return err
	})
	return absolute, err
}

func (s *Service) absolute(ctx context.Context, owner site.Site) (bool, error) {
	contextual, err := s.deps.Links.ListContextual(ctx, owner.ID)
	if err != nil {
		return false, err
	}
	forms, err := s.deps.Renders.URLForms(ctx, owner.ID)
	if err != nil {
		return false, err
	}
	return contextpack.LinkForm(owner.Settings.LinkURLForm, contextual, forms) == site.URLFormAbsolute, nil
}

func (s *Service) fill(ctx context.Context, snapshot *contextpack.Snapshot, page pagemap.Page, plan content.LinkContext) error {
	ids := snapshot.Neighborhood(page.ID, plan)
	from, err := s.deps.Links.ListFromPages(ctx, ids)
	if err != nil {
		return err
	}
	to, err := s.deps.Links.ListToPage(ctx, page.ID)
	if err != nil {
		return err
	}
	snapshot.Links = mergeLinks(from, to)
	if snapshot.Fields, err = s.deps.Fields.ListForPages(ctx, ids); err != nil {
		return err
	}
	bodies, err := s.deps.Bodies.ListCurrent(ctx, ids)
	if err != nil {
		return err
	}
	for i := range bodies {
		snapshot.Bodies[bodies[i].PageID] = bodies[i]
	}
	written, err := s.deps.Bodies.ListWritten(ctx, page.ID)
	if err != nil {
		return err
	}
	for i := range written {
		snapshot.Written[page.ID] = append(snapshot.Written[page.ID], written[i].BlockHash)
	}
	renders, err := s.deps.Renders.ListForPages(ctx, ids)
	if err != nil {
		return err
	}
	for i := range renders {
		snapshot.Renders[renders[i].PageID] = renders[i]
	}
	findings, err := s.deps.Findings.ListForPage(ctx, page.ID)
	if err != nil {
		return err
	}
	snapshot.Findings = slices.DeleteFunc(findings, func(f finding.Finding) bool { return f.Status != finding.Open })
	return nil
}

func mergeLinks(from, to []pagemap.PageLink) []pagemap.PageLink {
	out := slices.Clone(from)
	held := make(map[string]struct{}, len(from))
	for i := range from {
		held[from[i].ID] = struct{}{}
	}
	for i := range to {
		if _, dup := held[to[i].ID]; !dup {
			out = append(out, to[i])
		}
	}
	return out
}
