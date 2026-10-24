package sitemodel

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
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
	var view MapView
	err := s.onMap(ctx, req.SiteID, func(c context.Context) error {
		var err error
		view, err = s.siteMap(c, req)
		return err
	})
	return view, err
}

func (s *Service) onMap(ctx context.Context, siteID string, fn func(context.Context) error) error {
	stale := false
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		generation, built, err := s.deps.Maps.State(c, siteID)
		if err != nil {
			return err
		}
		if stale = built < generation; stale {
			return nil
		}
		return fn(c)
	})
	if err != nil || !stale {
		return err
	}
	if refreshErr := s.Refresh(ctx, siteID); refreshErr != nil {
		return refreshErr
	}
	return s.deps.Reader.Read(ctx, fn)
}

func (s *Service) siteMap(ctx context.Context, req MapRequest) (MapView, error) {
	view := MapView{Trail: []contextpack.Ref{}}
	var parent *contextpack.MapRow
	if req.ParentID != "" {
		row, err := s.nodeRow(ctx, req.SiteID, req.ParentID, "parentId")
		if err != nil {
			return MapView{}, err
		}
		parent = &row
	}
	q := contextpack.MapRowQuery{SiteID: req.SiteID}
	searching := req.Search != "" || req.PageTypeID != ""
	switch {
	case searching:
		q.Text, q.TypeID = graph.Key(req.Search), req.PageTypeID
		if parent != nil {
			q.Under = parent.Trail
		}
	case parent != nil:
		q.ParentID = &parent.PageID
	default:
		root := ""
		q.ParentID = &root
	}
	listed, err := s.deps.Maps.List(ctx, q, paging.Request{After: req.Cursor, Limit: clamped(req.Limit, mapDefaultLimit, mapMaxLimit)})
	if err != nil {
		return MapView{}, err
	}
	read, err := s.mapNodes(ctx, req.SiteID, listed.Items, parent, searching)
	if err != nil {
		return MapView{}, err
	}
	view.Nodes = paging.List[contextpack.MapNode]{Cursors: listed.Cursors, HasMore: listed.HasMore, Items: read.nodes}
	if parent != nil {
		node := read.node(*parent)
		view.Parent, view.Trail = &node, read.trail(*parent)
	}
	return view, s.bodiesOf(ctx, view.Nodes.Items)
}

func (s *Service) nodeRow(ctx context.Context, siteID, pageID, field string) (contextpack.MapRow, error) {
	missing := errors.New(errors.NotFound, "the site's map holds no such page").WithDetail(field, pageID)
	rows, err := s.deps.Maps.ListByIDs(ctx, []string{pageID})
	if err != nil {
		return contextpack.MapRow{}, err
	}
	if len(rows) == 0 || rows[0].SiteID != siteID {
		return contextpack.MapRow{}, missing
	}
	if rows[0].HubID != "" {
		if rows, err = s.deps.Maps.ListByIDs(ctx, []string{rows[0].HubID}); err != nil {
			return contextpack.MapRow{}, err
		}
	}
	if len(rows) == 0 || !rows[0].Row {
		return contextpack.MapRow{}, missing
	}
	return rows[0], nil
}

type mapRead struct {
	rows     map[string]contextpack.MapRow
	pages    map[string]pagemap.Page
	types    pagetype.Set
	findings map[string]int
	inbound  map[string]int
	outbound map[string]int
	nodes    []contextpack.MapNode
}

func (s *Service) mapNodes(ctx context.Context, siteID string, listed []contextpack.MapRow, parent *contextpack.MapRow, trails bool) (mapRead, error) {
	read := mapRead{rows: make(map[string]contextpack.MapRow, len(listed)+1)}
	counted := make([]string, 0, 2*len(listed)+2)
	wanted := map[string]struct{}{}
	hold := func(row contextpack.MapRow, withTrail bool) {
		read.rows[row.PageID] = row
		counted = append(counted, row.PageID)
		if row.TwinID != "" {
			counted = append(counted, row.TwinID)
			wanted[row.TwinID] = struct{}{}
		}
		if withTrail {
			for _, pageID := range row.Ancestors() {
				wanted[pageID] = struct{}{}
			}
		}
	}
	for i := range listed {
		hold(listed[i], trails)
	}
	if parent != nil {
		hold(*parent, true)
	}
	if err := s.moreRows(ctx, &read, wanted); err != nil {
		return mapRead{}, err
	}
	ids := make([]string, 0, len(read.rows))
	for pageID := range read.rows {
		ids = append(ids, pageID)
	}
	pages, err := s.deps.Pages.ListByIDs(ctx, ids)
	if err != nil {
		return mapRead{}, err
	}
	read.pages = make(map[string]pagemap.Page, len(pages))
	for i := range pages {
		read.pages[pages[i].ID] = pages[i]
	}
	if read.types, err = s.deps.Types.Set(ctx, siteID); err != nil {
		return mapRead{}, err
	}
	if read.findings, err = s.deps.Findings.CountOpen(ctx, counted); err != nil {
		return mapRead{}, err
	}
	if read.inbound, read.outbound, err = s.deps.Links.CountContextual(ctx, counted); err != nil {
		return mapRead{}, err
	}
	read.nodes = make([]contextpack.MapNode, 0, len(listed))
	for i := range listed {
		node := read.node(listed[i])
		if trails {
			node.Trail = read.trail(listed[i])
		}
		read.nodes = append(read.nodes, node)
	}
	return read, nil
}

func (s *Service) moreRows(ctx context.Context, read *mapRead, wanted map[string]struct{}) error {
	missing := make([]string, 0, len(wanted))
	for pageID := range wanted {
		if _, known := read.rows[pageID]; !known {
			missing = append(missing, pageID)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	extra, err := s.deps.Maps.ListByIDs(ctx, missing)
	if err != nil {
		return err
	}
	for i := range extra {
		read.rows[extra[i].PageID] = extra[i]
	}
	return nil
}

func (r mapRead) ref(pageID string) contextpack.Ref {
	row := r.rows[pageID]
	kind, _ := r.types.ByID(row.TypeID)
	return contextpack.Ref{
		PageID: row.PageID, Path: row.Path, Title: row.Title, Type: kind.Name, Preset: kind.Preset, OnSite: r.pages[pageID].WPID != nil,
	}
}

func (r mapRead) node(row contextpack.MapRow) contextpack.MapNode {
	page := r.pages[row.PageID]
	node := contextpack.MapNode{
		Page: r.ref(row.PageID), ParentReason: page.ParentReason, ParentMethod: page.ParentMethod, Technical: page.TechnicalReason,
		ReadOnly: page.ReadOnlyReason, Body: contextpack.BodyEmpty, LinksIn: r.inbound[row.PageID], LinksOut: r.outbound[row.PageID],
		Findings: r.findings[row.PageID], Children: row.Children, Descendants: row.Descendants,
	}
	if row.TwinID != "" {
		twin := r.ref(row.TwinID)
		node.Twin = &twin
		node.Findings += r.findings[row.TwinID]
		node.LinksIn += r.inbound[row.TwinID]
		node.LinksOut += r.outbound[row.TwinID]
	}
	return node
}

func (r mapRead) trail(row contextpack.MapRow) []contextpack.Ref {
	ancestors := row.Ancestors()
	out := make([]contextpack.Ref, 0, len(ancestors))
	for _, pageID := range ancestors {
		out = append(out, r.ref(pageID))
	}
	return out
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
