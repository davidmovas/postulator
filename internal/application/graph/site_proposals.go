package graph

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (s *Service) previewFromSite(ctx context.Context, rawSiteID string, pageIDs []string, pathPrefix string) (PreviewFromPagesResponse, error) {
	siteID := strings.TrimSpace(rawSiteID)
	if err := requireSite(siteID); err != nil {
		return PreviewFromPagesResponse{}, err
	}
	if _, err := s.sites.Get(ctx, siteID); err != nil {
		return PreviewFromPagesResponse{}, err
	}
	if s.maps == nil {
		return PreviewFromPagesResponse{}, errors.New(errors.Invalid, "no site model is wired for the site's proposals")
	}
	state, err := s.snapshot(ctx, siteID)
	if err != nil {
		return PreviewFromPagesResponse{}, err
	}
	var chosen []string
	if len(pageIDs) > 0 {
		pages, chosenErr := chosenPages(state.pages, pageIDs, pathPrefix)
		if chosenErr != nil {
			return PreviewFromPagesResponse{}, chosenErr
		}
		for i := range pages {
			chosen = append(chosen, pages[i].ID)
		}
		if len(chosen) == 0 {
			return PreviewFromPagesResponse{Entities: []ProposedEntity{}}, nil
		}
	}
	tree, err := s.maps.Map(ctx, siteID)
	if err != nil {
		return PreviewFromPagesResponse{}, err
	}

	byID := make(map[string]pagemap.Page, len(state.pages))
	for i := range state.pages {
		byID[state.pages[i].ID] = state.pages[i]
	}
	prefix := strings.TrimSpace(pathPrefix)
	response := PreviewFromPagesResponse{Entities: []ProposedEntity{}, PageTypes: map[string]string{}}
	proposals := tree.EntityProposals(chosen)
	for i := range proposals {
		if !strings.HasPrefix(proposals[i].Path, prefix) {
			continue
		}
		response.Entities = append(response.Entities, proposedFromSite(&proposals[i], byID[proposals[i].PageID]))
		if place, placed := tree.Place(proposals[i].PageID); placed && place.TypeID != "" {
			response.PageTypes[proposals[i].PageID] = place.TypeID
		}
	}
	response.Pages = len(response.Entities)
	return response, nil
}

func (s *Service) applyFromSite(ctx context.Context, siteID string, proposals []ProposedEntity, state *siteGraph,
	out *ApplyProposalsResponse, now time.Time) error {
	written := make(map[string]string, len(proposals))
	for _, at := range inParentOrder(proposals) {
		proposed := &proposals[at]
		if proposed.Clash != "" {
			out.Skipped++
			continue
		}
		page, hasPage, err := pageOf(state, proposed)
		if err != nil {
			return err
		}
		if !hasPage {
			return errors.New(errors.Invalid, "a site proposal names its page").WithDetail("field", "entities.pageId")
		}
		if _, taken := state.byPath[page.Path]; taken {
			out.Skipped++
			continue
		}
		scope := scopeOf(state, proposed, written)
		entityID, adopted, err := s.entityUnder(ctx, siteID, proposed, scope, state, out, now)
		if err != nil {
			return err
		}
		if entityID == "" || !s.mappable(state, page, entityID) {
			out.Skipped++
			continue
		}
		if err := s.mapPage(ctx, state, page, entityID, now); err != nil {
			return err
		}
		written[page.Path] = entityID
		out.Mapped++
		if !adopted {
			if err := s.entities.SetCanonicalPage(ctx, entityID, &page.ID, now); err != nil {
				return err
			}
		}
		if scope != nil {
			if err := s.approveParent(ctx, siteID, entityID, *scope, labelOf(proposed), state, out, now); err != nil {
				return err
			}
		}
		if err := s.carried(ctx, siteID, page.ID, now); err != nil {
			return err
		}
	}
	return nil
}

func inParentOrder(proposals []ProposedEntity) []int {
	byPath := make(map[string]int, len(proposals))
	for i := range proposals {
		if path := strings.TrimSpace(proposals[i].Path); path != "" {
			byPath[path] = i
		}
	}
	depth := make([]int, len(proposals))
	for i := range proposals {
		seen := map[int]struct{}{i: {}}
		for at := i; proposals[at].ParentEntityID == ""; {
			parent, inBatch := byPath[strings.TrimSpace(proposals[at].Parent)]
			if _, looped := seen[parent]; !inBatch || looped {
				break
			}
			seen[parent] = struct{}{}
			depth[i]++
			at = parent
		}
	}
	order := make([]int, len(proposals))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(depth[a], depth[b]) })
	return order
}

func scopeOf(state *siteGraph, proposed *ProposedEntity, written map[string]string) *string {
	if parentID := strings.TrimSpace(proposed.ParentEntityID); parentID != "" {
		for i := range state.entities {
			if state.entities[i].ID == parentID {
				return &parentID
			}
		}
		return nil
	}
	parent := strings.TrimSpace(proposed.Parent)
	if entityID, held := written[parent]; held {
		return &entityID
	}
	if entityID, held := state.byPath[parent]; held && parent != "" {
		return &entityID
	}
	return nil
}

func (s *Service) entityUnder(ctx context.Context, siteID string, proposed *ProposedEntity, scope *string, state *siteGraph,
	out *ApplyProposalsResponse, now time.Time) (entityID string, adopted bool, err error) {
	name := strings.TrimSpace(proposed.Name)
	held := ""
	if scope != nil {
		held = *scope
	}
	for i := range state.entities {
		current := ""
		if state.entities[i].ScopeID != nil {
			current = *state.entities[i].ScopeID
		}
		if current == held && graphdomain.Key(state.entities[i].Name) == graphdomain.Key(name) {
			return state.entities[i].ID, true, nil
		}
	}
	keywords, err := application.KeywordList(proposed.Keywords, "keywords")
	if err != nil {
		return "", false, err
	}
	facts := make([]graphdomain.Fact, 0, len(proposed.Facts))
	for _, fact := range proposed.Facts {
		facts = append(facts, graphdomain.Fact(fact))
	}
	entity, err := graphdomain.NewEntity(graphdomain.Entity{
		ID: id.New(), SiteID: siteID, Name: name, Facts: facts, Keywords: keywords, ScopeID: scope,
		Source: graphdomain.SourceUser, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return "", false, nil
	}
	if err := s.entities.Insert(ctx, entity); err != nil {
		if errors.IsCode(err, errors.Conflict) {
			return "", false, nil
		}
		return "", false, err
	}
	state.entities = append(state.entities, entity)
	state.labels = graphdomain.Labels(state.entities)
	out.Entities = append(out.Entities, entityView(entity))
	return entity.ID, false, nil
}

func (s *Service) mapPage(ctx context.Context, state *siteGraph, page pagemap.Page, entityID string, now time.Time) error {
	next := page
	next.EntityID = &entityID
	next.UpdatedAt = now
	if err := s.pages.Update(ctx, next); err != nil {
		return err
	}
	for i := range state.pages {
		if state.pages[i].ID == page.ID {
			state.pages[i] = next
		}
	}
	state.byPath[page.Path] = entityID
	return nil
}

func (s *Service) approveParent(ctx context.Context, siteID, from, to, label string, state *siteGraph,
	out *ApplyProposalsResponse, now time.Time) error {
	for i := range state.edges {
		edge := state.edges[i]
		if edge.FromEntityID != from || edge.ToEntityID != to || edge.Kind != graphdomain.EdgeParent {
			continue
		}
		if edge.Status == graphdomain.StatusApproved {
			return nil
		}
		if err := s.edges.SetStatus(ctx, edge.ID, graphdomain.StatusApproved); err != nil {
			return err
		}
		state.edges[i].Status = graphdomain.StatusApproved
		return nil
	}
	edge, err := graphdomain.NewEdge(graphdomain.Edge{
		ID: id.New(), SiteID: siteID, FromEntityID: from, ToEntityID: to, Kind: graphdomain.EdgeParent, Weight: parentEdgeWeight,
		Source: graphdomain.SourceUser, Status: graphdomain.StatusApproved, Reason: clip(parentReason(label, state.labels[to])), CreatedAt: now,
	})
	if err != nil {
		return err
	}
	if err := s.edges.Insert(ctx, edge); err != nil {
		return err
	}
	state.edges = append(state.edges, edge)
	out.Edges = append(out.Edges, edgeView(edge))
	return nil
}

func proposedFromSite(proposal *contextpack.EntityProposal, page pagemap.Page) ProposedEntity {
	facts := make([]Fact, 0, len(proposal.Facts))
	for _, fact := range proposal.Facts {
		facts = append(facts, Fact(fact))
	}
	return ProposedEntity{
		PageID: proposal.PageID, Path: proposal.Path, Name: proposal.Name, Label: proposal.Label,
		Keywords: application.KeywordViews(page.Keywords), Parent: proposal.Parent, ParentEntityID: proposal.ParentEntityID,
		ExistingEntityID: proposal.ExistingEntityID, Facts: facts, Clash: string(proposal.Clash),
	}
}
