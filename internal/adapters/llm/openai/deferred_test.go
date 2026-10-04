package openai_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/openai/openaitest"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
)

const loadedPages = `[{"type":"namespace","name":"pages","description":"The page map.","tools":[{"type":"function",` +
	`"name":"pages_get","description":"Reads a page.","parameters":{"type":"object","properties":{"id":{"type":"string"}},` +
	`"required":["id"]},"strict":false,"defer_loading":true}]}]`

func searchedAnswer() openaitest.Answer {
	return openaitest.Answer{
		Searches: []openaitest.Search{{Arguments: `{"paths":["pages"]}`, Tools: loadedPages}},
		Calls:    []openaitest.Call{{ID: "call_1", Namespace: "pages", Name: "pages_get", Arguments: `{"id":"p1"}`}},
		Usage:    openaitest.Usage{Input: 900, Output: 40},
	}
}

func deferredTools() []port.Tool {
	byID := &port.Schema{
		Type: port.SchemaObject, Properties: map[string]*port.Schema{"id": {Type: port.SchemaString}}, Required: []string{"id"},
	}
	pages := &port.ToolGroup{Name: "pages", Description: "The page map."}
	runs := &port.ToolGroup{Name: "runs", Description: "Runs and their items."}
	return []port.Tool{
		{Name: "pages_list", Description: "Lists pages.", Schema: &port.Schema{Type: port.SchemaObject}},
		{Name: "pages_get", Description: "Reads a page.", Schema: byID, Deferred: pages},
		{Name: "runs_get", Description: "Reads a run.", Schema: byID, Deferred: runs},
		{Name: "pages_tree", Description: "Reads the tree.", Schema: &port.Schema{Type: port.SchemaObject}, Deferred: pages},
	}
}

func deferredChat(messages ...port.Message) port.Request {
	return port.Request{
		Ref: ref("gpt-5.6-terra"), System: "You are the assistant.", Messages: messages,
		Tools: deferredTools(), MaxTokens: 2048, Effort: llm.EffortNone, Tier: llm.TierDefault,
		CacheKey: "chat:conv-1", Meta: port.CallMeta{ConversationID: "conv-1", Role: llm.RoleChat},
	}
}

func opening() []port.Message {
	return []port.Message{
		{Role: port.RoleDeveloper, Text: "Site: Example."},
		{Role: port.RoleUser, Text: "Read the page p1."},
	}
}

func sameJSON(t *testing.T, got json.RawMessage, want string) bool {
	t.Helper()

	var left, right any
	if err := json.Unmarshal(got, &left); err != nil {
		t.Fatalf("decode %s: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &right); err != nil {
		t.Fatalf("decode %s: %v", want, err)
	}
	return reflect.DeepEqual(left, right)
}

func TestAStreamHandsBackTheToolSearchBeforeTheCallItLoaded(t *testing.T) {
	t.Parallel()

	whole := searchedAnswer().Stream()
	bare := openaitest.Stream(whole.Events[0], whole.Events[len(whole.Events)-1])
	cases := []struct {
		name  string
		reply openaitest.Reply
	}{
		{name: "each item announced when it is done", reply: whole},
		{name: "only the final response names the items", reply: bare},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(tc.reply)

			deltas, err := newClient(server).Stream(t.Context(), deferredChat(opening()...))
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}
			got := listen(t, deltas)
			if got.err != nil || got.done == nil {
				t.Fatalf("stream = %+v, want a finished answer", got)
			}
			if want := []string{"search call", "search output", "call call_1"}; !reflect.DeepEqual(got.order, want) {
				t.Fatalf("order = %v, want %v: each item once, in the order the model produced it", got.order, want)
			}
			search, loaded := got.searches[0], got.searches[1]
			if search.Execution != "server" || !sameJSON(t, search.Payload, `{"paths":["pages"]}`) {
				t.Errorf("the search call = %+v, want the server's search for pages", search)
			}
			if loaded.Execution != "server" || !sameJSON(t, loaded.Payload, loadedPages) {
				t.Errorf("the search output = %+v, want the tools it loaded", loaded)
			}
			if call := got.calls[0]; call.Namespace != "pages" || call.Name != "pages_get" || string(call.Args) != `{"id":"p1"}` {
				t.Errorf("the call = %+v, want pages_get inside the pages namespace", call)
			}
		})
	}
}

func TestACompletedAnswerCarriesItsToolSearchAndTheNamespace(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(searchedAnswer().Reply())

	resp, err := newClient(server).Complete(t.Context(), deferredChat(opening()...))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(resp.Searches) != 2 || resp.Searches[0].Kind != port.SearchCall || resp.Searches[1].Kind != port.SearchOutput {
		t.Fatalf("searches = %+v, want the call and its output", resp.Searches)
	}
	if len(resp.Calls) != 1 || resp.Calls[0].Namespace != "pages" {
		t.Errorf("calls = %+v, want the namespaced call", resp.Calls)
	}
}

func TestASearchItemWithoutAPayloadIsReadAsAnEmptyOne(t *testing.T) {
	t.Parallel()

	body := `{"status":"completed","service_tier":"default","output":[` +
		`{"type":"tool_search_call","id":"tsc_1","call_id":null,"execution":"server","status":"completed","arguments":null},` +
		`{"type":"tool_search_output","id":"tso_1","call_id":null,"execution":"server","status":"completed"}],` +
		`"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`
	server := openaitest.New(t)
	server.Enqueue(openaitest.Reply{Status: http.StatusOK, Body: body})

	resp, err := newClient(server).Complete(t.Context(), deferredChat(opening()...))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(resp.Searches) != 2 || string(resp.Searches[0].Payload) != `{}` || string(resp.Searches[1].Payload) != `[]` {
		t.Fatalf("searches = %+v, want an empty argument object and an empty tool list", resp.Searches)
	}
}

func TestADeferredRoundSendsNamespacesAndReplaysTheSearchOfTheRoundBefore(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(searchedAnswer().Stream(), openaitest.Text("Page p1 is a draft.").Stream())
	client := newClient(server)

	deltas, err := client.Stream(t.Context(), deferredChat(opening()...))
	if err != nil {
		t.Fatalf("first round: %v", err)
	}
	first := listen(t, deltas)
	if first.err != nil || len(first.calls) != 1 || len(first.searches) != 2 {
		t.Fatalf("first round = %+v, want the search and the call", first)
	}

	next := opening()
	for i := range first.searches {
		next = append(next, port.Message{Role: port.RoleAssistant, Search: &first.searches[i]})
	}
	next = append(next,
		port.Message{Role: port.RoleAssistant, Call: &first.calls[0]},
		port.Message{Role: port.RoleTool, Result: &port.ToolResult{CallID: "call_1", Output: json.RawMessage(`{"status":"draft"}`)}},
	)

	deltas, err = client.Stream(t.Context(), deferredChat(next...))
	if err != nil {
		t.Fatalf("second round: %v", err)
	}
	if second := listen(t, deltas); second.err != nil || second.text() != "Page p1 is a draft." {
		t.Fatalf("second round = %+v, want the answer the server gave the replayed round", second)
	}

	requests := server.Requests()
	if len(requests) != 2 {
		t.Fatalf("the server saw %d requests, want two rounds", len(requests))
	}
	assertGolden(t, "deferred_request.json", requests[1].Raw)
}
