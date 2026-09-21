package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func repoRoot(t *testing.T) string {
	t.Helper()

	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}

	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		t.Fatal("the generator test must run inside the module")
	}
	return filepath.Dir(gomod)
}

func TestGeneratedFileIsInSync(t *testing.T) {
	t.Parallel()

	generated := filepath.Join(t.TempDir(), "events.ts")
	if err := write(generated); err != nil {
		t.Fatalf("write: %v", err)
	}

	fresh, err := os.ReadFile(generated)
	if err != nil {
		t.Fatalf("read the generated file: %v", err)
	}

	committed, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(generatedFile)))
	if err != nil {
		t.Fatalf("read the committed file: %v", err)
	}

	if !bytes.Equal(fresh, committed) {
		t.Fatalf("%s is out of date; run `task events`", generatedFile)
	}
}

func TestRenderCarriesTheEventNameTuple(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := render(&out, events.NewRegistry().Entries()); err != nil {
		t.Fatalf("render: %v", err)
	}

	rendered := out.String()
	if !strings.Contains(rendered, "export const eventTypes = [\n") {
		t.Fatal("the module carries no eventTypes tuple")
	}
	if !strings.Contains(rendered, "] as const;") {
		t.Fatal("the eventTypes tuple is not readonly")
	}
	for _, entry := range events.NewRegistry().Entries() {
		if !strings.Contains(rendered, "    \""+string(entry.Type)+"\",\n") {
			t.Errorf("the eventTypes tuple is missing %q", string(entry.Type))
		}
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	t.Parallel()

	var first, second bytes.Buffer
	if err := render(&first, events.NewRegistry().Entries()); err != nil {
		t.Fatalf("render: %v", err)
	}
	if err := render(&second, events.NewRegistry().Entries()); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("two renders of the same registry differ")
	}
}

type everyKindPayload struct {
	Name     string          `json:"name"`
	Count    int             `json:"count"`
	Duration int64           `json:"duration"`
	Amount   float64         `json:"amount"`
	Enabled  bool            `json:"enabled"`
	Paths    []string        `json:"paths"`
	Args     json.RawMessage `json:"args"`
}

func TestRenderMapsEverySupportedKind(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := render(&out, []events.Entry{{Type: events.Type("x.kinds"), Payload: everyKindPayload{}}}); err != nil {
		t.Fatalf("render: %v", err)
	}

	const want = `export interface everyKindPayload {
    name: string;
    count: number;
    duration: number;
    amount: number;
    enabled: boolean;
    paths: string[];
    args: unknown;
}`
	if !strings.Contains(out.String(), want) {
		t.Fatalf("render produced %s, want it to contain %s", out.String(), want)
	}
}

func TestRenderRejectsPayloadsItCannotType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		entry events.Entry
	}{
		{
			name: "field kind without a TypeScript mapping",
			entry: events.Entry{Type: events.Type("x.map"), Payload: struct {
				Labels map[string]string `json:"labels"`
			}{}},
		},
		{
			name: "field without a json tag",
			entry: events.Entry{Type: events.Type("x.untagged"), Payload: struct {
				Name string
			}{}},
		},
		{
			name:  "payload that is not a struct",
			entry: events.Entry{Type: events.Type("x.scalar"), Payload: "nope"},
		},
		{
			name: "field excluded from json",
			entry: events.Entry{Type: events.Type("x.excluded"), Payload: struct {
				Name string `json:"-"`
			}{}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := render(&bytes.Buffer{}, []events.Entry{tc.entry})
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("render error = %v, want an INVALID kernel error", err)
			}
		})
	}
}
