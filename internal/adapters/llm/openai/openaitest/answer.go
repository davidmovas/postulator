package openaitest

import (
	"encoding/json"
	"maps"
	"net/http"
	"strconv"
)

const (
	statusCompleted  = "completed"
	statusIncomplete = "incomplete"
	statusInProgress = "in_progress"

	reasonMaxOutput = "max_output_tokens"
	tierDefault     = "default"
	executionServer = "server"
	emptyObject     = "{}"
	emptyList       = "[]"
)

type Usage struct {
	Input      int
	Cached     int
	CacheWrite int
	Output     int
	Reasoning  int
}

type Call struct {
	ID        string
	Name      string
	Namespace string
	Arguments string
}

type Search struct {
	CallID    string
	Arguments string
	Tools     string
}

type Answer struct {
	Model      string
	Text       string
	Refusal    string
	Incomplete string
	Tier       string
	Searches   []Search
	Calls      []Call
	Chunks     []string
	Usage      Usage
}

func Text(text string) Answer {
	return Answer{Text: text}
}

func Structured(t TB, value any) Answer {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Errorf("encode the structured answer: %v", err)
		return Answer{}
	}
	return Answer{Text: string(encoded)}
}

func Calls(calls ...Call) Answer {
	return Answer{Calls: calls}
}

func Refusal(text string) Answer {
	return Answer{Refusal: text}
}

func Truncated(text string) Answer {
	return Answer{Text: text, Incomplete: reasonMaxOutput}
}

func (a Answer) Reply() Reply {
	return Reply{Status: http.StatusOK, Body: encode(a.response())}
}

func (a Answer) Stream() Reply {
	builder := a.opening()
	builder.list = append(builder.list, a.terminal(len(builder.list)))
	return Stream(builder.list...)
}

func (a Answer) StreamFailing(fault Fault) Reply {
	builder := a.opening()
	builder.fail(fault, a.model())
	return Stream(builder.list...)
}

func (a Answer) opening() *events {
	builder := &events{}
	pending := response(a.model(), statusInProgress, a.tier())
	builder.add("response.created", map[string]any{"response": pending})
	builder.add("response.in_progress", map[string]any{"response": pending})

	for index, item := range a.items() {
		a.streamItem(builder, index, item)
	}
	return builder
}

func (a Answer) terminal(sequence int) Event {
	name := "response.completed"
	if a.Incomplete != "" {
		name = "response.incomplete"
	}
	data := map[string]any{"type": name, "sequence_number": sequence, "response": a.response()}
	return Event{Name: name, Data: encode(data)}
}

func (a Answer) streamItem(builder *events, index int, item map[string]any) {
	switch item["type"] {
	case "message":
		a.streamMessage(builder, index, item)
	case "function_call":
		started := maps.Clone(item)
		started["arguments"] = ""
		started["status"] = statusInProgress
		builder.add("response.output_item.added", map[string]any{"output_index": index, "item": started})
		builder.add("response.function_call_arguments.delta", map[string]any{
			"output_index": index, "item_id": item["id"], "delta": item["arguments"],
		})
		builder.add("response.function_call_arguments.done", map[string]any{
			"output_index": index, "item_id": item["id"], "arguments": item["arguments"],
		})
		builder.add("response.output_item.done", map[string]any{"output_index": index, "item": item})
	default:
		builder.add("response.output_item.added", map[string]any{"output_index": index, "item": item})
		builder.add("response.output_item.done", map[string]any{"output_index": index, "item": item})
	}
}

func (a Answer) streamMessage(builder *events, index int, item map[string]any) {
	started := maps.Clone(item)
	started["content"] = []any{}
	started["status"] = statusInProgress
	builder.add("response.output_item.added", map[string]any{"output_index": index, "item": started})

	at := map[string]any{"output_index": index, "item_id": item["id"], "content_index": 0}
	if a.Refusal != "" {
		builder.add("response.content_part.added", with(at, "part", map[string]any{"type": "refusal", "refusal": ""}))
		builder.add("response.refusal.delta", with(at, "delta", a.Refusal))
		builder.add("response.refusal.done", with(at, "refusal", a.Refusal))
		builder.add("response.content_part.done", with(at, "part", map[string]any{"type": "refusal", "refusal": a.Refusal}))
	} else {
		builder.add("response.content_part.added", with(at, "part", textPart("")))
		for _, chunk := range a.chunks() {
			builder.add("response.output_text.delta", with(at, "delta", chunk))
		}
		builder.add("response.output_text.done", with(at, "text", a.Text))
		builder.add("response.content_part.done", with(at, "part", textPart(a.Text)))
	}
	builder.add("response.output_item.done", map[string]any{"output_index": index, "item": item})
}

func (a Answer) response() map[string]any {
	status := statusCompleted
	if a.Incomplete != "" {
		status = statusIncomplete
	}

	body := response(a.model(), status, a.tier())
	if a.Incomplete != "" {
		body["incomplete_details"] = map[string]any{"reason": a.Incomplete}
	}
	body["output"] = a.items()
	body["usage"] = map[string]any{
		"input_tokens": a.Usage.Input,
		"input_tokens_details": map[string]any{
			"cached_tokens": a.Usage.Cached, "cache_write_tokens": a.Usage.CacheWrite,
		},
		"output_tokens":         a.Usage.Output,
		"output_tokens_details": map[string]any{"reasoning_tokens": a.Usage.Reasoning},
		"total_tokens":          a.Usage.Input + a.Usage.Output,
	}
	return body
}

func (a Answer) items() []map[string]any {
	items := make([]map[string]any, 0, len(a.Calls)+2*len(a.Searches)+2)
	if a.Usage.Reasoning > 0 {
		items = append(items, map[string]any{"id": "rs_test", "type": "reasoning", "summary": []any{}})
	}
	for i, search := range a.Searches {
		items = append(items, search.items(i)...)
	}
	if a.Text != "" || a.Refusal != "" {
		items = append(items, a.message())
	}
	for i, call := range a.Calls {
		item := map[string]any{
			"id":        "fc_test_" + strconv.Itoa(i),
			"type":      "function_call",
			"status":    statusCompleted,
			"call_id":   call.ID,
			"name":      call.Name,
			"arguments": call.Arguments,
		}
		if call.Namespace != "" {
			item["namespace"] = call.Namespace
		}
		items = append(items, item)
	}
	return items
}

func (s Search) items(index int) []map[string]any {
	suffix := strconv.Itoa(index)
	return []map[string]any{
		{
			"id": "tsc_test_" + suffix, "type": "tool_search_call", "status": statusCompleted,
			"call_id": nullable(s.CallID), "execution": executionServer, "arguments": raw(s.Arguments, emptyObject),
		},
		{
			"id": "tso_test_" + suffix, "type": "tool_search_output", "status": statusCompleted,
			"call_id": nullable(s.CallID), "execution": executionServer, "tools": raw(s.Tools, emptyList),
		},
	}
}

func raw(value, fallback string) json.RawMessage {
	if value == "" {
		return json.RawMessage(fallback)
	}
	return json.RawMessage(value)
}

func (a Answer) message() map[string]any {
	part := textPart(a.Text)
	if a.Refusal != "" {
		part = map[string]any{"type": "refusal", "refusal": a.Refusal}
	}
	status := statusCompleted
	if a.Incomplete != "" {
		status = statusIncomplete
	}
	return map[string]any{
		"id":      "msg_test",
		"type":    "message",
		"status":  status,
		"role":    "assistant",
		"content": []any{part},
	}
}

func (a Answer) chunks() []string {
	if len(a.Chunks) > 0 {
		return a.Chunks
	}
	return []string{a.Text}
}

func (a Answer) model() string {
	if a.Model == "" {
		return DefaultModel
	}
	return a.Model
}

func (a Answer) tier() string {
	if a.Tier == "" {
		return tierDefault
	}
	return a.Tier
}

func textPart(text string) map[string]any {
	return map[string]any{"type": "output_text", "annotations": []any{}, "logprobs": []any{}, "text": text}
}

func with(base map[string]any, key string, value any) map[string]any {
	next := maps.Clone(base)
	next[key] = value
	return next
}
