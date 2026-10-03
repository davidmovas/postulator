//go:build e2e

package e2e_test

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	ourShelf      = "Postulator Vials "
	ourSmallShelf = "Postulator Small Vials "
	clientShelf   = "Postulator Client Shelf "
	vialKeyword   = "postulator vial"
)

var shelfPrefixes = []string{ourShelf, ourSmallShelf, clientShelf}

func (s *site) sweepShelves(t *testing.T) {
	t.Helper()

	searches := make([]string, 0, len(shelfPrefixes))
	for _, prefix := range shelfPrefixes {
		searches = append(searches, strings.TrimSpace(prefix))
	}
	s.dropTerms(t, wp.TaxonomyProductCategory, func(term siteTerm) bool {
		for _, prefix := range shelfPrefixes {
			if strings.HasPrefix(term.Name, prefix) {
				return true
			}
		}
		return false
	}, searches...)
}

func TestAProductKeepsTheClientsCategoryAndGainsOurs(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	requireStore(t, live)
	live.sweepShelves(t)
	defer live.sweepShelves(t)

	tag := strconv.FormatInt(time.Now().UnixNano()%1_000_000_000, 10)
	shelf := live.createTerm(t, wp.TaxonomyProductCategory, clientShelf+tag)
	vial := live.createProduct(t, "Postulator Vial "+tag, "postulator-vial-"+tag, shelf.ID)
	if !sameSet(vial.categoryIDs(), []int64{shelf.ID}) {
		t.Fatalf("the client filed the vial under %v, want %d", vial.categoryIDs(), shelf.ID)
	}
	vialPath := "/product/" + vial.Slug + "/"

	core := openCoreWith(t, fake.NewScripted(append(replies(),
		fake.Reply{Step: steps.NameGenerateBody, Match: vialPath, Text: productAnswer(vialKeyword)},
		fake.Reply{Step: steps.NameGenerateMeta, Match: vialPath, Text: metaOf(vialKeyword)},
	)...))
	siteID, checked := openFilingSite(t, core, live, "Docker Store With Shelves")
	if !checked.Plugin.Installed || checked.Commerce != "ready" {
		t.Fatalf("the check reports %+v, want the plugin and a store that can be edited", checked)
	}

	vials, small := ourShelf+tag, ourSmallShelf+tag
	sheet := filepath.Join(t.TempDir(), "shelves.csv")
	rows := "Category,Subcategory,Path,H1,Keywords\n" +
		vials + "," + small + ",/" + vial.Slug + "/,Postulator Vial " + tag + "," + vialKeyword + "\n"
	if err := os.WriteFile(sheet, []byte(rows), 0o600); err != nil {
		t.Fatalf("write the sheet: %v", err)
	}
	seen, err := core.Imports.Inspect(t.Context(), imports.InspectRequest{SiteID: siteID, Path: sheet})
	if err != nil {
		t.Fatalf("inspect the sheet: %v", err)
	}
	if seen.Detected.Options.RowType != importmap.RowProducts ||
		!slices.Equal(seen.Detected.Options.LevelColumns, []string{"Category", "Subcategory"}) {
		t.Fatalf("the sheet is detected as %+v, want its rows read as the store's products under two levels",
			seen.Detected.Options)
	}
	applied, err := core.Imports.Apply(t.Context(), imports.ApplyRequest{SiteID: siteID, Path: sheet, Mapping: seen.Detected})
	if err != nil || len(applied.Report.Errors) != 0 {
		t.Fatalf("apply the sheet: %v %+v", err, applied.Report.Errors)
	}

	page := pagesByPath(t, core.Pages, siteID)[vialPath]
	if page.WPType != string(pagemap.WPProduct) || page.WPID == nil || *page.WPID != int64(vial.ID) || page.EntityID == nil {
		t.Fatalf("the import left %s as %+v, want the client's product %d with an entity", vialPath, page, vial.ID)
	}
	chain := categoryChainOf(t, core, siteID, *page.EntityID)
	if !slices.Equal(chain, []string{vials, small}) {
		t.Fatalf("the product sits in the chain %v, want %v", chain, []string{vials, small})
	}

	started, err := core.Runs.Start(t.Context(), runs.StartRequest{
		SiteID: siteID, PageIDs: []string{page.ID}, TemplateID: productTemplate(t, core),
		PublishMode: string(run.PublishLive), Recipe: recipe(),
	})
	if err != nil {
		t.Fatalf("start the product run: %v", err)
	}
	awaitRun(t, core.Runs, started.RunID)
	items, err := core.Runs.ListItems(t.Context(), runs.ListItemsRequest{RunID: started.RunID, ListRequest: dto.ListRequest{Limit: 5}})
	if err != nil || len(items.Items) != 1 {
		t.Fatalf("the product run carries %+v (%v), want one item", items.Items, err)
	}
	if items.Items[0].Status != string(run.StatusCompleted) {
		t.Fatalf("the product item is %q: %s%s", items.Items[0].Status, items.Items[0].Error,
			artifactDump(t, core.Runs, items.Items[0].ID, string(run.ArtifactValidationReport)))
	}
	final := finalReport(t, core.Runs, items.Items[0].ID)
	if final.Publish == nil || final.Publish.Categories == nil {
		t.Fatalf("%s went up without a category write: %+v", vialPath, final.Publish)
	}
	written := final.Publish.Categories

	assertTheChainIsOnTheSite(t, live, wp.TaxonomyProductCategory, vialPath, written, chain)
	if created := createdIDsOf(written); !sameSet(created, termIDsOf(written)) {
		t.Errorf("the product run says it created %v of %v, want both levels", created, termIDsOf(written))
	}
	if !sameSet(written.Previous, []int64{shelf.ID}) || !sameSet(written.Added, termIDsOf(written)) {
		t.Errorf("the product run read %v and added %v, want the client's %d and then the chain %v",
			written.Previous, written.Added, shelf.ID, termIDsOf(written))
	}
	if held := live.product(t, vial.ID).categoryIDs(); !sameSet(held, append([]int64{shelf.ID}, termIDsOf(written)...)) {
		t.Errorf("the store files the vial under %v, want the client's %d beside our %v", held, shelf.ID, termIDsOf(written))
	}

	reverted, err := core.Runs.Revert(t.Context(), runs.RevertRequest{RunID: started.RunID})
	if err != nil {
		t.Fatalf("revert the product run: %v", err)
	}
	awaitRun(t, core.Runs, reverted.RunID)
	undone, err := core.Runs.ListItems(t.Context(), runs.ListItemsRequest{RunID: reverted.RunID, ListRequest: dto.ListRequest{Limit: 5}})
	if err != nil || len(undone.Items) != 1 {
		t.Fatalf("the revert carries %+v (%v), want one item", undone.Items, err)
	}
	result := revertResultOf(t, core, undone.Items[0].ID)
	if result.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert of %s answered %+v, want the product restored", vialPath, result)
	}
	if kept, found := findingOf(result.Findings, steps.CodeRevertTermsKept); !found {
		t.Errorf("the revert of %s carries %+v, want a %s finding", vialPath, result.Findings, steps.CodeRevertTermsKept)
	} else if ids := idsIn(kept.Details["termIds"]); !sameSet(ids, createdIDsOf(written)) {
		t.Errorf("the revert says it kept %v, want the %v the run created", ids, createdIDsOf(written))
	}

	back := live.product(t, vial.ID)
	if !sameSet(back.categoryIDs(), []int64{shelf.ID}) {
		t.Errorf("after the revert the store files the vial under %v, want only the client's %d", back.categoryIDs(), shelf.ID)
	}
	if back.Description != clientDescription {
		t.Errorf("after the revert the vial reads %q, want the client's own description", back.Description)
	}
	for i := range written.Terms {
		if onSite := live.termsNamed(t, wp.TaxonomyProductCategory, written.Terms[i].Name); len(onSite) != 1 {
			t.Errorf("after the revert the store holds %+v named %q, want the category kept", onSite, written.Terms[i].Name)
		}
	}
}
