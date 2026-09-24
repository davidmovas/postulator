package graph

import (
	"context"
	"strconv"
	"strings"
	"time"

	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const (
	parentEdgeWeight  = 1
	relatedEdgeWeight = 0.5
)

func (s *Service) ApplyProposals(ctx context.Context, req ApplyProposalsRequest) (ApplyProposalsResponse, error) {
	siteID := strings.TrimSpace(req.SiteID)
	if err := requireSite(siteID); err != nil {
		return ApplyProposalsResponse{}, err
	}
	if _, err := s.sites.Get(ctx, siteID); err != nil {
		return ApplyProposalsResponse{}, err
	}
	if len(req.Entities) == 0 {
		return ApplyProposalsResponse{}, errors.New(errors.Invalid, "there is no proposal to write").
			WithDetail("field", "entities")
	}
	for i := range req.Entities {
		if strings.TrimSpace(req.Entities[i].Name) == "" {
			return ApplyProposalsResponse{}, errors.New(errors.Invalid, "a proposal needs a name").
				WithDetail("field", "entities["+strconv.Itoa(i)+"].name")
		}
	}

	state, err := s.snapshot(ctx, siteID)
	if err != nil {
		return ApplyProposalsResponse{}, err
	}

	response := ApplyProposalsResponse{Entities: []Entity{}, Edges: []Edge{}}
	now := s.now()
	err = s.uow.Do(ctx, func(c context.Context) error {
		for i := range req.Entities {
			if adoptErr := s.adopt(c, siteID, &req.Entities[i], &state, &response, now); adoptErr != nil {
				return adoptErr
			}
		}
		return s.connect(c, siteID, req.Entities, &state, &response, now)
	})
	if err != nil {
		return ApplyProposalsResponse{}, err
	}

	if len(response.Entities) == 0 && len(response.Edges) == 0 && response.Mapped == 0 {
		return response, nil
	}
	if changedErr := s.changed(siteID); changedErr != nil {
		return ApplyProposalsResponse{}, changedErr
	}
	if response.Mapped > 0 {
		if pagesErr := s.pagesChanged(siteID); pagesErr != nil {
			return ApplyProposalsResponse{}, pagesErr
		}
	}
	return response, nil
}

func (s *Service) adopt(ctx context.Context, siteID string, proposed *ProposedEntity, state *siteGraph,
	out *ApplyProposalsResponse, now time.Time) error {
	name := strings.TrimSpace(proposed.Name)
	page, hasPage, err := pageOf(state, proposed)
	if err != nil {
		return err
	}
	if hasPage {
		if _, taken := state.byPath[page.Path]; taken {
			out.Skipped++
			return nil
		}
	}

	entityID, exists := state.byName[fold(name)]
	if !exists {
		entity, buildErr := graphdomain.NewEntity(graphdomain.Entity{
			ID:                id.New(),
			SiteID:            siteID,
			Name:              name,
			Kind:              kindOf(proposed.Kind),
			Intent:            strings.TrimSpace(proposed.Intent),
			PrimaryKeyword:    proposed.PrimaryKeyword,
			SecondaryKeywords: proposed.SecondaryKeywords,
			Anchors:           proposedAnchors(proposed.Anchors),
			Source:            graphdomain.SourceAI,
			CreatedAt:         now,
			UpdatedAt:         now,
		})
		if buildErr != nil {
			out.Skipped++
			return nil
		}
		if insertErr := s.entities.Insert(ctx, entity); insertErr != nil {
			return insertErr
		}
		state.entities = append(state.entities, entity)
		state.byName[fold(name)] = entity.ID
		entityID = entity.ID
		out.Entities = append(out.Entities, entityView(entity))
	}
	if !hasPage {
		if exists {
			out.Skipped++
		}
		return nil
	}

	if !s.mappable(state, page, entityID) {
		out.Skipped++
		return nil
	}
	next := page
	next.EntityID = &entityID
	next.UpdatedAt = now
	if updateErr := s.pages.Update(ctx, next); updateErr != nil {
		return updateErr
	}
	for i := range state.pages {
		if state.pages[i].ID == page.ID {
			state.pages[i] = next
		}
	}
	state.byPath[page.Path] = entityID
	out.Mapped++
	if !exists {
		if canonicalErr := s.entities.SetCanonicalPage(ctx, entityID, &page.ID, now); canonicalErr != nil {
			return canonicalErr
		}
	}
	return nil
}

func (s *Service) mappable(state *siteGraph, page pagemap.Page, entityID string) bool {
	var entity graphdomain.Entity
	for i := range state.entities {
		if state.entities[i].ID == entityID {
			entity = state.entities[i]
		}
	}
	g, err := graphdomain.New(state.entities, nil)
	if err != nil {
		return false
	}
	candidate := page
	candidate.EntityID = &entityID
	return pagemap.Cannibalization(candidate, entity, pagemap.NewIndex(state.pages), g).Allowed
}

func pageOf(state *siteGraph, proposed *ProposedEntity) (pagemap.Page, bool, error) {
	pageID := strings.TrimSpace(proposed.PageID)
	if pageID == "" {
		return pagemap.Page{}, false, nil
	}
	for i := range state.pages {
		if state.pages[i].ID == pageID {
			return state.pages[i], true, nil
		}
	}
	return pagemap.Page{}, false, errors.New(errors.NotFound, "page not found").WithDetail("pageId", pageID)
}

func (s *Service) connect(ctx context.Context, siteID string, proposals []ProposedEntity, state *siteGraph,
	out *ApplyProposalsResponse, now time.Time) error {
	for i := range proposals {
		proposed := &proposals[i]
		from, known := state.byName[fold(proposed.Name)]
		if !known {
			continue
		}

		if parent, ok := resolveRef(state, proposed.Parent); ok && parent != from {
			reason := parentReason(labelOf(proposed), strings.TrimSpace(proposed.Parent))
			if err := s.propose(ctx, siteID, from, parent, graphdomain.EdgeParent, parentEdgeWeight, reason, state, out, now); err != nil {
				return err
			}
		}
		for _, raw := range proposed.Related {
			related, ok := resolveRef(state, raw)
			if !ok || related == from {
				continue
			}
			reason := relatedReason(labelOf(proposed), strings.TrimSpace(raw))
			if err := s.propose(ctx, siteID, from, related, graphdomain.EdgeRelated, relatedEdgeWeight, reason, state, out, now); err != nil {
				return err
			}
		}
	}
	return nil
}

func labelOf(proposed *ProposedEntity) string {
	if path := strings.TrimSpace(proposed.Path); path != "" {
		return path
	}
	return strings.TrimSpace(proposed.Name)
}

func resolveRef(state *siteGraph, raw string) (string, bool) {
	ref := strings.TrimSpace(raw)
	if ref == "" {
		return "", false
	}
	if entityID, ok := state.byPath[ref]; ok {
		return entityID, true
	}
	entityID, ok := state.byName[fold(ref)]
	return entityID, ok
}

func (s *Service) propose(ctx context.Context, siteID, from, to string, kind graphdomain.EdgeKind,
	weight float64, reason string, state *siteGraph, out *ApplyProposalsResponse, now time.Time) error {
	edge, err := graphdomain.NewEdge(graphdomain.Edge{
		ID: id.New(), SiteID: siteID, FromEntityID: from, ToEntityID: to, Kind: kind, Weight: weight,
		Source: graphdomain.SourceAI, Status: graphdomain.StatusProposed, Reason: clip(reason), CreatedAt: now,
	})
	if err != nil {
		return nil
	}
	if connected(state.edges, edge) {
		return nil
	}
	if insertErr := s.edges.Insert(ctx, edge); insertErr != nil {
		if errors.IsCode(insertErr, errors.Conflict) {
			return nil
		}
		return insertErr
	}

	state.edges = append(state.edges, edge)
	out.Edges = append(out.Edges, edgeView(edge))
	return nil
}
