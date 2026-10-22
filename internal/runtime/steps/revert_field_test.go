package steps_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type fieldStand struct {
	deps      steps.Deps
	server    *wptest.Server
	page      pagemap.Page
	published steps.PublishResult
}

func revertOf(t *testing.T, deps steps.Deps, pageID string, fields ...string) (steps.RevertResult, run.Result) {
	t.Helper()

	parent := sourceRunID
	params := map[string]any{}
	if len(fields) > 0 {
		wanted := make([]any, 0, len(fields))
		for _, key := range fields {
			wanted = append(wanted, key)
		}
		params[steps.ParamRevertFields] = wanted
	}
	sc := &run.StepContext{
		Run:       run.Run{ID: "revert-run", SiteID: "site", Kind: run.KindRevert, ParentRunID: &parent},
		Item:      run.Item{ID: "revert-" + pageID, RunID: "revert-run", SiteID: "site", TargetID: pageID},
		Params:    params,
		Artifacts: map[run.ArtifactKind]run.Artifact{},
		Check:     run.NewCheckpoint(),
	}
	result, err := steps.Revert(deps).Run(t.Context(), sc)
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

func recordedFor(t *testing.T, deps *steps.Deps, page pagemap.Page, published steps.PublishResult) {
	t.Helper()

	source := newSourceRecord()
	source.record(t, page.ID, map[run.ArtifactKind]any{run.ArtifactPublishResult: published})
	deps.Pages, deps.Links, deps.Items, deps.Artifacts = newPageMap(page), &linkRecorder{}, source, source
}

func publishedPost(t *testing.T) fieldStand {
	t.Helper()

	deps, server := imageDeps(t)
	deps.Types = scopedTypes{keys: []field.Key{field.Excerpt}}
	seeded := server.Seed(wptest.Item{Type: wptest.TypePost, Title: "The owner's title", Slug: "espresso-post", Status: "publish",
		Content: clientText, Excerpt: "The owner's excerpt."})[0]
	if _, err := syncClient(t, server).SetSEOMeta(t.Context(), wp.TypePost, seeded.ID, wp.SEOMeta{Title: "The owner's SEO title"}, nil); err != nil {
		t.Fatalf("seed the owner's SEO title: %v", err)
	}
	sc := rewritten(publishContext(t))
	sc.Page.WPType, sc.Page.WPID, sc.Page.Slug, sc.Page.Path = pagemap.WPPost, &seeded.ID, "espresso-post", "/espresso-post/"
	withFields(t, sc, steps.FieldValue{Key: string(field.Excerpt), Value: "Pull a shot."})
	published := runPublish(t, deps, sc)
	if published.Version == 0 {
		t.Fatalf("the publish keeps no record version: %+v", published)
	}
	page := sc.Page
	recordedFor(t, &deps, page, published)
	return fieldStand{deps: deps, server: server, page: page, published: published}
}

func seoOf(t *testing.T, server *wptest.Server, wpID int64) wp.SEOMeta {
	t.Helper()

	meta, err := syncClient(t, server).GetSEOMeta(t.Context(), wp.TypePost, wpID)
	if err != nil {
		t.Fatalf("read the SEO meta: %v", err)
	}
	return meta
}

func outcomeOf(reverted steps.RevertResult, key field.Key) string {
	for _, held := range reverted.Fields {
		if held.Key == key {
			return held.Outcome
		}
	}
	return ""
}

func TestARevertPutsBackEachFieldItWrote(t *testing.T) {
	t.Parallel()

	s := publishedPost(t)
	if stored, _ := s.server.Lookup(*s.page.WPID); stored.Excerpt != "Pull a shot." || stored.Content == clientText {
		t.Fatalf("the publish wrote %+v, want the excerpt and the body ours", stored)
	}
	reverted, result := revertOf(t, s.deps, s.page.ID)
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
	}
	stored, _ := s.server.Lookup(*s.page.WPID)
	if stored.Excerpt != "The owner's excerpt." || stored.Content != clientText {
		t.Errorf("after the revert the post holds %q / %q, want the owner's", stored.Excerpt, stored.Content)
	}
	if meta := seoOf(t, s.server, *s.page.WPID); meta.Title != "The owner's SEO title" {
		t.Errorf("the SEO title is %q, want the owner's back", meta.Title)
	}
	if stored.Status != "publish" {
		t.Errorf("the status became %q: a revert never touches it", stored.Status)
	}
	for _, key := range []field.Key{field.Excerpt, field.SEOTitle} {
		if got := outcomeOf(reverted, key); got != steps.OutcomeRestored {
			t.Errorf("%s reads %q in the revert, want restored", key, got)
		}
	}
}

func TestAFieldChangedSinceOurWriteIsHandedBack(t *testing.T) {
	t.Parallel()

	s := publishedPost(t)
	held, _ := s.server.Lookup(*s.page.WPID)
	held.Excerpt = "The owner rewrote it after the run."
	s.server.Restore(held)

	reverted, _ := revertOf(t, s.deps, s.page.ID)
	stored, _ := s.server.Lookup(*s.page.WPID)
	if stored.Excerpt != "The owner rewrote it after the run." {
		t.Errorf("the owner's later excerpt became %q", stored.Excerpt)
	}
	if stored.Content != clientText {
		t.Errorf("the body = %q, want it put back beside the kept excerpt", stored.Content)
	}
	if got := outcomeOf(reverted, field.Excerpt); got != steps.OutcomeKept {
		t.Errorf("the excerpt reads %q, want kept", got)
	}
	finding, found := findingOf(reverted, steps.CodeRevertFieldKept)
	if !found || !strings.Contains(finding.Message, string(field.Excerpt)) {
		t.Errorf("findings = %+v, want the kept excerpt named", reverted.Findings)
	}
}

func TestTheSEOMetaIsNotPutBackOverTheClientsEdit(t *testing.T) {
	t.Parallel()

	s := publishedPost(t)
	if _, err := syncClient(t, s.server).SetSEOMeta(t.Context(), wp.TypePost, *s.page.WPID, wp.SEOMeta{Title: "The owner's newer SEO title"}, nil); err != nil {
		t.Fatalf("edit the SEO title: %v", err)
	}
	reverted, _ := revertOf(t, s.deps, s.page.ID)
	if meta := seoOf(t, s.server, *s.page.WPID); meta.Title != "The owner's newer SEO title" {
		t.Errorf("the SEO title became %q, want the owner's later edit kept", meta.Title)
	}
	if got := outcomeOf(reverted, field.SEOTitle); got != steps.OutcomeKept {
		t.Errorf("the SEO title reads %q, want kept", got)
	}
}

func TestARevertOfOneFieldLeavesTheOthers(t *testing.T) {
	t.Parallel()

	s := publishedPost(t)
	reverted, _ := revertOf(t, s.deps, s.page.ID, string(field.Excerpt))
	stored, _ := s.server.Lookup(*s.page.WPID)
	if stored.Excerpt != "The owner's excerpt." {
		t.Errorf("the excerpt is %q, want it put back", stored.Excerpt)
	}
	if stored.Content == clientText {
		t.Error("the body was put back too, though the revert named the excerpt alone")
	}
	if meta := seoOf(t, s.server, *s.page.WPID); meta.Title == "The owner's SEO title" {
		t.Error("the SEO title was put back too, though the revert named the excerpt alone")
	}
	if len(reverted.Fields) != 1 {
		t.Errorf("the revert reports %+v, want the excerpt alone", reverted.Fields)
	}
}

func TestASecondRevertPutsBackWhatTheFirstLeft(t *testing.T) {
	t.Parallel()

	s := publishedPost(t)
	revertOf(t, s.deps, s.page.ID, string(field.Excerpt))
	reverted, result := revertOf(t, s.deps, s.page.ID)
	if result.Next == run.TransitionPause {
		t.Fatalf("the second revert paused: %s", result.Message)
	}
	stored, _ := s.server.Lookup(*s.page.WPID)
	if stored.Excerpt != "The owner's excerpt." || stored.Content != clientText {
		t.Errorf("after the second revert the post holds %q / %q, want everything the owner's", stored.Excerpt, stored.Content)
	}
	if got := outcomeOf(reverted, field.Excerpt); got != steps.OutcomeRestored {
		t.Errorf("the excerpt the first revert put back reads %q, want restored with nothing written over it", got)
	}
}

func TestARevertRestoresOurBlocksAmongTheClientsEdits(t *testing.T) {
	t.Parallel()

	clientBody := "<p>The owner's first paragraph.</p>\n\n<p>The owner's second paragraph.</p>"
	kept := &content.KeptBody{Raw: clientBody, Hash: block.BodyHash(clientBody)}
	deps, server, _, sc := keptRunOver(t, clientBody, oursBody, kept)
	published := runPublish(t, deps, sc)
	if stored, _ := server.Lookup(published.WPID); stored.Content != clientBody+"\n\n"+oursBody {
		t.Fatalf("the publish wrote %q, want ours after the owner's paragraphs", stored.Content)
	}
	recordedFor(t, &deps, sc.Page, published)

	held, _ := server.Lookup(published.WPID)
	held.Content = strings.Replace(held.Content, "The owner's first paragraph.", "The owner reworded his first paragraph.", 1)
	server.Restore(held)

	reverted, result := revertOf(t, deps, sc.Page.ID)
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
	}
	want := "<p>The owner reworded his first paragraph.</p>\n\n<p>The owner's second paragraph.</p>"
	if stored, _ := server.Lookup(published.WPID); stored.Content != want {
		t.Errorf("after the revert the body is %q, want our block gone and the owner's later edit kept: %q", stored.Content, want)
	}
}

func TestACreatedPageTheClientEditedIsHandedBack(t *testing.T) {
	t.Parallel()

	for _, edited := range []bool{true, false} {
		deps, server := imageDeps(t)
		sc := publishContext(t)
		sc.Run.PublishMode = run.PublishLive
		published := runPublish(t, deps, sc)
		if !published.Created {
			t.Fatalf("the publish did not create the page: %+v", published)
		}
		page := sc.Page
		page.WPID = &published.WPID
		recordedFor(t, &deps, page, published)
		if edited {
			held, _ := server.Lookup(published.WPID)
			held.Content += "<p>The owner added a paragraph to the page we made.</p>"
			server.Restore(held)
		}

		reverted, result := revertOf(t, deps, page.ID)
		stored, _ := server.Lookup(published.WPID)
		switch {
		case edited && (result.Next != run.TransitionPause || reverted.Outcome == steps.OutcomeTrashed || stored.Status == "trash"):
			t.Errorf("a created page the owner edited = %+v / %s (status %s), want it handed back and left", reverted, result.Message, stored.Status)
		case !edited && reverted.Outcome != steps.OutcomeTrashed:
			t.Errorf("a created page still as we wrote it = %+v, want it trashed", reverted)
		}
	}
}

func TestAnOlderRunStillRevertsWhole(t *testing.T) {
	t.Parallel()

	s := publishedPost(t)
	older := s.published
	older.Version = 0
	older.Fields, older.Blocks = nil, nil
	recordedFor(t, &s.deps, s.page, older)

	reverted, result := revertOf(t, s.deps, s.page.ID, string(field.Excerpt))
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("an older record's revert answered %+v / %s", reverted, result.Message)
	}
	if stored, _ := s.server.Lookup(*s.page.WPID); stored.Content != clientText || stored.Excerpt != "The owner's excerpt." {
		t.Errorf("an older record put back %q / %q, want the whole page as before, whatever fields the revert named", stored.Content, stored.Excerpt)
	}
	if len(reverted.Fields) != 0 {
		t.Errorf("an older record reports fields %+v, want the whole revert", reverted.Fields)
	}
}

func TestAnOlderRunsSEOMetaIsPutBackOnlyWhileItHoldsWhatThatRunWrote(t *testing.T) {
	t.Parallel()

	for _, edited := range []bool{false, true} {
		s := publishedPost(t)
		client := syncClient(t, s.server)
		if _, err := client.SetSEOMeta(t.Context(), wp.TypePost, *s.page.WPID, wp.SEOMeta{Title: "Ours"}, nil); err != nil {
			t.Fatalf("write the older run's SEO title: %v", err)
		}
		if edited {
			if _, err := client.SetSEOMeta(t.Context(), wp.TypePost, *s.page.WPID, wp.SEOMeta{Title: "What the client wrote since"}, nil); err != nil {
				t.Fatalf("edit the SEO title: %v", err)
			}
		}
		older := s.published
		older.Version, older.Fields, older.Blocks = 0, nil, nil
		older.SEOApplied = []string{"title"}
		previous := wp.SEOMeta{Title: "The owner's SEO title"}
		older.PreviousMeta = &previous
		source := newSourceRecord()
		source.record(t, s.page.ID, map[run.ArtifactKind]any{
			run.ArtifactPublishResult: older,
			run.ArtifactMeta:          map[string]string{"title": "Ours", "description": "Ours."},
		})
		s.deps.Items, s.deps.Artifacts = source, source

		reverted, _ := revertOf(t, s.deps, s.page.ID)
		meta := seoOf(t, s.server, *s.page.WPID)
		want := "The owner's SEO title"
		if edited {
			want = "What the client wrote since"
		}
		if meta.Title != want {
			t.Errorf("edited %t: the SEO title is %q, want %q", edited, meta.Title, want)
		}
		if _, kept := findingOf(reverted, steps.CodeRevertMetaKept); kept != edited {
			t.Errorf("edited %t: the revert reported the meta kept = %t, want it kept only over the client's edit", edited, kept)
		}
	}
}

func TestAnOlderRunsCanonicalIsPutBackOnlyWhileItHoldsWhatThatRunWrote(t *testing.T) {
	t.Parallel()

	for _, edited := range []bool{false, true} {
		s := publishedPost(t)
		client := syncClient(t, s.server)
		if _, err := client.SetSEOMeta(t.Context(), wp.TypePost, *s.page.WPID, wp.SEOMeta{Canonical: "https://shop.example.com/ours/"}, nil); err != nil {
			t.Fatalf("write the older run's canonical: %v", err)
		}
		if edited {
			if _, err := client.SetSEOMeta(t.Context(), wp.TypePost, *s.page.WPID, wp.SEOMeta{Canonical: "https://shop.example.com/theirs/"}, nil); err != nil {
				t.Fatalf("edit the canonical: %v", err)
			}
		}
		older := s.published
		older.Version, older.Fields, older.Blocks = 0, nil, nil
		older.SEOApplied = append(slices.Clone(older.SEOApplied), "canonical")
		previous := wp.SEOMeta{Title: "The owner's SEO title", Canonical: "https://shop.example.com/before/"}
		older.PreviousMeta = &previous
		source := newSourceRecord()
		source.record(t, s.page.ID, map[run.ArtifactKind]any{
			run.ArtifactPublishResult: older,
			run.ArtifactMeta: map[string]string{
				"title": s.published.WrittenMeta.Title, "description": s.published.WrittenMeta.Description,
				"canonical": "https://shop.example.com/ours/",
			},
		})
		s.deps.Items, s.deps.Artifacts = source, source

		revertOf(t, s.deps, s.page.ID)
		meta := seoOf(t, s.server, *s.page.WPID)
		want := "https://shop.example.com/before/"
		if edited {
			want = "https://shop.example.com/theirs/"
		}
		if meta.Canonical != want {
			t.Errorf("edited %t: the canonical is %q, want %q", edited, meta.Canonical, want)
		}
	}
}
