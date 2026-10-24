package steps_test

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	theirParagraph  = "<p>The owner wrote this paragraph about research peptides.</p>"
	ourOwnParagraph = "<p>We wrote this paragraph about storing them cold.</p>"
	editedBody      = theirParagraph + "\n\n" + ourOwnParagraph
)

type editStand struct {
	deps   steps.Deps
	server *wptest.Server
	page   pagemap.Page
	bodies *bodyLog
	fields *fieldLog
}

type rewritingTypes struct{}

func (rewritingTypes) Set(_ context.Context, siteID string) (pagetype.Set, error) {
	kind := pagetype.Type{ID: "type-entity", SiteID: siteID, Name: "Entity", Preset: pagetype.Entity}
	scopes := pagetype.DefaultScopes(pagetype.Entity, kind.ID)
	for i := range scopes {
		if scopes[i].Key == field.Body {
			scopes[i].Rewrites = true
		}
	}
	return pagetype.NewSet([]pagetype.Type{kind}, scopes), nil
}

func newEditStand(t *testing.T, item wptest.Item, opts ...wptest.Option) editStand {
	t.Helper()

	deps, server := imageDepsWith(t, opts...)
	held := server.Seed(item)[0]
	page := pagemap.Page{
		ID: "page-edit", SiteID: "site", Path: "/peptides/", Slug: "peptides", WPType: pagemap.WPType(item.Type),
		Status: pagemap.StatusPublished, WPID: &held.ID, EntityID: pointer("parent"),
	}
	target := pagemap.Page{ID: "page-coffee", SiteID: "site", Path: "/coffee/", Slug: "coffee", WPType: pagemap.WPPage,
		Status: pagemap.StatusPublished, WPID: pointerOf(int64(9001)), EntityID: pointer("child")}
	bodies := &bodyLog{written: map[string][]string{page.ID: {block.Parse(item.Content).Blocks[len(block.Parse(item.Content).Blocks)-1].Hash}}}
	fields := &fieldLog{}
	deps.Pages, deps.Links, deps.Bodies, deps.Fields = newPageMap(page, target), &linkRecorder{}, bodies, fields
	deps.Types = scopedTypes{}
	return editStand{deps: deps, server: server, page: page, bodies: bodies, fields: fields}
}

func pointerOf[T any](value T) *T {
	return &value
}

func editContext(page pagemap.Page, edit run.Edit) *run.StepContext {
	return &run.StepContext{
		Run:       run.Run{ID: "edit-run", SiteID: "site", Kind: run.KindEdit, PublishMode: run.PublishLive},
		Item:      run.Item{ID: "edit-item", RunID: "edit-run", SiteID: "site", TargetID: page.ID, CurrentStep: "edit_page"},
		Params:    run.EditRecipe(edit)[0].Params,
		Artifacts: map[run.ArtifactKind]run.Artifact{},
		Check:     run.NewCheckpoint(),
	}
}

func (s editStand) edit(t *testing.T, edit run.Edit) (steps.PublishResult, run.Result) {
	t.Helper()

	result, err := drive(t, steps.EditPage(s.deps), editContext(s.page, edit))
	if err != nil {
		t.Fatalf("EditPage(%+v): %v", edit, err)
	}
	var published steps.PublishResult
	for i := range result.Artifacts {
		if result.Artifacts[i].Kind == run.ArtifactPublishResult {
			if err = json.Unmarshal(result.Artifacts[i].Blob, &published); err != nil {
				t.Fatalf("decode the publish result: %v", err)
			}
		}
	}
	return published, result
}

func (s editStand) stored(t *testing.T) wptest.Item {
	t.Helper()

	held, ok := s.server.Lookup(*s.page.WPID)
	if !ok {
		t.Fatal("the page left the site")
	}
	return held
}

func TestAnAgentReplacesOurBlockAndTheClientsBlockStandsByteForByte(t *testing.T) {
	t.Parallel()

	s := newEditStand(t, wptest.Item{Type: wptest.TypePage, Title: "Peptides", Slug: "peptides", Content: editedBody})
	published, result := s.edit(t, run.Edit{
		Op: run.EditReplace, Index: 1, HTML: `<p class="x">We keep them cold, and dark, for longer life.</p>`, Hash: block.BodyHash(editedBody),
	})
	if result.Next == run.TransitionPause {
		t.Fatalf("the edit paused: %s", result.Message)
	}
	held := s.stored(t)
	blocks := block.Parse(held.Content).Blocks
	if len(blocks) != 2 || strings.TrimSpace(blocks[0].Raw) != theirParagraph {
		t.Fatalf("the site holds %q, want the owner's paragraph first and untouched", held.Content)
	}
	if !strings.Contains(blocks[1].Raw, "for longer life") || strings.Contains(held.Content, "class=") {
		t.Fatalf("the site holds %q, want our new paragraph, scrubbed", held.Content)
	}
	if ours := s.bodies.ours(s.page.ID); !slices.Equal(ours, []string{blocks[1].Hash}) {
		t.Errorf("our blocks are %v, want the new one alone", ours)
	}
	if published.Version == 0 || published.PreviousContent != editedBody || published.ContentHash != block.BodyHash(held.Content) {
		t.Fatalf("the publish result = %+v, want a record a revert reads", published)
	}

	recordedFor(t, &s.deps, s.page, published)
	if reverted, _ := revertOf(t, s.deps, s.page.ID); reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v", reverted)
	}
	if back := s.stored(t); back.Content != editedBody {
		t.Errorf("after the revert the site holds %q, want %q byte for byte", back.Content, editedBody)
	}
}

func TestAnAgentInsertsABlockBetweenTheClientsAndOurs(t *testing.T) {
	t.Parallel()

	s := newEditStand(t, wptest.Item{Type: wptest.TypePage, Title: "Peptides", Slug: "peptides", Content: editedBody})
	if _, result := s.edit(t, run.Edit{
		Op: run.EditInsertAfter, Index: 0, HTML: "<h2>Storage</h2>", Hash: block.BodyHash(editedBody),
	}); result.Next == run.TransitionPause {
		t.Fatalf("the insert paused: %s", result.Message)
	}
	blocks := block.Parse(s.stored(t).Content).Blocks
	if len(blocks) != 3 || strings.TrimSpace(blocks[0].Raw) != theirParagraph || strings.TrimSpace(blocks[2].Raw) != ourOwnParagraph ||
		!strings.Contains(blocks[1].Raw, "<h2>Storage</h2>") {
		t.Fatalf("the site holds %+v, want the heading between the owner's paragraph and ours", blocks)
	}
	if ours := s.bodies.ours(s.page.ID); !slices.Contains(ours, blocks[1].Hash) || !slices.Contains(ours, blocks[2].Hash) || slices.Contains(ours, blocks[0].Hash) {
		t.Errorf("our blocks are %v, want the heading and our paragraph", ours)
	}
}

func TestAnAgentsBlockInABlockEditorBodyIsOneHTMLBlock(t *testing.T) {
	t.Parallel()

	theirs := "<!-- wp:paragraph -->\n<p>The owner's own opening.</p>\n<!-- /wp:paragraph -->"
	ours := "<!-- wp:html -->\n<p>Our earlier note.</p>\n<!-- /wp:html -->"
	body := theirs + "\n\n" + ours
	s := newEditStand(t, wptest.Item{Type: wptest.TypePage, Title: "Guide", Slug: "guide", Content: body})
	if _, result := s.edit(t, run.Edit{
		Op: run.EditInsertAfter, Index: 0, HTML: "<p>A note on the grind.</p>", Hash: block.BodyHash(body),
	}); result.Next == run.TransitionPause {
		t.Fatalf("the insert paused: %s", result.Message)
	}
	stored := block.Parse(s.stored(t).Content)
	if stored.Kind != block.Blocks || len(stored.Blocks) != 3 || strings.TrimSpace(stored.Blocks[0].Raw) != theirs {
		t.Fatalf("the site holds %+v, want three blocks of a block-editor body, the owner's first", stored)
	}
	if raw := stored.Blocks[1].Raw; !strings.Contains(raw, "<!-- wp:html -->") || !strings.Contains(raw, "A note on the grind.") {
		t.Errorf("the new block is %q, want one core/html block of ours", raw)
	}
}

func TestAnAgentEditOverAClientsBlockWaitsForRewritesOrAStillBody(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		edit run.Edit
	}{
		{name: "the client's block replaced", edit: run.Edit{Op: run.EditReplace, Index: 0, HTML: "<p>Ours.</p>", Hash: block.BodyHash(editedBody)}},
		{name: "the client's block deleted", edit: run.Edit{Op: run.EditDelete, Index: 0, Hash: block.BodyHash(editedBody)}},
		{name: "a body that moved since it was listed", edit: run.Edit{Op: run.EditDelete, Index: 1, Hash: block.BodyHash("<p>older</p>")}},
		{name: "a block past the end", edit: run.Edit{Op: run.EditDelete, Index: 5, Hash: block.BodyHash(editedBody)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := newEditStand(t, wptest.Item{Type: wptest.TypePage, Title: "Peptides", Slug: "peptides", Content: editedBody})
			_, result := s.edit(t, tc.edit)
			if result.Next != run.TransitionPause || result.Reason != run.PauseNeedsHuman {
				t.Fatalf("the edit answered %+v, want it held for a person", result)
			}
			if held := s.stored(t); held.Content != editedBody {
				t.Errorf("the refused edit wrote %q", held.Content)
			}
		})
	}
}

func TestAnAgentReplacesTheClientsBlockWhereItsTypeRewrites(t *testing.T) {
	t.Parallel()

	s := newEditStand(t, wptest.Item{Type: wptest.TypePage, Title: "Peptides", Slug: "peptides", Content: editedBody})
	s.deps.Types = rewritingTypes{}
	if _, result := s.edit(t, run.Edit{
		Op: run.EditReplace, Index: 0, HTML: "<p>Our rewrite of the owner's opening.</p>", Hash: block.BodyHash(editedBody),
	}); result.Next == run.TransitionPause {
		t.Fatalf("the rewrite paused: %s", result.Message)
	}
	blocks := block.Parse(s.stored(t).Content).Blocks
	if !strings.Contains(blocks[0].Raw, "Our rewrite") || !slices.Contains(s.bodies.ours(s.page.ID), blocks[0].Hash) {
		t.Fatalf("the site holds %+v and ours are %v, want the rewritten block ours", blocks, s.bodies.ours(s.page.ID))
	}
}

func TestAnAgentPlacesALinkInTheClientsBlockAndTheBlockStaysHis(t *testing.T) {
	t.Parallel()

	s := newEditStand(t, wptest.Item{Type: wptest.TypePage, Title: "Peptides", Slug: "peptides", Content: editedBody})
	if _, result := s.edit(t, run.Edit{
		Op: run.EditPlaceLink, Index: 0, TargetID: "page-coffee", Anchor: "research peptides", Hash: block.BodyHash(editedBody),
	}); result.Next == run.TransitionPause {
		t.Fatalf("the link paused: %s", result.Message)
	}
	blocks := block.Parse(s.stored(t).Content).Blocks
	if !strings.Contains(blocks[0].Raw, `href=`) || !strings.Contains(blocks[0].Raw, `/coffee/`) || strings.TrimSpace(blocks[1].Raw) != ourOwnParagraph {
		t.Fatalf("the site holds %+v, want the link in the owner's paragraph and ours untouched", blocks)
	}
	if slices.Contains(s.bodies.ours(s.page.ID), blocks[0].Hash) {
		t.Errorf("the owner's paragraph became ours by carrying a link of ours")
	}
}

func TestABlockEditWaitsForThePlugin(t *testing.T) {
	t.Parallel()

	s := newEditStand(t, wptest.Item{Type: wptest.TypePage, Title: "Peptides", Slug: "peptides", Content: editedBody}, wptest.WithoutPlugin())
	_, result := s.edit(t, run.Edit{Op: run.EditDelete, Index: 1, Hash: block.BodyHash(editedBody)})
	if result.Next != run.TransitionPause || s.stored(t).Content != editedBody {
		t.Fatalf("a block edit without the plugin answered %+v, want it held and the body untouched", result)
	}
}

func TestAnAgentSetsAFieldUnderTheValueItReadAndARevertPutsItBack(t *testing.T) {
	t.Parallel()

	s := newEditStand(t, wptest.Item{Type: wptest.TypePost, Title: "Espresso", Slug: "espresso", Status: "publish",
		Content: editedBody, Excerpt: "The owner's excerpt."})
	s.fields.values = []field.Value{field.NewValue(s.page.ID, field.Excerpt, "The owner's excerpt.", sqlitetest.Stamp)}
	published, result := s.edit(t, run.Edit{
		Op: run.EditSetField, Field: string(field.Excerpt), Value: "Pull a balanced shot.", Hash: field.Hash("The owner's excerpt."),
	})
	if result.Next == run.TransitionPause {
		t.Fatalf("the field edit paused: %s", result.Message)
	}
	if held := s.stored(t); held.Excerpt != "Pull a balanced shot." || held.Content != editedBody {
		t.Fatalf("the site holds %q over %q, want the new excerpt and the body untouched", held.Excerpt, held.Content)
	}
	if got := s.fields.recordedValue(field.Excerpt); got != "Pull a balanced shot." {
		t.Errorf("the model recorded %q as ours, want the excerpt the site stored", got)
	}
	if len(published.Fields) != 1 || published.Fields[0].Before != "The owner's excerpt." || len(published.Blocks) != 0 {
		t.Fatalf("the publish result = %+v, want the excerpt's change alone", published)
	}

	recordedFor(t, &s.deps, s.page, published)
	if reverted, _ := revertOf(t, s.deps, s.page.ID); reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v", reverted)
	}
	if back := s.stored(t); back.Excerpt != "The owner's excerpt." {
		t.Errorf("after the revert the excerpt is %q, want the owner's back", back.Excerpt)
	}
}

func TestAnAgentFieldEditIsHeldWhenTheValueMovedSinceItWasRead(t *testing.T) {
	t.Parallel()

	s := newEditStand(t, wptest.Item{Type: wptest.TypePost, Title: "Espresso", Slug: "espresso", Status: "publish",
		Content: editedBody, Excerpt: "The owner's newer excerpt."})
	s.fields.values = []field.Value{field.NewValue(s.page.ID, field.Excerpt, "The owner's excerpt.", sqlitetest.Stamp)}
	_, result := s.edit(t, run.Edit{
		Op: run.EditSetField, Field: string(field.Excerpt), Value: "Pull a balanced shot.", Hash: field.Hash("The owner's excerpt."),
	})
	if result.Next != run.TransitionPause || s.stored(t).Excerpt != "The owner's newer excerpt." {
		t.Fatalf("an edit over a moved excerpt answered %+v and left %q, want it held and his excerpt kept",
			result, s.stored(t).Excerpt)
	}
}
