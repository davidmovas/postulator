package template_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func canonical(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	return value
}

func TestMergePatchRFC7396Vectors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		target string
		patch  string
		want   string
	}{
		{`{"a":"b"}`, `{"a":"c"}`, `{"a":"c"}`},
		{`{"a":"b"}`, `{"b":"c"}`, `{"a":"b","b":"c"}`},
		{`{"a":"b"}`, `{"a":null}`, `{}`},
		{`{"a":"b","b":"c"}`, `{"a":null}`, `{"b":"c"}`},
		{`{"a":["b"]}`, `{"a":"c"}`, `{"a":"c"}`},
		{`{"a":"c"}`, `{"a":["b"]}`, `{"a":["b"]}`},
		{`{"a":{"b":"c"}}`, `{"a":{"b":"d","c":null}}`, `{"a":{"b":"d"}}`},
		{`{"a":[{"b":"c"}]}`, `{"a":[1]}`, `{"a":[1]}`},
		{`["a","b"]`, `["c","d"]`, `["c","d"]`},
		{`{"a":"b"}`, `["c"]`, `["c"]`},
		{`{"a":"foo"}`, `null`, `null`},
		{`{"a":"foo"}`, `"bar"`, `"bar"`},
		{`{"e":null}`, `{"a":1}`, `{"e":null,"a":1}`},
		{`[1,2]`, `{"a":"b","c":null}`, `{"a":"b"}`},
		{`{}`, `{"a":{"bb":{"ccc":null}}}`, `{"a":{"bb":{}}}`},
	}

	for _, tc := range cases {
		t.Run(tc.target+" + "+tc.patch, func(t *testing.T) {
			t.Parallel()

			got, err := template.MergePatch(json.RawMessage(tc.target), json.RawMessage(tc.patch))
			if err != nil {
				t.Fatalf("MergePatch: %v", err)
			}
			if !reflect.DeepEqual(canonical(t, got), canonical(t, json.RawMessage(tc.want))) {
				t.Errorf("MergePatch = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestMergePatchEdges(t *testing.T) {
	t.Parallel()

	same, err := template.MergePatch(json.RawMessage(`{"a":1}`), nil)
	if err != nil || string(same) != `{"a":1}` {
		t.Fatalf("empty patch must return the target: %s, %v", same, err)
	}
	if _, err = template.MergePatch(json.RawMessage(`{"a":1}`), json.RawMessage(`{`)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("broken patch code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = template.MergePatch(json.RawMessage(`{`), json.RawMessage(`{"a":1}`)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("broken target code = %q, want INVALID", errors.CodeOf(err))
	}
}

func TestResolveLayersOverrides(t *testing.T) {
	t.Parallel()

	base := validSpec()
	site := json.RawMessage(`{"linkRules":{"maxLinks":5},"tone":"warm"}`)
	page := json.RawMessage(`{"sections":[{"heading":"Only","intent":"one","targetWords":100,"required":true,"keywordRules":{"include":["x"],"primaryInHeading":false}}],"images":{"inline":0},"modelProfiles":{"writer":null,"editor":{"provider":"anthropic","model":"m"}}}`)

	resolved, err := template.Resolve(base, site, page)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Tone != "warm" {
		t.Errorf("tone = %q, want the site override", resolved.Tone)
	}
	if resolved.LinkRules.MaxLinks != 5 || resolved.LinkRules.UpDepth != 2 {
		t.Errorf("link rules = %+v, want maxLinks patched and upDepth kept", resolved.LinkRules)
	}
	if len(resolved.Sections) != 1 || resolved.Sections[0].Heading != "Only" || len(resolved.Sections[0].KeywordRules.Include) != 1 {
		t.Errorf("sections = %+v, want the page override to replace the list wholesale", resolved.Sections)
	}
	if !resolved.Images.Featured || resolved.Images.Inline != 0 || resolved.Images.Source != template.ImagesAI {
		t.Errorf("images = %+v, want inline patched and the rest kept", resolved.Images)
	}
	if _, writer := resolved.ModelProfiles["writer"]; writer || resolved.ModelProfiles["editor"].Provider != "anthropic" {
		t.Errorf("model profiles = %+v, want writer removed and editor added", resolved.ModelProfiles)
	}
	if len(resolved.Recipe) != 2 {
		t.Errorf("recipe = %+v, want untouched", resolved.Recipe)
	}

	untouched, err := template.Resolve(base, nil, nil)
	if err != nil || !reflect.DeepEqual(untouched, base) {
		t.Errorf("Resolve without overrides = %+v, %v; want the base", untouched, err)
	}
}

func TestResolveRejectsBrokenResults(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		site  string
		page  string
		field string
	}{
		{name: "invalid after merge", site: `{"length":{"min":9000}}`, field: "length.max"},
		{name: "sections removed", page: `{"sections":null}`, field: "sections"},
		{name: "unknown key", site: `{"tonee":"x"}`},
		{name: "wrong type", page: `{"length":{"min":"ten"}}`},
		{name: "broken json", site: `{`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := template.Resolve(validSpec(), json.RawMessage(tc.site), json.RawMessage(tc.page))
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID (err %v)", errors.CodeOf(err), err)
			}
			if tc.field != "" {
				if got := fieldOf(t, err); got != tc.field {
					t.Errorf("field = %q, want %q", got, tc.field)
				}
			}
		})
	}
}
