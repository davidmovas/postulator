package tools_test

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/llm"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func bounded(schema *llm.Schema, start float64) float64 {
	value := start
	if schema.Minimum != nil && value < *schema.Minimum {
		value = *schema.Minimum
	}
	if schema.Maximum != nil && value > *schema.Maximum {
		value = *schema.Maximum
	}
	return value
}

func (f fixture) idFor(tool, field string) (string, bool) {
	switch field {
	case "siteId":
		return f.site, true
	case "entityId", "fromEntityId", "parentEntityId":
		return f.entity, true
	case "toEntityId", "newParentId":
		return f.other, true
	case "pageId":
		return f.page, true
	case "templateId":
		return f.template, true
	case "cursor":
		return "", true
	case "id":
		switch {
		case strings.HasPrefix(tool, "sites_"):
			return f.site, true
		case strings.HasPrefix(tool, "pages_"):
			return f.page, true
		case strings.HasPrefix(tool, "templates_"):
			return f.template, true
		case strings.HasPrefix(tool, "graph_") && strings.Contains(tool, "entity"):
			return f.entity, true
		}
	}
	return "", false
}

type draft struct {
	tool     string
	seeded   fixture
	distinct string
}

func (d draft) of(schema *llm.Schema, everything bool, field string) any {
	switch schema.Type {
	case llm.SchemaObject:
		out := map[string]any{}
		for name, property := range schema.Properties {
			if !everything && !slices.Contains(schema.Required, name) {
				continue
			}
			out[name] = d.of(property, everything, name)
		}
		return out
	case llm.SchemaArray:
		if schema.Items == nil {
			return []any{}
		}
		return []any{d.of(schema.Items, everything, field)}
	case llm.SchemaString:
		if len(schema.Enum) > 0 {
			return schema.Enum[0]
		}
		if known, ok := d.seeded.idFor(d.tool, field); ok {
			return known
		}
		return "x" + d.distinct
	case llm.SchemaInteger:
		return int(bounded(schema, 1))
	case llm.SchemaNumber:
		return bounded(schema, 1)
	default:
		return false
	}
}

func encoded(t *testing.T, value any) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode the arguments: %v", err)
	}
	return raw
}

func callOf(t *testing.T, tool string, schema *llm.Schema, everything bool, seeded fixture, distinct string) map[string]any {
	t.Helper()

	built, ok := draft{tool: tool, seeded: seeded, distinct: distinct}.of(schema, everything, "").(map[string]any)
	if !ok {
		t.Fatalf("the schema %+v is not an object", schema)
	}
	return built
}

func TestACallCarryingOnlyTheRequiredFieldsIsAccepted(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous

	for _, tool := range registry.Build(binding) {
		t.Run(tool.Def.Name, func(t *testing.T) {
			t.Parallel()

			args := encoded(t, callOf(t, tool.Def.Name, tool.Def.Schema, false, seeded, "1"))
			if err := tool.Check(args); err != nil {
				t.Fatalf("the smallest call %s refuses is %s: %v", tool.Def.Name, args, err)
			}
		})
	}
}

func TestACallCarryingEveryFieldIsAccepted(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous

	for _, tool := range registry.Build(binding) {
		t.Run(tool.Def.Name, func(t *testing.T) {
			t.Parallel()

			args := encoded(t, callOf(t, tool.Def.Name, tool.Def.Schema, true, seeded, "1"))
			if err := tool.Check(args); err != nil {
				t.Fatalf("the fullest call %s refuses is %s: %v", tool.Def.Name, args, err)
			}
		})
	}
}

func TestARefusedCallNamesTheFieldTheModelGotWrong(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		tool string
		args string
		want []string
	}{
		{
			name: "a field the tool does not take",
			tool: "graph_create_entity",
			args: `{"name":"Creatine","kind":"topic","shade":"blue"}`,
			want: []string{"graph_create_entity", "shade", "only the fields its schema declares"},
		},
		{
			name: "a nested field the tool does not take",
			tool: "graph_create_entity",
			args: `{"name":"Creatine","kind":"topic","anchors":[{"text":"c","hue":"blue"}]}`,
			want: []string{"graph_create_entity", "hue"},
		},
		{
			name: "a field of the wrong shape",
			tool: "pages_create",
			args: `{"path":"/coffee/","title":42}`,
			want: []string{"pages_create", "title", "number"},
		},
	}

	registry, binding, _ := wired(t)
	binding.Mode = domainagent.ModeAutonomous

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := registry.Call(t.Context(), binding, tc.tool, json.RawMessage(tc.args))
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("the call answered %v, want an invalid argument", err)
			}
			for _, wanted := range tc.want {
				if !strings.Contains(err.Error(), wanted) {
					t.Errorf("the refusal reads %q and never says %q", err.Error(), wanted)
				}
			}
		})
	}
}

const itemToken = "[]"

func requiredPaths(schema *llm.Schema, prefix []string) [][]string {
	out := make([][]string, 0)
	switch schema.Type {
	case llm.SchemaObject:
		for _, name := range schema.Required {
			out = append(out, append(slices.Clone(prefix), name))
		}
		for name, property := range schema.Properties {
			out = append(out, requiredPaths(property, append(slices.Clone(prefix), name))...)
		}
	case llm.SchemaArray:
		if schema.Items != nil {
			out = append(out, requiredPaths(schema.Items, append(slices.Clone(prefix), itemToken))...)
		}
	default:
	}
	return out
}

func drop(value any, tokens []string) {
	switch {
	case len(tokens) == 0:
		return
	case tokens[0] == itemToken:
		items, ok := value.([]any)
		if !ok {
			return
		}
		for _, item := range items {
			drop(item, tokens[1:])
		}
	default:
		held, ok := value.(map[string]any)
		if !ok {
			return
		}
		if len(tokens) == 1 {
			delete(held, tokens[0])
			return
		}
		drop(held[tokens[0]], tokens[1:])
	}
}

func TestEveryRequiredFieldIsOneTheUseCaseRefusesToDoWithout(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous

	for _, tool := range registry.Build(binding) {
		wanted := requiredPaths(tool.Def.Schema, nil)
		if len(wanted) == 0 {
			continue
		}

		t.Run(tool.Def.Name, func(t *testing.T) {
			for at, tokens := range wanted {
				without := callOf(t, tool.Def.Name, tool.Def.Schema, true, seeded, strconv.Itoa(at))
				drop(without, tokens)

				_, err := registry.Call(t.Context(), binding, tool.Def.Name, encoded(t, without))
				switch {
				case err == nil:
					t.Errorf("%s ran without %s, so the model is refused a call the use case would accept",
						tool.Def.Name, strings.Join(tokens, "."))
				case errors.IsCode(err, errors.Internal):
					t.Errorf("%s answered an internal error without %s rather than refusing it: %v",
						tool.Def.Name, strings.Join(tokens, "."), err)
				}
			}
		})
	}
}
