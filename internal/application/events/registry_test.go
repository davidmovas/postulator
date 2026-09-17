package events_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

func declaredTypes(t *testing.T) map[events.Type]string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "type.go", nil, 0)
	if err != nil {
		t.Fatalf("parse type.go: %v", err)
	}

	declared := make(map[events.Type]string)
	for _, decl := range file.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.CONST {
			continue
		}
		for _, spec := range group.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			ident, ok := value.Type.(*ast.Ident)
			if !ok || ident.Name != "Type" {
				continue
			}
			for index, name := range value.Names {
				literal, ok := value.Values[index].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					t.Fatalf("%s is not declared as a string literal", name.Name)
				}
				unquoted, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatalf("unquote %s: %v", name.Name, err)
				}
				if previous, ok := declared[events.Type(unquoted)]; ok {
					t.Fatalf("%s and %s both declare the name %q", previous, name.Name, unquoted)
				}
				declared[events.Type(unquoted)] = name.Name
			}
		}
	}
	return declared
}

func TestTheRegistryHoldsExactlyTheDeclaredTypes(t *testing.T) {
	t.Parallel()

	declared := declaredTypes(t)
	if len(declared) == 0 {
		t.Fatal("no Type constants were found in type.go")
	}

	registered := make(map[events.Type]struct{})
	for _, entry := range events.NewRegistry().Entries() {
		if _, ok := registered[entry.Type]; ok {
			t.Fatalf("the registry holds %q twice", entry.Type)
		}
		registered[entry.Type] = struct{}{}
	}

	for eventType, name := range declared {
		if _, ok := registered[eventType]; !ok {
			t.Errorf("the constant %s declares %q, which the registry does not hold", name, eventType)
		}
	}
	for eventType := range registered {
		if _, ok := declared[eventType]; !ok {
			t.Errorf("the registry holds %q, which type.go does not declare", eventType)
		}
	}
}

func TestEveryRegisteredEntryHasADistinctStructPayload(t *testing.T) {
	t.Parallel()

	seen := make(map[reflect.Type]events.Type)
	for _, entry := range events.NewRegistry().Entries() {
		payload := reflect.TypeOf(entry.Payload)
		if payload == nil || payload.Kind() != reflect.Struct {
			t.Fatalf("%s carries a non-struct payload", entry.Type)
		}
		if other, ok := seen[payload]; ok {
			t.Fatalf("%s and %s share the payload type %s", entry.Type, other, payload)
		}
		seen[payload] = entry.Type
	}
}

func TestEveryPayloadFieldCarriesACamelCaseJSONTag(t *testing.T) {
	t.Parallel()

	camelCase := regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)
	for _, entry := range events.NewRegistry().Entries() {
		payload := reflect.TypeOf(entry.Payload)
		for index := range payload.NumField() {
			field := payload.Field(index)
			tag := field.Tag.Get("json")
			if !camelCase.MatchString(tag) {
				t.Errorf("%s.%s has the json tag %q, want camelCase", entry.Type, field.Name, tag)
			}
		}
	}
}

func TestLookupReportsRunEvents(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input events.Type
		found bool
		run   bool
	}{
		{name: "graph change is not a run event", input: events.GraphChanged, found: true},
		{name: "step retry is a run event", input: events.StepRetrying, found: true, run: true},
		{name: "unknown name", input: events.Type("nope")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			entry, ok := events.NewRegistry().Lookup(tc.input)
			if ok != tc.found {
				t.Fatalf("Lookup(%q) found = %v, want %v", tc.input, ok, tc.found)
			}
			if entry.Run != tc.run {
				t.Fatalf("Lookup(%q).Run = %v, want %v", tc.input, entry.Run, tc.run)
			}
		})
	}
}

func TestEnvelopeMarshalsTheAgreedShape(t *testing.T) {
	t.Parallel()

	runID := "6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90"
	at := dto.NewTime(time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC))

	cases := []struct {
		name  string
		input events.Envelope
		want  string
	}{
		{
			name:  "application event omits the run id",
			input: events.Envelope{Type: events.GraphChanged, Seq: 4, At: at, Payload: events.GraphChangedPayload{SiteID: "s1"}},
			want:  `{"type":"graph.changed","seq":4,"at":"2026-09-17T10:30:00Z","payload":{"siteId":"s1"}}`,
		},
		{
			name:  "run event carries the run id",
			input: events.Envelope{Type: events.RunStarted, Seq: 1, RunID: &runID, At: at, Payload: events.RunStartedPayload{RunID: runID}},
			want:  `{"type":"run.started","seq":1,"runId":"6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90","at":"2026-09-17T10:30:00Z","payload":{"runId":"6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90"}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("Marshal() error: %v", err)
			}
			if string(encoded) != tc.want {
				t.Fatalf("Marshal() = %s, want %s", encoded, tc.want)
			}
		})
	}
}
