package openai

import (
	"maps"
	"slices"
	"strings"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	formatJSONSchema   = "json_schema"
	maxFormatName      = 64
	fallbackFormatName = "answer"
	schemaNull         = "null"
)

func formatOf(schema *port.Schema, step string) (wireFormat, error) {
	if schema.Type != port.SchemaObject {
		return wireFormat{}, errors.New(errors.Invalid, "a structured answer must be a JSON object").
			WithDetail("type", string(schema.Type))
	}
	return wireFormat{Type: formatJSONSchema, Name: formatName(step), Schema: strictOf(schema), Strict: true}, nil
}

func formatName(step string) string {
	var builder strings.Builder
	for _, symbol := range step {
		if builder.Len() == maxFormatName {
			break
		}
		if nameSymbol(symbol) {
			builder.WriteRune(symbol)
			continue
		}
		builder.WriteByte('_')
	}
	if builder.Len() == 0 {
		return fallbackFormatName
	}
	return builder.String()
}

func nameSymbol(symbol rune) bool {
	switch {
	case symbol >= 'a' && symbol <= 'z', symbol >= 'A' && symbol <= 'Z', symbol >= '0' && symbol <= '9':
		return true
	default:
		return symbol == '_' || symbol == '-'
	}
}

func strictOf(schema *port.Schema) map[string]any {
	return shaped(schema, true, false)
}

func parametersOf(schema *port.Schema) map[string]any {
	if schema == nil {
		return map[string]any{"type": string(port.SchemaObject), "properties": map[string]any{}}
	}
	return shaped(schema, false, false)
}

func shaped(schema *port.Schema, strict, nullable bool) map[string]any {
	if schema == nil {
		schema = &port.Schema{Type: port.SchemaString}
	}

	node := map[string]any{"type": string(schema.Type)}
	if schema.Description != "" {
		node["description"] = schema.Description
	}
	if schema.Minimum != nil {
		node["minimum"] = *schema.Minimum
	}
	if schema.Maximum != nil {
		node["maximum"] = *schema.Maximum
	}
	if len(schema.Enum) > 0 {
		node["enum"] = slices.Clone(schema.Enum)
	}

	switch schema.Type {
	case port.SchemaObject:
		shapeObject(node, schema, strict)
	case port.SchemaArray:
		node["items"] = shaped(schema.Items, strict, false)
	}

	if nullable {
		return nullableOf(node, schema)
	}
	return node
}

func shapeObject(node map[string]any, schema *port.Schema, strict bool) {
	properties := make(map[string]any, len(schema.Properties))
	for name, child := range schema.Properties {
		optional := !slices.Contains(schema.Required, name)
		properties[name] = shaped(child, strict, strict && optional)
	}
	node["properties"] = properties

	if strict {
		required := slices.Sorted(maps.Keys(schema.Properties))
		if required == nil {
			required = []string{}
		}
		node["required"] = required
		node["additionalProperties"] = false
		return
	}
	if len(schema.Required) > 0 {
		node["required"] = slices.Clone(schema.Required)
	}
}

func nullableOf(node map[string]any, schema *port.Schema) map[string]any {
	if schema.Type == port.SchemaObject || schema.Type == port.SchemaArray {
		return map[string]any{"anyOf": []any{node, map[string]any{"type": schemaNull}}}
	}

	node["type"] = []string{string(schema.Type), schemaNull}
	if len(schema.Enum) > 0 {
		choices := make([]any, 0, len(schema.Enum)+1)
		for _, choice := range schema.Enum {
			choices = append(choices, choice)
		}
		node["enum"] = append(choices, nil)
	}
	return node
}
