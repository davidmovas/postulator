package sitemodel

import (
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type TreeShape string

const (
	TreeRoots     TreeShape = "roots"
	TreeChildren  TreeShape = "children"
	TreeSubtree   TreeShape = "subtree"
	TreeAncestors TreeShape = "ancestors"
	TreeSiblings  TreeShape = "siblings"
	TreeWalk      TreeShape = "walk"

	treeDefaultLimit = 50
	treeMaxLimit     = 200
	treeDefaultDepth = 2
	treeMaxDepth     = 32
)

func TreeShapes() []TreeShape {
	return []TreeShape{TreeRoots, TreeChildren, TreeSubtree, TreeAncestors, TreeSiblings, TreeWalk}
}

type TreeRequest struct {
	SiteID     string        `json:"siteId"`
	Shape      TreeShape     `json:"shape"`
	PageID     string        `json:"pageId,omitempty"`
	PageTypeID string        `json:"pageTypeId,omitempty"`
	Cursor     paging.Cursor `json:"cursor,omitempty"`
	Generation int64         `json:"generation,omitempty"`
	Depth      int           `json:"depth,omitempty"`
	Limit      int           `json:"limit,omitempty"`
}

type TreeAnswer struct {
	Node       *contextpack.MapNode             `json:"node,omitempty"`
	Nodes      paging.List[contextpack.MapNode] `json:"nodes"`
	Generation int64                            `json:"generation"`
	Stale      bool                             `json:"stale"`
}

func (s TreeShape) needsPage() bool {
	return s != TreeRoots && s != TreeWalk
}

func (s *Service) Tree(ctx context.Context, req TreeRequest) (TreeAnswer, error) {
	switch {
	case req.SiteID == "":
		return TreeAnswer{}, errors.New(errors.Invalid, "name the site whose tree to read").WithDetail("field", "siteId")
	case !slices.Contains(TreeShapes(), req.Shape):
		return TreeAnswer{}, errors.New(errors.Invalid, "the shape must be roots, children, subtree, ancestors, siblings or walk").
			WithDetail("field", "shape").WithDetail("value", string(req.Shape))
	case req.Shape.needsPage() && req.PageID == "":
		return TreeAnswer{}, errors.New(errors.Invalid, "the "+string(req.Shape)+" shape needs the page it starts from").
			WithDetail("field", "pageId")
	}
	var answer TreeAnswer
	err := s.onMap(ctx, req.SiteID, func(c context.Context) error {
		var err error
		answer, err = s.tree(c, req)
		return err
	})
	return answer, err
}

func (s *Service) tree(ctx context.Context, req TreeRequest) (TreeAnswer, error) {
	generation, _, err := s.deps.Maps.State(ctx, req.SiteID)
	if err != nil {
		return TreeAnswer{}, err
	}
	answer := TreeAnswer{Generation: generation, Stale: req.Generation != 0 && req.Generation != generation}

	var asked *contextpack.MapRow
	if req.Shape.needsPage() {
		row, rowErr := s.nodeRow(ctx, req.SiteID, req.PageID, "pageId")
		if rowErr != nil {
			return TreeAnswer{}, rowErr
		}
		asked = &row
	}

	var listed paging.List[contextpack.MapRow]
	if req.Shape == TreeAncestors {
		if listed.Items, err = s.ancestors(ctx, *asked); err != nil {
			return TreeAnswer{}, err
		}
	} else {
		listed, err = s.deps.Maps.List(ctx, treeQuery(req, asked), paging.Request{
			After: req.Cursor, Limit: clamped(req.Limit, treeDefaultLimit, treeMaxLimit),
		})
		if err != nil {
			return TreeAnswer{}, err
		}
	}

	read, err := s.mapNodes(ctx, req.SiteID, listed.Items, asked, false)
	if err != nil {
		return TreeAnswer{}, err
	}
	answer.Nodes = paging.List[contextpack.MapNode]{Cursors: listed.Cursors, HasMore: listed.HasMore, Items: read.nodes}
	if asked != nil {
		node := read.node(*asked)
		answer.Node = &node
	}
	if whyErr := s.whys(ctx, read, answer.Nodes.Items, answer.Node); whyErr != nil {
		return TreeAnswer{}, whyErr
	}
	return answer, s.bodiesOf(ctx, answer.Nodes.Items)
}

func treeQuery(req TreeRequest, asked *contextpack.MapRow) contextpack.MapRowQuery {
	q := contextpack.MapRowQuery{SiteID: req.SiteID, TypeID: req.PageTypeID}
	switch req.Shape {
	case TreeRoots:
		root := ""
		q.ParentID = &root
	case TreeChildren:
		q.ParentID = &asked.PageID
	case TreeSiblings:
		q.ParentID, q.Except = &asked.ParentID, asked.PageID
	case TreeSubtree:
		deepest := asked.Depth + clamped(req.Depth, treeDefaultDepth, treeMaxDepth)
		q.Under, q.MinDepth, q.MaxDepth, q.ByDepth = asked.Trail, asked.Depth+1, &deepest, true
	default:
		q.ByDepth = true
	}
	return q
}

func (s *Service) ancestors(ctx context.Context, row contextpack.MapRow) ([]contextpack.MapRow, error) {
	ids := row.Ancestors()
	held, err := s.deps.Maps.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]contextpack.MapRow, len(held))
	for i := range held {
		byID[held[i].PageID] = held[i]
	}
	out := make([]contextpack.MapRow, 0, len(ids))
	for _, pageID := range ids {
		if ancestor, found := byID[pageID]; found {
			out = append(out, ancestor)
		}
	}
	return out, nil
}
