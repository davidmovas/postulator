package runs

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/application/graph"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
)

type missingEdge struct {
	from pagemap.Page
	to   pagemap.Page
}

func briefOf(req StartRequest) run.Brief {
	if req.Brief == nil {
		return run.Brief{}
	}
	return *req.Brief
}

func (s *Service) directed(ctx context.Context, req StartRequest, siteID string, targets []string) ([]missingEdge, error) {
	if err := run.ValidateBrief(briefOf(req), req.Directives); err != nil {
		return nil, err
	}
	missing := make([]missingEdge, 0)
	for i := range req.Directives {
		directive := &req.Directives[i]
		pageID := strings.TrimSpace(directive.PageID)
		if !slices.Contains(targets, pageID) {
			return nil, invalid("a directive is for a page the run writes, and "+pageID+" is not one of them", "directives.pageId").
				WithDetail("pageId", pageID)
		}
		if len(directive.MustLink) == 0 {
			continue
		}
		plan, err := s.links.LinkContextOf(ctx, siteID, pageID)
		if err != nil {
			return nil, err
		}
		for _, link := range directive.MustLink {
			if _, owed := plan.ByPageID(link.PageID); owed {
				continue
			}
			gap, open, gapErr := s.unlinked(ctx, req, pageID, link.PageID)
			if gapErr != nil {
				return nil, gapErr
			}
			if open {
				missing = append(missing, gap)
			}
		}
	}
	return missing, nil
}

func (s *Service) unlinked(ctx context.Context, req StartRequest, fromID, toID string) (missingEdge, bool, error) {
	from, err := s.pages.Get(ctx, fromID)
	if err != nil {
		return missingEdge{}, false, err
	}
	to, err := s.pages.Get(ctx, toID)
	if err != nil {
		return missingEdge{}, false, err
	}
	if from.EntityID == nil || to.EntityID == nil {
		return missingEdge{}, false, invalid(from.Path+" cannot be made to link to "+to.Path+
			": the entity graph links pages through their entities, and one of the two carries none", "directives.mustLink").
			WithDetail("from", from.Path).WithDetail("to", to.Path)
	}
	if assumes(req.Assumed, *from.EntityID, *to.EntityID) {
		return missingEdge{}, false, nil
	}
	if !req.AddEdges {
		return missingEdge{}, false, invalid(from.Path+" does not link to "+to.Path+
			": no edge of the entity graph that the page's link policy follows connects their entities; add a related edge first, "+
			"or start the run with addEdges", "directives.mustLink").WithDetail("from", from.Path).WithDetail("to", to.Path)
	}
	return missingEdge{from: from, to: to}, true, nil
}

func assumes(assumed []AssumedEdge, a, b string) bool {
	return slices.ContainsFunc(assumed, func(edge AssumedEdge) bool {
		return edge.FromEntityID == a && edge.ToEntityID == b || edge.FromEntityID == b && edge.ToEntityID == a
	})
}

func (s *Service) addEdges(ctx context.Context, siteID string, missing []missingEdge) error {
	for i := range missing {
		gap := &missing[i]
		if _, err := s.edges.AddEdge(ctx, graph.AddEdgeRequest{
			SiteID: siteID, FromEntityID: *gap.from.EntityID, ToEntityID: *gap.to.EntityID,
			Kind: string(graphdomain.EdgeRelated), Weight: 1, Status: string(graphdomain.StatusApproved),
			Reason: "a run's directive asks " + gap.from.Path + " to link to " + gap.to.Path,
		}); err != nil {
			return err
		}
	}
	return nil
}
