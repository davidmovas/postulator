package sitemodel

import (
	"cmp"
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type LinksShape string

const (
	LinksPage    LinksShape = "page"
	LinksSection LinksShape = "section"
	LinksMap     LinksShape = "map"
	LinksOwed    LinksShape = "owed"
	LinksOrphans LinksShape = "orphans"
	LinksSuggest LinksShape = "suggest"

	SuggestParentEdge = "parent_edge"
	SuggestLink       = "link"

	linksDefaultLimit = 50
	linksMaxLimit     = 200
)

func LinksShapes() []LinksShape {
	return []LinksShape{LinksPage, LinksSection, LinksMap, LinksOwed, LinksOrphans, LinksSuggest}
}

type LinksRequest struct {
	SiteID  string        `json:"siteId"`
	View    LinksShape    `json:"view"`
	PageID  string        `json:"pageId,omitempty"`
	Cursor  paging.Cursor `json:"cursor,omitempty"`
	Limit   int           `json:"limit,omitempty"`
	Missing bool          `json:"missing,omitempty"`
}

type SectionTotals struct {
	Pages           int `json:"pages"`
	Required        int `json:"required"`
	Satisfied       int `json:"satisfied"`
	MissingRequired int `json:"missingRequired"`
	OffGraph        int `json:"offGraph"`
	Orphans         int `json:"orphans"`
}

type InboundLink struct {
	ID         string `json:"id"`
	FromPageID string `json:"fromPageId"`
	FromPath   string `json:"fromPath"`
	Anchor     string `json:"anchor"`
	Ours       bool   `json:"ours"`
}

type SectionLink struct {
	ID         string `json:"id"`
	FromPageID string `json:"fromPageId"`
	FromPath   string `json:"fromPath"`
	ToPageID   string `json:"toPageId"`
	ToPath     string `json:"toPath"`
	Anchor     string `json:"anchor"`
	Ours       bool   `json:"ours"`
	Inside     bool   `json:"inside"`
}

type OwedLink struct {
	FromPageID string `json:"fromPageId"`
	FromPath   string `json:"fromPath"`
	reports.RequiredLink
}

type Suggestion struct {
	Kind          string   `json:"kind"`
	EntityID      string   `json:"entityId,omitempty"`
	EntityName    string   `json:"entityName,omitempty"`
	GraphParents  []string `json:"graphParents,omitempty"`
	ToEntityID    string   `json:"toEntityId,omitempty"`
	ToEntityName  string   `json:"toEntityName,omitempty"`
	FromPageID    string   `json:"fromPageId"`
	FromPath      string   `json:"fromPath"`
	ToPageID      string   `json:"toPageId,omitempty"`
	ToPath        string   `json:"toPath,omitempty"`
	Relation      string   `json:"relation,omitempty"`
	Anchors       []string `json:"anchors,omitempty"`
	BlockedReason string   `json:"blockedReason,omitempty"`
}

type LinksAnswer struct {
	View        LinksShape                      `json:"view"`
	Root        *contextpack.Ref                `json:"root,omitempty"`
	Totals      *SectionTotals                  `json:"totals,omitempty"`
	Page        *reports.PageLinks              `json:"page,omitempty"`
	Inbound     *paging.List[InboundLink]       `json:"inbound,omitempty"`
	Pages       *paging.List[reports.PageAudit] `json:"pages,omitempty"`
	Links       *paging.List[SectionLink]       `json:"links,omitempty"`
	Owed        *paging.List[OwedLink]          `json:"owed,omitempty"`
	Suggestions *paging.List[Suggestion]        `json:"suggestions,omitempty"`
}

type section struct {
	snapshot contextpack.Snapshot
	ids      []string
	inside   map[string]struct{}
	audited  []reports.PageLinks
}

func (s *Service) Links(ctx context.Context, req LinksRequest) (LinksAnswer, error) {
	switch {
	case req.SiteID == "":
		return LinksAnswer{}, errors.New(errors.Invalid, "name the site whose links to read").WithDetail("field", "siteId")
	case !slices.Contains(LinksShapes(), req.View):
		return LinksAnswer{}, errors.New(errors.Invalid, "the view must be page, section, map, owed, orphans or suggest").
			WithDetail("field", "view").WithDetail("value", string(req.View))
	case req.View == LinksPage && req.PageID == "":
		return LinksAnswer{}, errors.New(errors.Invalid, "the page view needs the page").WithDetail("field", "pageId")
	}
	var answer LinksAnswer
	err := s.onMap(ctx, req.SiteID, func(c context.Context) error {
		read, root, err := s.section(c, req)
		if err != nil {
			return err
		}
		answer = LinksAnswer{View: req.View, Root: root}
		page := paging.Request{After: req.Cursor, Limit: clamped(req.Limit, linksDefaultLimit, linksMaxLimit)}
		switch req.View {
		case LinksPage:
			return s.pageLinks(c, read, req.PageID, page, &answer)
		case LinksSection, LinksOrphans:
			return sectionPages(read, req.View == LinksOrphans, page, &answer)
		case LinksMap:
			return s.sectionLinks(c, read, page, &answer)
		case LinksOwed:
			listed, owedErr := byKey(owedLinks(read, req.Missing), func(row OwedLink) string { return row.FromPath },
				func(row OwedLink) string { return row.FromPageID + ":" + row.TargetEntityID + ":" + row.Relation }, page)
			answer.Owed = &listed
			return owedErr
		default:
			listed, suggestErr := byKey(suggestions(read), func(row Suggestion) string { return row.Kind + "\x00" + row.FromPath },
				func(row Suggestion) string {
					return row.FromPageID + ":" + row.EntityID + ":" + row.ToPageID + ":" + row.Relation
				}, page)
			answer.Suggestions = &listed
			return suggestErr
		}
	})
	return answer, err
}

func (s *Service) section(ctx context.Context, req LinksRequest) (section, *contextpack.Ref, error) {
	snapshot, err := s.base(ctx, req.SiteID)
	if err != nil {
		return section{}, nil, err
	}
	read := section{snapshot: snapshot, inside: map[string]struct{}{}}
	var root *contextpack.Ref
	if req.PageID != "" {
		row, rowErr := s.nodeRow(ctx, req.SiteID, req.PageID, "pageId")
		if rowErr != nil {
			return section{}, nil, rowErr
		}
		if read.ids, err = s.deps.Maps.IDsUnder(ctx, req.SiteID, row.Trail, ""); err != nil {
			return section{}, nil, err
		}
		if page, found := snapshot.Index.ByID(row.PageID); found {
			ref := snapshot.Ref(page)
			root = &ref
		}
	} else {
		pages := snapshot.Index.Pages()
		for i := range pages {
			if pages[i].Status != pagemap.StatusArchived {
				read.ids = append(read.ids, pages[i].ID)
			}
		}
	}
	for _, pageID := range read.ids {
		read.inside[pageID] = struct{}{}
	}
	audited, err := s.deps.Audits.LinkAuditPages(ctx, reports.LinkAuditPagesRequest{SiteID: req.SiteID, PageIDs: read.ids})
	if err != nil {
		return section{}, nil, err
	}
	read.audited = audited.Pages
	slices.SortFunc(read.audited, func(a, b reports.PageLinks) int {
		return cmp.Or(cmp.Compare(a.Page.Path, b.Page.Path), cmp.Compare(a.Page.PageID, b.Page.PageID))
	})
	return read, root, nil
}

func sectionPages(read section, orphansOnly bool, page paging.Request, answer *LinksAnswer) error {
	totals := SectionTotals{}
	rows := make([]reports.PageAudit, 0, len(read.audited))
	for i := range read.audited {
		row := read.audited[i].Page
		totals.Pages++
		totals.Required += row.Required
		totals.Satisfied += row.Satisfied
		totals.MissingRequired += row.MissingRequired
		totals.OffGraph += row.OffGraph
		if row.Orphan {
			totals.Orphans++
		}
		if !orphansOnly || row.Orphan {
			rows = append(rows, row)
		}
	}
	listed, err := byKey(rows, func(row reports.PageAudit) string { return row.Path }, func(row reports.PageAudit) string { return row.PageID }, page)
	answer.Totals, answer.Pages = &totals, &listed
	return err
}

func (s *Service) pageLinks(ctx context.Context, read section, pageID string, page paging.Request, answer *LinksAnswer) error {
	for i := range read.audited {
		if read.audited[i].Page.PageID == pageID {
			held := read.audited[i]
			answer.Page = &held
		}
	}
	if answer.Page == nil {
		return errors.New(errors.NotFound, "the site holds no live page of this id").WithDetail("pageId", pageID)
	}
	into, err := s.deps.Links.ListToPage(ctx, pageID)
	if err != nil {
		return err
	}
	rows := make([]InboundLink, 0, len(into))
	for i := range into {
		link := into[i]
		from, found := read.snapshot.Index.ByID(link.FromPageID)
		if link.Type != pagemap.LinkContextual || !found || link.FromPageID == pageID {
			continue
		}
		rows = append(rows, InboundLink{ID: link.ID, FromPageID: from.ID, FromPath: from.Path, Anchor: link.AnchorText, Ours: link.Ours})
	}
	listed, err := byKey(rows, func(row InboundLink) string { return row.FromPath }, func(row InboundLink) string { return row.ID }, page)
	answer.Inbound = &listed
	return err
}

func (s *Service) sectionLinks(ctx context.Context, read section, page paging.Request, answer *LinksAnswer) error {
	held, err := s.deps.Links.ListFromPages(ctx, read.ids)
	if err != nil {
		return err
	}
	rows := make([]SectionLink, 0, len(held))
	for i := range held {
		link := held[i]
		if link.Type != pagemap.LinkContextual || link.ToPageID == nil || *link.ToPageID == link.FromPageID {
			continue
		}
		from, fromFound := read.snapshot.Index.ByID(link.FromPageID)
		to, toFound := read.snapshot.Index.ByID(*link.ToPageID)
		if !fromFound || !toFound {
			continue
		}
		_, inside := read.inside[to.ID]
		rows = append(rows, SectionLink{
			ID: link.ID, FromPageID: from.ID, FromPath: from.Path, ToPageID: to.ID, ToPath: to.Path, Anchor: link.AnchorText,
			Ours: link.Ours, Inside: inside,
		})
	}
	listed, err := byKey(rows, func(row SectionLink) string { return row.FromPath }, func(row SectionLink) string { return row.ID }, page)
	answer.Links = &listed
	return err
}

func owedLinks(read section, missingOnly bool) []OwedLink {
	out := make([]OwedLink, 0)
	for i := range read.audited {
		held := &read.audited[i]
		for j := range held.Required {
			required := &held.Required[j]
			if missingOnly && required.State != string(reports.LinkMissing) {
				continue
			}
			out = append(out, OwedLink{FromPageID: held.Page.PageID, FromPath: held.Page.Path, RequiredLink: *required})
		}
	}
	return out
}

func suggestions(read section) []Suggestion {
	disagreements := read.snapshot.Map(nil).ParentDisagreements(read.ids)
	owed := owedLinks(read, true)
	out := make([]Suggestion, 0, len(disagreements)+len(owed))
	for i := range disagreements {
		disagreement := &disagreements[i]
		entity, _ := read.snapshot.Graph.Entity(disagreement.EntityID)
		suggestion := Suggestion{
			Kind: SuggestParentEdge, EntityID: entity.ID, EntityName: entity.Name, GraphParents: disagreement.GraphParents,
			ToEntityID: disagreement.Tree.EntityID, FromPageID: disagreement.PageID, FromPath: disagreement.Path,
			ToPageID: disagreement.Tree.PageID, ToPath: disagreement.Tree.Path,
		}
		if parent, found := read.snapshot.Graph.Entity(disagreement.Tree.EntityID); found {
			suggestion.ToEntityName = parent.Name
		}
		out = append(out, suggestion)
	}
	for i := range owed {
		held := &owed[i]
		out = append(out, Suggestion{
			Kind: SuggestLink, EntityID: held.TargetEntityID, FromPageID: held.FromPageID, FromPath: held.FromPath,
			ToPageID: held.TargetPageID, ToPath: held.TargetPath, Relation: held.Relation, Anchors: held.AnchorsAllowed,
			BlockedReason: held.BlockedReason,
		})
	}
	return out
}

func byKey[T any](rows []T, key, id func(T) string, page paging.Request) (paging.List[T], error) {
	slices.SortFunc(rows, func(a, b T) int { return cmp.Or(cmp.Compare(key(a), key(b)), cmp.Compare(id(a), id(b))) })
	keyset := paging.Keyset[T]{
		IDColumn: "id",
		ID:       id,
		Keys:     []paging.SortKey[T]{paging.TextKey("key", "key", func(row T) any { return key(row) })},
	}
	return keyset.Page(rows, page)
}
