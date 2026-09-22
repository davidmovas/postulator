package steps_test

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	neighborBefore = `<h1>Coffee</h1><p>We roast every espresso blend we sell.</p>`
	updatedBefore  = `<h1>Filter</h1><p>What a human wrote about filter coffee.</p>`
	sourceRunID    = "source-run"
)

type pageMap struct {
	items map[string]pagemap.Page
	order []string
}

func newPageMap(pages ...pagemap.Page) *pageMap {
	held := &pageMap{items: make(map[string]pagemap.Page, len(pages))}
	for i := range pages {
		held.items[pages[i].ID] = pages[i]
		held.order = append(held.order, pages[i].ID)
	}
	return held
}

func (p *pageMap) ListBySite(context.Context, string) ([]pagemap.Page, error) {
	out := make([]pagemap.Page, 0, len(p.order))
	for _, pageID := range p.order {
		out = append(out, p.items[pageID])
	}
	return out, nil
}

func (p *pageMap) Get(_ context.Context, pageID string) (pagemap.Page, error) {
	page, ok := p.items[pageID]
	if !ok {
		return pagemap.Page{}, errors.New(errors.NotFound, "no such page").WithDetail("pageId", pageID)
	}
	return page, nil
}

func (p *pageMap) Insert(_ context.Context, page pagemap.Page) error {
	p.items[page.ID] = page
	p.order = append(p.order, page.ID)
	return nil
}

func (p *pageMap) Update(_ context.Context, page pagemap.Page) error {
	p.items[page.ID] = page
	return nil
}

type sourceRecord struct {
	items     []run.Item
	artifacts map[string][]run.Artifact
}

func newSourceRecord() *sourceRecord {
	return &sourceRecord{artifacts: make(map[string][]run.Artifact)}
}

func (s *sourceRecord) ByRun(_ context.Context, runID string) ([]run.Item, error) {
	if runID != sourceRunID {
		return nil, nil
	}
	return s.items, nil
}

func (s *sourceRecord) ByItem(_ context.Context, itemID string) ([]run.Artifact, error) {
	return s.artifacts[itemID], nil
}

func (s *sourceRecord) record(t *testing.T, pageID string, blobs map[run.ArtifactKind]any) {
	t.Helper()

	itemID := "item-" + pageID
	s.items = append(s.items, run.Item{ID: itemID, RunID: sourceRunID, SiteID: "site", TargetID: pageID})
	for kind, value := range blobs {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("encode the %s of %s: %v", kind, pageID, err)
		}
		s.artifacts[itemID] = append(s.artifacts[itemID], run.Artifact{
			ID: itemID + "-" + string(kind), RunID: sourceRunID, ItemID: itemID, Step: "publish",
			Kind: kind, Blob: encoded,
		})
	}
}

type revertStand struct {
	deps   steps.Deps
	server *wptest.Server
	pages  *pageMap
	links  *linkRecorder
	source *sourceRecord
	wpIDs  map[string]int64
}

func newRevertStand(t *testing.T, opts ...wptest.Option) *revertStand {
	t.Helper()

	deps, server := imageDepsWith(t, opts...)
	seeded := server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Coffee", Slug: "coffee", Content: neighborBefore},
		wptest.Item{Type: wptest.TypePage, Title: "Filter", Slug: "filter", Content: updatedBefore},
	)

	neighbor, updated := seeded[0].ID, seeded[1].ID
	espresso, ristretto := int64(101), int64(102)
	pages := newPageMap(
		pagemap.Page{
			ID: "page-parent", SiteID: "site", Path: "/coffee/", Slug: "coffee", WPType: pagemap.WPPage,
			Status: pagemap.StatusPublished, EntityID: pointer("parent"), WPID: &neighbor,
			ContentHash: wp.ContentHash(neighborBefore),
		},
		pagemap.Page{
			ID: "page-filter", SiteID: "site", Path: "/filter/", Slug: "filter", WPType: pagemap.WPPage,
			Status: pagemap.StatusPublished, WPID: &updated,
		},
		pagemap.Page{
			ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", Slug: "espresso", WPType: pagemap.WPPage,
			Status: pagemap.StatusPublished, EntityID: pointer("child"), WPID: &espresso,
		},
		pagemap.Page{
			ID: "page-ristretto", SiteID: "site", Path: "/coffee/ristretto/", Slug: "ristretto",
			WPType: pagemap.WPPage, Status: pagemap.StatusPublished, WPID: &ristretto,
		},
	)

	links := &linkRecorder{}
	deps.Pages = pages
	deps.Links = links
	source := newSourceRecord()
	deps.Items = source
	deps.Artifacts = source

	return &revertStand{
		deps: deps, server: server, pages: pages, links: links, source: source,
		wpIDs: map[string]int64{
			"page-parent": neighbor, "page-filter": updated,
			"page-child": espresso, "page-ristretto": ristretto,
		},
	}
}

func (s *revertStand) created(t *testing.T, pageID string) {
	t.Helper()

	wpID := s.wpIDs[pageID]
	body := `<h1>New</h1><p>What the run wrote to ` + pageID + `.</p>`
	s.server.Seed(wptest.Item{Type: wptest.TypePage, Title: "New", Slug: pageID, Content: body})

	held := s.server.Items()
	for i := range held {
		if held[i].Slug == pageID {
			wpID = held[i].ID
		}
	}
	page := s.pages.items[pageID]
	page.WPID = &wpID
	page.ContentHash = wp.ContentHash(body)
	s.pages.items[pageID] = page
	s.wpIDs[pageID] = wpID

	s.source.record(t, pageID, map[run.ArtifactKind]any{
		run.ArtifactPublishResult: steps.PublishResult{
			WPID: wpID, Created: true, ContentHash: wp.ContentHash(body), SEOApplied: []string{}, Skipped: []string{},
		},
	})
}

func (s *revertStand) updated(t *testing.T, pageID, body string) {
	t.Helper()
	s.updatedOverMeta(t, pageID, body, nil)
}

func (s *revertStand) updatedOverMeta(t *testing.T, pageID, body string, previous *wp.SEOMeta) {
	t.Helper()

	wpID := s.wpIDs[pageID]
	client := syncClient(t, s.server)
	if _, err := client.PutRaw(t.Context(), wpID, body, ""); err != nil {
		t.Fatalf("write the run body to %s: %v", pageID, err)
	}

	page := s.pages.items[pageID]
	page.ContentHash = wp.ContentHash(body)
	s.pages.items[pageID] = page

	applied := []string{"title"}
	if previous != nil {
		applied = []string{"title", "description"}
		if _, err := client.SetSEOMeta(t.Context(), wpID, wp.SEOMeta{
			Title: "written by the run", Description: "written by the run",
		}); err != nil {
			t.Fatalf("write the run meta to %s: %v", pageID, err)
		}
	}

	s.source.record(t, pageID, map[run.ArtifactKind]any{
		run.ArtifactPublishResult: steps.PublishResult{
			WPID: wpID, Created: false, ContentHash: wp.ContentHash(body),
			PreviousContent: updatedBefore, PreviousContentHash: wp.ContentHash(updatedBefore),
			PreviousMeta: previous,
			SEOApplied:   applied, Skipped: []string{},
		},
	})
}

func findingOf(reverted steps.RevertResult, code string) (content.Finding, bool) {
	for i := range reverted.Findings {
		if reverted.Findings[i].Code == code {
			return reverted.Findings[i], true
		}
	}
	return content.Finding{}, false
}

func TestRevertPutsBackTheSEOMetaTheRunReplaced(t *testing.T) {
	t.Parallel()

	stand := newRevertStand(t, wptest.WithSEOPlugin("yoast"))
	stand.updatedOverMeta(t, "page-filter", runBody, &wp.SEOMeta{Title: "what a human wrote"})

	reverted, _ := stand.revert(t, "page-filter")
	if reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("reverted = %+v", reverted)
	}
	if kept, found := findingOf(reverted, steps.CodeRevertMetaKept); found {
		t.Fatalf("the revert gave up on meta it could put back: %+v", kept)
	}

	stored, ok := stand.server.Lookup(stand.wpIDs["page-filter"])
	if !ok {
		t.Fatal("the page is gone from the site")
	}
	if stored.Meta["_yoast_wpseo_title"] != "what a human wrote" {
		t.Errorf("title = %q, want the value the run replaced", stored.Meta["_yoast_wpseo_title"])
	}
	if stored.Meta["_yoast_wpseo_metadesc"] != "" {
		t.Errorf("description = %q, want a field that carried nothing before the run cleared",
			stored.Meta["_yoast_wpseo_metadesc"])
	}
}

func TestRevertKeepsTheSEOMetaItHasNoCopyOf(t *testing.T) {
	t.Parallel()

	stand := newRevertStand(t)
	stand.updated(t, "page-filter", runBody)

	reverted, _ := stand.revert(t, "page-filter")
	if reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("reverted = %+v", reverted)
	}

	kept, found := findingOf(reverted, steps.CodeRevertMetaKept)
	if !found {
		t.Fatalf("the revert said nothing about the meta it left: %+v", reverted.Findings)
	}
	if kept.Severity != content.SeverityWarn {
		t.Errorf("the finding = %+v, want a warning", kept)
	}
	if kept.Details["reason"] != steps.ReasonRevertNoMeta {
		t.Errorf("the finding = %+v, want the reason %q", kept, steps.ReasonRevertNoMeta)
	}
}

func (s *revertStand) relinked(t *testing.T, pageID, neighborID, body string) {
	t.Helper()

	wpID := s.wpIDs[neighborID]
	client := syncClient(t, s.server)
	if _, err := client.PutRaw(t.Context(), wpID, body, ""); err != nil {
		t.Fatalf("write the relinked body to %s: %v", neighborID, err)
	}

	neighbor := s.pages.items[neighborID]
	neighbor.ContentHash = wp.ContentHash(body)
	s.pages.items[neighborID] = neighbor

	itemID := "item-" + pageID
	encoded, err := json.Marshal(steps.RelinkResult{
		Linked: 1,
		Neighbors: []steps.NeighborResult{{
			PageID: neighborID, Path: s.pages.items[neighborID].Path, WPID: wpID,
			Outcome: steps.OutcomeLinked, Anchor: "espresso",
			Before: steps.NeighborBefore{Hash: wp.ContentHash(neighborBefore), HTML: neighborBefore},
		}},
	})
	if err != nil {
		t.Fatalf("encode the relink result: %v", err)
	}
	s.source.artifacts[itemID] = append(s.source.artifacts[itemID], run.Artifact{
		ID: itemID + "-relink", RunID: sourceRunID, ItemID: itemID, Step: "relink_neighbors",
		Kind: run.ArtifactRelinkResult, Blob: encoded,
	})
}

func (s *revertStand) revert(t *testing.T, pageID string) (steps.RevertResult, run.Result) {
	t.Helper()

	parent := sourceRunID
	sc := &run.StepContext{
		Run:       run.Run{ID: "revert-run", SiteID: "site", Kind: run.KindRevert, ParentRunID: &parent},
		Item:      run.Item{ID: "revert-" + pageID, RunID: "revert-run", SiteID: "site", TargetID: pageID},
		Params:    map[string]any{},
		Artifacts: map[run.ArtifactKind]run.Artifact{},
		Check:     run.NewCheckpoint(),
	}

	result, err := steps.Revert(s.deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("Revert %s: %v", pageID, err)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Kind != run.ArtifactRevertResult {
		t.Fatalf("Revert produced %+v", result.Artifacts)
	}

	var reverted steps.RevertResult
	if err = json.Unmarshal(result.Artifacts[0].Blob, &reverted); err != nil {
		t.Fatalf("decode the revert result: %v", err)
	}
	return reverted, result
}

func (s *revertStand) body(t *testing.T, wpID int64) string {
	t.Helper()

	stored, ok := s.server.Lookup(wpID)
	if !ok {
		t.Fatalf("the site holds nothing at %d", wpID)
	}
	return stored.Content
}

func (s *revertStand) status(t *testing.T, wpID int64) string {
	t.Helper()

	stored, ok := s.server.Lookup(wpID)
	if !ok {
		t.Fatalf("the site holds nothing at %d", wpID)
	}
	return stored.Status
}

const runBody = `<h1>Filter</h1><p>What the run wrote about filter coffee.</p>`

func relinkedBody() string {
	return neighborBefore + `<p>Try our <a href="/coffee/espresso/">espresso</a>.</p>`
}

func TestRevertPutsBackWhatTheRunWroteToTheSite(t *testing.T) {
	t.Parallel()

	stand := newRevertStand(t)
	stand.created(t, "page-child")
	stand.created(t, "page-ristretto")
	stand.updated(t, "page-filter", runBody)
	stand.relinked(t, "page-filter", "page-parent", relinkedBody())

	for _, pageID := range []string{"page-ristretto", "page-child", "page-filter"} {
		reverted, result := stand.revert(t, pageID)
		if result.Next == run.TransitionPause {
			t.Fatalf("the revert of %s paused: %s", pageID, result.Message)
		}
		if reverted.Outcome != steps.OutcomeTrashed && reverted.Outcome != steps.OutcomeRestored {
			t.Fatalf("the revert of %s answered %+v", pageID, reverted)
		}
	}

	if got := stand.body(t, stand.wpIDs["page-filter"]); got != updatedBefore {
		t.Fatalf("the updated page holds %q, want the body the run replaced", got)
	}
	if got := stand.body(t, stand.wpIDs["page-parent"]); got != neighborBefore {
		t.Fatalf("the neighbor holds %q, want the content the relink replaced", got)
	}
	for _, pageID := range []string{"page-child", "page-ristretto"} {
		if got := stand.status(t, stand.wpIDs[pageID]); got != "trash" {
			t.Fatalf("the page the run created is %q on the site, want it in the trash", got)
		}
		page := stand.pages.items[pageID]
		if page.WPID != nil || page.Status != pagemap.StatusPlanned || page.ContentHash != "" {
			t.Fatalf("the local row of %s reads %+v, want it planned again", pageID, page)
		}
		if !page.Observed.Empty() || page.LastSyncedAt != nil || page.Drift {
			t.Fatalf("the local row of %s still mirrors the site: %+v", pageID, page)
		}
		if len(stand.links.byPage[pageID]) != 0 {
			t.Fatalf("the local row of %s still carries %d links", pageID, len(stand.links.byPage[pageID]))
		}
	}
}

func TestRevertPausesTheItemAHumanEditedSince(t *testing.T) {
	t.Parallel()

	stand := newRevertStand(t)
	stand.created(t, "page-child")
	stand.updated(t, "page-filter", runBody)

	edited := runBody + `<p>And a sentence a human added afterwards.</p>`
	if _, err := syncClient(t, stand.server).PutRaw(t.Context(), stand.wpIDs["page-filter"], edited, ""); err != nil {
		t.Fatalf("edit the page on the site: %v", err)
	}

	reverted, result := stand.revert(t, "page-filter")
	if result.Next != run.TransitionPause || result.Reason != run.PauseNeedsHuman {
		t.Fatalf("the revert answered %q / %q, want a pause for a human", result.Next, result.Reason)
	}
	if reverted.Outcome != steps.OutcomeNeedsHand {
		t.Fatalf("the revert answered %+v", reverted)
	}
	for _, hash := range []string{wp.ContentHash(runBody), wp.ContentHash(edited)} {
		if !slices.ContainsFunc(reverted.Findings, func(f content.Finding) bool {
			return strings.Contains(f.Message, hash)
		}) {
			t.Fatalf("the finding does not name %s: %+v", hash, reverted.Findings)
		}
	}
	if got := stand.body(t, stand.wpIDs["page-filter"]); got != edited {
		t.Fatalf("the page was written although a human had edited it: %q", got)
	}

	other, otherResult := stand.revert(t, "page-child")
	if otherResult.Next == run.TransitionPause || other.Outcome != steps.OutcomeTrashed {
		t.Fatalf("the other item of the run was held back too: %+v", other)
	}
}

func TestRevertWithoutThePluginOnlyTakesBackWhatItCreated(t *testing.T) {
	t.Parallel()

	stand := newRevertStand(t, wptest.WithoutPlugin())
	stand.created(t, "page-child")
	stand.source.record(t, "page-filter", map[run.ArtifactKind]any{
		run.ArtifactPublishResult: steps.PublishResult{
			WPID: stand.wpIDs["page-filter"], Created: false, ContentHash: wp.ContentHash(runBody),
			SEOApplied: []string{}, Skipped: []string{},
		},
	})

	created, createdResult := stand.revert(t, "page-child")
	if createdResult.Next == run.TransitionPause || created.Outcome != steps.OutcomeTrashed {
		t.Fatalf("a created page was not taken off a site without the plugin: %+v", created)
	}

	reverted, result := stand.revert(t, "page-filter")
	if result.Next != run.TransitionPause || result.Reason != run.PauseNeedsHuman {
		t.Fatalf("the updated page answered %q / %q, want a pause for a human", result.Next, result.Reason)
	}
	if !strings.Contains(reverted.Detail, "revision") {
		t.Fatalf("the refusal does not name the WordPress revisions: %q", reverted.Detail)
	}
}

func TestRevertRunAgainIsAQuietSecondPass(t *testing.T) {
	t.Parallel()

	stand := newRevertStand(t)
	stand.created(t, "page-child")
	stand.updated(t, "page-filter", runBody)
	stand.relinked(t, "page-filter", "page-parent", relinkedBody())

	for range 2 {
		reverted, result := stand.revert(t, "page-filter")
		if result.Next == run.TransitionPause {
			t.Fatalf("the second pass paused: %s", result.Message)
		}
		if reverted.Outcome != steps.OutcomeRestored || len(reverted.Neighbors) != 1 {
			t.Fatalf("the second pass answered %+v", reverted)
		}
	}
	if got := stand.body(t, stand.wpIDs["page-filter"]); got != updatedBefore {
		t.Fatalf("the page holds %q after two reverts", got)
	}
	if got := stand.body(t, stand.wpIDs["page-parent"]); got != neighborBefore {
		t.Fatalf("the neighbor holds %q after two reverts", got)
	}
}
