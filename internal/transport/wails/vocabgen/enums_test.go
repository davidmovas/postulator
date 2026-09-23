package main

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/tools"
)

var narrowed = map[string]bool{
	"templates_create.parameters.pageKind":   true,
	"templates_update.parameters.pageKind":   true,
	"templates_list.parameters.pageKind":     true,
	"graph_list_edges.parameters.sort.field": true,
	"imports_export.parameters.format":       true,
}

var choiceNames = []string{
	"kind", "source", "status", "scope", "wpType", "anchorStrategy", "role", "mode", "format",
	"reasoningEffort", "risk", "relation", "linkClass", "publishMode", "step", "field", "action",
	"reason", "origin",
}

var freeText = map[string]bool{
	"provider": true,
	"model":    true,
}

var prose = map[string]bool{
	"graph_add_edge.parameters.reason": true,
}

func vocabularies(t *testing.T) map[string]string {
	t.Helper()

	root := repoRoot(t)
	known := make(map[string]string, len(catalog()))
	for _, entry := range catalog() {
		if entry.pkg == "" {
			continue
		}
		values, err := constValues(filepath.Join(root, entry.pkg), entry.typeName)
		if err != nil {
			t.Fatalf("read %s.%s: %v", entry.pkg, entry.typeName, err)
		}
		known[settled(values)] = entry.export
	}
	return known
}

func settled(values []string) string {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	return strings.Join(sorted, ",")
}

func schemas(t *testing.T) map[string]*llm.Schema {
	t.Helper()
	return selected(t, func(tools.Risk) bool { return true })
}

func selected(t *testing.T, keep func(tools.Risk) bool) map[string]*llm.Schema {
	t.Helper()

	built := tools.New(tools.Deps{}).Build(tools.Binding{})
	out := make(map[string]*llm.Schema, len(built))
	for i := range built {
		if built[i].Def.Schema == nil {
			t.Fatalf("%s has no argument schema", built[i].Def.Name)
		}
		if keep(built[i].Def.Risk) {
			out[built[i].Def.Name] = built[i].Def.Schema
		}
	}
	return out
}

func eachProperty(schema *llm.Schema, path string, visit func(path, name string, property *llm.Schema)) {
	if schema == nil {
		return
	}
	for name, property := range schema.Properties {
		visit(path+"."+name, name, property)
		eachProperty(property, path+"."+name, visit)
	}
	if schema.Items != nil {
		eachProperty(schema.Items, path+"[]", visit)
	}
}

func TestEveryToolChoiceIsADomainVocabulary(t *testing.T) {
	t.Parallel()

	known := vocabularies(t)
	for name, schema := range schemas(t) {
		eachProperty(schema, name+".parameters", func(path, _ string, property *llm.Schema) {
			if len(property.Enum) == 0 {
				return
			}
			if _, ok := known[settled(property.Enum)]; ok {
				return
			}
			if narrowed[path] {
				return
			}
			t.Errorf("%s offers %v, which is no domain vocabulary; a partial list teaches the model a wrong value",
				path, property.Enum)
		})
	}
}

func TestEveryToolFieldThatNamesAChoiceOffersOne(t *testing.T) {
	t.Parallel()

	for name, schema := range schemas(t) {
		eachProperty(schema, name+".parameters", func(path, field string, property *llm.Schema) {
			if property.Type != llm.SchemaString || len(property.Enum) > 0 || freeText[field] || prose[path] {
				return
			}
			if !slices.Contains(choiceNames, field) {
				return
			}
			t.Errorf("%s is a bare string, so the model has to guess the value the domain will accept", path)
		})
	}
}

func TestEveryFieldOfEveryToolSaysWhatItIsFor(t *testing.T) {
	t.Parallel()

	for name, schema := range schemas(t) {
		eachProperty(schema, name+".parameters", func(path, _ string, property *llm.Schema) {
			if property.Description == "" {
				t.Errorf("%s reaches the model with no description", path)
			}
		})
	}
}
