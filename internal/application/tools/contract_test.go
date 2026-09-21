package tools_test

import (
	"regexp"
	"testing"

	"github.com/davidmovas/postulator/internal/application/llm"
)

const (
	maxToolNameLength        = 64
	maxToolDescriptionLength = 1024
	maxToolsPerRequest       = 128
)

var toolName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func walk(t *testing.T, name, path string, schema *llm.Schema, depth int) {
	t.Helper()

	if schema == nil {
		t.Errorf("%s: %s has no schema, which a provider reads as a missing type", name, path)
		return
	}
	if schema.Type == "" {
		t.Errorf("%s: %s declares no type", name, path)
	}
	if depth > 6 {
		t.Errorf("%s: %s nests deeper than a provider will read", name, path)
		return
	}
	if schema.Type == llm.SchemaArray && schema.Items == nil {
		t.Errorf("%s: %s is an array without items", name, path)
	}
	if len(schema.Enum) > 0 && schema.Type != llm.SchemaString {
		t.Errorf("%s: %s carries an enum on a %s", name, path, schema.Type)
	}
	for _, required := range schema.Required {
		if _, ok := schema.Properties[required]; !ok {
			t.Errorf("%s: %s requires %q, which it does not declare", name, path, required)
		}
	}
	for field, property := range schema.Properties {
		walk(t, name, path+"."+field, property, depth+1)
	}
	if schema.Items != nil {
		walk(t, name, path+"[]", schema.Items, depth+1)
	}
}

func TestEveryToolSatisfiesTheFunctionCallingContract(t *testing.T) {
	t.Parallel()

	registry, binding := wired(t)
	built := registry.Build(binding)

	if len(built) > maxToolsPerRequest {
		t.Errorf("the agent offers %d tools, more than the %d a request may carry", len(built), maxToolsPerRequest)
	}

	seen := make(map[string]bool, len(built))
	for i := range built {
		def := built[i].Def
		if seen[def.Name] {
			t.Errorf("%s is offered twice, and a provider refuses a repeated name", def.Name)
		}
		seen[def.Name] = true

		if !toolName.MatchString(def.Name) || len(def.Name) > maxToolNameLength {
			t.Errorf("%q is not a name a provider accepts", def.Name)
		}
		if def.Description == "" {
			t.Errorf("%s has no description", def.Name)
		}
		if len(def.Description) > maxToolDescriptionLength {
			t.Errorf("%s describes itself in %d characters, over the %d a provider reads",
				def.Name, len(def.Description), maxToolDescriptionLength)
		}
		if def.Schema == nil {
			continue
		}
		if def.Schema.Type != llm.SchemaObject {
			t.Errorf("%s takes a %s rather than an object", def.Name, def.Schema.Type)
		}
		walk(t, def.Name, "parameters", def.Schema, 0)
	}
}
