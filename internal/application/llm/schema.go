package llm

import (
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type SchemaType string

const (
	SchemaObject  SchemaType = "object"
	SchemaArray   SchemaType = "array"
	SchemaString  SchemaType = "string"
	SchemaNumber  SchemaType = "number"
	SchemaInteger SchemaType = "integer"
	SchemaBoolean SchemaType = "boolean"
)

type Schema struct {
	Items       *Schema            `json:"items,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Type        SchemaType         `json:"type"`
	Title       string             `json:"title,omitempty"`
	Description string             `json:"description,omitempty"`
	Enum        []string           `json:"enum,omitempty"`
	Required    []string           `json:"required,omitempty"`
}

var timeType = reflect.TypeOf(time.Time{})

func SchemaFor[T any]() (*Schema, error) {
	return schemaOf(reflect.TypeFor[T](), nil)
}

func schemaOf(t reflect.Type, path []reflect.Type) (*Schema, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: SchemaString}, nil
	case reflect.Bool:
		return &Schema{Type: SchemaBoolean}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: SchemaInteger}, nil
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: SchemaNumber}, nil
	case reflect.Slice, reflect.Array:
		items, err := schemaOf(t.Elem(), path)
		if err != nil {
			return nil, err
		}
		return &Schema{Type: SchemaArray, Items: items}, nil
	case reflect.Struct:
		return structSchema(t, path)
	default:
		return nil, errors.New(errors.Invalid, "a structured response cannot be derived from this type").
			WithDetail("type", t.String()).
			WithDetail("kind", t.Kind().String())
	}
}

func structSchema(t reflect.Type, path []reflect.Type) (*Schema, error) {
	if t == timeType {
		return &Schema{Type: SchemaString, Description: "an RFC3339 timestamp"}, nil
	}
	if slices.Contains(path, t) {
		return nil, errors.New(errors.Invalid, "a structured response cannot be derived from a recursive type").
			WithDetail("type", t.String())
	}
	path = append(path, t)

	schema := &Schema{Type: SchemaObject, Title: t.Name(), Properties: map[string]*Schema{}, Required: []string{}}
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name, optional, skip := fieldName(field)
		if skip {
			continue
		}
		if field.Anonymous && field.Tag.Get("json") == "" {
			embedded, err := schemaOf(field.Type, path)
			if err != nil {
				return nil, err
			}
			if embedded.Type != SchemaObject {
				return nil, errors.New(errors.Invalid, "an embedded field must be a struct").WithDetail("type", field.Type.String())
			}
			for key, value := range embedded.Properties {
				schema.Properties[key] = value
			}
			schema.Required = append(schema.Required, embedded.Required...)
			continue
		}

		property, err := schemaOf(field.Type, path)
		if err != nil {
			return nil, err
		}
		if description := field.Tag.Get("description"); description != "" {
			property.Description = description
		}
		if values := field.Tag.Get("enum"); values != "" {
			property.Enum = strings.Split(values, ",")
		}

		schema.Properties[name] = property
		if !optional && field.Type.Kind() != reflect.Pointer {
			schema.Required = append(schema.Required, name)
		}
	}
	slices.Sort(schema.Required)
	return schema, nil
}

func fieldName(field reflect.StructField) (name string, optional, skip bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}

	name, rest, _ := strings.Cut(tag, ",")
	if name == "" {
		name = field.Name
	}
	return name, slices.Contains(strings.Split(rest, ","), "omitempty"), false
}
