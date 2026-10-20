package tools_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/field"
)

func rewritingType(t *testing.T, seeded fixture, name string) pagetypes.Type {
	t.Helper()

	created, err := seeded.types.Create(t.Context(), pagetypes.CreateRequest{SiteID: seeded.site, Name: name, Preset: "hub"})
	if err != nil {
		t.Fatalf("create the type %s: %v", name, err)
	}
	body := scopeNamed(created, field.Body)
	if _, err = seeded.types.SetScope(t.Context(), pagetypes.SetScopeRequest{
		PageTypeID: created.ID, Field: string(field.Body), Enabled: true, AIWrites: true, Rewrites: true, Links: body.Links,
	}); err != nil {
		t.Fatalf("let a person switch the body of %s to rewriting: %v", name, err)
	}
	return created
}

func scopeNamed(kind pagetypes.Type, key field.Key) pagetypes.Scope {
	for i := range kind.Scopes {
		if kind.Scopes[i].Field == string(key) {
			return kind.Scopes[i]
		}
	}
	return pagetypes.Scope{Field: string(key), Links: field.NoLinks()}
}

func waitsForAPerson(t *testing.T, out any) tools.Confirmation {
	t.Helper()

	asked, ok := out.(tools.Confirmation)
	if !ok || asked.Status != domainagent.ConfirmationRequired {
		t.Fatalf("the call answered %+v, want it held for a person to confirm", out)
	}
	return asked
}

func typeOfThePage(t *testing.T, seeded fixture) string {
	t.Helper()

	held, err := seeded.pages.Get(t.Context(), pages.GetRequest{ID: seeded.page})
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if held.Page.PageTypeID == nil {
		return ""
	}
	return *held.Page.PageTypeID
}

func everyMode() []domainagent.Mode {
	return []domainagent.Mode{domainagent.ModeConfirm, domainagent.ModeAutonomous}
}

func TestTypingAPageIntoARewritingTypeWaitsForAPerson(t *testing.T) {
	t.Parallel()

	for _, mode := range everyMode() {
		t.Run(string(mode), func(t *testing.T) {
			t.Parallel()

			registry, binding, seeded := wired(t)
			binding.Mode = mode
			rewriting := rewritingType(t, seeded, "Rewritten")
			before := typeOfThePage(t, seeded)
			args := json.RawMessage(`{"id":"` + seeded.page + `","pageTypeId":"` + rewriting.ID + `"}`)

			out, err := registry.Call(t.Context(), binding, "pages_update", args)
			if err != nil {
				t.Fatalf("pages_update: %v", err)
			}
			waitsForAPerson(t, out)
			if got := typeOfThePage(t, seeded); got != before {
				t.Fatalf("the page was typed %q before a person confirmed", got)
			}

			binding.Approved = true
			if _, err = registry.Call(t.Context(), binding, "pages_update", args); err != nil {
				t.Fatalf("the approved call: %v", err)
			}
			if got := typeOfThePage(t, seeded); got != rewriting.ID {
				t.Fatalf("the approved call left the page typed %q, want %q", got, rewriting.ID)
			}
		})
	}
}

func TestTypingAPageIntoAnOrdinaryTypeRunsInAutonomousMode(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	plain, err := seeded.types.Create(t.Context(), pagetypes.CreateRequest{SiteID: seeded.site, Name: "Plain", Preset: "hub"})
	if err != nil {
		t.Fatalf("create the type: %v", err)
	}

	out, err := registry.Call(t.Context(), binding, "pages_update",
		json.RawMessage(`{"id":"`+seeded.page+`","pageTypeId":"`+plain.ID+`"}`))
	if err != nil {
		t.Fatalf("pages_update: %v", err)
	}
	if _, asked := out.(tools.Confirmation); asked {
		t.Fatalf("a type that keeps the client's text asked for a person: %+v", out)
	}
	if got := typeOfThePage(t, seeded); got != plain.ID {
		t.Fatalf("the page is typed %q, want %q", got, plain.ID)
	}
}

func TestReassigningPagesIntoARewritingTypeWaitsForAPerson(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	rewriting := rewritingType(t, seeded, "Rewritten")
	leaving, err := seeded.types.Create(t.Context(), pagetypes.CreateRequest{SiteID: seeded.site, Name: "Leaving", Preset: "hub"})
	if err != nil {
		t.Fatalf("create the type: %v", err)
	}
	if _, err = seeded.pages.SetType(t.Context(), pages.SetTypeRequest{PageID: seeded.page, PageTypeID: leaving.ID}); err != nil {
		t.Fatalf("type the page: %v", err)
	}

	out, err := registry.Call(t.Context(), binding, "pagetypes_delete",
		json.RawMessage(`{"id":"`+leaving.ID+`","reassignTo":"`+rewriting.ID+`"}`))
	if err != nil {
		t.Fatalf("pagetypes_delete: %v", err)
	}
	waitsForAPerson(t, out)
	if got := typeOfThePage(t, seeded); got != leaving.ID {
		t.Fatalf("the page moved to %q before a person confirmed", got)
	}
}

func TestAnImportTypingPagesIntoARewritingTypeWaitsForAPerson(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	rewritingType(t, seeded, "Rewritten")
	sheet := filepath.Join(t.TempDir(), "pages.csv")
	if err := os.WriteFile(sheet, []byte("URL,Title,Page Type\n/coffee/,Coffee,Rewritten\n"), 0o600); err != nil {
		t.Fatalf("write the sheet: %v", err)
	}
	before := typeOfThePage(t, seeded)

	args, err := json.Marshal(map[string]string{"path": sheet})
	if err != nil {
		t.Fatalf("encode the arguments: %v", err)
	}
	out, err := registry.Call(t.Context(), binding, "imports_apply", args)
	if err != nil {
		t.Fatalf("imports_apply: %v", err)
	}
	waitsForAPerson(t, out)
	if got := typeOfThePage(t, seeded); got != before {
		t.Fatalf("the import typed the page %q before a person confirmed", got)
	}
}

func TestEnablingAFieldUnderRewritesWaitsForAPerson(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	rewriting := rewritingType(t, seeded, "Rewritten")
	if _, err := seeded.types.SetScope(t.Context(), pagetypes.SetScopeRequest{
		PageTypeID: rewriting.ID, Field: string(field.Excerpt), AIWrites: true, Rewrites: true, Links: field.NoLinks(),
	}); err != nil {
		t.Fatalf("let a person mark the excerpt as rewritten while it is off: %v", err)
	}

	out, err := registry.Call(t.Context(), binding, "pagetypes_set_scope",
		json.RawMessage(`{"pageTypeId":"`+rewriting.ID+`","field":"excerpt","enabled":true}`))
	if err != nil {
		t.Fatalf("pagetypes_set_scope: %v", err)
	}
	waitsForAPerson(t, out)
}

func TestTrashingAPageOnTheSiteWaitsForAPerson(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous

	out, err := registry.Call(t.Context(), binding, "pages_delete", json.RawMessage(`{"id":"`+seeded.page+`","onSite":true}`))
	if err != nil {
		t.Fatalf("pages_delete: %v", err)
	}
	asked := waitsForAPerson(t, out)
	if !strings.Contains(asked.Summary, "trash") {
		t.Errorf("the confirmation says %q, want it to say the page goes to the trash", asked.Summary)
	}
	if _, err = seeded.pages.Get(t.Context(), pages.GetRequest{ID: seeded.page}); err != nil {
		t.Fatalf("the page was deleted before a person confirmed: %v", err)
	}
}

func TestTheConfirmationSaysThePagesOwnTextWillBeRewritten(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	rewriting := rewritingType(t, seeded, "Rewritten")

	out, err := registry.Call(t.Context(), binding, "pages_update",
		json.RawMessage(`{"id":"`+seeded.page+`","pageTypeId":"`+rewriting.ID+`"}`))
	if err != nil {
		t.Fatalf("pages_update: %v", err)
	}
	asked := waitsForAPerson(t, out)
	for _, said := range []string{"/coffee/", "Rewritten", "own text will be rewritten"} {
		if !strings.Contains(asked.Summary, said) {
			t.Errorf("the confirmation says %q, want it to name %q", asked.Summary, said)
		}
	}
}
