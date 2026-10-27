package tools_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestTheGraphReadsAnswerEveryView(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeConfirm
	added := called[graph.AddEdgeResponse](t, registry, tools.Binding{SiteID: binding.SiteID, Mode: domainagent.ModeAutonomous},
		"graph_edit", `{"op":"add_edge","entityId":"`+seeded.entity+`","toEntityId":"`+seeded.other+`","kind":"related","weight":0.5}`)

	listed := called[tools.GraphReadResponse](t, registry, binding, "graph_read", `{"view":"entities"}`)
	if listed.Entities == nil || len(listed.Entities.Items) != 2 {
		t.Errorf("graph_read entities = %+v, want both seeded entities", listed.Entities)
	}
	detail := called[tools.GraphReadResponse](t, registry, binding, "graph_read", `{"view":"entity","entityId":"`+seeded.entity+`"}`)
	if detail.Entity == nil || detail.Entity.Name != "Coffee" || len(detail.Entity.Related) != 1 || detail.Entity.ParentWhy == "" {
		t.Errorf("graph_read entity = %+v, want Coffee with its related edge and why it has no parent", detail.Entity)
	}
	edges := called[tools.GraphReadResponse](t, registry, binding, "graph_read", `{"view":"edges","kind":"related"}`)
	if edges.Edges == nil || len(edges.Edges.Items) != 1 || edges.Edges.Items[0].ID != added.Edge.ID {
		t.Errorf("graph_read edges = %+v, want the one related edge", edges.Edges)
	}
	adoption := called[tools.GraphReadResponse](t, registry, binding, "graph_read", `{"view":"adoption"}`)
	if adoption.Adoption == nil || adoption.Adoption.Hash == "" {
		t.Errorf("graph_read adoption = %+v, want a preview with its hash", adoption.Adoption)
	}
	if _, err := registry.Call(t.Context(), binding, "graph_read", json.RawMessage(`{"view":"everything"}`)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown view = %v, want Invalid", err)
	}
}

func TestEveryGraphChangeIsSaidInOneSentence(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	added := called[graph.AddEdgeResponse](t, registry, tools.Binding{SiteID: binding.SiteID, Mode: domainagent.ModeAutonomous},
		"graph_edit", `{"op":"add_edge","entityId":"`+seeded.entity+`","toEntityId":"`+seeded.other+`","kind":"related","status":"proposed"}`)
	binding.Mode = domainagent.ModeConfirm
	cases := map[string]struct {
		tool string
		args string
		want string
	}{
		"create":        {"graph_edit", `{"op":"create","name":"Grinders"}`, "Add the entity Grinders"},
		"a tree":        {"graph_edit", `{"op":"create","entities":[{"name":"Grinders"},{"name":"Burr Grinders","parentName":"Grinders"}]}`, "Add 2 entities to the graph: Grinders, Burr Grinders"},
		"update":        {"graph_edit", `{"op":"update","entityId":"` + seeded.entity + `","intent":"Buy beans"}`, "Change the intent of the entity Coffee"},
		"delete":        {"graph_edit", `{"op":"delete","entityId":"` + seeded.entity + `"}`, "Delete the entity Coffee and the edges that touch it"},
		"anchors":       {"graph_edit", `{"op":"anchors","entityId":"` + seeded.entity + `","anchors":[{"text":"fresh beans"}]}`, "Let pages link to Coffee with “fresh beans”"},
		"a parent":      {"graph_edit", `{"op":"add_edge","entityId":"` + seeded.other + `","toEntityId":"` + seeded.entity + `","kind":"parent","weight":1}`, "Put Espresso under Coffee"},
		"approve":       {"graph_edit", `{"op":"approve_edge","edgeId":"` + added.Edge.ID + `"}`, "Approve the related edge between"},
		"reject":        {"graph_edit", `{"op":"reject_edge","edgeId":"` + added.Edge.ID + `"}`, "Reject the related edge between"},
		"delete edge":   {"graph_edit", `{"op":"delete_edge","edgeId":"` + added.Edge.ID + `"}`, "Delete the related edge between"},
		"move":          {"graph_edit", `{"op":"move","entityId":"` + seeded.other + `","parentId":"` + seeded.entity + `"}`, "Move Espresso under Coffee"},
		"second":        {"graph_edit", `{"op":"move","entityId":"` + seeded.other + `","parentId":"` + seeded.entity + `","keepBoth":true}`, "Give Espresso a second parent, Coffee"},
		"rescore":       {"graph_edit", `{"op":"rescore"}`, "Recompute the authority score"},
		"related":       {"graph_propose", `{"op":"related","entityId":"` + seeded.entity + `"}`, "related edges around Coffee"},
		"cards":         {"graph_propose", `{"op":"cards","cards":[{"entityId":"` + seeded.entity + `","definition":"Roasted beans."}]}`, "Propose 1 entity cards"},
		"from pages":    {"graph_propose", `{"op":"pages","pathPrefix":"/coffee/"}`, "the pages without an entity under /coffee/"},
		"adopt parents": {"graph_edit", `{"op":"adopt_parents","hash":"` + adoptionHash(t, registry, binding) + `"}`, "the parent the page tree shows"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			out, err := registry.Call(t.Context(), binding, tc.tool, json.RawMessage(tc.args))
			if err != nil {
				t.Fatalf("%s %s: %v", tc.tool, name, err)
			}
			if asked := waitsForAPerson(t, out); !strings.Contains(asked.Summary, tc.want) {
				t.Errorf("the card says %q, want %q", asked.Summary, tc.want)
			}
		})
	}
	if _, err := registry.Call(t.Context(), binding, "graph_edit", json.RawMessage(`{"op":"adopt_parents"}`)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an adoption without the preview's hash = %v, want Invalid", err)
	}
	if _, err := registry.Call(t.Context(), binding, "graph_edit", json.RawMessage(`{"op":"adopt_parents","hash":"stale"}`)); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("an adoption under a stale hash = %v, want Conflict", err)
	}
}

func adoptionHash(t *testing.T, registry *tools.Registry, binding tools.Binding) string {
	t.Helper()

	preview := called[tools.GraphReadResponse](t, registry, binding, "graph_read", `{"view":"adoption"}`)
	if preview.Adoption == nil {
		t.Fatal("graph_read adoption answered no preview")
	}
	return preview.Adoption.Hash
}
