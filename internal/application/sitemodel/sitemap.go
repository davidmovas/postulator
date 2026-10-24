package sitemodel

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	mapDefaultLimit = 100
	mapMaxLimit     = 200
)

type MapRequest struct {
	SiteID     string        `json:"siteId"`
	ParentID   string        `json:"parentId,omitempty"`
	Search     string        `json:"search,omitempty"`
	PageTypeID string        `json:"pageTypeId,omitempty"`
	Cursor     paging.Cursor `json:"cursor,omitempty"`
	Limit      int           `json:"limit,omitempty"`
}

type MapView struct {
	Parent *contextpack.MapNode             `json:"parent,omitempty"`
	Trail  []contextpack.Ref                `json:"trail"`
	Nodes  paging.List[contextpack.MapNode] `json:"nodes"`
}

func (s *Service) Map(ctx context.Context, siteID string) (contextpack.Map, error) {
	var tree contextpack.Map
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		snapshot, err := s.base(c, siteID)
		if err != nil {
			return err
		}
		tree = snapshot.Map(nil)
		return nil
	})
	return tree, err
}

func (s *Service) SiteMap(ctx context.Context, req MapRequest) (MapView, error) {
	if req.SiteID == "" {
		return MapView{}, errors.New(errors.Invalid, "name the site whose map to read").WithDetail("field", "siteId")
	}
	view := MapView{Trail: []contextpack.Ref{}}
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		tree, err := s.mapped(c, req.SiteID)
		if err != nil {
			return err
		}
		if req.ParentID != "" {
			parent, found := tree.Node(req.ParentID)
			if !found {
				return errors.New(errors.NotFound, "the site's map holds no such page").WithDetail("parentId", req.ParentID)
			}
			view.Parent, view.Trail = &parent, tree.Trail(parent.Page.PageID)
		}
		var nodes []contextpack.MapNode
		if req.Search != "" || req.PageTypeID != "" {
			nodes = tree.Matches(contextpack.MapQuery{Under: req.ParentID, Text: req.Search, TypeID: req.PageTypeID})
		} else {
			nodes = tree.Children(req.ParentID)
		}
		view.Nodes, err = byPath(nodes, func(node contextpack.MapNode) contextpack.Ref { return node.Page }, req.Cursor,
			clamped(req.Limit, mapDefaultLimit, mapMaxLimit))
		if err != nil {
			return err
		}
		return s.bodiesOf(c, view.Nodes.Items)
	})
	return view, err
}

func (s *Service) mapped(ctx context.Context, siteID string) (contextpack.Map, error) {
	snapshot, err := s.base(ctx, siteID)
	if err != nil {
		return contextpack.Map{}, err
	}
	if snapshot.Findings, err = s.deps.Findings.ListBySite(ctx, siteID, finding.Open); err != nil {
		return contextpack.Map{}, err
	}
	audit, err := s.deps.Audits.LinkAudit(ctx, reports.LinkAuditRequest{SiteID: siteID})
	if err != nil {
		return contextpack.Map{}, err
	}
	return snapshot.Map(linkFacts(audit)), nil
}

func (s *Service) bodiesOf(ctx context.Context, nodes []contextpack.MapNode) error {
	if len(nodes) == 0 {
		return nil
	}
	ids := make([]string, 0, len(nodes))
	for i := range nodes {
		ids = append(ids, nodes[i].Page.PageID)
	}
	bodies, err := s.deps.Bodies.ListCurrent(ctx, ids)
	if err != nil {
		return err
	}
	written, err := s.deps.Bodies.ListWrittenFor(ctx, ids)
	if err != nil {
		return err
	}
	ours := make(map[string][]string, len(written))
	for i := range written {
		ours[written[i].PageID] = append(ours[written[i].PageID], written[i].BlockHash)
	}
	raw := make(map[string]string, len(bodies))
	for i := range bodies {
		raw[bodies[i].PageID] = bodies[i].Raw
	}
	for i := range nodes {
		id := nodes[i].Page.PageID
		nodes[i].Body = contextpack.BodyOf(raw[id], ours[id])
	}
	return nil
}
