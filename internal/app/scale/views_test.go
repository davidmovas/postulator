//go:build !race

package scale_test

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func (s *scaled) read(t *testing.T, binding tools.Binding, tool, args string) []byte {
	t.Helper()

	out, err := s.core.Tools.Call(t.Context(), binding, tool, json.RawMessage(args))
	if err != nil {
		t.Fatalf("%s %s: %v", tool, args, err)
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("encode what %s answered: %v", tool, err)
	}
	return encoded
}

type listed struct {
	Items      []json.RawMessage `json:"items"`
	HasMore    bool              `json:"hasMore"`
	NextCursor string            `json:"nextCursor"`
}

func (s *scaled) walkAll(t *testing.T, binding tools.Binding, tool, args string, list func([]byte) listed, key func(json.RawMessage) string) map[string]int {
	t.Helper()

	ceiling := agent.MaxToolResultBytes(s.core.Settings)
	seen := map[string]int{}
	cursor := ""
	for calls := 0; ; calls++ {
		asked := args
		if cursor != "" {
			asked = args[:len(args)-1] + `,"cursor":"` + cursor + `"}`
		}
		encoded := s.read(t, binding, tool, asked)
		if len(encoded) > ceiling {
			t.Fatalf("%s %s answered %d bytes on call %d, over the %d-byte cap", tool, args, len(encoded), calls, ceiling)
		}
		page := list(encoded)
		for _, item := range page.Items {
			seen[key(item)]++
		}
		if !page.HasMore {
			return seen
		}
		if page.NextCursor == "" {
			t.Fatalf("%s %s says there is more on call %d and gives no cursor", tool, args, calls)
		}
		cursor = page.NextCursor
	}
}

func nodesAt(field string) func([]byte) listed {
	return func(encoded []byte) listed {
		var answer map[string]json.RawMessage
		var page listed
		if json.Unmarshal(encoded, &answer) != nil {
			return page
		}
		held := encoded
		if field != "" {
			held = answer[field]
		}
		if json.Unmarshal(held, &page) != nil {
			return listed{}
		}
		return page
	}
}

func byField(name string) func(json.RawMessage) string {
	return func(item json.RawMessage) string {
		var held map[string]json.RawMessage
		if json.Unmarshal(item, &held) != nil {
			return ""
		}
		value := held[name]
		if name == "page" {
			var ref contextpack.Ref
			if json.Unmarshal(value, &ref) == nil {
				return ref.PageID
			}
		}
		var text string
		if json.Unmarshal(value, &text) == nil {
			return text
		}
		return string(value)
	}
}

func TestEveryToolViewFitsTheCapAtScale(t *testing.T) {
	t.Parallel()

	s := generate(t, largeBlocks)
	binding := tools.Binding{SiteID: s.siteID, Mode: domainagent.ModeAutonomous}
	ceiling := agent.MaxToolResultBytes(s.core.Settings)
	hub, deep, product := s.id(t, "/c010/"), s.id(t, "/c010/s1/t1/"), s.id(t, "/product/item-013-1-4/")

	views := []struct{ tool, args string }{
		{"sites", `{"view":"list"}`},
		{"sites", `{"view":"get"}`},
		{"sites", `{"view":"overview"}`},
		{"pages_find", `{}`},
		{"pages_find", `{"limit":200}`},
		{"pages_find", `{"query":"article"}`},
		{"pages_find", `{"gap":"unmapped","limit":200}`},
		{"pages_find", `{"under":"` + hub + `","limit":200}`},
		{"pages_tree", `{"view":"roots","limit":200}`},
		{"pages_tree", `{"view":"children","pageId":"` + hub + `","limit":200}`},
		{"pages_tree", `{"view":"subtree","pageId":"` + hub + `","depth":32,"limit":200}`},
		{"pages_tree", `{"view":"ancestors","pageId":"` + deep + `"}`},
		{"pages_tree", `{"view":"siblings","pageId":"` + deep + `","limit":200}`},
		{"pages_tree", `{"view":"walk","limit":200}`},
		{"pages_get", `{"pageId":"` + hub + `"}`},
		{"pages_get", `{"pageId":"` + product + `"}`},
		{"pages_get", `{"view":"pack","pageId":"` + hub + `"}`},
		{"pages_blocks", `{"op":"list","pageId":"` + hub + `","text":true,"limit":200}`},
		{"pages_fields", `{"op":"get","pageId":"` + product + `","limit":200}`},
		{"links", `{"view":"page","pageId":"` + hub + `","limit":200}`},
		{"links", `{"view":"section","limit":200}`},
		{"links", `{"view":"map","limit":200}`},
		{"links", `{"view":"owed","limit":200}`},
		{"links", `{"view":"orphans","limit":200}`},
		{"links", `{"view":"suggest","pageId":"` + hub + `","limit":200}`},
		{"graph_read", `{"view":"entities","limit":200}`},
		{"graph_read", `{"view":"edges","limit":200}`},
		{"graph_read", `{"view":"adoption","limit":200}`},
		{"findings", `{"op":"list","limit":200}`},
		{"runs", `{"view":"list","limit":200}`},
		{"pagetypes", `{"op":"list"}`},
		{"pagetypes", `{"op":"get","pageTypeId":"` + s.typeOf(t, pagetype.Category) + `"}`},
		{"findings", `{"op":"list"}`},
		{"pagetypes_classify", `{"op":"rules"}`},
		{"templates", `{"op":"list","limit":200}`},
		{"models", `{"view":"list"}`},
		{"models", `{"view":"profiles"}`},
		{"schedules", `{"op":"list","limit":200}`},
		{"imports", `{"op":"mappings"}`},
	}
	for _, view := range views {
		started := time.Now()
		encoded := s.read(t, binding, view.tool, view.args)
		t.Logf("%s %s: %d bytes in %s", view.tool, view.args, len(encoded), time.Since(started).Round(time.Millisecond))
		if len(encoded) > ceiling {
			t.Errorf("%s %s answered %d bytes, over the %d-byte cap the assistant reads", view.tool, view.args, len(encoded), ceiling)
		}
	}

	rows, err := sqlite.NewPageMapRepo(s.core.Store).ListBySite(t.Context(), s.siteID)
	if err != nil {
		t.Fatalf("read the map: %v", err)
	}
	onMap := 0
	for i := range rows {
		if rows[i].Row {
			onMap++
		}
	}
	entities, err := sqlite.NewEntityRepo(s.core.Store).ListBySite(t.Context(), s.siteID)
	if err != nil {
		t.Fatalf("read the graph: %v", err)
	}
	walks := []struct {
		name, tool, args string
		list             func([]byte) listed
		key              func(json.RawMessage) string
		want             int
	}{
		{"the whole tree", "pages_tree", `{"view":"walk","limit":200}`, nodesAt("nodes"), byField("page"), onMap},
		{"every page found", "pages_find", `{"limit":200}`, nodesAt(""), byField("page"), onMap},
		{"every entity", "graph_read", `{"view":"entities","limit":200}`, nodesAt("entities"), byField("id"), len(entities)},
	}
	for _, walk := range walks {
		started := time.Now()
		seen := s.walkAll(t, binding, walk.tool, walk.args, walk.list, walk.key)
		t.Logf("%s walked in %s", walk.name, time.Since(started).Round(time.Millisecond))
		twice := 0
		for _, count := range seen {
			if count > 1 {
				twice++
			}
		}
		if len(seen) != walk.want || twice > 0 {
			t.Errorf("%s by cursor read %d distinct of %d with %d read twice", walk.name, len(seen), walk.want, twice)
		}
		t.Logf("%s: %d rows by cursor at %s pages, none missed and none twice", walk.name, len(seen), strconv.Itoa(len(s.plan.specs)+2*s.blocks))
	}

	var tree sitemodel.TreeAnswer
	if err = json.Unmarshal(s.read(t, binding, "pages_tree", `{"view":"walk","limit":200}`), &tree); err != nil || tree.Generation == 0 {
		t.Fatalf("the walk answered no generation: %+v, %v", tree.Generation, err)
	}

	opened, err := s.core.Agent.CreateConversation(t.Context(), agent.CreateConversationRequest{
		SiteID: s.siteID, Title: "Scale plan", Mode: string(domainagent.ModeConfirm),
	})
	if err != nil {
		t.Fatalf("open a conversation: %v", err)
	}
	binding.ConversationID = opened.Conversation.ID
	var adoption tools.GraphReadResponse
	if err = json.Unmarshal(s.read(t, binding, "graph_read", `{"view":"adoption"}`), &adoption); err != nil || adoption.Adoption == nil {
		t.Fatalf("the adoption preview: %+v, %v", adoption, err)
	}
	started := time.Now()
	steps := `[{"tool":"pages_manage","args":"{\"op\":\"set_types\",\"pageTypeId\":\"` + s.typeOf(t, pagetype.Guide) +
		`\",\"select\":{\"under\":[\"` + hub + `\"],\"ofType\":\"` + s.typeOf(t, pagetype.Info) + `\"}}","reason":"the hub's info pages are guides"},` +
		`{"tool":"graph_edit","args":"{\"op\":\"adopt_parents\",\"hash\":\"` + adoption.Adoption.Hash + `\"}","reason":"the graph takes the tree's parents"}]`
	proposed := s.read(t, binding, "plans", `{"op":"propose","title":"Retype a hub's info pages and adopt the tree","steps":`+steps+`}`)
	t.Logf("a plan of two steps over %d pages proposed in %s, %d bytes", len(s.plan.specs)+2*s.blocks,
		time.Since(started).Round(time.Millisecond), len(proposed))
	var plan tools.PlansResponse
	if err = json.Unmarshal(proposed, &plan); err != nil || plan.Plan == nil || len(plan.Plan.Steps) != 2 || len(proposed) > ceiling {
		t.Fatalf("the plan at scale = %s, %v, want two checked steps within the cap", proposed, err)
	}
	for _, step := range plan.Plan.Steps {
		if step.Summary == "" {
			t.Errorf("step %d says nothing of what it will do", step.Number)
		}
	}
}
