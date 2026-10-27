package tools_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func onTheSite(t *testing.T, seeded fixture) {
	t.Helper()

	repo := sqlite.NewPageRepo(seeded.store)
	page, err := repo.Get(t.Context(), seeded.page)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	wpID := int64(41)
	page.WPID, page.EntityID = &wpID, &seeded.entity
	if err = repo.Update(t.Context(), page); err != nil {
		t.Fatalf("put the page on the site: %v", err)
	}
}

func TestRunsStartTakesAFieldScopeAndShowsItsEstimateOnTheCard(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	onTheSite(t, seeded)
	binding.Mode = domainagent.ModeConfirm
	args := `{"op":"start","pageIds":["` + seeded.page + `"],"templateId":"` + seeded.template +
		`","fields":["seo_description"],"rewrite":["` + seeded.page + `"],"publishMode":"publish","brief":{"goal":"Sell the freshest beans"}}`

	out, err := registry.Call(t.Context(), binding, "runs_control", json.RawMessage(args))
	if err != nil {
		t.Fatalf("runs_control start: %v", err)
	}
	asked, ok := out.(tools.Confirmation)
	if !ok {
		t.Fatalf("runs_control start in confirm mode answered %+v, want a card", out)
	}
	for _, said := range []string{"/coffee/", "seo_description", "estimated", "own text", "Sell the freshest beans"} {
		if !strings.Contains(asked.Summary, said) {
			t.Errorf("the card says %q, want it to name %q", asked.Summary, said)
		}
	}

	binding.Mode = domainagent.ModeAutonomous
	started, startErr := registry.Call(t.Context(), binding, "runs_control", json.RawMessage(args))
	if _, waited := started.(tools.Confirmation); waited {
		t.Fatalf("an autonomous start answered %+v, want the start itself, as the mode says", started)
	}
	t.Logf("the autonomous start answered %v, the fixture's engine knowing no step", startErr)

	priced, err := registry.Call(t.Context(), binding, "runs_control", json.RawMessage(strings.Replace(args, `"op":"start"`, `"op":"estimate"`, 1)))
	if _, isEstimate := priced.(runs.EstimateResponse); err != nil || !isEstimate {
		t.Errorf("runs_control estimate = %+v, %v, want the estimate and no card", priced, err)
	}
}

func TestARunStartInAPlanIsPricedAndOffersItsBriefAndDirectivesForEditing(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	onTheSite(t, seeded)
	other := called[pages.CreateResponse](t, registry, tools.Binding{SiteID: binding.SiteID, Mode: domainagent.ModeAutonomous},
		"pages_manage", `{"op":"create","path":"/coffee/grinders/","entityId":"`+seeded.other+`"}`)
	args := `{"op":"start","select":{"pageIds":["` + seeded.page + `"]},"brief":{"goal":"Sell the freshest beans"},` +
		`"directives":[{"pageId":"` + seeded.page + `","notes":"Lead with storage","mustLink":[{"pageId":"` + other.Page.ID + `","anchor":"grinders"}]}],"addEdges":true}`

	check, err := registry.Validate(t.Context(), binding, "runs_control", json.RawMessage(args))
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	names := make([]string, 0, len(check.Editable))
	for _, field := range check.Editable {
		names = append(names, field.Name+"="+field.Path)
	}
	for _, want := range []string{"goal=brief.goal", "notes for /coffee/=directives.0.notes", "anchor from /coffee/ to /coffee/grinders/=directives.0.mustLink.0.anchor"} {
		if !strings.Contains(strings.Join(names, " "), want) {
			t.Errorf("the editable fields are %v, want %q among them", names, want)
		}
	}
	if !strings.Contains(check.Summary, "with a directive for 1 page") {
		t.Errorf("the step says %q, want its directive named", check.Summary)
	}

	editable(t, seeded)
	repo := sqlite.NewPageRepo(seeded.store)
	target, err := repo.Get(t.Context(), other.Page.ID)
	if err != nil {
		t.Fatalf("read the target: %v", err)
	}
	target.WPID = new(int64(42))
	if err = repo.Update(t.Context(), target); err != nil {
		t.Fatalf("put the target on the site: %v", err)
	}
	listed := called[pagetypes.ListResponse](t, registry, binding, "pagetypes", `{"op":"list"}`)
	checks := map[string]struct {
		tool, args, want string
		choices          int
	}{
		"a field's value": {"pages_fields", `{"op":"set","pageId":"` + seeded.page + `","key":"excerpt","value":"Fresh beans.","hash":"` +
			field.Hash("The owner's excerpt.") + `"}`, "value=value", 0},
		"a link's anchor": {"pages_blocks", `{"op":"place_link","pageId":"` + seeded.page + `","index":0,"hash":"` + block.BodyHash(coffeeBody) +
			`","targetId":"` + target.ID + `","anchor":"coffee"}`, "anchor=anchor", 0},
		"a type to choose": {"pages_manage", `{"op":"set_types","pageTypeId":"` + listed.Types[0].ID + `","select":{"pageIds":["` + seeded.page + `"]}}`,
			"type=pageTypeId", len(listed.Types)},
	}
	for name, tc := range checks {
		held, checkErr := registry.Validate(t.Context(), binding, tc.tool, json.RawMessage(tc.args))
		if checkErr != nil {
			t.Fatalf("%s: Validate: %v", name, checkErr)
		}
		if len(held.Editable) != 1 || held.Editable[0].Name+"="+held.Editable[0].Path != tc.want || len(held.Editable[0].Choices) != tc.choices {
			t.Errorf("%s: the editable fields are %+v, want %s with %d choices", name, held.Editable, tc.want, tc.choices)
		}
	}

	if _, err = registry.Call(t.Context(), tools.Binding{SiteID: binding.SiteID, ConversationID: binding.ConversationID, Mode: domainagent.ModeAutonomous},
		"runs_control", json.RawMessage(strings.Replace(args, `,"addEdges":true`, "", 1))); !errors.IsCode(err, errors.Invalid) ||
		!strings.Contains(err.Error(), "/coffee/grinders/") {
		t.Errorf("a must-link the graph does not connect = %v, want it refused naming the target", err)
	}
}
