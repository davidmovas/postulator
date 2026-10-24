package main

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func measuredPages(t *testing.T) []pageRow {
	t.Helper()

	models, err := catalog.New(noOverrides{})
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	rows, err := measurePages(t.Context(), newCounters(), models)
	if err != nil {
		t.Fatalf("measurePages: %v", err)
	}
	return rows
}

func rowAt(t *testing.T, rows []pageRow, stand, path string) pageRow {
	t.Helper()

	for i := range rows {
		if rows[i].Stand == stand && rows[i].Path == path {
			return rows[i]
		}
	}
	t.Fatalf("no row for %s %s", stand, path)
	return pageRow{}
}

func TestEveryStandPageIsARowMeasuredOrSaidWhyNot(t *testing.T) {
	t.Parallel()

	rows := measuredPages(t)
	want := 0
	for _, name := range sitemodeltest.Stands() {
		model, err := sitemodeltest.Stand(name)
		if err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
		for i := range model.Pages {
			if model.Pages[i].Status != pagemap.StatusArchived {
				want++
			}
		}
	}
	if len(rows) != want {
		t.Fatalf("%d rows, want one per unarchived page of both stands, %d", len(rows), want)
	}

	measured := 0
	for i := range rows {
		row := &rows[i]
		switch {
		case row.Skipped != "":
			if row.Writer != nil || row.Judge != nil {
				t.Errorf("%s %s is skipped (%s) and measured", row.Stand, row.Path, row.Skipped)
			}
		case row.Writer == nil || row.Judge == nil || row.Linker == nil:
			t.Errorf("%s %s is neither skipped nor measured whole: %+v", row.Stand, row.Path, row)
		default:
			measured++
		}
	}
	if measured != 9 {
		t.Errorf("%d pages measured, want the nine mapped pages with a template (six hubs and entities and a product on the replica, two on vanilla)", measured)
	}
	if product := rowAt(t, rows, "replica", "/product/bpc-157-peptide/"); product.Skipped != "" || product.Writer == nil {
		t.Errorf("the product the capture gave an entity through its proposal is skipped for %q, want it measured", product.Skipped)
	}
	if unmapped := rowAt(t, rows, "replica", "/product/tb-500-peptide/"); !strings.Contains(unmapped.Skipped, "unmapped") {
		t.Errorf("a product no entity carries is skipped for %q, want the reason it has no entity", unmapped.Skipped)
	}
	if cart := rowAt(t, rows, "replica", "/cart/"); cart.Skipped == "" {
		t.Error("a technical page is measured, want it skipped")
	}
}

func TestAWritersPromptIsSplitIntoItsParts(t *testing.T) {
	t.Parallel()

	rows := measuredPages(t)
	for _, at := range []struct{ stand, path string }{{"replica", "/peptides/"}, {"replica", "/tb-500/"}, {"vanilla", "/guide/setup/"}} {
		row := rowAt(t, rows, at.stand, at.path)
		for name, asked := range map[string]*prompt{"writer": row.Writer, "judge": row.Judge} {
			if asked == nil {
				t.Fatalf("%s %s has no %s prompt", at.stand, at.path, name)
			}
			if asked.Instructions.Tokens <= 0 || asked.Pack.Tokens <= 0 || asked.Rest.Tokens <= 0 || asked.Schema.Tokens <= 0 {
				t.Errorf("%s %s %s = %+v, want every part counted", at.stand, at.path, name, asked)
			}
			if want := asked.Instructions.Tokens + asked.Pack.Tokens + asked.Rest.Tokens + asked.Schema.Tokens; asked.Total.Tokens != want {
				t.Errorf("%s %s %s totals %d tokens, want its parts' %d", at.stand, at.path, name, asked.Total.Tokens, want)
			}
			if len(asked.Sections) == 0 || asked.Sections[0].Name != "page" {
				t.Errorf("%s %s %s sections = %+v, want the pack section by section, the page first", at.stand, at.path, name, asked.Sections)
			}
			if asked.Model == "" {
				t.Errorf("%s %s %s names no model", at.stand, at.path, name)
			}
		}
		if !hasSection(row.Writer.Sections, "rules") {
			t.Errorf("%s %s's writer prompt carries no field rules: %+v", at.stand, at.path, row.Writer.Sections)
		}
	}
	if !rowAt(t, rows, "replica", "/peptides/").NoBody || rowAt(t, rows, "replica", "/tb-500/").NoBody {
		t.Error("an empty hub must be marked as having no body, and the hub with the client's text must not")
	}
}

func TestTheLinkerCountsEachOwedPhraseOnce(t *testing.T) {
	t.Parallel()

	for _, row := range measuredPages(t) {
		if row.Linker == nil || row.Linker.Call == nil {
			continue
		}
		if row.Linker.Phrases <= 0 || row.Linker.Phrases >= row.Linker.Calls {
			t.Errorf("%s %s owes %d phrases in %d refused calls, want each phrase tried more than once and counted once",
				row.Stand, row.Path, row.Linker.Phrases, row.Linker.Calls)
		}
	}
}

func hasSection(sections []measure, name string) bool {
	for _, section := range sections {
		if section.Name == name {
			return true
		}
	}
	return false
}
