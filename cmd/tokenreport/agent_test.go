package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/tiktoken-go/tokenizer"
)

func measuredAgent(t *testing.T) agentReport {
	t.Helper()

	counter, err := newCounter(tokenizer.O200kBase)
	if err != nil {
		t.Fatalf("newCounter: %v", err)
	}
	measured, err := measureAgent(t.Context(), counter)
	if err != nil {
		t.Fatalf("measureAgent: %v", err)
	}
	return measured
}

func TestTheAgentsRoundIsCountedPartByPart(t *testing.T) {
	t.Parallel()

	measured := measuredAgent(t)

	if len(measured.Instructions) != 2 || measured.Instructions[0].Name != "confirm" || measured.Instructions[1].Name != "autonomous" {
		t.Fatalf("instructions = %+v, want the confirm and the autonomous mode", measured.Instructions)
	}
	for _, row := range measured.Instructions {
		if row.Tokens <= 0 || row.Bytes <= row.Tokens {
			t.Errorf("the %s instructions count %d tokens in %d bytes", row.Name, row.Tokens, row.Bytes)
		}
	}

	if len(measured.Schemas) != 93 {
		t.Fatalf("%d schemas, want the registry's 93 tools", len(measured.Schemas))
	}
	if !slices.IsSortedFunc(measured.Schemas, func(a, b measure) int { return b.Tokens - a.Tokens }) {
		t.Error("the schemas are not sorted by tokens, the widest first")
	}
	bytes, tokens := 0, 0
	for _, row := range measured.Schemas {
		bytes += row.Bytes
		tokens += row.Tokens
	}
	if measured.AllSchemas.Bytes != bytes || measured.AllSchemas.Tokens != tokens {
		t.Errorf("all schemas = %+v, want the sum of the rows, %d bytes and %d tokens", measured.AllSchemas, bytes, tokens)
	}
	if measured.Deferred.Tokens <= 0 || measured.Deferred.Tokens >= measured.AllSchemas.Tokens {
		t.Errorf("the deferred round = %+v, want fewer tokens than every schema's %d", measured.Deferred, measured.AllSchemas.Tokens)
	}

	if len(measured.Stands) != 2 {
		t.Fatalf("%d stands, want the replica and vanilla", len(measured.Stands))
	}
	for _, stand := range measured.Stands {
		if stand.Brief.Tokens <= 0 || stand.Context.Tokens <= 0 {
			t.Errorf("the %s's brief or context is empty: %+v", stand.Stand, stand)
		}
		if stand.Instructions.Tokens <= stand.Brief.Tokens {
			t.Errorf("the %s's instructions with the brief count %d tokens, no more than the brief's %d", stand.Stand, stand.Instructions.Tokens, stand.Brief.Tokens)
		}
		if want := stand.Instructions.Tokens + measured.AllSchemas.Tokens + stand.Context.Tokens; stand.Round.Tokens != want {
			t.Errorf("the %s's round = %d tokens, want the instructions, every schema and the context, %d", stand.Stand, stand.Round.Tokens, want)
		}
		if want := stand.Instructions.Tokens + measured.Deferred.Tokens + stand.Context.Tokens; stand.DeferredRound.Tokens != want {
			t.Errorf("the %s's deferred round = %d tokens, want %d", stand.Stand, stand.DeferredRound.Tokens, want)
		}
	}
}

func TestTheReportPrintsTheAgentsRound(t *testing.T) {
	t.Parallel()

	var table strings.Builder
	if err := measuredAgent(t).write(&table); err != nil {
		t.Fatalf("write: %v", err)
	}
	for _, want := range []string{"THE AGENT, PER ROUND", "templates_create", "every schema", "deferred", "replica", "vanilla", "brief"} {
		if !strings.Contains(table.String(), want) {
			t.Errorf("the table lacks %q:\n%s", want, table.String())
		}
	}
}
