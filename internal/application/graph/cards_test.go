package graph_test

import (
	stderrors "errors"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func carded(t *testing.T) (h harness, entityID string) {
	t.Helper()

	h = newHarness(t)
	created, err := h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{SiteID: h.siteID, Name: "BPC-157"})
	if err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}
	if _, err = h.service.UpdateEntity(t.Context(), graph.UpdateEntityRequest{
		ID: created.Entity.ID, Facts: &[]graph.Fact{{Label: "Form", Value: "Powder"}},
	}); err != nil {
		t.Fatalf("UpdateEntity: %v", err)
	}
	return h, created.Entity.ID
}

func cardOf(entityID, definition string) graph.CardInput {
	return graph.CardInput{EntityID: entityID, Definition: definition, Reason: "read from the hub's own text"}
}

func reasonOf(err error) string {
	held, ok := stderrors.AsType[*errors.Error](err)
	if !ok {
		return ""
	}
	if reason, said := held.Details["reason"].(string); said {
		return reason
	}
	return ""
}

func TestAProposedCardIsWrittenOnlyOnceAccepted(t *testing.T) {
	t.Parallel()

	h, entityID := carded(t)
	proposed, err := h.service.ProposeCards(t.Context(), graph.ProposeCardsRequest{
		SiteID: h.siteID, Cards: []graph.CardInput{cardOf(entityID, "A peptide studied for tissue repair.")},
	})
	if err != nil {
		t.Fatalf("ProposeCards: %v", err)
	}
	if len(proposed.Proposals) != 1 || proposed.Proposals[0].Proposed.Definition != "A peptide studied for tissue repair." ||
		len(proposed.Proposals[0].Proposed.Facts) != 1 || proposed.Proposals[0].Current.Definition != "" {
		t.Fatalf("proposals = %+v, want the definition proposed and the facts kept", proposed.Proposals)
	}
	held, err := h.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: entityID})
	if err != nil || held.Entity.Definition != "" {
		t.Fatalf("the entity after a proposal = %+v, %v; want nothing written", held.Entity, err)
	}

	listed, err := h.service.CardProposals(t.Context(), graph.CardProposalsRequest{SiteID: h.siteID})
	if err != nil || len(listed.Items) != 1 || listed.Items[0].FindingID != proposed.Proposals[0].FindingID {
		t.Fatalf("CardProposals = %+v, %v", listed, err)
	}

	h.recorder.Reset()
	shown := graph.AcceptCardRequest{FindingID: proposed.Proposals[0].FindingID, ProposalHash: proposed.Proposals[0].ProposalHash}
	accepted, err := h.service.AcceptCard(t.Context(), shown)
	if err != nil || accepted.Entity.Definition != "A peptide studied for tissue repair." || len(accepted.Entity.Facts) != 1 {
		t.Fatalf("AcceptCard = %+v, %v", accepted, err)
	}
	if got := h.recorder.Events(); len(got) != 1 || got[0].Type != events.GraphChanged {
		t.Errorf("accepting published %v, want graph.changed", got)
	}
	settled, err := sqlite.NewFindingRepo(h.store).Get(t.Context(), proposed.Proposals[0].FindingID)
	if err != nil || settled.Status != finding.Resolved || settled.Resolution != graph.ResolutionAccepted || settled.ResolvedBy != graph.ResolvedByPerson {
		t.Errorf("the finding after accepting = %+v, %v", settled, err)
	}
	if _, err = h.service.AcceptCard(t.Context(), shown); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("accepting twice = %v, want Conflict", err)
	}
	if listed, err = h.service.CardProposals(t.Context(), graph.CardProposalsRequest{SiteID: h.siteID}); err != nil || len(listed.Items) != 0 {
		t.Errorf("open proposals after accepting = %+v, %v", listed, err)
	}
}

func TestACardThatChangedSinceItsProposalIsRefused(t *testing.T) {
	t.Parallel()

	h, entityID := carded(t)
	proposed, err := h.service.ProposeCards(t.Context(), graph.ProposeCardsRequest{
		SiteID: h.siteID, Cards: []graph.CardInput{cardOf(entityID, "A peptide.")},
	})
	if err != nil {
		t.Fatalf("ProposeCards: %v", err)
	}
	if _, err = h.service.UpdateEntity(t.Context(), graph.UpdateEntityRequest{ID: entityID, Definition: new("Written by a person.")}); err != nil {
		t.Fatalf("UpdateEntity: %v", err)
	}
	listed, err := h.service.CardProposals(t.Context(), graph.CardProposalsRequest{SiteID: h.siteID})
	if err != nil || len(listed.Items) != 1 || !listed.Items[0].Changed {
		t.Errorf("the listed proposal = %+v, %v; want it marked as changed since", listed.Items, err)
	}
	_, err = h.service.AcceptCard(t.Context(), graph.AcceptCardRequest{FindingID: proposed.Proposals[0].FindingID, ProposalHash: proposed.Proposals[0].ProposalHash})
	if !errors.IsCode(err, errors.Conflict) || reasonOf(err) != graph.ReasonCardChanged {
		t.Errorf("accepting a stale proposal = %v, want Conflict with %s", err, graph.ReasonCardChanged)
	}
	held, err := h.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: entityID})
	if err != nil || held.Entity.Definition != "Written by a person." {
		t.Errorf("the person's definition = %q, %v; want it kept", held.Entity.Definition, err)
	}
}

func TestAcceptingWritesOnlyTheProposalThePersonSaw(t *testing.T) {
	t.Parallel()

	h, entityID := carded(t)
	seen, err := h.service.ProposeCards(t.Context(), graph.ProposeCardsRequest{SiteID: h.siteID, Cards: []graph.CardInput{cardOf(entityID, "A peptide.")}})
	if err != nil {
		t.Fatalf("ProposeCards: %v", err)
	}
	if _, err = h.service.ProposeCards(t.Context(), graph.ProposeCardsRequest{SiteID: h.siteID, Cards: []graph.CardInput{cardOf(entityID, "Something else entirely.")}}); err != nil {
		t.Fatalf("ProposeCards again: %v", err)
	}
	_, err = h.service.AcceptCard(t.Context(), graph.AcceptCardRequest{FindingID: seen.Proposals[0].FindingID, ProposalHash: seen.Proposals[0].ProposalHash})
	if !errors.IsCode(err, errors.Conflict) || reasonOf(err) != graph.ReasonProposalChanged {
		t.Errorf("accepting the proposal as it was first seen = %v, want Conflict with %s", err, graph.ReasonProposalChanged)
	}
	held, err := h.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: entityID})
	if err != nil || held.Entity.Definition != "" {
		t.Errorf("the entity = %+v, %v; want nothing written", held.Entity, err)
	}
	if _, err = h.service.AcceptCard(t.Context(), graph.AcceptCardRequest{FindingID: seen.Proposals[0].FindingID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("accepting without saying which proposal = %v, want Invalid", err)
	}
}

func TestADismissedProposalStaysDismissed(t *testing.T) {
	t.Parallel()

	h, entityID := carded(t)
	request := graph.ProposeCardsRequest{SiteID: h.siteID, Cards: []graph.CardInput{cardOf(entityID, "A peptide.")}}
	proposed, err := h.service.ProposeCards(t.Context(), request)
	if err != nil {
		t.Fatalf("ProposeCards: %v", err)
	}
	if _, err = h.service.DismissCard(t.Context(), graph.DismissCardRequest{FindingID: proposed.Proposals[0].FindingID}); err != nil {
		t.Fatalf("DismissCard: %v", err)
	}
	again, err := h.service.ProposeCards(t.Context(), request)
	if err != nil || len(again.Proposals) != 0 || len(again.Skipped) != 1 || again.Skipped[0].Reason != graph.ReasonCardDismissed {
		t.Errorf("the same proposal again = %+v, %v; want it skipped as dismissed", again, err)
	}
	other, err := h.service.ProposeCards(t.Context(), graph.ProposeCardsRequest{SiteID: h.siteID, Cards: []graph.CardInput{cardOf(entityID, "A protein fragment.")}})
	if err != nil || len(other.Proposals) != 1 || other.Proposals[0].FindingID != proposed.Proposals[0].FindingID {
		t.Errorf("another proposal = %+v, %v; want the same finding opened again", other, err)
	}
}

func TestApplyingCardsWritesThemAtOnce(t *testing.T) {
	t.Parallel()

	h, entityID := carded(t)
	applied, err := h.service.ApplyCards(t.Context(), graph.ApplyCardsRequest{SiteID: h.siteID, Cards: []graph.CardInput{
		{EntityID: entityID, Definition: "A peptide.", Facts: []graph.Fact{}},
	}})
	if err != nil || len(applied.Entities) != 1 || applied.Entities[0].Definition != "A peptide." || len(applied.Entities[0].Facts) != 0 {
		t.Fatalf("ApplyCards = %+v, %v; want the definition written and the facts cleared by an empty list", applied, err)
	}
	unchanged, err := h.service.ApplyCards(t.Context(), graph.ApplyCardsRequest{SiteID: h.siteID, Cards: []graph.CardInput{
		{EntityID: entityID, Definition: "A peptide."},
	}})
	if err != nil || len(unchanged.Entities) != 0 || len(unchanged.Skipped) != 1 || unchanged.Skipped[0].Reason != graph.ReasonCardUnchanged {
		t.Errorf("applying the same card = %+v, %v; want it skipped as unchanged", unchanged, err)
	}
}

func TestACardProposalRefusesWhatItCannotWrite(t *testing.T) {
	t.Parallel()

	h, entityID := carded(t)
	other := sqlitetest.Site(t, h.store, "elsewhere")
	foreign, err := h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{SiteID: other.ID, Name: "Elsewhere"})
	if err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}
	cases := []struct {
		name  string
		cards []graph.CardInput
		code  errors.Code
	}{
		{name: "no card", cards: nil, code: errors.Invalid},
		{name: "an unknown entity", cards: []graph.CardInput{cardOf("missing", "x")}, code: errors.NotFound},
		{name: "an entity of another site", cards: []graph.CardInput{cardOf(foreign.Entity.ID, "x")}, code: errors.NotFound},
		{name: "a fact with no value", cards: []graph.CardInput{{EntityID: entityID, Facts: []graph.Fact{{Label: "Form"}}}}, code: errors.Invalid},
		{name: "nothing proposed", cards: []graph.CardInput{{EntityID: entityID}}, code: errors.Invalid},
		{name: "one entity twice", cards: []graph.CardInput{cardOf(entityID, "a"), cardOf(entityID, "b")}, code: errors.Invalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := h.service.ProposeCards(t.Context(), graph.ProposeCardsRequest{SiteID: h.siteID, Cards: tc.cards}); !errors.IsCode(err, tc.code) {
				t.Errorf("ProposeCards = %v, want %s", err, tc.code)
			}
		})
	}
	if _, err := h.service.AcceptCard(t.Context(), graph.AcceptCardRequest{FindingID: "missing", ProposalHash: "seen"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("accepting an unknown proposal = %v, want NotFound", err)
	}
	if _, err := h.service.CardProposals(t.Context(), graph.CardProposalsRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("CardProposals with no site = %v, want Invalid", err)
	}
}
