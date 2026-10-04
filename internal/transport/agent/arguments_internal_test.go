package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func bound(value float64) *float64 {
	return &value
}

func anchorsSchema() *llm.Schema {
	return &llm.Schema{
		Type:     llm.SchemaObject,
		Required: []string{"entityId"},
		Properties: map[string]*llm.Schema{
			"entityId": {Type: llm.SchemaString},
			"kind":     {Type: llm.SchemaString, Enum: []string{"hub", "category"}},
			"weight":   {Type: llm.SchemaNumber, Minimum: bound(0), Maximum: bound(1)},
			"limit":    {Type: llm.SchemaInteger, Minimum: bound(1), Maximum: bound(200)},
			"strict":   {Type: llm.SchemaBoolean},
			"tags":     {Type: llm.SchemaArray, Items: &llm.Schema{Type: llm.SchemaString, Enum: []string{"a", "b"}}},
			"loose":    {Type: llm.SchemaArray},
			"anchors": {
				Type: llm.SchemaArray,
				Items: &llm.Schema{
					Type:     llm.SchemaObject,
					Required: []string{"text"},
					Properties: map[string]*llm.Schema{
						"text":   {Type: llm.SchemaString},
						"source": {Type: llm.SchemaString},
					},
				},
			},
		},
	}
}

func TestTheArgumentsAreCheckedAgainstTheToolSchema(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		args  string
		reads []string
	}{
		{name: "arguments that fit", args: `{"entityId":"e1","kind":"hub","weight":0.5,"limit":3,"strict":true,"tags":["a"],"loose":[1,"x"],"anchors":[{"text":"espresso"}]}`},
		{name: "an optional field left null", args: `{"entityId":"e1","kind":null}`},
		{name: "an integer written with a fraction of nothing", args: `{"entityId":"e1","limit":3.0}`},
		{name: "a required field missing", args: `{"kind":"hub"}`, reads: []string{"entityId: required parameter missing"}},
		{name: "no arguments at all", args: `null`, reads: []string{"entityId: required parameter missing"}},
		{name: "a value outside the enum", args: `{"entityId":"e1","kind":"beverage"}`, reads: []string{"kind: value not in enum; it takes one of hub, category"}},
		{name: "a number over the maximum", args: `{"entityId":"e1","weight":5}`, reads: []string{"weight: number too large; the maximum is 1"}},
		{name: "a number under the minimum", args: `{"entityId":"e1","weight":-0.5}`, reads: []string{"weight: number too small; the minimum is 0"}},
		{name: "an integer with a fraction", args: `{"entityId":"e1","limit":2.5}`, reads: []string{"limit: expected an integer"}},
		{name: "an integer over the maximum", args: `{"entityId":"e1","limit":500}`, reads: []string{"limit: integer too large; the maximum is 200"}},
		{name: "a string where a number goes", args: `{"entityId":"e1","weight":"high"}`, reads: []string{"weight: expected a number"}},
		{name: "a number where a string goes", args: `{"entityId":7}`, reads: []string{"entityId: expected a string"}},
		{name: "a word where a flag goes", args: `{"entityId":"e1","strict":"yes"}`, reads: []string{"strict: expected true or false"}},
		{name: "an object where a list goes", args: `{"entityId":"e1","tags":{"a":1}}`, reads: []string{"tags: expected a list"}},
		{name: "a list item outside the enum", args: `{"entityId":"e1","tags":["a","z"]}`, reads: []string{"tags[1]: value not in enum"}},
		{
			name: "a required field missing inside a list", args: `{"entityId":"e1","anchors":[{"text":"a"},{"source":"user"}]}`,
			reads: []string{"anchors[1].text: required parameter missing"},
		},
		{name: "a list item that is not an object", args: `{"entityId":"e1","anchors":["espresso"]}`, reads: []string{"anchors[0]: expected an object"}},
		{name: "a list where the arguments go", args: `[1,2]`, reads: []string{"the arguments: expected an object"}},
		{
			name: "every problem is named at once", args: `{"kind":"beverage","weight":9}`,
			reads: []string{"entityId: required parameter missing", "kind: value not in enum", "weight: number too large"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := check("graph_set_anchors", anchorsSchema(), json.RawMessage(tc.args))
			if len(tc.reads) == 0 {
				if err != nil {
					t.Fatalf("check = %v, want the arguments accepted", err)
				}
				return
			}
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("check = %v, want INVALID", err)
			}
			for _, read := range tc.reads {
				if !strings.Contains(err.Error(), read) {
					t.Fatalf("the refusal reads %q, which never says %q", err.Error(), read)
				}
			}
			if !strings.Contains(err.Error(), "graph_set_anchors") {
				t.Fatalf("the refusal does not name its tool: %q", err.Error())
			}
		})
	}
}

func TestArgumentsThatAreNotJSONAreRefused(t *testing.T) {
	t.Parallel()

	if err := check("pages_tree", anchorsSchema(), json.RawMessage(`{"entityId":`)); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("check = %v, want INVALID", err)
	}
	if err := check("pages_tree", nil, json.RawMessage(`{"anything":1}`)); err != nil {
		t.Fatalf("a tool without a schema refused %v", err)
	}
}
