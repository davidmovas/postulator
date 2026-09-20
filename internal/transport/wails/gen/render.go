package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var rawMessageType = reflect.TypeFor[json.RawMessage]()

func render(w io.Writer, entries []events.Entry) error {
	var out bytes.Buffer

	out.WriteString("export type EventType =\n")
	for index, entry := range entries {
		terminator := "\n"
		if index == len(entries)-1 {
			terminator = ";\n\n"
		}
		fmt.Fprintf(&out, "    | %q%s", string(entry.Type), terminator)
	}

	out.WriteString("export const eventTypes = [\n")
	for _, entry := range entries {
		fmt.Fprintf(&out, "    %q,\n", string(entry.Type))
	}
	out.WriteString("] as const;\n\n")

	for _, entry := range entries {
		block, err := renderInterface(entry)
		if err != nil {
			return err
		}
		out.WriteString(block)
	}

	out.WriteString("export interface EventPayloads {\n")
	for _, entry := range entries {
		fmt.Fprintf(&out, "    %q: %s;\n", string(entry.Type), reflect.TypeOf(entry.Payload).Name())
	}
	out.WriteString("}\n\n")

	out.WriteString("export interface Envelope<T extends EventType = EventType> {\n")
	out.WriteString("    type: T;\n")
	out.WriteString("    seq: number;\n")
	out.WriteString("    runId?: string;\n")
	out.WriteString("    at: string;\n")
	out.WriteString("    payload: EventPayloads[T];\n")
	out.WriteString("}\n")

	_, err := w.Write(out.Bytes())
	return err
}

func renderInterface(entry events.Entry) (string, error) {
	payload := reflect.TypeOf(entry.Payload)
	if payload == nil || payload.Kind() != reflect.Struct {
		return "", errors.New(errors.Invalid, "event payload must be a struct").
			WithDetail("type", string(entry.Type))
	}

	if payload.NumField() == 0 {
		return fmt.Sprintf("export interface %s {}\n\n", payload.Name()), nil
	}

	var block strings.Builder
	fmt.Fprintf(&block, "export interface %s {\n", payload.Name())
	for index := range payload.NumField() {
		field := payload.Field(index)
		name, ok := jsonName(field)
		if !ok {
			return "", errors.New(errors.Invalid, "event payload field needs a json tag").
				WithDetail("type", string(entry.Type)).
				WithDetail("field", field.Name)
		}
		rendered, err := tsType(field.Type)
		if err != nil {
			return "", errors.New(errors.Invalid, "event payload field has an unsupported type").
				WithDetail("type", string(entry.Type)).
				WithDetail("field", field.Name)
		}
		fmt.Fprintf(&block, "    %s: %s;\n", name, rendered)
	}
	block.WriteString("}\n\n")
	return block.String(), nil
}

func jsonName(field reflect.StructField) (string, bool) {
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return "", false
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" || name == "-" {
		return "", false
	}
	return name, true
}

func tsType(field reflect.Type) (string, error) {
	if field == rawMessageType {
		return "unknown", nil
	}

	switch field.Kind() {
	case reflect.String:
		return "string", nil
	case reflect.Int, reflect.Int64, reflect.Float64:
		return "number", nil
	case reflect.Bool:
		return "boolean", nil
	default:
		return "", errors.New(errors.Invalid, "unsupported field kind")
	}
}
