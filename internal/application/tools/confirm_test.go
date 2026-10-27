package tools_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
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
			args := json.RawMessage(`{"op":"update","pageId":"` + seeded.page + `","pageTypeId":"` + rewriting.ID + `"}`)

			out, err := registry.Call(t.Context(), binding, "pages_manage", args)
			if err != nil {
				t.Fatalf("pages_manage update: %v", err)
			}
			waitsForAPerson(t, out)
			if got := typeOfThePage(t, seeded); got != before {
				t.Fatalf("the page was typed %q before a person confirmed", got)
			}

			binding.Approved = true
			if _, err = registry.Call(t.Context(), binding, "pages_manage", args); err != nil {
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

	out, err := registry.Call(t.Context(), binding, "pages_manage",
		json.RawMessage(`{"op":"update","pageId":"`+seeded.page+`","pageTypeId":"`+plain.ID+`"}`))
	if err != nil {
		t.Fatalf("pages_manage update: %v", err)
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

	out, err := registry.Call(t.Context(), binding, "pagetypes",
		json.RawMessage(`{"op":"delete","pageTypeId":"`+leaving.ID+`","reassignTo":"`+rewriting.ID+`"}`))
	if err != nil {
		t.Fatalf("pagetypes delete: %v", err)
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

	args, err := json.Marshal(map[string]string{"op": "apply", "path": sheet})
	if err != nil {
		t.Fatalf("encode the arguments: %v", err)
	}
	out, err := registry.Call(t.Context(), binding, "imports", args)
	if err != nil {
		t.Fatalf("imports apply: %v", err)
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

	out, err := registry.Call(t.Context(), binding, "pagetypes",
		json.RawMessage(`{"op":"scope","pageTypeId":"`+rewriting.ID+`","field":"excerpt","enabled":true}`))
	if err != nil {
		t.Fatalf("pagetypes scope: %v", err)
	}
	waitsForAPerson(t, out)
}

func TestTrashingAPageOnTheSiteWaitsForAPerson(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous

	out, err := registry.Call(t.Context(), binding, "pages_manage", json.RawMessage(`{"op":"delete","pageId":"`+seeded.page+`","onSite":true}`))
	if err != nil {
		t.Fatalf("pages_manage delete: %v", err)
	}
	asked := waitsForAPerson(t, out)
	if !strings.Contains(asked.Summary, "trash") {
		t.Errorf("the confirmation says %q, want it to say the page goes to the trash", asked.Summary)
	}
	if _, err = seeded.pages.Get(t.Context(), pages.GetRequest{ID: seeded.page}); err != nil {
		t.Fatalf("the page was deleted before a person confirmed: %v", err)
	}
}

func TestAnEntityProposalWaitsForAPersonInEitherMode(t *testing.T) {
	t.Parallel()

	for _, mode := range everyMode() {
		t.Run(string(mode), func(t *testing.T) {
			t.Parallel()

			registry, binding, seeded := wired(t)
			binding.Mode = mode
			apply := json.RawMessage(`{"op":"apply_proposals","proposalsOf":"site","proposals":[{"pageId":"` + seeded.page + `","path":"/coffee/","name":"Coffee Beans","parentEntityId":"` + seeded.entity + `"}]}`)
			out, err := registry.Call(t.Context(), binding, "graph_edit", apply)
			if err != nil {
				t.Fatalf("graph_edit apply_proposals: %v", err)
			}
			asked := waitsForAPerson(t, out)
			for _, said := range []string{"Coffee Beans", "/coffee/", "approved"} {
				if !strings.Contains(asked.Summary, said) {
					t.Errorf("the confirmation says %q, want it to name %q", asked.Summary, said)
				}
			}
			held, err := seeded.pages.Get(t.Context(), pages.GetRequest{ID: seeded.page})
			if err != nil || held.Page.EntityID != nil {
				t.Fatalf("the page was mapped before a person confirmed: %+v %v", held.Page.EntityID, err)
			}

			proposed, err := registry.Call(t.Context(), binding, "graph_propose", json.RawMessage(`{"op":"pages","pathPrefix":"/coffee/"}`))
			if err != nil {
				t.Fatalf("graph_propose pages: %v", err)
			}
			if asked := waitsForAPerson(t, proposed); !strings.Contains(asked.Summary, "/coffee/") {
				t.Errorf("the confirmation says %q, want it to name the pages it proposes for", asked.Summary)
			}
		})
	}
}

func TestResolvingAFindingWithAProposalReadsAndWritesNothing(t *testing.T) {
	t.Parallel()

	registry, binding, _ := wired(t)
	for _, tool := range registry.Build(binding) {
		if tool.Def.Name != "findings" {
			continue
		}
		if risk := tool.RiskOf(json.RawMessage(`{"op":"resolve","findingId":"f","option":"propose_entity"}`)); risk != tools.RiskRead {
			t.Errorf("a proposal option is %s, want read: it only answers the preview to show", risk)
		}
		if risk := tool.RiskOf(json.RawMessage(`{"op":"resolve","findingId":"f","option":"ignore"}`)); risk != tools.RiskWrite {
			t.Errorf("a model change is %s, want write", risk)
		}
		return
	}
	t.Fatal("no findings tool")
}

func TestTheConfirmationSaysThePagesOwnTextWillBeRewritten(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	rewriting := rewritingType(t, seeded, "Rewritten")

	out, err := registry.Call(t.Context(), binding, "pages_manage",
		json.RawMessage(`{"op":"update","pageId":"`+seeded.page+`","pageTypeId":"`+rewriting.ID+`"}`))
	if err != nil {
		t.Fatalf("pages_manage update: %v", err)
	}
	asked := waitsForAPerson(t, out)
	for _, said := range []string{"/coffee/", "Rewritten", "own text will be rewritten"} {
		if !strings.Contains(asked.Summary, said) {
			t.Errorf("the confirmation says %q, want it to name %q", asked.Summary, said)
		}
	}

	many, err := registry.Call(t.Context(), binding, "pages_manage",
		json.RawMessage(`{"op":"set_types","pageTypeId":"`+rewriting.ID+`","select":{"pageIds":["`+seeded.page+`"]}}`))
	if err != nil {
		t.Fatalf("pages_manage set_types: %v", err)
	}
	asked = waitsForAPerson(t, many)
	for _, said := range []string{"1 page (/coffee/)", "Rewritten", "own text will be rewritten"} {
		if !strings.Contains(asked.Summary, said) {
			t.Errorf("the bulk confirmation says %q, want it to name %q", asked.Summary, said)
		}
	}
}

func TestEveryPageChangeIsSaidInOneSentenceTheCardAndThePlanShow(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeConfirm
	cases := map[string]struct {
		args string
		want []string
	}{
		"create":        {`{"op":"create","path":"/coffee/grinders/"}`, []string{"Plan the page /coffee/grinders/"}},
		"update":        {`{"op":"update","pageId":"` + seeded.page + `","title":"Coffee","h1":"Coffee"}`, []string{"title, H1 of /coffee/"}},
		"delete":        {`{"op":"delete","pageId":"` + seeded.page + `"}`, []string{"Delete /coffee/ from the page map"}},
		"map":           {`{"op":"map","pageId":"` + seeded.page + `","entityId":"` + seeded.entity + `"}`, []string{"Map /coffee/ to the entity Coffee"}},
		"unmap":         {`{"op":"unmap","pageId":"` + seeded.page + `"}`, []string{"Detach /coffee/"}},
		"canonical":     {`{"op":"canonical","pageId":"` + seeded.page + `","entityId":"` + seeded.entity + `"}`, []string{"canonical page of Coffee"}},
		"preview_link":  {`{"op":"preview_link","pageId":"` + seeded.page + `"}`, []string{"one-hour link", "/coffee/"}},
		"set_templates": {`{"op":"set_templates","templateId":"` + seeded.template + `","select":{"pageIds":["` + seeded.page + `"]}}`, []string{"1 page (/coffee/)", "Guide"}},
		"back to rules": {`{"op":"set_types","reset":true,"select":{"pageIds":["` + seeded.page + `"]}}`, []string{"back to the classification rules"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			out, err := registry.Call(t.Context(), binding, "pages_manage", json.RawMessage(tc.args))
			if err != nil {
				t.Fatalf("pages_manage %s: %v", name, err)
			}
			asked := waitsForAPerson(t, out)
			for _, said := range tc.want {
				if !strings.Contains(asked.Summary, said) {
					t.Errorf("the card says %q, want %q", asked.Summary, said)
				}
			}
		})
	}

	other := sqlitetest.Site(t, seeded.store, "elsewhere")
	foreign := sqlitetest.Page(t, seeded.store, other.ID, "/elsewhere/")
	if _, err := registry.Call(t.Context(), binding, "pages_manage", json.RawMessage(`{"op":"unmap","pageId":"`+foreign.ID+`"}`)); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("a page of another site = %v, want NotFound", err)
	}
	if _, err := registry.Call(t.Context(), binding, "pages_manage", json.RawMessage(`{"op":"set_types","reset":true}`)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a bulk change naming no page = %v, want Invalid", err)
	}
}
