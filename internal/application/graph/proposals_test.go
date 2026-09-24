package graph_test

import (
	"slices"
	"strings"
	"testing"

	appgraph "github.com/davidmovas/postulator/internal/application/graph"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const keywordProposal = `{"entities":[
	{"keyword":"trail running shoes","name":"Trail Running Shoes","kind":"topic","intent":"pick a pair",
	 "secondaryKeywords":["trail shoes"],"anchors":["trail running shoes"],"parentName":"Coffee","relatedNames":["Road Running Shoes"]},
	{"keyword":"road running shoes","name":"Road Running Shoes","kind":"topic","intent":"pick a pair",
	 "secondaryKeywords":[],"anchors":["road running shoes"],"parentName":"","relatedNames":["Trail Running Shoes"]},
	{"keyword":"socks","name":"Socks","kind":"topic","intent":"","secondaryKeywords":[],"anchors":[],"parentName":"","relatedNames":[]}
]}`

func TestProposeFromPagesTakesOnlyTheChosenPages(t *testing.T) {
	t.Parallel()

	f := newProposeFixture(t, &scriptedModel{replies: []string{proposal}}, fixedProfiles{})
	f.page(t, "/coffee/", "Coffee")
	espresso := f.page(t, "/coffee/espresso/", "Espresso")
	f.page(t, "/coffee/filter/", "Filter coffee")
	f.page(t, "/tea/", "Tea")

	if _, err := f.service.ProposeFromPages(t.Context(), appgraph.ProposeFromPagesRequest{
		SiteID: f.siteID, PageIDs: []string{espresso.ID},
	}); err != nil {
		t.Fatalf("ProposeFromPages: %v", err)
	}

	prompt := f.model.calls[0].Messages[0].Text
	if !strings.Contains(prompt, "/coffee/espresso/") || strings.Contains(prompt, "/tea/") || strings.Contains(prompt, "- /coffee/ ") {
		t.Fatalf("the prompt does not hold the chosen page alone:\n%s", prompt)
	}

	mapped, err := f.pages.Get(t.Context(), espresso.ID)
	if err != nil || mapped.EntityID == nil {
		t.Fatalf("the chosen page was not mapped: %+v, %v", mapped, err)
	}
	if _, err = f.service.ProposeFromPages(t.Context(), appgraph.ProposeFromPagesRequest{
		SiteID: f.siteID, PageIDs: []string{espresso.ID},
	}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("a page that already carries an entity = %v, want INVALID", err)
	}
	if _, err = f.service.ProposeFromPages(t.Context(), appgraph.ProposeFromPagesRequest{
		SiteID: f.siteID, PageIDs: []string{"missing"},
	}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("an unknown page = %v, want NOT_FOUND", err)
	}
}

func TestProposeFromPagesTakesABranch(t *testing.T) {
	t.Parallel()

	f := newProposeFixture(t, &scriptedModel{replies: []string{proposal}}, fixedProfiles{})
	f.page(t, "/coffee/", "Coffee")
	f.page(t, "/coffee/espresso/", "Espresso")
	f.page(t, "/tea/", "Tea")

	if _, err := f.service.ProposeFromPages(t.Context(), appgraph.ProposeFromPagesRequest{
		SiteID: f.siteID, PathPrefix: "/coffee/",
	}); err != nil {
		t.Fatalf("ProposeFromPages: %v", err)
	}

	prompt := f.model.calls[0].Messages[0].Text
	if !strings.Contains(prompt, "/coffee/espresso/") || strings.Contains(prompt, "/tea/") {
		t.Fatalf("the prompt does not hold the branch alone:\n%s", prompt)
	}
}

func TestPreviewFromPagesWritesNothing(t *testing.T) {
	t.Parallel()

	f := newProposeFixture(t, &scriptedModel{replies: []string{proposal}}, fixedProfiles{})
	hub := f.page(t, "/coffee/", "Coffee")
	f.page(t, "/coffee/espresso/", "Espresso")
	f.page(t, "/coffee/filter/", "Filter coffee")
	existing := f.entity(t, "Espresso")

	out, err := f.service.PreviewFromPages(t.Context(), appgraph.PreviewFromPagesRequest{SiteID: f.siteID})
	if err != nil {
		t.Fatalf("PreviewFromPages: %v", err)
	}
	if out.Pages != 3 || len(out.Entities) != 3 || out.Skipped != 1 || out.Tokens != 12 {
		t.Fatalf("preview = %+v", out)
	}

	byName := make(map[string]appgraph.ProposedEntity, len(out.Entities))
	for _, proposed := range out.Entities {
		byName[proposed.Name] = proposed
	}
	if byName["Coffee"].PageID != hub.ID || byName["Coffee"].Path != "/coffee/" || byName["Coffee"].Kind != "hub" {
		t.Fatalf("the hub proposal = %+v", byName["Coffee"])
	}
	if byName["Espresso"].ExistingEntityID != existing.ID {
		t.Fatalf("a proposal naming an entity that exists does not say so: %+v", byName["Espresso"])
	}
	if byName["Espresso"].Parent != "/coffee/" || !slices.Equal(byName["Espresso"].Related, []string{"/coffee/filter/"}) {
		t.Fatalf("the espresso proposal lost its neighbors: %+v", byName["Espresso"])
	}

	entities, err := f.service.ListEntities(t.Context(), appgraph.ListEntitiesRequest{SiteID: f.siteID})
	if err != nil || len(entities.Items) != 1 {
		t.Fatalf("a preview wrote entities: %+v, %v", entities.Items, err)
	}
	unmapped, err := f.pages.Get(t.Context(), hub.ID)
	if err != nil || unmapped.EntityID != nil {
		t.Fatalf("a preview mapped a page: %+v, %v", unmapped, err)
	}
}

func TestApplyProposalsWritesWhatWasChosen(t *testing.T) {
	t.Parallel()

	f := newProposeFixture(t, &scriptedModel{replies: []string{proposal}}, fixedProfiles{})
	hub := f.page(t, "/coffee/", "Coffee")
	espresso := f.page(t, "/coffee/espresso/", "Espresso")
	f.page(t, "/coffee/filter/", "Filter coffee")
	existing := f.entity(t, "Espresso")

	preview, err := f.service.PreviewFromPages(t.Context(), appgraph.PreviewFromPagesRequest{SiteID: f.siteID})
	if err != nil {
		t.Fatalf("PreviewFromPages: %v", err)
	}
	chosen := make([]appgraph.ProposedEntity, 0, 2)
	for _, proposed := range preview.Entities {
		if proposed.Name != "Filter Coffee" {
			chosen = append(chosen, proposed)
		}
	}

	applied, err := f.service.ApplyProposals(t.Context(), appgraph.ApplyProposalsRequest{SiteID: f.siteID, Entities: chosen})
	if err != nil {
		t.Fatalf("ApplyProposals: %v", err)
	}
	if len(applied.Entities) != 1 || applied.Entities[0].Name != "Coffee" || applied.Mapped != 2 {
		t.Fatalf("applied = %+v, want the hub created and two pages mapped", applied)
	}
	if len(applied.Edges) != 1 || applied.Edges[0].Kind != string(graphdomain.EdgeParent) {
		t.Fatalf("edges = %+v, want the parent edge alone, the sibling being left out", applied.Edges)
	}

	mappedEspresso, err := f.pages.Get(t.Context(), espresso.ID)
	if err != nil || mappedEspresso.EntityID == nil || *mappedEspresso.EntityID != existing.ID {
		t.Fatalf("the espresso page was not mapped to the entity that exists: %+v, %v", mappedEspresso, err)
	}
	mappedHub, err := f.pages.Get(t.Context(), hub.ID)
	if err != nil || mappedHub.EntityID == nil {
		t.Fatalf("the hub page was not mapped: %+v, %v", mappedHub, err)
	}
	created, err := f.service.GetEntity(t.Context(), appgraph.GetEntityRequest{ID: *mappedHub.EntityID})
	if err != nil || created.Entity.CanonicalPageID == nil || *created.Entity.CanonicalPageID != hub.ID {
		t.Fatalf("the new entity did not take its page as canonical: %+v, %v", created.Entity, err)
	}

	again, err := f.service.ApplyProposals(t.Context(), appgraph.ApplyProposalsRequest{SiteID: f.siteID, Entities: chosen})
	if err != nil || len(again.Entities) != 0 || again.Mapped != 0 || again.Skipped != 2 {
		t.Fatalf("applying the same proposals again = %+v, %v; want everything skipped", again, err)
	}
	if _, err = f.service.ApplyProposals(t.Context(), appgraph.ApplyProposalsRequest{
		SiteID: f.siteID, Entities: []appgraph.ProposedEntity{{Name: " ", Kind: "topic"}},
	}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("a nameless proposal = %v, want INVALID", err)
	}
}

func TestProposeFromKeywordsProposesWithoutPages(t *testing.T) {
	t.Parallel()

	f := newProposeFixture(t, &scriptedModel{replies: []string{keywordProposal}}, fixedProfiles{})
	coffee := f.entity(t, "Coffee")

	out, err := f.service.ProposeFromKeywords(t.Context(), appgraph.ProposeFromKeywordsRequest{
		SiteID: f.siteID, Keywords: []string{"trail running shoes", " road running shoes ", "Trail Running Shoes"}, ParentEntityID: coffee.ID,
	})
	if err != nil {
		t.Fatalf("ProposeFromKeywords: %v", err)
	}

	prompt := f.model.calls[0].Messages[0].Text
	for _, want := range []string{"- trail running shoes", "- road running shoes", "Coffee (", "sits under Coffee"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the prompt does not carry %q:\n%s", want, prompt)
		}
	}
	if strings.Count(prompt, "running shoes") != 2 {
		t.Errorf("a keyword given twice reached the model twice:\n%s", prompt)
	}

	if len(out.Entities) != 2 || out.Skipped != 1 {
		t.Fatalf("proposals = %+v, want the two given keywords and the invented one skipped", out)
	}
	trail := out.Entities[0]
	if trail.PageID != "" || trail.Path != "" || trail.PrimaryKeyword != "trail running shoes" || trail.Parent != "Coffee" {
		t.Fatalf("the trail proposal = %+v", trail)
	}
	if road := out.Entities[1]; road.Parent != "Coffee" || !slices.Equal(road.Related, []string{"Trail Running Shoes"}) {
		t.Fatalf("the road proposal did not take the parent asked for: %+v", road)
	}

	applied, err := f.service.ApplyProposals(t.Context(), appgraph.ApplyProposalsRequest{SiteID: f.siteID, Entities: out.Entities})
	if err != nil {
		t.Fatalf("ApplyProposals: %v", err)
	}
	if len(applied.Entities) != 2 || applied.Mapped != 0 || len(applied.Edges) != 3 {
		t.Fatalf("applied = %+v, want two entities, no page and three edges", applied)
	}
	for _, entity := range applied.Entities {
		if entity.CanonicalPageID != nil {
			t.Fatalf("an entity from a keyword took a canonical page: %+v", entity)
		}
	}

	if _, err = f.service.ProposeFromKeywords(t.Context(), appgraph.ProposeFromKeywordsRequest{SiteID: f.siteID}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("no keywords = %v, want INVALID", err)
	}
}

func TestThePagesPromptCarriesTheKeywordsOfThePage(t *testing.T) {
	t.Parallel()

	f := newProposeFixture(t, &scriptedModel{replies: []string{proposal}}, fixedProfiles{})
	keyed := f.page(t, "/coffee/", "Coffee")
	keyed.PrimaryKeyword = "best coffee beans"
	keyed.Keywords = []string{"arabica", "robusta"}
	if err := f.pages.Update(t.Context(), keyed); err != nil {
		t.Fatalf("update the page: %v", err)
	}

	if _, err := f.service.PreviewFromPages(t.Context(), appgraph.PreviewFromPagesRequest{SiteID: f.siteID}); err != nil {
		t.Fatalf("PreviewFromPages: %v", err)
	}
	prompt := f.model.calls[0].Messages[0].Text
	for _, want := range []string{"primary keyword: best coffee beans", "keywords: arabica, robusta", "stands for that keyword"} {
		if !strings.Contains(prompt+f.model.calls[0].System, want) {
			t.Errorf("the prompt does not carry %q", want)
		}
	}
}
