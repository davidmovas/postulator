package openai

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/content"
)

type StrictOrigin struct {
	Origin string `json:"origin"`
}

type strictPart struct {
	Heading string  `json:"heading"`
	Weight  float64 `json:"weight,omitempty"`
}

type strictProbe struct {
	At      time.Time    `json:"at"`
	Count   *int         `json:"count" minimum:"1" maximum:"9"`
	Owner   *strictPart  `json:"owner,omitempty"`
	Name    string       `json:"name" description:"the name"`
	Kind    string       `json:"kind" enum:"hub,topic"`
	Mood    string       `json:"mood,omitempty" enum:"calm,loud"`
	Note    string       `json:"note,omitempty"`
	Skipped string       `json:"-"`
	Tags    []string     `json:"tags,omitempty"`
	Parts   []strictPart `json:"parts"`
	Detail  strictPart   `json:"detail"`
	StrictOrigin
}

func schemaFor[T any](t *testing.T) *port.Schema {
	t.Helper()

	schema, err := port.SchemaFor[T]()
	if err != nil {
		t.Fatalf("derive the schema: %v", err)
	}
	return schema
}

func canonical(t *testing.T, value any) string {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var decoded any
	if err = json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	again, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		t.Fatalf("encode again: %v", err)
	}
	return string(again)
}

func canonicalText(t *testing.T, text string) string {
	t.Helper()

	var decoded any
	if err := json.Unmarshal([]byte(text), &decoded); err != nil {
		t.Fatalf("decode the expected json: %v", err)
	}
	return canonical(t, decoded)
}

func assertStrict(t *testing.T, node map[string]any, path string) {
	t.Helper()

	if branches, ok := node["anyOf"].([]any); ok {
		if len(branches) != 2 {
			t.Fatalf("%s: anyOf = %v, want the value and null", path, branches)
		}
		for i, branch := range branches {
			nested, isObject := branch.(map[string]any)
			if !isObject {
				t.Fatalf("%s: anyOf[%d] = %v, want a schema", path, i, branch)
			}
			assertStrict(t, nested, path+".anyOf")
		}
		return
	}

	if node["type"] == nil {
		t.Fatalf("%s: %v carries no type", path, node)
	}
	if items, ok := node["items"].(map[string]any); ok {
		assertStrict(t, items, path+"[]")
	}
	if node["type"] != "object" {
		return
	}

	if node["additionalProperties"] != false {
		t.Errorf("%s: additionalProperties = %v, want false", path, node["additionalProperties"])
	}
	properties, ok := node["properties"].(map[string]any)
	if !ok {
		t.Fatalf("%s: properties = %v, want an object", path, node["properties"])
	}
	required, ok := node["required"].([]string)
	if !ok {
		t.Fatalf("%s: required = %T, want a list of names", path, node["required"])
	}
	names := make([]string, 0, len(properties))
	for name, property := range properties {
		names = append(names, name)
		nested, isObject := property.(map[string]any)
		if !isObject {
			t.Fatalf("%s.%s = %v, want a schema", path, name, property)
		}
		assertStrict(t, nested, path+"."+name)
	}
	slices.Sort(names)
	if !reflect.DeepEqual(names, required) {
		t.Errorf("%s: required = %v, want every property %v", path, required, names)
	}
}

func TestEveryStructuredAnswerBecomesAStrictSchema(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		schema *port.Schema
	}{
		{name: "the writer's draft", schema: schemaFor[content.DraftAnswer](t)},
		{name: "the writer's product", schema: schemaFor[content.ProductAnswer](t)},
		{name: "the linker's repair", schema: schemaFor[content.RepairResponse](t)},
		{name: "a probe of every shape", schema: schemaFor[strictProbe](t)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			format, err := formatOf(tc.schema, "generate_body")
			if err != nil {
				t.Fatalf("formatOf: %v", err)
			}
			if !format.Strict || format.Type != "json_schema" {
				t.Errorf("format = %+v, want a strict json_schema", format)
			}
			assertStrict(t, format.Schema, "$")
		})
	}
}

func TestTheProductAnswerKeepsTheDraftItEmbeds(t *testing.T) {
	t.Parallel()

	strict := strictOf(schemaFor[content.ProductAnswer](t))
	properties, ok := strict["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties = %v, want an object", strict["properties"])
	}
	for _, name := range []string{"title", "h1", "sections", "summary", "shortDescription", "specifications"} {
		if _, present := properties[name]; !present {
			t.Errorf("the product schema lacks %s: %v", name, properties)
		}
	}
}

func TestAnOptionalPropertyBecomesNullable(t *testing.T) {
	t.Parallel()

	const want = `{
	  "type": "object",
	  "additionalProperties": false,
	  "required": ["at", "count", "detail", "kind", "mood", "name", "note", "origin", "owner", "parts", "tags"],
	  "properties": {
	    "at": {"type": "string", "description": "an RFC3339 timestamp"},
	    "count": {"type": ["integer", "null"], "minimum": 1, "maximum": 9},
	    "detail": {
	      "type": "object", "additionalProperties": false, "required": ["heading", "weight"],
	      "properties": {"heading": {"type": "string"}, "weight": {"type": ["number", "null"]}}
	    },
	    "kind": {"type": "string", "enum": ["hub", "topic"]},
	    "mood": {"type": ["string", "null"], "enum": ["calm", "loud", null]},
	    "name": {"type": "string", "description": "the name"},
	    "note": {"type": ["string", "null"]},
	    "origin": {"type": "string"},
	    "owner": {"anyOf": [
	      {
	        "type": "object", "additionalProperties": false, "required": ["heading", "weight"],
	        "properties": {"heading": {"type": "string"}, "weight": {"type": ["number", "null"]}}
	      },
	      {"type": "null"}
	    ]},
	    "parts": {"type": "array", "items": {
	      "type": "object", "additionalProperties": false, "required": ["heading", "weight"],
	      "properties": {"heading": {"type": "string"}, "weight": {"type": ["number", "null"]}}
	    }},
	    "tags": {"anyOf": [{"type": "array", "items": {"type": "string"}}, {"type": "null"}]}
	  }
	}`

	if got := canonical(t, strictOf(schemaFor[strictProbe](t))); got != canonicalText(t, want) {
		t.Errorf("strict schema =\n%s\nwant\n%s", got, canonicalText(t, want))
	}
}

func TestANullForAnOptionalFieldDecodesToItsZeroValue(t *testing.T) {
	t.Parallel()

	const answer = `{"at":"2026-10-03T12:00:00Z","count":null,"detail":{"heading":"h","weight":null},"kind":"hub",` +
		`"mood":null,"name":"Koffein","note":null,"origin":"sheet","owner":null,"parts":[{"heading":"p","weight":null}],"tags":null}`

	var decoded strictProbe
	if err := json.Unmarshal([]byte(answer), &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := strictProbe{
		At:           time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC),
		Name:         "Koffein",
		Kind:         "hub",
		Parts:        []strictPart{{Heading: "p"}},
		Detail:       strictPart{Heading: "h"},
		StrictOrigin: StrictOrigin{Origin: "sheet"},
	}
	if !reflect.DeepEqual(decoded, want) {
		t.Errorf("decoded = %+v, want %+v", decoded, want)
	}
}

func TestAnEmptyObjectIsStillClosed(t *testing.T) {
	t.Parallel()

	got := canonical(t, strictOf(schemaFor[struct{}](t)))
	if want := canonicalText(t, `{"type":"object","properties":{},"required":[],"additionalProperties":false}`); got != want {
		t.Errorf("strict schema = %s, want %s", got, want)
	}
}

func TestAStructuredAnswerMustBeAnObject(t *testing.T) {
	t.Parallel()

	if _, err := formatOf(schemaFor[[]string](t), "list"); err == nil {
		t.Fatal("an array answer was accepted, want it refused before it reaches the provider")
	}
}

func TestTheFormatIsNamedAfterTheStep(t *testing.T) {
	t.Parallel()

	cases := []struct {
		step string
		want string
	}{
		{step: "generate_body", want: "generate_body"},
		{step: "repair-links", want: "repair-links"},
		{step: "", want: "answer"},
		{step: "judge page", want: "judge_page"},
		{step: "générer", want: "g_n_rer"},
		{step: strings.Repeat("a", 80), want: strings.Repeat("a", 64)},
	}

	for _, tc := range cases {
		if got := formatName(tc.step); got != tc.want {
			t.Errorf("formatName(%q) = %q, want %q", tc.step, got, tc.want)
		}
	}
}

func TestToolParametersStayLoose(t *testing.T) {
	t.Parallel()

	minimum := 1.0
	schema := &port.Schema{
		Type: port.SchemaObject,
		Properties: map[string]*port.Schema{
			"limit":  {Type: port.SchemaInteger, Minimum: &minimum, Description: "how many"},
			"status": {Type: port.SchemaString, Enum: []string{"draft", "publish"}},
			"ids":    {Type: port.SchemaArray},
			"filter": {Type: port.SchemaObject},
			"loose":  nil,
		},
		Required: []string{"status"},
	}

	const want = `{
	  "type": "object",
	  "required": ["status"],
	  "properties": {
	    "filter": {"type": "object", "properties": {}},
	    "ids": {"type": "array", "items": {"type": "string"}},
	    "limit": {"type": "integer", "minimum": 1, "description": "how many"},
	    "loose": {"type": "string"},
	    "status": {"type": "string", "enum": ["draft", "publish"]}
	  }
	}`
	if got := canonical(t, parametersOf(schema)); got != canonicalText(t, want) {
		t.Errorf("parameters =\n%s\nwant\n%s", got, canonicalText(t, want))
	}

	if got := canonical(t, parametersOf(nil)); got != canonicalText(t, `{"type":"object","properties":{}}`) {
		t.Errorf("parameters of no schema = %s, want an empty object", got)
	}
}
