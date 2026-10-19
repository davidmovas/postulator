package steps_test

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	clientText = "<p>Step-by-step guides for handling research peptides in the laboratory.</p>"
	oursBody   = "<p>Espresso is a way to make coffee.</p>"
)

type bodyLog struct {
	mu      sync.Mutex
	bodies  map[string]string
	written map[string][]string
}

func (b *bodyLog) Record(_ context.Context, pageID, raw string, _ time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.bodies == nil {
		b.bodies = map[string]string{}
	}
	b.bodies[pageID] = raw
	return nil
}

func (b *bodyLog) ReplaceWritten(_ context.Context, pageID string, hashes []string, _ string, _ time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.written == nil {
		b.written = map[string][]string{}
	}
	b.written[pageID] = append([]string(nil), hashes...)
	return nil
}

func (b *bodyLog) ours(pageID string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.written[pageID]...)
}

func keptDraft(t *testing.T, kept *content.KeptBody) []byte {
	t.Helper()

	var draft content.ContentDraft
	if err := json.Unmarshal([]byte(goodDraft), &draft); err != nil {
		t.Fatalf("decode the draft: %v", err)
	}
	draft.Kept = kept
	encoded, err := json.Marshal(draft)
	if err != nil {
		t.Fatalf("encode the draft: %v", err)
	}
	return encoded
}

func keptRun(t *testing.T, ours string, kept *content.KeptBody) (steps.Deps, *wptest.Server, *bodyLog, *run.StepContext) {
	t.Helper()

	deps, server := imageDeps(t)
	log := &bodyLog{}
	deps.Bodies = log
	seeded := server.Seed(wptest.Item{Type: "page", Title: "Espresso", Slug: "espresso", Status: "publish", Content: clientText})
	sc := publishContext(t)
	sc.Page.WPID = &seeded[0].ID
	sc.Artifacts[run.ArtifactBodyHTML] = run.Artifact{Kind: run.ArtifactBodyHTML, Blob: []byte("<h1>Espresso</h1>" + ours)}
	sc.Artifacts[run.ArtifactDraft] = run.Artifact{Kind: run.ArtifactDraft, Blob: keptDraft(t, kept)}
	return deps, server, log, sc
}

func TestTheClientsBlocksAreKeptWhenRewritesIsOff(t *testing.T) {
	t.Parallel()

	kept := &content.KeptBody{Raw: clientText, Hash: block.BodyHash(clientText)}
	deps, server, log, sc := keptRun(t, oursBody, kept)

	published := runPublish(t, deps, sc)
	stored, _ := server.Lookup(published.WPID)
	if stored.Content != clientText+"\n\n"+oursBody {
		t.Fatalf("the site holds %q, want the client's paragraph byte for byte and ours after it, with no h1", stored.Content)
	}
	if published.PreviousContent != clientText {
		t.Errorf("the publish kept %q as what it replaced, want the client's body", published.PreviousContent)
	}
	ours := log.ours(sc.Page.ID)
	if len(ours) == 0 {
		t.Fatal("no block was recorded as ours")
	}
	for _, hash := range ours {
		if hash == block.Hash(clientText) {
			t.Errorf("the client's paragraph was recorded as ours")
		}
	}
}

func TestTwoRunsLeaveOneCopyOfOurSections(t *testing.T) {
	t.Parallel()

	first := &content.KeptBody{Raw: clientText, Hash: block.BodyHash(clientText)}
	deps, server, log, sc := keptRun(t, oursBody, first)
	published := runPublish(t, deps, sc)
	stored, _ := server.Lookup(published.WPID)

	again := "<p>Espresso is pulled under nine bars.</p>"
	second := &content.KeptBody{Raw: stored.Content, Hash: block.BodyHash(stored.Content), Written: log.ours(sc.Page.ID)}
	sc.Artifacts[run.ArtifactBodyHTML] = run.Artifact{Kind: run.ArtifactBodyHTML, Blob: []byte("<h1>Espresso</h1>" + again)}
	sc.Artifacts[run.ArtifactDraft] = run.Artifact{Kind: run.ArtifactDraft, Blob: keptDraft(t, second)}
	runPublish(t, deps, sc)

	rewritten, _ := server.Lookup(published.WPID)
	if rewritten.Content != clientText+"\n\n"+again {
		t.Errorf("after two runs the site holds %q, want the client's text and only the second run's part", rewritten.Content)
	}
	if strings.Contains(rewritten.Content, oursBody) {
		t.Errorf("the first run's part is still there:\n%s", rewritten.Content)
	}
}

func TestAClientEditAfterTheReadPauses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		edit func(server *wptest.Server, id int64)
	}{
		{name: "an edit before the publish reads", edit: func(server *wptest.Server, id int64) {
			server.Rewrite(id, clientText+"<p>Added by the owner.</p>")
		}},
		{name: "an edit between the read and the write", edit: func(server *wptest.Server, id int64) {
			server.EditBeforeNextRawWrite(id, clientText+"<p>Added by the owner.</p>")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			kept := &content.KeptBody{Raw: clientText, Hash: block.BodyHash(clientText)}
			deps, server, _, sc := keptRun(t, oursBody, kept)
			tc.edit(server, *sc.Page.WPID)

			result, err := steps.Publish(deps).Run(t.Context(), sc)
			if err != nil {
				t.Fatalf("Publish: %v", err)
			}
			if result.Next != run.TransitionPause || !strings.Contains(result.Message, steps.ReasonClientChanged) {
				t.Fatalf("result = %+v, want the page paused for the owner's edit", result)
			}
			stored, _ := server.Lookup(*sc.Page.WPID)
			if strings.Contains(stored.Content, oursBody) {
				t.Errorf("our part reached the site over the owner's edit: %q", stored.Content)
			}
		})
	}
}

func TestAKeptBodyWithoutThePluginPauses(t *testing.T) {
	t.Parallel()

	kept := &content.KeptBody{Raw: clientText, Hash: block.BodyHash(clientText)}
	deps, server := imageDepsWith(t, wptest.WithoutPlugin())
	deps.Bodies = &bodyLog{}
	seeded := server.Seed(wptest.Item{Type: "page", Title: "Espresso", Slug: "espresso", Status: "publish", Content: clientText})
	sc := publishContext(t)
	sc.Page.WPID = &seeded[0].ID
	sc.Artifacts[run.ArtifactDraft] = run.Artifact{Kind: run.ArtifactDraft, Blob: keptDraft(t, kept)}

	result, err := steps.Publish(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Next != run.TransitionPause || !strings.Contains(result.Message, steps.ReasonKeptNeedsPlugin) {
		t.Fatalf("result = %+v, want the page paused for the plugin", result)
	}
}

func TestGenerateBodyKeepsTheClientsTextUnlessRewritesIsOn(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		written  bool
		rewrite  bool
		wantKept bool
	}{
		{name: "the client's text is kept", wantKept: true},
		{name: "a body that is all ours is rewritten", written: true},
		{name: "a run that rewrites the page", rewrite: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			stub := stubOf(t, deps)
			stub.bodies = map[string]block.Snapshot{"page-child": {PageID: "page-child", Side: block.Current, Raw: clientText, Kind: block.Classic}}
			if tc.written {
				stub.written = map[string][]string{"page-child": {block.Hash(clientText)}}
			}
			recorder := &promptRecorder{reply: goodDraft}
			deps.LLM = recorder
			sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: linkContextBlob(t, deps)})
			if tc.rewrite {
				sc.Params[run.ParamRewrite] = []any{"page-child"}
			}

			result, err := steps.GenerateBody(deps).Run(t.Context(), sc)
			if err != nil {
				t.Fatalf("GenerateBody: %v", err)
			}
			var draft content.ContentDraft
			if err := json.Unmarshal(result.Artifacts[0].Blob, &draft); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if (draft.Kept != nil) != tc.wantKept {
				t.Fatalf("kept = %+v, want kept %t", draft.Kept, tc.wantKept)
			}
			if tc.wantKept && (draft.Kept.Raw != clientText || draft.Kept.Hash != block.BodyHash(clientText)) {
				t.Errorf("kept = %+v, want the body and hash the writer read", draft.Kept)
			}
			prompt := recorder.last
			if tc.wantKept && !strings.Contains(prompt, "keep it as it stands") {
				t.Errorf("the writer was not told to keep the client's text:\n%s", prompt)
			}
			if tc.rewrite && !strings.Contains(prompt, "may be rewritten") {
				t.Errorf("the writer was not told the run rewrites the page:\n%s", prompt)
			}
		})
	}
}

func TestTheClientsLinkSatisfiesWhatThePageOwesAndIsNotPlacedTwice(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	blob := linkContextBlob(t, deps)
	theirs := `<p>Read the <a href="/coffee/">coffee</a> guide first.</p>`
	kept := &content.KeptBody{Raw: theirs, Hash: block.BodyHash(theirs)}
	body := []byte(`<h1>Espresso</h1><p>Espresso is a way to make coffee.</p>`)
	artifacts := map[run.ArtifactKind][]byte{run.ArtifactLinkContext: blob, run.ArtifactBodyHTML: body, run.ArtifactDraft: keptDraft(t, kept)}

	inserted, err := steps.InsertLinks(deps).Run(t.Context(), unitContext(t, artifacts))
	if err != nil {
		t.Fatalf("InsertLinks: %v", err)
	}
	if strings.Contains(string(inserted.Artifacts[0].Blob), `href="/coffee/"`) {
		t.Errorf("our part links the parent the client's text already links: %s", inserted.Artifacts[0].Blob)
	}

	artifacts[run.ArtifactBodyHTML] = inserted.Artifacts[0].Blob
	sc := unitContext(t, artifacts)
	sc.Check = inserted.Checkpoint
	validated, err := steps.Validate(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	var report steps.ValidationReport
	if err = json.Unmarshal(validated.Artifacts[0].Blob, &report); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, item := range report.Compliance.Items {
		if item.Code == content.CodeTargetMissing {
			t.Errorf("compliance = %+v, want the client's link to count for the owed parent", report.Compliance.Items)
		}
	}
}

func TestADisabledFieldIsNeitherWrittenNorSent(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	deps.Types = disabledSEO{}
	seeded := server.Seed(wptest.Item{Type: "page", Title: "The owner's title", Slug: "espresso", Status: "publish", Content: oursBody})
	sc := publishContext(t)
	sc.Page.WPID = &seeded[0].ID

	published := runPublish(t, deps, sc)
	stored, _ := server.Lookup(published.WPID)
	if stored.Title != "The owner's title" {
		t.Errorf("the title became %q, want the owner's kept: its scope is off", stored.Title)
	}
	if len(published.SEOApplied) != 0 || !slices.Contains(published.Skipped, steps.CodeSEOMetaKept) {
		t.Errorf("seo applied %v, skipped %v; want the snippet kept", published.SEOApplied, published.Skipped)
	}
}

type disabledSEO struct{}

func (disabledSEO) Set(_ context.Context, siteID string) (pagetype.Set, error) {
	kind := pagetype.Type{ID: "type-entity", SiteID: siteID, Name: "Entity", Preset: pagetype.Entity}
	body := pagetype.DefaultScope(pagetype.Entity, field.Body, kind.ID)
	title := pagetype.DefaultScope(pagetype.Entity, field.SEOTitle, kind.ID)
	title.Enabled = false
	description := pagetype.DefaultScope(pagetype.Entity, field.SEODescription, kind.ID)
	description.AIWrites = false
	return pagetype.NewSet([]pagetype.Type{kind}, []field.Scope{body, title, description}), nil
}
