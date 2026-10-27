package tools_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func liveAt(t *testing.T, seeded fixture, path string, wpType pagemap.WPType, wpID int64) string {
	t.Helper()

	record := pagemap.Page{
		ID: id.New(), SiteID: seeded.site, Path: path, Slug: pagemap.Slug(path), WPType: wpType, WPID: &wpID,
		ParentReason: pagemap.ParentNone, Title: path, H1: path, Keywords: keyword.Of(), Notes: []note.Note{},
		Status: pagemap.StatusPublished, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if err := sqlite.NewPageRepo(seeded.store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the page %s: %v", path, err)
	}
	return record.ID
}

func TestTheAgentPreviewsTheRulesFreelyAndAppliesThemOnlyAsThePersonConfirms(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	product := liveAt(t, seeded, "/mug/", pagemap.WPProduct, 41)
	liveAt(t, seeded, "/learn/", pagemap.WPPage, 42)

	preview := called[tools.PagetypesClassifyResponse](t, registry, binding, "pagetypes_classify", `{"op":"preview"}`)
	if preview.Hash == "" || preview.Moving != 1 || len(preview.Groups) != 1 {
		t.Fatalf("the preview = %+v, want the one product moving", preview)
	}
	group := preview.Groups[0]
	if group.Group != 1 || group.From != "untyped" || group.To != "Product" || group.Pages != 1 ||
		!slices.Equal(group.Samples, []string{"/mug/"}) || len(group.Rules) != 1 || group.Rules[0].Name != "Products" {
		t.Fatalf("the group = %+v, want the untyped product moving to Product by the products' rule", group)
	}
	if preview.Waiting != 1 {
		t.Errorf("waiting = %d, want the page whose render the listing rule needs", preview.Waiting)
	}

	_, err := registry.Call(t.Context(), binding, "pagetypes_classify", json.RawMessage(`{"op":"apply","hash":"stale"}`))
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("an apply over a stale hash = %v, want CONFLICT before anything is proposed", err)
	}

	_, err = registry.Call(t.Context(), binding, "pagetypes_classify", json.RawMessage(`{"op":"apply","hash":"`+preview.Hash+`","groups":[2]}`))
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("an apply naming a group the preview lacks = %v, want INVALID", err)
	}

	apply := `{"op":"apply","hash":"` + preview.Hash + `","groups":[1]}`
	out, err := registry.Call(t.Context(), binding, "pagetypes_classify", json.RawMessage(apply))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	asked := waitsForAPerson(t, out)
	if !strings.Contains(asked.Summary, "Type 1 pages by the site's rules: 1 untyped → Product") {
		t.Errorf("the summary = %q, want the count and the group in plain words", asked.Summary)
	}
	if held, getErr := seeded.pages.Get(t.Context(), pages.GetRequest{ID: product}); getErr != nil || held.Page.PageTypeID != nil {
		t.Fatalf("the product before the person confirmed = %+v %v, want it untyped", held.Page, getErr)
	}

	binding.Approved = true
	applied := called[tools.PagetypesClassifyResponse](t, registry, binding, "pagetypes_classify", apply)
	if applied.Applied == nil || applied.Applied.Moved != 1 {
		t.Fatalf("the confirmed apply = %+v, want the one product typed", applied)
	}
	held, err := seeded.pages.Get(t.Context(), pages.GetRequest{ID: product})
	if err != nil || held.Page.PageTypeID == nil || held.Page.TypeReason != string(pagemap.TypeByRule) || held.Page.TypeRuleID == "" {
		t.Fatalf("the product after the apply = %+v %v, want it typed by its rule", held.Page, err)
	}
}
