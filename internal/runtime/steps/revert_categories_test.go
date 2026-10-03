package steps_test

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

var itemRoute = regexp.MustCompile(`^/wp-json/wp/v2/(pages|posts)/\d+$`)

func publishedForRevert(t *testing.T, deps steps.Deps, sc *run.StepContext) (steps.Deps, steps.PublishResult) {
	t.Helper()

	recorded := &pagemap.Page{}
	deps.Pages = pageList{recorded: recorded}
	published := runPublish(t, deps, sc)

	source := newSourceRecord()
	source.record(t, recorded.ID, map[run.ArtifactKind]any{run.ArtifactPublishResult: published})
	deps.Pages = newPageMap(*recorded)
	deps.Links = &linkRecorder{}
	deps.Items = source
	deps.Artifacts = source
	return deps, published
}

func filedItem(t *testing.T, server *wptest.Server, wpType string) (item wptest.Item, own int64) {
	t.Helper()

	own = server.SeedCategory(wptest.Category{Name: "News"}).ID
	item = server.Seed(wptest.Item{
		Type: wpType, Title: "Espresso", Slug: "espresso", Content: updatedBefore, Categories: []int64{own},
	})[0]
	return item, own
}

func termsKeptSays(t *testing.T, reverted steps.RevertResult, names ...string) {
	t.Helper()

	kept, found := findingOf(reverted, steps.CodeRevertTermsKept)
	if !found {
		t.Fatalf("the revert said nothing of the categories the run created: %+v", reverted.Findings)
	}
	if kept.Severity != content.SeverityInfo {
		t.Errorf("the finding = %+v, want it told as information", kept)
	}
	for _, name := range names {
		if !strings.Contains(kept.Message, name) {
			t.Errorf("the finding reads %q, want it to name %s", kept.Message, name)
		}
	}
}

func TestRevertTakesBackOnlyTheCategoriesTheRunAdded(t *testing.T) {
	t.Parallel()

	for _, wpType := range []string{wptest.TypePage, wptest.TypePost} {
		t.Run("a "+wpType, func(t *testing.T) {
			t.Parallel()

			deps, server := imageDeps(t)
			filed(&deps)
			item, own := filedItem(t, server, wpType)
			sc := publishContext(t)
			sc.Page.WPType = pagemap.WPType(wpType)
			deps, published := publishedForRevert(t, deps, sc)
			if published.Categories == nil || len(published.Categories.Added) != 2 {
				t.Fatalf("categories = %+v, want two added", published.Categories)
			}

			for range 2 {
				reverted, result := runRevert(t, deps, "page-child")
				if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
					t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
				}
				if failed, found := findingOf(reverted, steps.CodeRevertCategoriesKept); found {
					t.Fatalf("the revert kept the categories: %+v", failed)
				}
				termsKeptSays(t, reverted, "Drinks", "Coffee")
			}

			stored, _ := server.Lookup(item.ID)
			if !slices.Equal(stored.Categories, []int64{own}) {
				t.Errorf("the item carries %v, want only its own %d", stored.Categories, own)
			}
			if stored.Content != updatedBefore {
				t.Errorf("the item holds %q, want the body the run replaced", stored.Content)
			}
			if len(server.Categories()) != 3 {
				t.Errorf("the site holds %+v, want the categories the run created kept", server.Categories())
			}
		})
	}
}

func TestRevertTakesEveryCategoryOffAnItemThatHadNone(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	filed(&deps)
	item := server.Seed(wptest.Item{Type: wptest.TypePost, Title: "Espresso", Slug: "espresso", Content: updatedBefore})[0]
	sc := publishContext(t)
	sc.Page.WPType = pagemap.WPPost
	deps, _ = publishedForRevert(t, deps, sc)

	server.ResetRequests()
	reverted, result := runRevert(t, deps, "page-child")
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
	}
	stored, _ := server.Lookup(item.ID)
	if len(stored.Categories) != 0 {
		t.Errorf("the post carries %v, want none", stored.Categories)
	}
	sent := false
	for _, request := range server.Requests() {
		if request.Method == http.MethodPost && itemRoute.MatchString(request.Path) {
			sent = sent || strings.Contains(string(request.Body), `"categories":[]`)
		}
	}
	if !sent {
		t.Error("the revert never sent an empty category list")
	}
}

func TestRevertTrashesACreatedPageAndKeepsTheCategoriesItCreated(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	filed(&deps)
	deps, published := publishedForRevert(t, deps, publishContext(t))

	reverted, result := runRevert(t, deps, "page-child")
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeTrashed {
		t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
	}
	termsKeptSays(t, reverted, "Drinks", "Coffee")
	if stored, _ := server.Lookup(published.WPID); stored.Status != "trash" {
		t.Errorf("the page is %q, want it in the trash", stored.Status)
	}
	if len(server.Categories()) != 2 {
		t.Errorf("the site holds %+v, want the categories kept", server.Categories())
	}
}

func TestRevertSaysSoWhenItCannotTakeTheCategoriesBack(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	filed(&deps)
	item, own := filedItem(t, server, wptest.TypePage)
	deps, published := publishedForRevert(t, deps, publishContext(t))
	deps.WordPress = oneClient{client: behind(t, server, func(w http.ResponseWriter, r *http.Request, forward http.Handler) {
		if r.Method == http.MethodPost && itemRoute.MatchString(r.URL.Path) {
			failWith(t, w, http.StatusInternalServerError)
			return
		}
		forward.ServeHTTP(w, r)
	})}

	reverted, result := runRevert(t, deps, "page-child")
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
	}
	kept, found := findingOf(reverted, steps.CodeRevertCategoriesKept)
	if !found || kept.Severity != content.SeverityWarn || kept.Details["reason"] == "" {
		t.Fatalf("the revert said %+v, want a warning that the categories stay", reverted.Findings)
	}
	stored, _ := server.Lookup(item.ID)
	if !sameSet(stored.Categories, append([]int64{own}, published.Categories.Added...)) {
		t.Errorf("the page carries %v, want it left as the run filed it", stored.Categories)
	}
	if stored.Content != updatedBefore {
		t.Errorf("the page holds %q, want its body back whatever its categories", stored.Content)
	}
}

func TestRevertTakesBackOnlyTheProductCategoriesTheRunAdded(t *testing.T) {
	t.Parallel()

	h := newProductHarness(t, storeProduct())
	filed(&h.deps)
	assigned := fileProduct(t, h, "Machines")
	published := runPublish(t, h.deps, storeContext(t, h.held.ID))
	if published.Categories == nil || len(published.Categories.Added) != 2 {
		t.Fatalf("categories = %+v, want two added", published.Categories)
	}
	p := recordedProduct(t, h, published)

	reverted, result := runRevert(t, p.deps, "page-child")
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
	}
	termsKeptSays(t, reverted, "Drinks", "Coffee")

	stored, _ := p.server.Lookup(p.held.ID)
	if !slices.Equal(stored.Categories, assigned) {
		t.Errorf("the product carries %v, want only the client's %v", stored.Categories, assigned)
	}
	if stored.Excerpt != "<p>old short</p>" || stored.Content != "<p>old description</p>" {
		t.Errorf("the product holds %q / %q, want what the run replaced", stored.Excerpt, stored.Content)
	}
	if len(productCategories(p.server)) != 3 {
		t.Errorf("the store holds %+v, want the categories the run created kept", productCategories(p.server))
	}
}
