package graph

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
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
	response := PreviewFromPagesResponse{Entities: []ProposedEntity{}}
	proposals := tree.EntityProposals(chosen)
	for i := range proposals {
		if !strings.HasPrefix(proposals[i].Path, prefix) {
			continue
		}
		response.Entities = append(response.Entities, proposedFromSite(&proposals[i], byID[proposals[i].PageID]))
	}
	response.Pages = len(response.Entities)
	return response, nil
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
